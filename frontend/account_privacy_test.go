package main

import (
	"github.com/k1y0miiii/applemusic-tui/engine"
	"os"
	"path/filepath"
	"testing"
)

func TestDisconnectErasesPersonalHistoryAndRetainsPreferences(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AMTUI_CONFIG_DIR", dir)
	for _, name := range []string{"search-history.json", "listening-stats.json", "listening-stats.json.tmp", "theme"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	original := &listeningHistory{Days: map[string]map[string]listeningRecord{"today": {"song": {Title: "Private fixture"}}}}
	m := model{listenHistory: original, searchHistory: []string{"private search"}}
	m.recent = []engine.Track{{ID: "account-history"}}
	m.localRecent = []engine.Track{{ID: "local-history"}}
	m.recentSource = "spotify"
	m.recentSyncBusy = true
	m.recentSyncToken = 7
	if err := m.clearSpotifyPersonalData(); err != nil {
		t.Fatal(err)
	}
	if len(original.Days) != 0 || len(m.searchHistory) != 0 {
		t.Fatal("personal data retained in memory")
	}
	if len(m.recent) != 0 || len(m.localRecent) != 0 || m.recentSource != "" || m.recentSyncBusy || m.recentSyncToken == 7 {
		t.Fatal("sign-out retained account history or allowed an old refresh to restore it")
	}
	for _, name := range []string{"search-history.json", "listening-stats.json", "listening-stats.json.tmp"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatal("history remains", name)
		}
	}
	if data, err := os.ReadFile(filepath.Join(dir, "theme")); err != nil || string(data) != "fixture" {
		t.Fatal("preferences lost", err)
	}
	original.save() // The shutdown saver must not resurrect the old account's history.
	restored := loadListeningHistory()
	if len(restored.Days) != 0 {
		t.Fatal("shutdown restored erased history")
	}
}
