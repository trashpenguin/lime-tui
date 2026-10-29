package main

import (
	"bytes"
	"context"
	"encoding/json"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/k1y0miiii/applemusic-tui/engine"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func coverPNG(c color.Color) []byte {
	var body bytes.Buffer
	_ = png.Encode(&body, solidImage(64, 64, c))
	return body.Bytes()
}

func TestCoverRetriesTransientFailureAndStopsAtLimit(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.WriteHeader(503)
			return
		}
		w.Write(coverPNG(color.RGBA{R: 210, A: 255}))
	}))
	defer server.Close()
	m := mouseModel(140, 40, 0)
	m.artCache = newArtCache(8)
	m.st.Now = engine.Track{ID: "fixture", Art: server.URL}
	m, command := m.requestCurrentArtwork()
	if command == nil {
		t.Fatal("initial cover fetch missing")
	}
	updated, _ := m.Update(command())
	m = updated.(model)
	if m.art != nil {
		t.Fatal("failed cover was accepted")
	}
	m, command = m.requestCurrentArtwork()
	if command != nil {
		t.Fatal("cover retry ignored cooldown")
	}
	state := m.coverRequests["fixture"]
	state.Next = time.Now().Add(-time.Second)
	m.coverRequests["fixture"] = state
	m, command = m.requestCurrentArtwork()
	if command == nil {
		t.Fatal("failed cover was never retried")
	}
	updated, _ = m.Update(command())
	m = updated.(model)
	if m.art == nil || requests != 2 {
		t.Fatal("retry did not restore the current cover")
	}
	m.art = nil
	m.artCache = newArtCache(8)
	m.coverRequests = map[string]coverRequest{"fixture": {URL: server.URL, Attempts: 3}}
	_, command = m.requestCurrentArtwork()
	if command != nil {
		t.Fatal("cover retry limit was exceeded")
	}
}

func TestCoverChangesWhenURLChangesForSameTrack(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(coverPNG(color.RGBA{B: 230, A: 255})) }))
	defer server.Close()
	old := solidImage(64, 64, color.RGBA{R: 220, A: 255})
	m := mouseModel(140, 40, 0)
	m.artCache = newArtCache(8)
	m.artCache.putURL("fixture", "https://old.invalid/cover", old)
	m.art = old
	m.artFor = "fixture"
	m.artSourceURL = "https://old.invalid/cover"
	m.st.Now = engine.Track{ID: "fixture", Art: server.URL}
	m, command := m.requestCurrentArtwork()
	if command == nil || m.art != nil {
		t.Fatal("changed cover URL kept stale artwork")
	}
	updated, _ := m.Update(command())
	m = updated.(model)
	_, _, blue, _ := m.art.At(0, 0).RGBA()
	if blue == 0 {
		t.Fatal("new cover was not displayed")
	}
}

func TestMetadataFallbackCoverIsCachedForBothURLs(t *testing.T) {
	cache := newArtCache(8)
	img := solidImage(8, 8, color.White)
	cache.putSourceURL("fixture", "https://old.invalid/cover", "https://new.invalid/cover", img)
	for _, url := range []string{"https://old.invalid/cover", "https://new.invalid/cover"} {
		if _, ok := cache.compatible("fixture", url); !ok {
			t.Fatal("fallback would be downloaded repeatedly")
		}
	}
	if _, ok := cache.compatible("fixture", "https://changed.invalid/cover"); ok {
		t.Fatal("genuinely changed cover reused stale artwork")
	}
}

func TestDecodedCoversAreSquareAndMemoryBounded(t *testing.T) {
	var body bytes.Buffer
	png.Encode(&body, solidImage(640, 480, color.RGBA{G: 190, A: 255}))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(body.Bytes()) }))
	defer server.Close()
	img, err := fetchArtwork(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds() != image.Rect(0, 0, 256, 256) {
		t.Fatalf("cached cover bounds: %v", img.Bounds())
	}
}

func TestLibraryAndSearchDisplaySelectedCover(t *testing.T) {
	m := mouseModel(140, 40, 0)
	m.artCache = newArtCache(8)
	m.libraryOpen = true
	m.libraryCategory = 1
	m.sTab = 2
	m.sRes.Albums = []engine.Track{{ID: "fixture", Kind: "album", Title: "Album with cover"}}
	m.artCache.put("fixture", solidImage(64, 64, color.RGBA{R: 180, G: 90, A: 255}))
	if !strings.Contains(m.libraryView(), "▀") {
		t.Fatal("library album cover is not rendered")
	}
	m.libraryOpen = false
	m.searchOpen = true
	if !strings.Contains(m.searchView(), "▀") {
		t.Fatal("search album cover is not rendered")
	}
}

func TestClickingCoverPlaysSelectedAlbumRatherThanCoveredRow(t *testing.T) {
	m := mouseModel(140, 40, 0)
	m.libraryOpen = true
	m.libraryCategory = 1
	m.sTab = 2
	m.sRes.Albums = []engine.Track{{ID: "first", Kind: "album", Title: "Selected album"}, {ID: "second", Kind: "album", Title: "Other album"}}
	box, ok := renderedBrowserBox(m.libraryView())
	if !ok {
		t.Fatal("library border missing")
	}
	updated, command := m.updateBrowserMouse(tea.MouseMsg{X: box.x + box.width - 4, Y: box.y + 7, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if got := updated.(model); got.loading != "Selected album" || command == nil {
		t.Fatal("cover click did not play its selected album")
	}
}

func TestLateCoverResponseCannotReplaceNewTrack(t *testing.T) {
	m := mouseModel(140, 40, 0)
	m.artCache = newArtCache(8)
	m.st.Now = engine.Track{ID: "new", Art: "new-url"}
	m.coverRequests = map[string]coverRequest{"old": {URL: "old-url", Pending: true, Token: 1}}
	updated, _ := m.Update(coverMsg{ID: "old", RequestedURL: "old-url", ResolvedURL: "old-url", Token: 1, Image: solidImage(2, 2, color.White)})
	if updated.(model).art != nil {
		t.Fatal("old album cover leaked into the new track")
	}
}

func TestObservedPlaysKeepRecentCoverEntriesBounded(t *testing.T) {
	m := mouseModel(140, 40, 0)
	for i := 0; i < 30; i++ {
		m.rememberPlayedCover(engine.Track{ID: string(rune('a' + i)), Art: "cover-url"})
	}
	if len(m.recent) != 20 || m.recent[0].Art != "cover-url" {
		t.Fatal("recent cover history is not bounded or loses artwork")
	}
	first := m.recent[3]
	m.rememberPlayedCover(first)
	if len(m.recent) != 20 || m.recent[0].ID != first.ID {
		t.Fatal("replaying a track did not move its cover to the front")
	}
}

func TestLiveSpotifyCoverFetchAndRender(t *testing.T) {
	input := os.Getenv("SPOTIFY_COVER_TEST_URLS")
	if input == "" {
		t.Skip("optional live cover probe")
	}
	var urls []string
	if err := json.Unmarshal([]byte(input), &urls); err != nil {
		t.Fatal(err)
	}
	for _, url := range urls {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		img, err := fetchArtwork(ctx, url)
		cancel()
		if err != nil {
			t.Fatal(err)
		}
		if img.Bounds().Dx() > 256 || img.Bounds().Dy() > 256 {
			t.Fatal("cover is not bounded")
		}
		if rows := renderArtwork(img, 24, 12); len(rows) != 12 || !strings.Contains(rows[0], "▀") {
			t.Fatal("live cover did not render")
		}
	}
}

func TestLiveSpotifyMissingCoverFallback(t *testing.T) {
	input := os.Getenv("SPOTIFY_COVER_TEST_ITEMS")
	if input == "" {
		t.Skip("optional signed-in cover probe")
	}
	var tracks []engine.Track
	if err := json.Unmarshal([]byte(input), &tracks); err != nil {
		t.Fatal(err)
	}
	eng, err := engine.Connect(func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()
	tested := false
	for _, track := range tracks {
		if track.Art == "" {
			message := fetchCoverCmd(eng, track, 1)().(coverMsg)
			if message.Err != nil || message.Image == nil || message.ResolvedURL == "" {
				t.Fatalf("metadata fallback failed: %v", message.Err)
			}
			tested = true
		}
	}
	if !tested {
		t.Fatal("probe had no missing-cover fixture")
	}
}
