package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAudioQualitySavesPreferenceForNextLaunch(t *testing.T) {
	t.Setenv("AMTUI_CONFIG_DIR", t.TempDir())
	t.Setenv(nativeBitrateEnv, "160")

	m := mouseModel(100, 38, 0)
	for _, key := range []tea.KeyType{tea.KeyF6, tea.KeyDown, tea.KeyEnter} {
		next, _ := m.Update(tea.KeyMsg{Type: key})
		m = next.(model)
	}
	if !m.audioQualityOpen || savedAudioQuality() != 2 || m.audioQualityNotice != "Saved. Restart Lime to apply." {
		t.Fatal("the selected quality was not saved for the next launch")
	}
}
