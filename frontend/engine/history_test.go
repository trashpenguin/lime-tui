package engine

import (
	"encoding/json"
	"net"
	"testing"
	"time"
)

func TestHistoryRPCMapsSpotifyMetadataAndPreservesRetryDeadline(t *testing.T) {
	for _, failed := range []bool{false, true} {
		client, server := net.Pipe()
		rpc := newRPC(client, client)
		eng := &Engine{rpc: rpc}
		go func() {
			defer server.Close()
			var request map[string]any
			if json.NewDecoder(server).Decode(&request) != nil {
				return
			}
			var err any
			if failed {
				err = "Spotify history rate-limited"
			}
			_ = json.NewEncoder(server).Encode(map[string]any{"id": request["id"], "result": map[string]any{
				"recent": []map[string]any{{"id": "fixture", "title": "Remote play", "kind": "songs", "durMs": 120000, "art": "https://example.invalid/cover"}},
				"err":    err, "retryAfterSeconds": 7200,
			}})
		}()
		page, err := eng.RecentHistory()
		rpc.close()
		if (err != nil) != failed || page.RetryAfter != 2*time.Hour {
			t.Fatal("lost error or shortened server cooldown", err)
		}
		if len(page.Tracks) != 1 || page.Tracks[0].Kind != "song" || page.Tracks[0].Duration != 2*time.Minute || page.Tracks[0].Art == "" {
			t.Fatal("Spotify history metadata lost")
		}
	}
}
