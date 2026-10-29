package engine

import (
	"testing"
	"time"
)

func TestLegacyQueueKindsRemainUsableAfterUpgrade(t *testing.T) {
	for original, want := range map[string]string{"songs": "song", "albums": "album", "playlists": "playlist", "artists": "artist", "song": "song"} {
		if got := (wireTrack{Kind: original}).track().Kind; got != want {
			t.Fatalf("kind %s mapped to %s", original, got)
		}
	}
}

func TestCatalogActionsHaveEnoughTimeToComplete(t *testing.T) {
	if got := methodTimeout("save_item"); got <= 30*time.Second {
		t.Fatalf("save_item timeout=%s, backend allows 30 seconds", got)
	}
	if got := methodTimeout("save_playlist"); got <= 30*time.Second {
		t.Fatalf("save_playlist timeout=%s, want time for a multi-track save", got)
	}
}
