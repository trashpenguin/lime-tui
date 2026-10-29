<p align="center"><img src="assets/lime.png" alt="Lime TUI logo" width="96"></p>

# Lime TUI v1.0.1

**Your music. Your terminal.**

A Spotify terminal player for Windows, combining Pomme's Go/Bubble Tea interface with a native Rust playback engine adapted from spotatui.

[Download Windows x64](https://github.com/trashpenguin/lime-tui/releases/download/v1.0.1/LimeTUI-1.0.1-windows-x64.zip) · [Release notes](https://github.com/trashpenguin/lime-tui/releases/tag/v1.0.1) · [Changelog](CHANGELOG.md)

**Initial preview:** the binaries are unsigned. Spotify distribution authorization and broader clean-machine/hardware qualification remain unresolved. See [release readiness](RELEASE-READINESS.md) and the [Spotify integration review](SPOTIFY-DISTRIBUTION-REVIEW.md). Publishing this repository does not establish Spotify approval.

## Features

- Native Spotify playback with a local queue, albums, playlists, search and library browsing.
- Clickable Play All, Add to Queue and Next controls in the library, with progress, confirmed results and retryable errors.
- Live visualizer driven by Lime's outgoing audio samples.
- Keyboard and mouse navigation, clear focus indicators and selectable themes.
- Album artwork, including metadata fallback for missing playlist covers.
- Spotify timed lyrics with current-line highlighting and LRCLIB fallback.
- Queue editing, shuffle/repeat, seek, volume, sleep timer and mini player.
- Windows media controls and an original Lime executable/shortcut icon.
- Windows DPAPI protection for cached Spotify credentials and optional Last.fm session tokens.
- Local preferences and paused queue restoration; sign-out clears local Spotify history.
- Bounded artwork/metadata caches and a lean two-process architecture.

## Start

Extract the Windows ZIP into a folder and run **Lime.exe**, or **Start Lime.cmd**. Keep `lime-engine.exe` beside it. A Spotify **Premium** account and internet connection are required. On first launch the app opens Spotify's pairing page and displays the pairing code in the terminal. Approve the pairing yourself in your browser. No developer client ID is required, and the Spotify desktop app is not required for playback.

Windows 10/11 x64 is the supported target. If Windows reports a missing VCRUNTIME/MSVCP DLL, install Microsoft's Visual C++ x64 Redistributable. This initial preview is unsigned.

`Lime.exe --preview` prints a static sample interface without login, network requests or playback. `--help` shows the controls; `--version` shows the version.

## Controls

| Key | Action |
| --- | --- |
| `/`, then Enter | Search songs, albums, playlists and artists |
| Enter | Play/open the selected item |
| Space | Play/pause |
| `n` / `p` | Next/previous; previous restarts after the first 3 seconds |
| `a` / `A` while browsing | Append to queue / insert after current track |
| `L` | Library; Tab cycles songs, albums, playlists, artists and liked songs |
| `r` in the library | Reload the current page / retry a failed library load |
| Left/right in the player | Seek 5 seconds |
| Up/down in the player | Volume |
| `s` / `r` | Shuffle upcoming queue / repeat off → queue → track |
| Delete / Ctrl+Up / Ctrl+Down | Remove/reorder queue selection |
| Ctrl+Delete | Clear queue |
| `S` twice | Save the queue as a private Spotify playlist |
| `t` / `v` | Theme / live visualizer mode |
| `c` | Reload album covers |
| `l` / click lyrics | Full-screen lyrics; Esc returns |
| `[` / `]` / backslash | Lyric timing −0.5s / +0.5s / reset |
| `m` / `T` | Mini player / sleep timer |
| `a` outside search | Account and sign-out menu |
| `R` | Reconnect the audio engine, leaving the track paused |
| `?` | Full context-sensitive help |
| `q` / Ctrl+C | Quit |

Windows media keys use the system media session when Windows makes this player active. A handoff to another Spotify device can interrupt local playback; press R to reconnect if needed. The queue is managed by this app; editing it from another Spotify Connect client is not synchronized.

Enabling shuffle rearranges the upcoming queue once; the visible queue shows the actual playback order. Turning shuffle off keeps that order. With repeat off, playback stops at the end of the queue.

In the library, select an album or playlist and click **Play All** to replace the
queue, **Add to Queue** to append it, or **Next** to insert it after the current
track. Enter, `a` and `A` perform the same actions. The page stays open to show
progress and confirmed results; Esc returns to the player. Failed actions keep
their error visible so you can retry. Press `r` or click **Retry** after a page
load fails.

## Memory choices

- Spotify only: no other streaming sources, Lua plugins, AI DJ, browser frontend or desktop webview.
- 160 kbps native playback; audio disk caching and autoplay are disabled.
- Metadata caches capped at 256 entries per metadata type (upstream uses 4,000).
- Saved-album browsing fetches album metadata; it fetches track lists when you open/play an album.
- The frontend chooses the image nearest 256 pixels and keeps eight covers.
- Queue capped at 1,000 tracks; catalog/search pages at 50 items.
- Go's default soft memory limit is 64 MiB. It covers the Go runtime, not the Rust process, native allocations or total working set. Set `GOMEMLIMIT` before launching to override it.

The visualizer analyzes Lime’s outgoing audio with a bounded stereo FFT as the output mixer consumes samples. Paused or unavailable output is silent; device buffering and terminal rendering can add a small delay. It does not record microphone or system audio. Lyrics use Spotify first and LRCLIB as a fallback, and can be unavailable for some tracks. Lower memory than spotatui is a design goal, not a measured playback guarantee. Compare total process memory during the same playback scenario before drawing conclusions.

## Data and sessions

New-install profile: `%LOCALAPPDATA%\LimeTUI`. If that directory does not exist and `%LOCALAPPDATA%\PommeSpotifyTUI` does, Lime uses the existing directory in place, preserving login, queue and settings. It has its own settings, queue, playback position and Spotify credential cache. It does not use your Apple Music Pomme profile or installed spotatui profile. Queue/position restore is **paused** on startup. State is checkpointed every 15 seconds and at normal shutdown.

Set `LIME_DATA_DIR` (or legacy `SPOTIFY_TUI_DATA_DIR`) to a separate absolute directory to override the profile. Authentication stays in the Rust engine. Cached Spotify credentials are encrypted with Windows current-user DPAPI in `credentials.dpapi`. A legacy JSON login is removed only after encrypted storage is verified. Optional Last.fm session tokens use `lastfm.dpapi`; manual configuration files are not encrypted. Keep the profile private and do not upload it. On another Windows account/machine, signing in again may be required. DPAPI protects stored credentials against other accounts; it does not protect against software running as your Windows user. The account menu removes the cached Spotify login, queue and local Spotify search/listening history, then closes Lime. Filesystem backups/snapshots are outside that deletion. Theme/preferences remain.

To route audio through Wave Link, use the Windows per-app output setting for `lime-engine.exe`. This build follows the system default output; it does not implement Pomme's Apple backend `-Output` flag or global shortcut settings.

## Build the matching source

The source ZIP contains `frontend/` and the modified spotatui tree in `backend/`. You need Go 1.27.3+, Rust 1.90+ (tested with Rust 1.99), and Visual Studio C++ Build Tools/Windows SDK.

```powershell
.\Test.ps1
.\Build.ps1
```

Use `-GoExecutable C:\path\to\go.exe` if Go is not on PATH. `Build.ps1` creates a portable folder under `dist/`. Cargo uses the included lockfile. The application uses Spotify's internal interfaces through spotatui; service changes may require updating that backend. Rebuild the binaries after changing source.

See `VALIDATION.md` for the exact verification performed and remaining live checks, and `NOTICE.md` for attribution and upstream commit IDs.

Environment overrides: `LIME_ENGINE` selects a backend executable; `LIME_COLOR=always|never` controls RGB. Legacy `SPOTIFY_TUI_BACKEND` and `SPOTIFY_TUI_COLOR` remain supported. Lime-prefixed values take precedence.

The original Lime icon is provided as SVG, PNG and a multi-resolution ICO under `assets/`. Windows Terminal retains its own application/taskbar identity; Explorer and shortcuts to Lime.exe use the embedded Lime icon.

Optional Start Menu shortcut: right-click `Create Lime Shortcut.ps1` and choose Run with PowerShell after extraction, or run `powershell -NoProfile -ExecutionPolicy Bypass -File ".\Create Lime Shortcut.ps1"` from the extracted folder. It creates a shortcut with the embedded Lime icon and requires no administrator access. Keep the extracted folder in place; deleting the shortcut removes the Start Menu entry.

## Security and release checks

`Test.ps1 -Security -PythonExecutable python` runs frontend tests/vet, the full selected Rust library suite, strict backend lint, govulncheck and the pinned Windows dependency audit. The audit preserves raw matches and fails on unassessed findings; assessment validity is bound to the Cargo lockfile and reviewed source hashes. See DEPENDENCY-ASSESSMENTS.json, DEPENDENCY-AUDIT.json and VEX.json.

`Sign-Windows.ps1` supports publisher certificates available through Windows and verifies SHA-256 signatures/timestamps. No public signing identity is bundled. Read SIGNING.md and SPOTIFY-DISTRIBUTION-REVIEW.md; this initial release is an unsigned preview and is not Spotify-approved.

The optional `scripts/PlaybackSoak.py` requires `--allow-audio`, a cached Premium profile, a Spotify album ID, and a separate state directory. It plays at 15% volume and writes a metadata-free process-memory report. Never point tests at the real profile's state directory or distribute private test state/catalog fixtures. Native tests are opt-in with NATIVE_LIVE_TEST=1 and a valid NATIVE_LIVE_TRACK_ID; run them with temporary AMTUI_CONFIG_DIR and XDG_STATE_HOME, and the existing profile's XDG_CONFIG_HOME/XDG_CACHE_HOME. The runtime data and credentials are never included in the ZIP.

## Credits and license

Lime TUI builds on [Pomme TUI](https://github.com/trashpenguin/pomme-tui), the MIT frontend from [applemusic-tui](https://github.com/k1y0miiii/applemusic-tui), and [spotatui](https://github.com/LargeModGames/spotatui). All upstream notices remain intact. See [NOTICE.md](NOTICE.md), [LICENSE](LICENSE) and the dependency notices. The backend source retains its upstream package/module versions; the Lime application release is v1.0.1.
