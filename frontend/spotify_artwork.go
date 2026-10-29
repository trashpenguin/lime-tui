package main

import (
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/k1y0miiii/applemusic-tui/engine"
	"image"
	"strings"
	"time"
)

type coverRequest struct {
	URL      string
	Token    uint64
	Attempts int
	Pending  bool
	Next     time.Time
}
type coverMsg struct {
	ID, RequestedURL, ResolvedURL string
	Token                         uint64
	Image                         image.Image
	Err                           error
}

func fetchCoverCmd(eng *engine.Engine, track engine.Track, token uint64) tea.Cmd {
	return func() tea.Msg {
		result := coverMsg{ID: track.ID, RequestedURL: track.Art, ResolvedURL: track.Art, Token: token}
		fetch := func(url string) (image.Image, error) {
			ctx, cancel := context.WithTimeout(context.Background(), artworkTimeout)
			defer cancel()
			return fetchArtwork(ctx, artworkURL(url, artworkPx))
		}
		if track.Art != "" {
			result.Image, result.Err = fetch(track.Art)
			if result.Err == nil {
				return result
			}
		}
		if eng != nil {
			if url, err := eng.Artwork(track.Kind, track.ID); err == nil && url != "" && url != track.Art {
				result.ResolvedURL = url
				result.Image, result.Err = fetch(url)
				return result
			} else if err != nil {
				result.Err = err
			}
		}
		if result.Err == nil {
			result.Err = fmt.Errorf("no cover is available")
		}
		return result
	}
}

func (m model) requestCover(track engine.Track) (model, tea.Cmd) {
	if track.ID == "" || !configBool(m.cfg, "artwork.enabled", true) {
		return m, nil
	}
	if _, ok := m.artCache.compatible(track.ID, track.Art); ok {
		return m, nil
	}
	if track.Art == "" && m.eng == nil {
		return m, nil
	}
	if m.coverRequests == nil {
		m.coverRequests = make(map[string]coverRequest)
	}
	state := m.coverRequests[track.ID]
	if state.URL != track.Art {
		state = coverRequest{URL: track.Art}
	}
	if state.Pending || state.Attempts >= 3 || time.Now().Before(state.Next) {
		return m, nil
	}
	pending := 0
	for _, r := range m.coverRequests {
		if r.Pending {
			pending++
		}
	}
	if pending >= 4 {
		return m, nil
	}
	if len(m.coverRequests) >= 64 {
		for id, r := range m.coverRequests {
			if !r.Pending && id != m.st.Now.ID && id != track.ID {
				delete(m.coverRequests, id)
			}
		}
	}
	m.coverSerial++
	state.Token = m.coverSerial
	state.Attempts++
	state.Pending = true
	m.coverRequests[track.ID] = state
	return m, fetchCoverCmd(m.eng, track, state.Token)
}

func (m model) requestCurrentArtwork() (model, tea.Cmd) {
	track := m.st.Now
	if track.ID == "" {
		m.art = nil
		m.artFor = ""
		m.artSourceURL = ""
		return m, nil
	}
	if m.artFor != track.ID || m.artSourceURL != track.Art {
		m.art = nil
		m.artFor = track.ID
		m.artSourceURL = track.Art
	}
	if m.art != nil {
		return m, nil
	}
	if img, ok := m.artCache.compatible(track.ID, track.Art); ok {
		m.art = img
		m.applyAutoTheme()
		return m, nil
	}
	return m.requestCover(track)
}

func (m model) requestVisibleArtwork() (model, tea.Cmd) {
	if m.phase != phaseReady {
		return m, nil
	}
	var commands []tea.Cmd
	var command tea.Cmd
	m, command = m.requestCurrentArtwork()
	if command != nil {
		commands = append(commands, command)
	}
	if m.libraryOpen || m.searchOpen {
		list := m.curList()
		if !m.sBusy && m.sSel >= 0 && m.sSel < len(list) {
			m, command = m.requestCover(list[m.sSel])
			if command != nil {
				commands = append(commands, command)
			}
		}
	}
	m, command = m.requestRecentArtwork()
	if command != nil {
		commands = append(commands, command)
	}
	return m, tea.Batch(commands...)
}

// Keep a separate bounded local fallback. Account history stays in Spotify's order.
func (m *model) rememberPlayedCover(track engine.Track) {
	if track.ID == "" {
		return
	}
	items := []engine.Track{track}
	oldItems := m.localRecent
	if len(oldItems) == 0 && m.recentSource != "spotify" {
		oldItems = m.recent
	}
	for _, old := range oldItems {
		if old.ID != track.ID && len(items) < 20 {
			items = append(items, old)
		}
	}
	m.localRecent = items
	if m.recentSource != "spotify" {
		m.recent = items
		m.sRes.Recent = items
		m.recentSel = min(m.recentSel, len(items)-1)
	}
}

func withVisibleArtwork(next tea.Model, command tea.Cmd) (tea.Model, tea.Cmd) {
	m := next.(model)
	m, artwork := m.requestVisibleArtwork()
	return m, tea.Batch(command, artwork)
}

func coverPlaceholder(w, h int, label string) []string {
	rows := make([]string, h)
	for y := range rows {
		text := ""
		if y == h/2 {
			text = label
		}
		text = pad(text, w)
		rows[y] = lipgloss.NewStyle().Foreground(fgDim).Background(lipgloss.Color("#18241F")).Render(text)
	}
	return rows
}

func (m model) coverLabel(id string) string {
	if state, ok := m.coverRequests[id]; ok {
		if state.Pending {
			return " Loading cover…"
		}
		if state.Attempts >= 3 {
			return " Cover unavailable"
		}
		if state.Attempts > 0 {
			return " Retrying cover…"
		}
	}
	return " Album artwork"
}

// Show the selected cover beside the list without changing the row hit targets.
func (m model) browserCoverRows(body []string, width, rows int) []string {
	list := m.curList()
	if width < 78 || rows < 8 || m.sBusy || m.sSel < 0 || m.sSel >= len(list) || !configBool(m.cfg, "artwork.enabled", true) {
		return body
	}
	selected := list[m.sSel]
	height := min(12, rows)
	coverWidth := height * 2
	leftWidth := width - coverWidth - artworkGap
	for len(body) < rows {
		body = append(body, "")
	}
	cover := coverPlaceholder(coverWidth, height, m.coverLabel(selected.ID))
	if img, ok := m.artCache.compatible(selected.ID, selected.Art); ok {
		cover = renderArtwork(img, coverWidth, height)
	}
	for i := 0; i < rows; i++ {
		right := ""
		if i < len(cover) {
			right = cover[i]
		} else if i == height {
			right = lipgloss.NewStyle().Foreground(accentHi).Bold(true).Render(" " + strings.ToUpper(selected.Kind) + " COVER")
		} else if i == height+1 {
			right = " Click cover to play"
		}
		body[i] = pad(body[i], leftWidth) + strings.Repeat(" ", artworkGap) + pad(right, coverWidth)
	}
	return body
}

func (m model) browserCoverColumn(width, rows int) int {
	if width < 78 || rows < 8 || !configBool(m.cfg, "artwork.enabled", true) {
		return width
	}
	return width - min(12, rows)*2
}
