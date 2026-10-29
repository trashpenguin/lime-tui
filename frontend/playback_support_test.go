package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/k1y0miiii/applemusic-tui/engine"
)

func TestSavingPlaybackReportDoesNotCancelAudioInitialization(t *testing.T) {
	m := mouseModel(100, 34, 1)
	m.libraryOpen = true
	m.loading = "Track loading"
	m.initPending = true
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyF12})
	if cmd == nil || !next.(model).initPending {
		t.Fatal("F12 did not preserve the pending player")
	}
	next, _ = next.(model).Update(playbackReportMsg{traceAvailable: true})
	got := next.(model)
	if !got.initPending || got.loading != "Track loading" || !strings.Contains(got.note, playbackReportName) {
		t.Fatal("support notification changed playback state")
	}
}

func TestPlaybackReportContainsNoCatalogOrCredentials(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"AMTUI_CONFIG_DIR", "AMTUI_NATIVE_BACKEND", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME"} {
		t.Setenv(name, "")
	}
	t.Setenv("LIME_DATA_DIR", dir)
	report := systemDiagnostics()
	report["playback"] = engine.PlaybackDiagnostics{Connected: true, Events: []engine.PlaybackTraceEvent{{Event: "stopped", PositionMs: 1000}}}
	path := filepath.Join(dir, playbackReportName)
	if err := writePlaybackSupport(path, report); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if json.Unmarshal(data, &decoded) != nil || decoded["playback"] == nil {
		t.Fatal("invalid support report")
	}
	for _, forbidden := range []string{dir, "access_token", "username", "password", "track_id", "artist", "profile_path"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("support report exposes %q", forbidden)
		}
	}
}
