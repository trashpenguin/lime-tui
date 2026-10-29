package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/k1y0miiii/applemusic-tui/engine"
	"github.com/k1y0miiii/applemusic-tui/lyrics"
	"github.com/muesli/termenv"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFocusedPanelHasVisibleMarkerWithoutColor(t *testing.T) {
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.Ascii)
	view := panel("LIBRARY", "Albums", 38, 4, true, accentHi)
	if !strings.Contains(view, "ACTIVE") || !strings.Contains(view, "▶") {
		t.Fatal("active panel has no non-color focus indicator")
	}
	if strings.Contains(panel("LIBRARY", "Albums", 38, 4, false, accentHi), "ACTIVE") {
		t.Fatal("inactive panel is labelled active")
	}
}

func TestLyricsViewFitsAndShowsSyncMode(t *testing.T) {
	for _, size := range [][2]int{{70, 22}, {100, 34}, {140, 40}} {
		m := mouseModel(size[0], size[1], 1)
		m.lyricsOpen = true
		m.ly = lyrics.ParseLRC("[00:00]An earlier line\n[00:01]The current line")
		m.st.Pos = 2 * time.Second
		view := m.View()
		if !strings.Contains(view, "SYNCED") {
			t.Fatal("lyrics mode is missing")
		}
		if lipgloss.Width(view) > m.w || lipgloss.Height(view) > m.h {
			t.Fatalf("full lyrics overflows %dx%d", m.w, m.h)
		}
	}
}

func TestExportSpotifyRendererPreview(t *testing.T) {
	dir := os.Getenv("SPOTIFY_RENDER_PREVIEW_DIR")
	if dir == "" {
		t.Skip("optional rendered preview")
	}
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.TrueColor)
	m := mouseModel(140, 40, 3)
	m.focus = focusLibrary
	m.librarySelection = 2
	m.st = demoState()
	cover := image.NewNRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			cover.SetNRGBA(x, y, color.NRGBA{R: uint8(40 + x/2), G: uint8(35 + y/3), B: uint8(100 + (x+y)/4), A: 255})
		}
	}
	m.art = cover
	m.artFor = m.st.Now.ID
	m.artCache = newArtCache(8)
	m.artCache.put(m.st.Now.ID, cover)
	m.recent = m.st.Queue[:3]
	m.vizBands = simulatedBands(1.7, true)
	m.st.Pos = 11 * time.Second
	m.ly = lyrics.ParseLRC("[00:00]A quieter interface for your music\n[00:05]Browse albums and playlists with a click\n[00:10]The current line stands out clearly\n[00:18]Open full-screen lyrics with l\n[00:25]Keep your focus on the song")
	m.ly.Source = "Preview"
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "home.ansi"), []byte(m.View()), 0600); err != nil {
		t.Fatal(err)
	}
	m.lyricsOpen = true
	if err := os.WriteFile(filepath.Join(dir, "lyrics.ansi"), []byte(m.View()), 0600); err != nil {
		t.Fatal(err)
	}
	m.lyricsOpen = false
	m.libraryOpen = true
	m.libraryCategory = 1
	m.sTab = 2
	m.sRes.Albums = []engine.Track{{ID: "cover-preview", Kind: "album", Title: "Cover rendering preview", Artist: "Sample metadata"}, {ID: "next-preview", Kind: "album", Title: "Another album", Artist: "Sample metadata"}}
	m.artCache.put("cover-preview", cover)
	if err := os.WriteFile(filepath.Join(dir, "library.ansi"), []byte(m.View()), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestCurrentLyricMarkerTracksTimestampAndSeek(t *testing.T) {
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.Ascii)
	m := mouseModel(140, 40, 1)
	m.ly = lyrics.ParseLRC("[00:01]Earlier line\n[00:10]Current line\n[00:20]Later line")
	m.st.Pos = 11 * time.Second
	if !strings.Contains(m.lyricsText(50, 8), "▶ Current line") {
		t.Fatal("current lyric is not marked")
	}
	m.st.Pos = 21 * time.Second
	view := m.lyricsText(50, 8)
	if !strings.Contains(view, "▶ Later line") || strings.Contains(view, "▶ Current line") {
		t.Fatal("lyric marker did not follow seek")
	}
}

func TestLibraryAlbumClickStartsSelectedAlbum(t *testing.T) {
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.Ascii)
	m := mouseModel(140, 40, 1)
	m.libraryOpen = true
	m.libraryCategory = 1
	m.sTab = 2
	m.sRes.Albums = []engine.Track{{ID: "0123456789012345678901", Title: "Clickable Album", Kind: "album"}}
	y := -1
	for i, line := range strings.Split(m.View(), "\n") {
		if strings.Contains(line, "Clickable Album") {
			y = i
			break
		}
	}
	if y < 0 {
		t.Fatal("album is not rendered")
	}
	updated, cmd := m.Update(tea.MouseMsg{X: m.w / 2, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	got := updated.(model)
	if !got.libraryOpen || !got.libraryActionBusy || got.loading != "Clickable Album" || cmd == nil {
		t.Fatal("clicking the visible album did not start it")
	}
}

func TestSearchAlbumTabCanBeClicked(t *testing.T) {
	defer lipgloss.SetColorProfile(lipgloss.ColorProfile())
	lipgloss.SetColorProfile(termenv.Ascii)
	m := mouseModel(140, 40, 1)
	m.searchOpen = true
	m.sInput = true
	x, y := -1, -1
	for i, line := range strings.Split(m.View(), "\n") {
		if at := strings.Index(line, "ALBUMS"); at >= 0 {
			x, y = lipgloss.Width(line[:at])+2, i
			break
		}
	}
	updated, _ := m.Update(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if got := updated.(model); got.sTab != 2 || got.sInput {
		t.Fatal("clicking Albums did not switch the search tab")
	}
}
