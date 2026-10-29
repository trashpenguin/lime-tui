package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg" // Apple serves JPEG artwork
	_ "image/png"  // …but decode PNG too rather than fail
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// artworkPx is the fetch size, and so the ceiling on the cover's detail: a
// cover of N columns samples N pixels across and 2N down. The cover grows with
// the panel, so this has to cover a tall terminal, not just a 24-column stamp.
const artworkPx = 256

// artworkTimeout bounds the cover fetch; artwork is decoration, never a stall.
const artworkTimeout = 5 * time.Second

// artworkPlaceholder matches anything still in braces after the substitutions
// below.
var artworkPlaceholder = regexp.MustCompile(`\{[^{}]*\}`)

// artworkURL fills MusicKit's placeholders. An empty template stays empty so
// callers can treat "" as "no artwork".
//
// Apple hands out two shapes of template: `{w}x{h}bb.jpg`, with the crop code
// and format already baked in, and `{w}x{h}{c}.{f}`, where both are left to the
// caller. Filling only {w}, {h} and {f} leaves a literal {c} in the URL, and
// mzstatic answers that with 400 and a JSON body — which decodes to no image,
// so the cover silently comes up blank while the baked-in ones work.
//
// Any leftover placeholder is dropped rather than guessed at: Apple serves
// `128x128.jpg` with no crop code perfectly well, so removing an unknown one
// yields a URL that works, and a placeholder Apple adds later fails the same
// safe way instead of 400ing.
func artworkURL(template string, px int) string {
	if template == "" {
		return ""
	}
	s := strconv.Itoa(px)
	// {f} is replaced rather than dropped — losing it would leave a URL ending
	// in a bare dot.
	r := strings.NewReplacer("{w}", s, "{h}", s, "{f}", "jpg")
	return artworkPlaceholder.ReplaceAllString(r.Replace(template), "")
}

func fetchArtwork(ctx context.Context, url string) (image.Image, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("artwork: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, (4<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(body) > 4<<20 {
		return nil, fmt.Errorf("artwork exceeds 4 MiB")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 2048 || cfg.Height > 2048 {
		return nil, fmt.Errorf("artwork dimensions exceed limit")
	}
	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return normalizeArtwork(img), nil
}

// Cache one small square instead of retaining a multi-megabyte original.
func normalizeArtwork(img image.Image) image.Image {
	b := img.Bounds()
	side := min(b.Dx(), b.Dy())
	size := min(side, artworkPx)
	if size <= 0 {
		return nil
	}
	x0, y0 := b.Min.X+(b.Dx()-side)/2, b.Min.Y+(b.Dy()-side)/2
	out := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			// Pixel-center sampling with bilinear interpolation keeps small covers clear.
			fx := float64(x) * float64(side-1) / float64(max(1, size-1))
			fy := float64(y) * float64(side-1) / float64(max(1, size-1))
			xi, yi := int(fx), int(fy)
			dx, dy := fx-float64(xi), fy-float64(yi)
			c00 := color.NRGBAModel.Convert(img.At(x0+xi, y0+yi)).(color.NRGBA)
			c10 := color.NRGBAModel.Convert(img.At(x0+min(xi+1, side-1), y0+yi)).(color.NRGBA)
			c01 := color.NRGBAModel.Convert(img.At(x0+xi, y0+min(yi+1, side-1))).(color.NRGBA)
			c11 := color.NRGBAModel.Convert(img.At(x0+min(xi+1, side-1), y0+min(yi+1, side-1))).(color.NRGBA)
			blend := func(a, b, c, d uint8) uint8 {
				return uint8((float64(a)*(1-dx)+float64(b)*dx)*(1-dy) + (float64(c)*(1-dx)+float64(d)*dx)*dy + .5)
			}
			out.SetNRGBA(x, y, color.NRGBA{R: blend(c00.R, c10.R, c01.R, c11.R), G: blend(c00.G, c10.G, c01.G, c11.G), B: blend(c00.B, c10.B, c01.B, c11.B), A: blend(c00.A, c10.A, c01.A, c11.A)})
		}
	}
	return out
}

// artCache keeps the last few decoded covers so switching back and forth in a
// queue does not refetch. ponytail: insertion-order eviction, not true LRU —
// the working set is a handful of tracks.
type artCache struct {
	max     int
	order   []string
	items   map[string]image.Image
	urls    map[string]string
	aliases map[string]string
}

func newArtCache(max int) *artCache {
	return &artCache{max: max, items: map[string]image.Image{}}
}

// get and put are nil-safe: artwork is decoration, so a model built without a
// cache (tests, future entry points) just goes uncached instead of crashing.
func (c *artCache) get(key string) (image.Image, bool) {
	if c == nil {
		return nil, false
	}
	img, ok := c.items[key]
	return img, ok
}

func (c *artCache) put(key string, img image.Image) {
	if c == nil || c.max <= 0 {
		return
	}
	if img == nil {
		return
	}
	if c.items == nil {
		c.items = map[string]image.Image{}
	}
	if _, ok := c.items[key]; ok {
		c.items[key] = img
		return
	}
	if len(c.order) >= c.max {
		delete(c.items, c.order[0])
		delete(c.urls, c.order[0])
		delete(c.aliases, c.order[0])
		c.order = c.order[1:]
	}
	c.order = append(c.order, key)
	c.items[key] = img
}

func (c *artCache) compatible(key, url string) (image.Image, bool) {
	img, ok := c.get(key)
	if !ok || img == nil {
		return nil, false
	}
	resolved, known := c.urls[key]
	return img, url == "" || !known || resolved == url || c.aliases[key] == url
}
func (c *artCache) putURL(key, url string, img image.Image) {
	if c == nil || c.max <= 0 || img == nil {
		return
	}
	c.put(key, img)
	if c.urls == nil {
		c.urls = map[string]string{}
	}
	c.urls[key] = url
}

func (c *artCache) putSourceURL(key, source, resolved string, img image.Image) {
	if c == nil || c.max <= 0 || img == nil {
		return
	}
	c.putURL(key, resolved, img)
	if c.aliases == nil {
		c.aliases = map[string]string{}
	}
	c.aliases[key] = source
}

// renderArtwork draws img into w columns and h rows using the upper half block:
// the glyph's foreground paints the upper pixel and its background the lower
// one, so every cell carries two vertical pixels.
func renderArtwork(img image.Image, w, h int) []string {
	if img == nil || w <= 0 || h <= 0 {
		return nil
	}
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return nil
	}
	sample := func(cx, cy int) lipgloss.Color {
		x := b.Min.X + cx*b.Dx()/w
		y := b.Min.Y + cy*b.Dy()/(h*2)
		r, g, bl, _ := img.At(x, y).RGBA()
		return lipgloss.Color("#" +
			hex2(uint8(r>>8)) + hex2(uint8(g>>8)) + hex2(uint8(bl>>8)))
	}
	rows := make([]string, h)
	for row := 0; row < h; row++ {
		var sb strings.Builder
		for col := 0; col < w; col++ {
			top := sample(col, row*2)
			bottom := sample(col, row*2+1)
			sb.WriteString(lipgloss.NewStyle().
				Foreground(top).Background(bottom).Render("▀"))
		}
		rows[row] = sb.String()
	}
	return rows
}

// artworkMinCols is where a cover stops being worth drawing: below this the
// half-block sampling is mush, so the lyrics get the whole panel instead.
const artworkMinCols = 16

// artworkBaseCols is the size the cover always gets when the panel can fit it,
// even when the song's lines are long. Only growth past it is negotiable.
const artworkBaseCols = 24

// artworkGap is the breathing room between the cover and the lyrics. One column
// reads as the text being glued to the artwork's edge.
const artworkGap = 3

// artworkLayout sizes the cover for a lyrics panel of w columns and h rows.
// Terminal cells are roughly 1:2, so a square needs cols = 2*rows: the cover is
// drawn as tall as the panel and only shrinks to leave the lyrics the width
// they actually use — song lines are short, and a fixed cap left most of a wide
// panel empty. lyricsWant is that used width; the lyrics never get less than
// minLyrics. Returns (0, 0) when what is left over is too small to be a cover.
func artworkLayout(w, h, minLyrics, lyricsWant int) (cols, rows int) {
	if w <= 0 || h <= 0 {
		return 0, 0
	}
	if lyricsWant < minLyrics {
		lyricsWant = minLyrics
	}
	// Width the cover may take: whatever the lyrics leave over, but never less
	// than the base size — on a narrow panel one long line would otherwise
	// squeeze the cover away entirely.
	free := max(w-artworkGap-lyricsWant, artworkBaseCols)
	rows = min(h, free/2)
	cols = rows * 2
	if cols < artworkMinCols || w-cols-artworkGap < minLyrics {
		return 0, 0
	}
	return cols, rows
}

func hex2(v uint8) string {
	const digits = "0123456789ABCDEF"
	return string([]byte{digits[v>>4], digits[v&0x0F]})
}
