package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"strings"
)

type browserBox struct{ x, y, width int }

// Derive the overlay rectangle from the rendered borders. This keeps mouse
// targets aligned when titles, terminal size or centering change.
func renderedBrowserBox(view string) (browserBox, bool) {
	for y, line := range strings.Split(view, "\n") {
		line = ansi.Strip(line)
		for i, r := range line {
			if r == '╔' || r == '╭' {
				return browserBox{lipgloss.Width(line[:i]), y, lipgloss.Width(strings.TrimRight(line[i:], " "))}, true
			}
		}
	}
	return browserBox{}, false
}

func (m model) updateBrowserMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress || m.saveConfirmOpen {
		return m, nil
	}
	wheel := msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown
	if wheel {
		key := tea.KeyMsg{Type: tea.KeyDown}
		if msg.Button == tea.MouseButtonWheelUp {
			key.Type = tea.KeyUp
		}
		switch {
		case m.libraryOpen:
			return m.updateLibraryView(key)
		case m.searchOpen:
			m.sInput = false
			return m.updateSearch(key)
		case m.discoveryOpen:
			return m.updateDiscovery(key)
		case m.queueOpen:
			return m.updateQueueManager(key)
		}
	}
	if msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	var view string
	switch {
	case m.libraryOpen:
		view = m.libraryView()
	case m.searchOpen:
		view = m.searchView()
	case m.discoveryOpen:
		view = m.discoveryView()
	case m.queueOpen:
		view = m.queueManagerView()
	}
	box, ok := renderedBrowserBox(view)
	if !ok || msg.X <= box.x || msg.X >= box.x+box.width-1 {
		return m, nil
	}
	enter := tea.KeyMsg{Type: tea.KeyEnter}
	switch {
	case m.libraryOpen:
		rows := max(4, min(18, m.h-14))
		for y, line := range strings.Split(view, "\n") {
			if y != msg.Y || y != box.y+2+rows {
				continue
			}
			plain := ansi.Strip(line)
			for _, label := range m.libraryButtons() {
				at := strings.Index(plain, label)
				if at < 0 {
					continue
				}
				x := lipgloss.Width(plain[:at])
				if msg.X >= x && msg.X < x+lipgloss.Width(label) {
					key := enter
					if strings.HasPrefix(label, "[a ") {
						key = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
					}
					if strings.HasPrefix(label, "[A ") {
						key = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}}
					}
					if strings.HasPrefix(label, "[r ") {
						key = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
					}
					return m.updateLibraryView(key)
				}
			}
		}
		if m.sBusy || m.libraryErr != "" {
			return m, nil
		}
		row := msg.Y - box.y - 2
		start := max(0, m.sSel-rows+1)
		index := start + row
		if msg.X >= box.x+1+m.browserCoverColumn(box.width-2, rows) {
			index = m.sSel
		}
		if row < 0 || row >= rows || index >= len(m.curList()) {
			return m, nil
		}
		m.sSel = index
		return m.updateLibraryView(enter)
	case m.searchOpen:
		if msg.Y == box.y+1 {
			m.sInput = true
			return m, nil
		}
		if msg.Y == box.y+2 {
			lines := strings.Split(view, "\n")
			line := ansi.Strip(lines[msg.Y])
			for i, name := range tabNames {
				if at := strings.Index(line, name); at >= 0 {
					x := lipgloss.Width(line[:at])
					if msg.X >= x-1 && msg.X < x+len(name)+1 {
						m.sTab = i
						m.sSel = 0
						m.sInput = false
						return m, nil
					}
				}
			}
			return m, nil
		}
		if m.sBusy {
			return m, nil
		}
		row := msg.Y - box.y - 4
		rows := m.searchRows()
		start := max(0, m.sSel-rows+1)
		index := start + row
		if msg.X >= box.x+1+m.browserCoverColumn(box.width-2, rows) {
			index = m.sSel
		}
		if row < 0 || row >= rows || index >= len(m.curList()) {
			return m, nil
		}
		m.sSel = index
		m.sInput = false
		return m.updateSearch(enter)
	case m.queueOpen:
		rows := max(5, min(20, m.h-13))
		row := msg.Y - box.y - 2
		start := max(0, min(m.selIdx-rows/2, len(m.st.Queue)-rows))
		index := start + row
		if row < 0 || row >= rows || index >= len(m.st.Queue) {
			return m, nil
		}
		m.selIdx = index
		return m.updateQueueManager(enter)
	case m.discoveryOpen:
		if m.discoveryBusy || m.discoveryErr != "" {
			return m, nil
		}
		rows := max(5, min(m.h-15, m.h*55/100))
		row := msg.Y - box.y - 2
		if row < 0 || row >= rows {
			return m, nil
		}
		var indices []int
		selectedLine := -1
		if m.discoveryDetail {
			for i := range m.discoveryTracks {
				indices = append(indices, i)
				if i == m.discoveryTrackSel {
					selectedLine = i
				}
			}
		} else {
			choice := 0
			for _, section := range m.discoveryFeed.Sections {
				indices = append(indices, -1)
				for range section.Items {
					if choice == m.discoverySel {
						selectedLine = len(indices)
					}
					indices = append(indices, choice)
					choice++
				}
			}
		}
		index := max(0, selectedLine-rows+1) + row
		if index >= len(indices) || indices[index] < 0 {
			return m, nil
		}
		if m.discoveryDetail {
			m.discoveryTrackSel = indices[index]
		} else {
			m.discoverySel = indices[index]
		}
		return m.updateDiscovery(enter)
	}
	return m, nil
}
