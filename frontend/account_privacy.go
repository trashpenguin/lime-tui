package main

import (
	"errors"
	"fmt"
	"github.com/k1y0miiii/applemusic-tui/engine"
	"github.com/k1y0miiii/applemusic-tui/lyrics"
	"os"
	"path/filepath"
)

// Keep settings/theme, but erase Spotify-derived personal data on disconnect.
func (m *model) clearSpotifyPersonalData() error {
	m.searchHistory = nil
	m.historyIndex = -1
	m.sQuery = ""
	// Clear the original shared object too: main's shutdown saver holds this pointer.
	if m.listenHistory != nil {
		m.listenHistory.Days = make(map[string]map[string]listeningRecord)
	}
	m.listenLastTrack = ""
	m.recent = nil
	m.recentArtRequested = nil
	m.sRes = engine.SearchResults{}
	m.statsRes = engine.SearchResults{}
	m.discoveryFeed = engine.Discovery{}
	m.discoveryTracks = nil
	m.discoveryTitle = ""
	m.libraryDetailTitle = ""
	m.libraryDetailID, m.libraryActionStatus = "", ""
	m.libraryActionBusy, m.libraryActionFailed, m.libraryPlayRequested = false, false, false
	m.libraryActionToken++ // discard completions from the disconnected account
	m.ly = lyrics.Lyrics{}
	m.lyFor = ""
	m.art = nil
	m.artFor = ""
	m.artCache = newArtCache(8)
	m.coverRequests = nil
	m.scrKey = ""
	m.scrPlayed = 0
	m.st = engine.State{}
	dir := configDir()
	if dir == "" {
		return errors.New("local profile directory is unavailable")
	}
	var failures []error
	for _, name := range []string{"search-history.json", "listening-stats.json", "listening-stats.json.tmp"} {
		if err := os.Remove(filepath.Join(dir, name)); err != nil && !os.IsNotExist(err) {
			failures = append(failures, fmt.Errorf("remove %s: %w", name, err))
		}
	}
	return errors.Join(failures...)
}
