# Changelog

## v1.0.3 — 2026-10-30

- Treat premature decoder/stream endings as failures rather than normal queue completion. Attempt one bounded startup reload, then show an actionable error.
- Track decoder request identities so late events from an old load of the same song cannot stop its replacement.
- Respect Pause and device handoff during recovery; Space can explicitly retry a failed track.
- Add F12 support-report export with a bounded, non-persistent event trace and no account/catalog/credential data.
- Validate ordinary starts, forced early-end recovery, one-retry limits, stale-event rejection and pause cancellation. Intermittent spontaneous failures were not reproduced in the sampled starts.


## v1.0.2 — 2026-10-30

- Add an independent Spotify account-history request, up to 50 recent tracks, with automatic refresh and H to refresh from the player.
- Label the feed SPOTIFY or LOCAL; preserve Spotify ordering and keep local plays as a separate fallback.
- Respect Retry-After with a persisted, non-personal cooldown; bound response size and clear pending account history on sign-out.
- Live account access returned HTTP 429 during validation. Remote success/order, failures and cooldown handling have local protocol/UI tests; real imported account contents remain unverified.


## v1.0.1 — 2026-10-29

- Add clickable Play All, Add to Queue and Next controls to library browsing.
- Show pending actions, confirmed queue results and persistent errors; add page retry/reload.
- Keep the library visible during playback loading, block duplicate pending actions and confirm playback from actual state.
- Replace decorative bars with a stereo FFT of audio consumed by Lime’s output mixer.
- Stop reporting synthetic motion when the live source fails; show UNAVAILABLE.
- Verify queued versus consumed samples, frequency response, pause silence and live playback.

## v1.0.0 — 2026-10-29

First GitHub upload and initial Windows x64 preview.

- Spotify native playback, search/library browsing, albums/playlists and editable queue.
- Album artwork with fallback/retries, synchronized lyrics, themes, keyboard/mouse navigation and clear panel focus.
- Windows executable branding, media controls and optional Start Menu shortcut.
- Windows DPAPI credential storage, verified legacy-login migration and sign-out data cleanup.
- Dependency updates, source-bound security assessments, support diagnostics and signing tools.
- Full frontend/Rust checks, live playback/repeat/end tests and a ten-minute native playback soak.

Known limits: unsigned binaries, unresolved Spotify distribution authorization and incomplete clean-machine/hardware qualification. This release is a preview, not a certification of consumer readiness. See RELEASE-READINESS.md.
