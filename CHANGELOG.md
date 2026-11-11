# Changelog

## v1.1.0 — 2026-11-12

- Add Spotify Recent to Stats. Press s to switch between Lime history and rankings from up to 50 recent Spotify plays. Listening time and streaks use local history.
- Show Spotify sync, retry times, and local fallback in Recently Played. Manual refresh honors Spotify's cooldown.
- Put album artwork beside track details in the mini player. Highlight active shuffle and repeat in both players.
- Add F6 Audio Quality with 96, 160, and 320 kbps preferences for native playback. Changes apply after restarting Lime. The Soloist panel explains how to select Lossless from Spotify Connect on a phone when available.
- Fix Escape routing when help is open over lyrics and duplicate spaces in search input. Update contextual key help.

## v1.0.8 — 2026-11-09

- Add Spotify catalog search previews and artwork cards for Library artists, albums, and playlists. Open artist songs and albums, or browse a collection before playing it with e.
- Make listening stats interactive. Play ranked tracks and browse ranked artists and albums.
- Add contextual help on F1, reserve r/R for repeat, move engine reload to F5, and clear the focused queue with Ctrl+X.
- Refresh the mini player and add a Lime Windows Terminal profile with a tab icon.
- Add a live Soloist visualizer on Windows build 20348 or newer using process-scoped audio capture.
- Keep Soloist playback on the Lime queue when Spotify attempts to start a different song at track end. With repeat off, stop at the last queued track.

## v1.0.6 — 2026-10-31

## v1.0.7 — 2026-11-09

- Add a per-user Windows x64 installer with Start menu shortcuts and an uninstaller that keeps the user profile.
- Download the Microsoft Visual C++ x64 Runtime if needed and verify its publisher before launching it.
- Offer to download and verify Spotify Soloist for accounts affected by native audio-key errors. Users supply their own Spotify Premium account and Soloist key and pair once.
- Offer the installed Soloist path in Lime's existing setup prompt. The portable package remains available.

- Add ring, dots and ribbon visualizers, bringing the total to nine. Press v to cycle styles; retain the selection across launches and show the live style in the panel title.
- Use the existing native spectrum and bounded history, with no new dependencies or audio capture. Soloist still has no audio tap.
- Follow native Spotify Connect song changes made on a phone while Lime is the playback device, including tracks outside Lime's local queue. Update metadata, artwork, lyrics and progress while rejecting stale decoder events.
- Let Spotify advance contexts selected from a phone and wait for track metadata before scheduling lyrics.
- Add regression coverage for rendering bounds, audio response, frequency response, silence, history direction, persisted styles and phone-controlled queue ownership.
- Remain an unsigned preview. Phone-driven playback requires manual verification on the user's device; broader hardware qualification and Spotify distribution authorization remain unresolved.

## v1.0.5 — 2026-10-30

- Add official Spotify Soloist playback for accounts where native librespot audio-key requests fail. Soloist is user-installed from Spotify and requires the user’s own key and initial device pairing.
- Add F8 and Setup Soloist.cmd with hidden key entry, current-user Windows DPAPI storage, a native-mode switch, and sign-out cleanup.
- Route queue starts, play/pause, seek, volume, next/previous, shuffle and repeat through the selected engine. Confirm playback using events and respect pauses and device handoffs.
- Bind Soloist control to an ephemeral loopback port, verify endpoint ownership, bound frames and command queues, redact remote errors, and terminate the owned child on shutdown or parent death.
- Label Soloist’s visualizer NO AUDIO TAP; no synthetic audio visualization is shown.
- Verify startup and pairing state with Spotify’s official Windows build and controlled transport/queue tests. Playback on the affected friend’s account awaits that user’s pairing and validation.


## v1.0.4 — 2026-10-30

- Preserve bounded track-load failure stages (metadata, availability, format, download, audio key, decode or seek) and numeric error categories in F12 reports; show a specific error in the player. Raw logs, tokens, track IDs and account data remain excluded.
- Display PLAYBACK ERROR in the visualizer heading when audio playback has failed. A connected spectrum reader no longer implies successful playback.
- This diagnostics preview does not establish or fix the cause of the reported failure on another Premium account. That PC must reproduce the problem and export a new F12 report.

- Retry the native visualizer reader after a five-second delay when a spectrum request fails, without interrupting playback.
- Open Explorer launches, the portable launcher and Start Menu shortcuts in a 152-column by 38-row Windows Terminal window; existing shell launches retain their window.
- Keep the existing console available when Windows Terminal is absent.

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
