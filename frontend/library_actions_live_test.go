package main

import (
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/k1y0miiii/applemusic-tui/engine"
)

func TestNativeLiveLibraryPlayAllAndQueue(t *testing.T) {
	if os.Getenv("NATIVE_LIVE_TEST") != "1" {
		t.Skip("requires signed-in native playback and isolated queue")
	}
	eng, err := engine.Connect(func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()
	defer eng.Pause()
	if err := eng.SetVolume(15); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		kind, id string
		tab      int
	}{
		{"album", os.Getenv("NATIVE_LIVE_ALBUM_ID"), 2},
		{"playlist", os.Getenv("NATIVE_LIVE_PLAYLIST_ID"), 3},
	} {
		if len(item.id) != 22 {
			t.Fatal("requires album and playlist fixture IDs")
		}
		m := libraryActionModel(100, 34)
		m.eng = eng
		m.sTab = item.tab
		selected := engine.Track{Kind: item.kind, ID: item.id, Title: "Live " + item.kind}
		if item.tab == 2 {
			m.sRes.Albums = []engine.Track{selected}
		} else {
			m.sRes.Playlists = []engine.Track{selected}
		}
		next, cmd := m.updateLibraryView(tea.KeyMsg{Type: tea.KeyEnter})
		m = next.(model)
		next, _ = m.Update(cmd())
		m = next.(model)
		if m.libraryActionFailed {
			t.Fatal(m.libraryActionStatus)
		}
		deadline := time.Now().Add(45 * time.Second)
		var state engine.State
		for time.Now().Before(deadline) {
			state, err = eng.State()
			if err != nil {
				t.Fatal(err)
			}
			next, _ = m.Update(stateMsg{st: state})
			m = next.(model)
			if state.Playing && !m.audioInitializing() && !m.libraryPlayRequested {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
		if !state.Playing || !strings.HasPrefix(m.libraryActionStatus, "Playing:") {
			t.Fatal("Play All never reached confirmed playback", m.libraryActionStatus)
		}
		before := len(state.Queue)
		next, cmd = m.updateLibraryView(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		m = next.(model)
		if strings.HasPrefix(m.libraryActionStatus, "Added") {
			t.Fatal("optimistic queue confirmation")
		}
		next, _ = m.Update(cmd())
		m = next.(model)
		state, err = eng.State()
		if err != nil {
			t.Fatal(err)
		}
		if m.libraryActionFailed || len(state.Queue) != 2*before || !strings.HasPrefix(m.libraryActionStatus, "Added to queue:") {
			t.Fatal("Add to Queue did not append the full item", m.libraryActionStatus)
		}
		t.Logf("%s: Play All confirmed %d tracks; Add to Queue confirmed %d tracks", item.kind, before, len(state.Queue))
	}
}
