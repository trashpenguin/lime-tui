package main

import (
	"github.com/charmbracelet/lipgloss"
	"math"
	"strings"
)

// spectrumPanel uses bounded spectrum history; memory does not grow with playback.
func spectrumPanel(w, h, mode int, bands [32]float64, history [48][32]float64) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	var out strings.Builder
	ramp := []rune(" .:-=+*#%@")
	// A fixed history window, read oldest to newest, draws an envelope of the
	// actual spectrum without allocating another audio buffer.
	var ribbon [48]float64
	if mode == vizRibbon {
		for i, frame := range history {
			for _, value := range frame {
				ribbon[i] += spectrumLevel(value) / 32
			}
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			band := min(31, x*32/w)
			level := spectrumLevel(bands[band])
			c := ' '
			switch mode {
			case vizWave:
				if y == h-1-int(level*float64(h-1)) {
					c = '*'
				}
			case vizMirror:
				if math.Abs(float64(y)-float64(h-1)/2) <= level*float64(h)/2 {
					c = '|'
				}
			case vizWaterfall:
				level = spectrumLevel(history[min(47, y*48/h)][band])
				c = ramp[int(level*float64(len(ramp)-1))]
			case vizRing:
				// Correct for terminal cells being taller than they are wide.
				rx := math.Max(0.5, math.Min(float64(w-1)/2, float64(h-1)))
				ry := math.Max(0.5, rx/2)
				dx := (float64(x) - float64(w-1)/2) / rx
				dy := (float64(y) - float64(h-1)/2) / ry
				angle := (math.Atan2(dy, dx) + math.Pi) / (2 * math.Pi) * 32
				i := int(angle) % 32
				fraction := angle - math.Floor(angle)
				level = spectrumLevel(bands[i])*(1-fraction) + spectrumLevel(bands[(i+1)%32])*fraction
				if level > 0.01 && math.Abs(math.Hypot(dx, dy)-(0.25+0.65*level)) <= 0.55/ry {
					c = '*'
				}
			case vizDots:
				if x%2 == 0 && (h-1-y)%2 == 0 && level > 0.01 && float64(h-1-y) < level*float64(h) {
					c = 'o'
				}
			case vizRibbon:
				level = ribbon[47-min(47, x*48/w)]
				distance := math.Abs(float64(y) - float64(h-1)/2)
				if level > 0.01 && distance <= level*float64(h)/2 {
					c = '.'
					if distance+1 >= level*float64(h)/2 {
						c = '+'
					}
				}
			}
			out.WriteRune(c)
		}
		if y < h-1 {
			out.WriteByte('\n')
		}
	}
	return lipgloss.NewStyle().Foreground(accentHi).Render(out.String())
}

func spectrumLevel(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return max(0., min(1., value))
}
