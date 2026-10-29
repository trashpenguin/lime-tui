package main

import (
	"encoding/json"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	"github.com/muesli/termenv"
	"image"
	"image/color"
	"os"
	"strings"
)

func configureTerminalColors() {
	configureTerminalColorsFor(term.IsTerminal(os.Stdout.Fd()))
}

func configureTerminalColorsFor(interactive bool) {
	mode := strings.ToLower(strings.TrimSpace(firstEnvironment("LIME_COLOR", "SPOTIFY_TUI_COLOR")))
	// A full-screen visual player needs RGB for artwork. NO_COLOR/TERM can be
	// inherited from an agent's log environment, rather than from the terminal.
	// Keep redirected output plain and provide an app-specific opt-out.
	profile := termenv.Ascii
	if mode == "always" || (interactive && mode != "never") {
		profile = termenv.TrueColor
	}
	lipgloss.SetColorProfile(profile)
}

func (m model) themeButtonLabel() string {
	name := m.themeName
	if name == "" {
		name = themes[0].name
	}
	return " [t Theme: " + name + "] "
}
func (m model) themeButtonX() int { return m.w - 1 - lipgloss.Width(m.themeButtonLabel()) }

func (m model) headerBar() string {
	width := m.w - 2
	button := lipgloss.NewStyle().Foreground(accentHi).Background(selBg).Bold(true).Render(m.themeButtonLabel())
	left := lipgloss.NewStyle().Foreground(accentHi).Bold(true).Render(" LIME TUI ") +
		lipgloss.NewStyle().Foreground(fgFaint).Render(" / Search ") + lipgloss.NewStyle().Foreground(fgBright).Render(m.sQuery) +
		lipgloss.NewStyle().Foreground(fgFaint).Render("  Tab focus · l lyrics")
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(borderDim).Width(width).
		Render(pad(left, max(0, width-lipgloss.Width(button)-1)) + " " + button)
}

func (m model) cycleTheme() (tea.Model, tea.Cmd) {
	next := nextTheme(m.themeName)
	m.themeName = next.name
	applyTheme(next)
	saveThemeName(next.name)
	m.applyAutoTheme()
	m.note, m.noteAt = "theme · "+next.name, m.t
	return m, nil
}

// Run this in the real host terminal to inspect startup, not a forced-color
// preview renderer. No Spotify login, account data or settings are accessed.
func writeTerminalColorDiagnostics(path string) error {
	configureTerminalColors()
	profile := "plain"
	if lipgloss.ColorProfile() == termenv.TrueColor {
		profile = "truecolor"
	}
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for x := 0; x < 2; x++ {
		img.Set(x, 0, color.RGBA{R: 255, A: 255})
		img.Set(x, 1, color.RGBA{B: 255, A: 255})
	}
	cover := strings.Join(renderArtwork(img, 1, 1), "")
	applyTheme(themes[0])
	first := panel("THEME CHECK", "color", 40, 3, true, accentHi)
	applyTheme(themes[1])
	second := panel("THEME CHECK", "color", 40, 3, true, accentHi)
	applyTheme(themes[0])
	report := map[string]any{"version": version, "stdout_is_terminal": term.IsTerminal(os.Stdout.Fd()), "windows_terminal": os.Getenv("WT_SESSION") != "", "inherited_no_color": os.Getenv("NO_COLOR") != "", "term": os.Getenv("TERM"), "app_color_mode": firstEnvironment("LIME_COLOR", "SPOTIFY_TUI_COLOR"), "profile": profile,
		"cover_rgb_foreground": strings.Contains(cover, "38;2;255;0;0"), "cover_rgb_background": strings.Contains(cover, "48;2;0;0;255"), "theme_rgb_changes": first != second && strings.Contains(first, "38;2;") && strings.Contains(second, "38;2;")}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(path, data, 0600); err != nil {
		return err
	}
	fmt.Println("Terminal color check:", profile)
	return nil
}

func (m model) lyricsTitle() string {
	title := "LYRICS"
	if len(m.ly.Lines) > 0 {
		if m.ly.Synced {
			title += " · SYNCED"
		} else {
			title += " · PLAIN TEXT"
		}
		if m.ly.Source != "" {
			title += " · " + m.ly.Source
		}
	}
	if m.lyricOffset != 0 {
		title += fmt.Sprintf(" · %+.1fs", m.lyricOffset.Seconds())
	}
	return title
}

func (m model) fullLyricsView() string {
	width := max(1, m.w-6)
	rows := max(1, m.h-9)
	heading := lipgloss.NewStyle().Foreground(accentHi).Bold(true).Render(pad(" "+m.st.Now.Title, width))
	artist := lipgloss.NewStyle().Foreground(fgDim).Render(pad(" "+m.st.Now.Artist+" · "+fmtTime(m.st.Pos)+" / "+fmtTime(m.st.Dur), width))
	box := panel(m.lyricsTitle(), m.lyricsText(width, rows), width, rows+1, true, accentHi)
	hints := lipgloss.NewStyle().Foreground(fgDim).Render("Esc/l Back · Space Pause · ←/→ Seek · [ / ] Timing ±0.5s · \\ Reset")
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, strings.Join([]string{heading, artist, box, hints}, "\n"))
}
