# Spotify Connect synchronization — v1.0.6 preview, 2026-10-31

When Spotify on a phone controls Lime as its selected playback device, Lime now
follows the native player's confirmed track changes. It updates the active queue
entry, title, artist, album, cover and progress, and schedules lyrics after track
metadata arrives. Songs outside Lime's queue are added within its existing cap.

Newer decoder requests can take control; older buffered events cannot overwrite
the current selection. Spotify owns automatic advancement of a context selected
from the phone, so Lime does not start a competing local queue song at its end.

This change targets native playback with sound coming from Lime on the PC.
Following playback whose active device is the phone and Soloist synchronization
are separate paths and are not changed by this patch.

Validation: all Go packages and vet, 1,498 Rust tests (11 opt-in tests ignored),
and strict native Clippy passed. Regression tests cover phone selections in and
outside the queue, same-song reselection, stale request rejection, queue bounds,
local load ownership and waiting for metadata before fetching lyrics.

Manual verification remains: select Lime in the phone's Spotify device picker,
change to a different song and check title, cover, lyrics and progress. Try a song
outside the previous Lime queue, then let it finish to check Spotify's next song.
Included in the v1.0.6 preview release.
