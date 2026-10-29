package main

import (
	"os"
	"path/filepath"
	"runtime/debug"
)

func setupSpotifyProfile() {
	root := firstEnvironment("LIME_DATA_DIR", "SPOTIFY_TUI_DATA_DIR")
	if root == "" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			base, _ = os.UserConfigDir()
		}
		root = defaultProfile(base)
	}
	root, _ = filepath.Abs(root)
	os.Setenv("AMTUI_CONFIG_DIR", filepath.Join(root, "settings"))
	backend := firstEnvironment("LIME_ENGINE", "SPOTIFY_TUI_BACKEND")
	if backend == "" {
		if executable, err := os.Executable(); err == nil {
			backend = filepath.Join(filepath.Dir(executable), "lime-engine.exe")
		}
	}
	if backend != "" {
		os.Setenv("AMTUI_NATIVE_BACKEND", backend)
	}
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	os.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	os.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	if os.Getenv("GOMEMLIMIT") == "" {
		debug.SetMemoryLimit(64 << 20)
	}
}

// Keep existing profiles in place: no credential copying or concurrent migration.
func defaultProfile(base string) string {
	current := filepath.Join(base, "LimeTUI")
	legacy := filepath.Join(base, "PommeSpotifyTUI")
	if _, err := os.Stat(current); os.IsNotExist(err) {
		if info, err := os.Stat(legacy); err == nil && info.IsDir() {
			return legacy
		}
	}
	return current
}
func firstEnvironment(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}
