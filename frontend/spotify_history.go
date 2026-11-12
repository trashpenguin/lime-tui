package main

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/k1y0miiii/applemusic-tui/engine"
)

type spotifyHistoryMsg struct {
	token      uint64
	tracks     []engine.Track
	retryAfter time.Duration
	err        error
}

func (m *model) syncSpotifyHistory() tea.Cmd {
	m.recentSyncOptedIn = true
	if m.eng == nil || m.recentSyncBusy || time.Now().Before(m.recentNextSync) {
		return nil
	}
	m.recentSyncBusy = true
	m.recentSyncToken++
	token, eng := m.recentSyncToken, m.eng
	return func() tea.Msg {
		page, err := eng.RecentHistory()
		return spotifyHistoryMsg{token: token, tracks: page.Tracks, retryAfter: page.RetryAfter, err: err}
	}
}

func (m model) applySpotifyHistory(msg spotifyHistoryMsg) (tea.Model, tea.Cmd) {
	if msg.token != m.recentSyncToken {
		return m, nil
	}
	m.recentSyncBusy = false
	wait := max(time.Minute, msg.retryAfter)
	if msg.err != nil && strings.HasPrefix(msg.err.Error(), "Spotify history rate-limited") {
		m.recentRateLimitDelay = min(15*time.Minute, max(2*time.Minute, 2*m.recentRateLimitDelay))
		wait = max(wait, m.recentRateLimitDelay)
	} else if msg.err == nil {
		m.recentRateLimitDelay = 0
	}
	m.recentNextSync = time.Now().Add(wait)
	previous := m.recentSource
	selectedID := ""
	if m.recentSel >= 0 && m.recentSel < len(m.recent) {
		selectedID = m.recent[m.recentSel].ID
	}
	m.recentSyncError = ""
	if msg.err != nil {
		m.recentSyncError = msg.err.Error()
		m.recentSource = "local"
		m.recent = append([]engine.Track(nil), m.localRecent...)
		if previous != "local" {
			m.note, m.noteAt = msg.err.Error()+" · showing Lime plays", m.t
		}
	} else {
		m.recentSource = "spotify"
		m.recent = append([]engine.Track(nil), msg.tracks[:min(50, len(msg.tracks))]...)
	}
	m.recentSel = max(0, min(m.recentSel, len(m.recent)-1))
	if m.focus == focusRecent && selectedID != "" {
		for i, track := range m.recent {
			if track.ID == selectedID {
				m.recentSel = i
				break
			}
		}
	}
	m.sRes.Recent = m.recent
	if m.statsOpen && m.statsSpotify {
		m.statsClickID = ""
		return m.requestStatsArtists()
	}
	return withVisibleArtwork(m, nil)
}
