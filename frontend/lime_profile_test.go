package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLimeProfileCompatibility(t *testing.T) {
	base := t.TempDir()
	current := filepath.Join(base, "LimeTUI")
	legacy := filepath.Join(base, "PommeSpotifyTUI")
	if got := defaultProfile(base); got != current {
		t.Fatal(got)
	}
	if err := os.Mkdir(legacy, 0700); err != nil {
		t.Fatal(err)
	}
	if got := defaultProfile(base); got != legacy {
		t.Fatal("legacy profile lost", got)
	}
	if err := os.Mkdir(current, 0700); err != nil {
		t.Fatal(err)
	}
	if got := defaultProfile(base); got != current {
		t.Fatal("current profile ignored", got)
	}
}
func TestLimeEnvironmentPrecedence(t *testing.T) {
	t.Setenv("LIME_DATA_DIR", "")
	t.Setenv("SPOTIFY_TUI_DATA_DIR", "legacy")
	if firstEnvironment("LIME_DATA_DIR", "SPOTIFY_TUI_DATA_DIR") != "legacy" {
		t.Fatal("legacy override lost")
	}
	t.Setenv("LIME_DATA_DIR", "lime")
	if firstEnvironment("LIME_DATA_DIR", "SPOTIFY_TUI_DATA_DIR") != "lime" {
		t.Fatal("Lime override ignored")
	}
	if themeByName("spotify") != themeByName("lime") {
		t.Fatal("legacy theme lost")
	}
}
