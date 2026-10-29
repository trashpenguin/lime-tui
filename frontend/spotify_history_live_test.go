package main

import (
	"os"
	"testing"

	"github.com/k1y0miiii/applemusic-tui/engine"
)

// Read-only opt-in. This verifies the actual source label, not an assumed import.
func TestNativeLiveHistoryReportsActualSource(t *testing.T) {
	if os.Getenv("NATIVE_LIVE_TEST") != "1" {
		t.Skip("requires signed-in backend and isolated queue state")
	}
	eng, err := engine.Connect(func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()
	m := mouseModel(100, 34, 1)
	m.eng = eng
	message := m.syncSpotifyHistory()().(spotifyHistoryMsg)
	next, _ := m.Update(message)
	m = next.(model)
	if message.err != nil {
		if m.recentSource != "local" || m.recentSyncBusy {
			t.Fatal("unavailable Spotify history was misrepresented")
		}
		t.Logf("Verified LOCAL fallback; live Spotify import unavailable: %v; retry after %s", message.err, message.retryAfter)
	} else {
		if m.recentSource != "spotify" || len(m.recent) > 50 {
			t.Fatal("account history source/count incorrect")
		}
		t.Logf("Verified SPOTIFY account history: %d recent track records", len(m.recent))
	}
}
