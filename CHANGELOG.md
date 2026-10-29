# Changelog

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
