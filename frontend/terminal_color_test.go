package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"image"
	"image/color"
	"os"
	"strings"
	"testing"
)

func TestInteractiveStartupRestoresCoverColorsWithInheritedNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("TERM", "dumb")
	t.Setenv("SPOTIFY_TUI_COLOR", "")
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.Ascii)
	configureTerminalColorsFor(true)
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for x := 0; x < 2; x++ {
		img.Set(x, 0, color.RGBA{R: 255, A: 255})
		img.Set(x, 1, color.RGBA{B: 255, A: 255})
	}
	rendered := strings.Join(renderArtwork(img, 1, 1), "")
	if !strings.Contains(rendered, "38;2;255;0;0") || !strings.Contains(rendered, "48;2;0;0;255") {
		t.Fatal("interactive startup still strips the cover's RGB foreground/background")
	}
}

func TestColorChoiceIsAppScopedAndRedirectedOutputStaysPlain(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("TERM", "dumb")
	t.Setenv("SPOTIFY_TUI_COLOR", "")
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	configureTerminalColorsFor(false)
	if lipgloss.ColorProfile() != termenv.Ascii {
		t.Fatal("redirected output contains color by default")
	}
	t.Setenv("SPOTIFY_TUI_COLOR", "always")
	configureTerminalColorsFor(false)
	if lipgloss.ColorProfile() != termenv.TrueColor {
		t.Fatal("explicit app color mode was ignored")
	}
	t.Setenv("SPOTIFY_TUI_COLOR", "never")
	configureTerminalColorsFor(true)
	if lipgloss.ColorProfile() != termenv.Ascii {
		t.Fatal("explicit app monochrome mode was ignored")
	}
	if os.Getenv("NO_COLOR") != "1" || os.Getenv("TERM") != "dumb" {
		t.Fatal("app color setup changed inherited environment settings")
	}
}

func TestThemeButtonChangesPaletteAndDoesNotOpenSearch(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("TERM", "dumb")
	t.Setenv("SPOTIFY_TUI_COLOR", "")
	t.Setenv("AMTUI_CONFIG_DIR", t.TempDir())
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	defer applyTheme(themes[0])
	configureTerminalColorsFor(true)
	applyTheme(themes[0])
	m := mouseModel(140, 40, 0)
	m.themeName = "spotify"
	updated, _ := m.Update(tea.MouseMsg{X: m.themeButtonX() + 4, Y: 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m = updated.(model)
	if m.themeName != "catppuccin" || m.searchOpen || loadThemeName() != "catppuccin" {
		t.Fatal("Theme button did not switch and save the theme")
	}
	if !strings.Contains(m.View(), "38;2;245;194;231") || !strings.Contains(m.View(), "Theme: catppuccin") {
		t.Fatal("new theme is not visible")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m = updated.(model)
	if m.themeName != "gruvbox" || !strings.Contains(m.View(), "38;2;254;128;25") {
		t.Fatal("keyboard theme switch did not change the palette")
	}
}

func TestThemeChangesRenderedColorsUnderLauncherEnvironment(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("TERM", "dumb")
	t.Setenv("SPOTIFY_TUI_COLOR", "")
	t.Setenv("AMTUI_CONFIG_DIR", t.TempDir())
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	defer applyTheme(themes[0])
	lipgloss.SetColorProfile(termenv.Ascii)
	configureTerminalColorsFor(true)
	applyTheme(themes[0])
	m := mouseModel(140, 40, 0)
	m.themeName = "spotify"
	before := m.View()
	theme := nextTheme(m.themeName)
	m.themeName = theme.name
	applyTheme(theme)
	after := m.View()
	if !strings.Contains(before, "38;2;217;255;160") || !strings.Contains(after, "38;2;245;194;231") {
		t.Fatal("switching theme does not change rendered RGB colors")
	}
}
