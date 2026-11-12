package main

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const nativeBitrateEnv = "LIME_NATIVE_BITRATE"

var nativeAudioQualities = [...]struct{ bitrate, label string }{
	{"96", "Normal · 96 kbps"},
	{"160", "High · 160 kbps"},
	{"320", "Very High · 320 kbps"},
}

func savedAudioQuality() int {
	data, _ := os.ReadFile(filepath.Join(configDir(), "audio-quality"))
	for i, quality := range nativeAudioQualities {
		if strings.TrimSpace(string(data)) == quality.bitrate {
			return i
		}
	}
	return 1
}

func (m model) soloistAudioQuality() bool {
	return m.eng != nil && m.eng.Soloist()
}

func (m model) updateAudioQuality(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc", "ctrl+c", "f6":
		m.audioQualityOpen = false
	case "up", "k":
		m.audioQualitySel = max(0, m.audioQualitySel-1)
	case "down", "j":
		m.audioQualitySel = min(len(nativeAudioQualities)-1, m.audioQualitySel+1)
	case "enter":
		if m.soloistAudioQuality() {
			return m, nil
		}
		dir := configDir()
		err := os.MkdirAll(dir, 0o700)
		if err == nil {
			err = os.WriteFile(filepath.Join(dir, "audio-quality"), []byte(nativeAudioQualities[m.audioQualitySel].bitrate+"\n"), 0o600)
		}
		m.audioQualityNotice = "Saved. Restart Lime to apply."
		if err != nil {
			m.audioQualityNotice = "Could not save audio quality."
		}
	}
	return m, nil
}

func (m model) audioQualityView() string {
	width := max(1, min(64, m.w-4))
	faint := lipgloss.NewStyle().Foreground(fgFaint)
	active := lipgloss.NewStyle().Foreground(accentHi).Bold(true)
	lines := []string{faint.Render("Actual stream format: unavailable from engine"), ""}
	if m.soloistAudioQuality() {
		lines = append(lines,
			active.Render("Spotify Soloist · Spotify Connect"),
			"On your phone, select Lime Soloist in Spotify's device picker.",
			"Open Change quality settings and choose Lossless",
			"if Spotify offers it for your Premium account.",
			"", faint.Render("Esc back"))
	} else {
		saved := savedAudioQuality()
		for i, quality := range nativeAudioQualities {
			label := "  " + quality.label
			if i == saved {
				label += " · saved"
			}
			if quality.bitrate == os.Getenv(nativeBitrateEnv) {
				label += " · used this session"
			}
			if i == m.audioQualitySel {
				label = active.Render("› " + label[2:])
			}
			lines = append(lines, label)
		}
		lines = append(lines, "", faint.Render("Native playback uses compressed audio."),
			faint.Render("↑↓ select · Enter save · Esc back"))
		if m.audioQualityNotice != "" {
			lines = append(lines, m.audioQualityNotice)
		} else if nativeAudioQualities[saved].bitrate != os.Getenv(nativeBitrateEnv) {
			lines = append(lines, "Restart Lime to apply the saved preference.")
		}
	}
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, width, "…")
	}
	box := panel("AUDIO QUALITY", strings.Join(lines, "\n"), width, len(lines)+1, true, accentHi)
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, box)
}
