package main

import (
	"github.com/charmbracelet/lipgloss"
	"math"
	"strings"
	"testing"
)

func TestExtraVisualizerBounds(t *testing.T) {
	for _, mode := range []int{vizWave, vizMirror, vizWaterfall, vizRing, vizDots, vizRibbon} {
		for _, size := range [][2]int{{1, 1}, {17, 9}, {80, 24}} {
			output := spectrumPanel(size[0], size[1], mode, fullBands(0.8), [48][32]float64{})
			rows := strings.Split(output, "\n")
			if len(rows) != size[1] {
				t.Fatalf("mode %d height %d", mode, len(rows))
			}
			for _, row := range rows {
				if lipgloss.Width(row) != size[0] {
					t.Fatalf("mode %d width %d", mode, lipgloss.Width(row))
				}
			}
		}
	}
}

func TestNewVisualizersReactToAudioAndStaySilentWithoutIt(t *testing.T) {
	var quiet, history [48][32]float64
	for i := range history {
		history[i] = fullBands(float64(48-i) / 48)
	}
	seen := map[string]bool{}
	for _, mode := range []int{vizRing, vizDots, vizRibbon} {
		silent := spectrumPanel(60, 14, mode, [32]float64{}, quiet)
		if strings.TrimSpace(silent) != "" {
			t.Fatalf("%s drew energy without audio", vizModeName(mode))
		}
		active := spectrumPanel(60, 14, mode, fullBands(0.8), history)
		if active == silent || seen[active] {
			t.Fatalf("%s did not draw a distinct audio response", vizModeName(mode))
		}
		seen[active] = true
		if got := spectrumPanel(0, 14, mode, fullBands(1), history); got != "" {
			t.Fatal("empty panel should render nothing")
		}
		if got := spectrumPanel(60, -1, mode, fullBands(1), history); got != "" {
			t.Fatal("negative panel should render nothing")
		}
	}
}

func TestRingAndDotsFollowFrequencyAndRibbonFollowsHistory(t *testing.T) {
	var bass, treble [32]float64
	for i := 0; i < 8; i++ {
		bass[i], treble[31-i] = 1, 1
	}
	for _, mode := range []int{vizRing, vizDots} {
		if spectrumPanel(60, 14, mode, bass, [48][32]float64{}) == spectrumPanel(60, 14, mode, treble, [48][32]float64{}) {
			t.Fatalf("%s ignores frequency changes", vizModeName(mode))
		}
	}
	var history [48][32]float64
	history[0] = fullBands(1)
	rows := strings.Split(spectrumPanel(48, 9, vizRibbon, bass, history), "\n")
	for _, row := range rows {
		if strings.TrimSpace(row[:47]) != "" || row[47] == ' ' {
			t.Fatal("newest history frame must appear at the right edge")
		}
	}
	if spectrumPanel(48, 9, vizRibbon, bass, history) != spectrumPanel(48, 9, vizRibbon, treble, history) {
		t.Fatal("ribbon should follow recorded history rather than invent a waveform")
	}
}

func TestSpectrumRejectsNonFiniteLevels(t *testing.T) {
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), -1} {
		if spectrumLevel(value) != 0 {
			t.Fatal("invalid spectrum level was not silenced")
		}
	}
}

func TestProgressClampsToTrack(t *testing.T) {
	m := model{}
	for _, filled := range []int{-10, 0, 5, 10, 100} {
		if got := lipgloss.Width(m.progressBar(10, filled)); got != 10 {
			t.Fatalf("filled %d width %d", filled, got)
		}
	}
	if m.progressBar(-1, 5) != "" {
		t.Fatal("negative width must be empty")
	}
}
