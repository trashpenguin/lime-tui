package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSpotifyProfileIsSeparateAndRespectsOverride(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SPOTIFY_TUI_DATA_DIR", root)
	t.Setenv("AMTUI_CONFIG_DIR", "")
	t.Setenv("AMTUI_NATIVE_BACKEND", "")
	for _, key := range []string{"XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME"} {
		t.Setenv(key, "")
	}
	setupSpotifyProfile()
	if got := configDir(); got != filepath.Join(root, "settings") {
		t.Fatalf("settings = %q", got)
	}
	for key, part := range map[string]string{"XDG_CONFIG_HOME": "config", "XDG_CACHE_HOME": "cache", "XDG_STATE_HOME": "state"} {
		if got := os.Getenv(key); got != filepath.Join(root, part) {
			t.Errorf("%s = %q", key, got)
		}
	}
}

func TestSpotifyPairingFitsBootScreen(t *testing.T) {
	applyTheme(themes[0])
	m := model{w: 140, h: 40, phase: phaseBoot, status: "Spotify: enter TEST-CODE at https://www.spotify.com/pair"}
	view := m.View()
	if !strings.Contains(view, "TEST-CODE") || !strings.Contains(view, "Premium required") {
		t.Fatal("pairing code or subscription requirement is missing")
	}
	if strings.Contains(strings.ToLower(view), "apple music") {
		t.Fatal("Apple Music branding remains")
	}
}
