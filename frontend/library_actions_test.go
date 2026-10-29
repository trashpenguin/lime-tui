package main

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/k1y0miiii/applemusic-tui/engine"
)

func libraryActionModel(w, h int) model {
	m := mouseModel(w, h, 1)
	m.libraryOpen, m.sTab, m.libraryCategory = true, 2, 1
	m.sRes.Albums = []engine.Track{{ID: "0123456789012345678901", Kind: "album", Title: "Selected album"}}
	return m
}

func TestLibraryButtonsClickAndBlockDuplicateRequests(t *testing.T) {
	for _, label := range []string{"[Enter Play All]", "[a Add to Queue]", "[A Next]"} {
		m := libraryActionModel(100, 34)
		x, y := -1, -1
		for row, line := range strings.Split(m.View(), "\n") {
			plain := ansi.Strip(line)
			if at := strings.Index(plain, label); at >= 0 {
				x, y = lipgloss.Width(plain[:at])+1, row
				break
			}
		}
		if x < 0 {
			t.Fatalf("missing button %s", label)
		}
		next, cmd := m.Update(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
		got := next.(model)
		if cmd == nil || !got.libraryActionBusy || !got.libraryOpen {
			t.Fatalf("button %s did not begin action in context", label)
		}
		if strings.Contains(got.libraryActionStatus, "Added") {
			t.Fatal("reported success before acknowledgement")
		}
		_, duplicate := got.updateLibraryView(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		if duplicate != nil {
			t.Fatal("duplicate queue mutation while pending")
		}
	}
}

func TestLibraryFailureStaysVisibleAndActionCanBeRetried(t *testing.T) {
	m := libraryActionModel(100, 34)
	m.eng = nil
	next, cmd := m.updateLibraryView(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = next.(model)
	next, _ = m.Update(cmd())
	m = next.(model)
	if m.libraryActionBusy || !m.libraryActionFailed || !m.libraryOpen || !strings.Contains(m.View(), "Could not add to queue") {
		t.Fatal("queue failure disappeared or falsely succeeded")
	}
	_, retry := m.updateLibraryView(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if retry == nil {
		t.Fatal("failed action cannot be retried")
	}
}

func TestLibraryQueueSuccessRequiresAcknowledgementAndIgnoresStaleResults(t *testing.T) {
	m := libraryActionModel(100, 34)
	next, _ := m.updateLibraryView(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = next.(model)
	next, _ = m.Update(libraryActionMsg{token: m.libraryActionToken - 1, action: "queue", title: "old item", err: errors.New("old error")})
	m = next.(model)
	if !m.libraryActionBusy {
		t.Fatal("stale completion changed current operation")
	}
	next, _ = m.Update(libraryActionMsg{token: m.libraryActionToken, action: "queue", title: "Selected album"})
	m = next.(model)
	if m.libraryActionBusy || m.libraryActionFailed || m.libraryActionStatus != "Added to queue: Selected album" {
		t.Fatal("acknowledged queue success missing")
	}
}

func TestLibraryActionsAndErrorsFitSupportedTerminalSizes(t *testing.T) {
	for _, size := range [][2]int{{70, 22}, {100, 34}, {140, 45}} {
		for _, state := range []string{"ready", "loading", "error", "action error"} {
			m := libraryActionModel(size[0], size[1])
			switch state {
			case "loading":
				m.sBusy = true
			case "error":
				m.libraryErr = strings.Repeat("Network unavailable ", 20)
			case "action error":
				m.libraryActionFailed = true
				m.libraryActionStatus = strings.Repeat("Could not queue · ", 20)
			}
			view := m.View()
			if lipgloss.Width(view) > m.w || lipgloss.Height(view) > m.h {
				t.Fatalf("%s overflows %dx%d: %dx%d", state, m.w, m.h, lipgloss.Width(view), lipgloss.Height(view))
			}
		}
	}
}

func TestLibraryLoadErrorCanBeRetriedWithoutPlayingStaleItems(t *testing.T) {
	m := libraryActionModel(100, 34)
	m.libraryErr = "Network unavailable"
	_, play := m.updateLibraryView(tea.KeyMsg{Type: tea.KeyEnter})
	if play != nil {
		t.Fatal("load error allowed stale item playback")
	}
	next, retry := m.updateLibraryView(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if retry == nil || !next.(model).sBusy || next.(model).libraryErr != "" {
		t.Fatal("reload did not enter loading state")
	}
}

func TestLibraryTitleCannotMasqueradeAsActionButton(t *testing.T) {
	m := libraryActionModel(100, 34)
	m.sRes.Albums[0].Title = "[a Add to Queue]"
	box, ok := renderedBrowserBox(m.View())
	if !ok {
		t.Fatal("missing library panel")
	}
	next, cmd := m.Update(tea.MouseMsg{X: box.x + 5, Y: box.y + 2, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if cmd == nil || !strings.HasPrefix(next.(model).libraryActionStatus, "Loading playback:") {
		t.Fatal("album metadata was treated as a queue control")
	}
}
