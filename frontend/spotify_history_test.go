package main

import (
	"errors"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
	"testing"
	"time"

	"github.com/k1y0miiii/applemusic-tui/engine"
)

func TestHistoryRefreshKeyKeepsThePendingRequestGeneration(t *testing.T) {
	m := mouseModel(100, 34, 1)
	m.eng = &engine.Engine{}
	next, cmd := m.updateKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
	m = next.(model)
	if cmd == nil || !m.recentSyncBusy || m.recentSyncToken != 1 {
		t.Fatal("refresh lost its pending state/token")
	}
}

func TestSpotifyHistoryPreservesAccountOrderAndIsNotReplacedByLocalPlayback(t *testing.T) {
	m := mouseModel(100, 34, 1)
	m.recentSyncToken = 3
	remote := []engine.Track{{ID: "spotify-new", Title: "Played on phone"}, {ID: "spotify-old", Title: "Played on desktop"}}
	next, _ := m.Update(spotifyHistoryMsg{token: 3, tracks: remote})
	m = next.(model)
	if m.recentSource != "spotify" || m.recent[0].ID != "spotify-new" || !strings.Contains(m.recentTitle(), "SPOTIFY") {
		t.Fatal("account history missing or mislabeled")
	}
	m.rememberPlayedCover(engine.Track{ID: "lime", Title: "Playing in Lime"})
	if m.recent[0].ID != "spotify-new" || m.localRecent[0].ID != "lime" {
		t.Fatal("local playback overwrote Spotify order or lost fallback")
	}
}

func TestSpotifyHistoryRateLimitUsesLocalFallbackAndHonorsLongRetry(t *testing.T) {
	m := mouseModel(100, 34, 1)
	m.recentSyncToken = 1
	m.recentSyncBusy = true
	m.localRecent = []engine.Track{{ID: "local", Title: "Lime play"}}
	deadline := time.Now().Add(2 * time.Hour)
	next, _ := m.Update(spotifyHistoryMsg{token: 1, err: errors.New("Spotify history rate-limited"), retryAfter: 2 * time.Hour})
	m = next.(model)
	if m.recentSyncBusy || m.recentSource != "local" || len(m.recent) != 1 || !strings.Contains(m.recentTitle(), "LOCAL") {
		t.Fatal("fallback is missing or pretending to be Spotify")
	}
	if m.recentNextSync.Before(deadline) {
		t.Fatal("retry would violate Spotify cooldown")
	}
}

func TestEmptySpotifyHistoryClearsPreviousAccountResults(t *testing.T) {
	m := mouseModel(100, 34, 1)
	m.recent = []engine.Track{{ID: "old"}}
	m.recentSel = 9
	m.recentSyncToken = 2
	next, _ := m.Update(spotifyHistoryMsg{token: 2, tracks: []engine.Track{}})
	m = next.(model)
	if len(m.recent) != 0 || m.recentSel != 0 || m.recentSource != "spotify" {
		t.Fatal("empty account history retained stale tracks")
	}
}

func TestDisconnectedOrReplacedSessionDiscardsOldHistoryCompletion(t *testing.T) {
	m := mouseModel(100, 34, 1)
	m.recentSyncToken = 10
	m.recentSource = "local"
	next, _ := m.Update(spotifyHistoryMsg{token: 9, tracks: []engine.Track{{ID: "old-account"}}})
	m = next.(model)
	if m.recentSource != "local" || len(m.recent) != 0 {
		t.Fatal("stale account results were applied")
	}
}

func TestLegacyLibraryResultsCannotOverwriteSpotifyHistory(t *testing.T) {
	m := mouseModel(100, 34, 1)
	m.recentSource = "spotify"
	m.recent = []engine.Track{{ID: "spotify"}}
	next, _ := m.Update(searchMsg{res: engine.SearchResults{Recent: []engine.Track{{ID: "local"}}}})
	m = next.(model)
	if m.recent[0].ID != "spotify" {
		t.Fatal("library query overwrote account history")
	}
	if len(m.sRes.Recent) != 1 || m.sRes.Recent[0].ID != "spotify" {
		t.Fatal("search Recent tab lost account history")
	}
}

func TestFocusedHistorySelectionSurvivesNewAccountPlays(t *testing.T) {
	m := mouseModel(100, 34, 1)
	m.focus = focusRecent
	m.recentSyncToken = 2
	m.recent = []engine.Track{{ID: "selected"}}
	m.recentSel = 0
	next, _ := m.Update(spotifyHistoryMsg{token: 2, tracks: []engine.Track{{ID: "new"}, {ID: "selected"}}})
	m = next.(model)
	if m.recentSel != 1 {
		t.Fatal("refresh moved selection to a different track")
	}
}
