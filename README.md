<p align="center"><img src="assets/lime.png" alt="Lime TUI logo" width="96"></p>

# Lime TUI v1.1.1

**Your music. Your terminal.**

A Spotify terminal player for Windows, combining Pomme's Go/Bubble Tea interface with a native Rust playback engine adapted from spotatui.

[Download Windows setup](https://github.com/trashpenguin/lime-tui/releases/download/v1.1.1/LimeTUI-1.1.1-setup-x64.exe) · [Portable Windows ZIP](https://github.com/trashpenguin/lime-tui/releases/download/v1.1.1/LimeTUI-1.1.1-windows-x64.zip) · [Release notes](https://github.com/trashpenguin/lime-tui/releases/tag/v1.1.1) · [Changelog](CHANGELOG.md)

**Initial preview:** the binaries are unsigned. Spotify distribution authorization and broader clean-machine/hardware qualification remain unresolved. See [release readiness](RELEASE-READINESS.md) and the [Spotify integration review](SPOTIFY-DISTRIBUTION-REVIEW.md). Publishing this repository does not establish Spotify approval.

## Features

- Native Spotify playback with a local queue, albums, playlists, search and library browsing.
- Clickable Play All, Add to Queue and Next controls in the library, with progress, confirmed results and retryable errors.
- Spotify account-history sync when available, with explicit SPOTIFY / LOCAL labels and a local-play fallback.
- Nine live visualizer styles driven by Lime's outgoing audio samples.
- Keyboard and mouse navigation, clear focus indicators and selectable themes.
- Album artwork, including metadata fallback for missing playlist covers.
- Spotify timed lyrics with current-line highlighting and LRCLIB fallback.
- Queue editing, shuffle/repeat, seek, volume, sleep timer and mini player.
- Windows media controls and an original Lime executable/shortcut icon.
- Windows DPAPI protection for cached Spotify credentials and optional Last.fm session tokens.
- Local preferences and paused queue restoration; sign-out clears local Spotify history.
- Bounded artwork/metadata caches and a lean two-process architecture.

## Official playback for audio-key failures

If pairing works but songs fail with `audio_key` / error 14, choose Spotify Soloist in the Windows installer. For portable installs, run **Setup Soloist.cmd** or press **F8** on the player. Each user generates their own key. See [Soloist setup](SOLOIST-SETUP.md). Windows encrypts the local key settings. Pair **Lime Soloist** once in the Spotify app’s device menu. This mode keeps Lime’s catalog, queue and transport interface. Native playback remains the default.

## Start

When you have the Windows setup executable, run it and open **Lime TUI** from Start. Setup installs the app and its playback engine for your Windows user. If the Microsoft Visual C++ x64 Runtime is missing, setup downloads and verifies Microsoft's installer. Windows may ask for administrator permission for that component. You can uninstall Lime from Windows Settings. Your Spotify profile is stored separately.

Choose **Install and configure Spotify Soloist** in setup if you hit an audio-key error. Setup downloads Soloist directly from Spotify and verifies its signature. Generate your own key using the dashboard link in setup. At Lime's configuration prompt, press **Enter** to use the installed copy, then paste your key into the hidden prompt. Then open Lime and pair **Lime Soloist** from the Spotify app on the same network. The **Configure Spotify Soloist** shortcut also works after installation.

With Windows Terminal installed, double-clicking Lime opens a new 152-column × 38-row window. `Start Lime.cmd`, the Start Menu shortcut, and `Lime.exe --window` use the same size. Font and display scaling determine the pixel dimensions. Running `Lime.exe` from an existing shell keeps that window. Without Windows Terminal, Lime uses the existing console.

Extract the Windows ZIP into a folder and run **Lime.exe**, or **Start Lime.cmd**. Keep `lime-engine.exe` beside it. A Spotify **Premium** account and internet connection are required. On first launch the app opens Spotify's pairing page and displays the pairing code in the terminal. Approve the pairing yourself in your browser. No developer client ID is required, and the Spotify desktop app is not required for playback.

Windows 10/11 x64 is the supported target. If the portable build reports a missing VCRUNTIME/MSVCP DLL, install [Microsoft's Visual C++ x64 Redistributable](https://aka.ms/vc14/vc_redist.x64.exe). This initial preview is unsigned.

`Lime.exe --preview` prints a static sample interface without login, network requests or playback. `--help` shows the controls; `--version` shows the version.

## What's new in v1.1.1

Lime restores its queue and current position paused after a restart. Press **Ctrl+Z** to undo the last removal or clear and **g** to jump to the current track. The player identifies Spotify device handoffs and explains how to resume playback on this PC.

Search opens the first category with results if the selected category is empty. Returning from an album or artist restores your search position. Spotify history is opt-in for each session. Press **H** on the player or switch to Spotify Recent in Stats to load it.

**F6 Audio Quality** shows the saved preference and the preference used for this session. Soloist errors now distinguish pairing, build expiry, and playback rejection. **F12** can export a support report from the startup error screen.

## Controls

| Key | Action |
| --- | --- |
| `/`, then Enter | Search songs, albums, playlists and artists |
| Enter | Play a song or open the selected artist, album, or playlist |
| `e` on an album or playlist | Play the whole collection |
| Tab in Stats | Switch between ranked tracks, artists, and albums |
| `s` in Stats | Switch between Lime history and Spotify Recent |
| Space | Play/pause |
| `n` / `p` | Next/previous; previous restarts after the first 3 seconds |
| `a` / `A` while browsing | Append to queue / insert after current track |
| `F6` | Open Audio Quality |
| `F8` | Set up official Soloist playback or switch back to native |
| `F12` | Save a playback support report beside Lime.exe |
| `H` on the player screen | Refresh Spotify history (honors server cooldowns) |
| `L` | Open the selected Library category. Press Esc to choose another. |
| `r` in the library | Reload the current page / retry a failed library load |
| Left/right in the player | Seek 5 seconds |
| Up/down in the player | Volume |
| `s` / `r` or `R` | Shuffle upcoming queue / repeat off → queue → track |
| Delete / Ctrl+Up / Ctrl+Down | Remove/reorder queue selection |
| Ctrl+X | Clear the focused queue |
| Ctrl+Z | Undo the last remove or clear in the focused queue |
| `g` in the queue | Jump to the current track |
| `S` twice | Save the queue as a private Spotify playlist |
| `t` / `v` | Theme / live visualizer mode |
| `c` | Reload album covers |
| `l` / click lyrics | Full-screen lyrics; Esc returns |
| `[` / `]` / backslash | Lyric timing −0.5s / +0.5s / reset |
| `m` / `T` | Mini player / sleep timer |
| `a` outside search | Account and sign-out menu |
| `F5` | Reconnect the audio engine, leaving the track paused |
| `F1` / `?` | Context-sensitive help. In the search field, ? types a question mark. |
| `q` / Ctrl+C | Quit |

Press `v` to cycle bars, torus, sphere, wave, mirror, waterfall, ring, dots and
ribbon. Lime remembers the selected style; the live panel title shows its name.
All nine styles support native playback. Soloist visualization uses only Soloist's output and requires Windows build 20348 or newer.

When the phone's Spotify app selects Lime as the playback device, native playback
follows confirmed song changes, including songs outside Lime's current queue.

Windows media keys use the system media session when Windows makes this player active. A handoff to another Spotify device can interrupt local playback; press F5 to reconnect if needed. Lime restores the queue, current track, and last checkpointed position paused. Press Space to resume. During playback, Lime checkpoints the position every five seconds. Ctrl+Z can undo the last remove or clear in this session until a subsequent queue edit replaces or invalidates it.

Enabling shuffle rearranges the upcoming queue once; the visible queue shows the actual playback order. Turning shuffle off keeps that order. With repeat off, playback stops at the end of the queue.

In the library, select an album or playlist and click **Play All** to replace the
queue, **Add to Queue** to append it, or **Next** to insert it after the current
track. Enter, `a` and `A` perform the same actions. The page stays open to show
progress and confirmed results; Esc returns to the player. Failed actions keep
their error visible so you can retry. Press `r` or click **Retry** after a page
load fails.

## Playback recovery and support

A stream that ends unexpectedly during startup gets one fresh load of the same track. A repeated or later premature ending shows an error instead of silently being treated as a finished song. Normal queue endings still follow repeat settings. Pause, track changes and device handoffs cancel pending recovery; recovery does not reclaim another device’s playback. Space explicitly retries after a playback error, and F5 reconnects.

If a song still stops, press **F12** soon afterward. Lime writes **Lime-playback-report.json beside Lime.exe**. It contains build/system information and the last 64 playback event types, request counters, positions and connection flags. It excludes account names, song/artist identifiers, credentials, tokens, profile paths and raw logs. No report is uploaded automatically. Attach this file to a bug report so the failure can be investigated even if you forgot the song. The portable folder must be writable.

## Recently played

Lime requests up to 50 recent track plays from [Spotify’s account-history endpoint](https://developer.spotify.com/documentation/web-api/reference/get-recently-played). The panel shows **SPOTIFY** for account results and **LOCAL** for plays observed in Lime when the account endpoint is unavailable. Account history is opt-in for the current session. Press `H` from the player or select Spotify Recent in Stats to start it. Local Stats opens by default and shows Lime's measured listening time and streaks. Account history refreshes at most once per minute. Repeated rate limits increase the retry delay up to 15 minutes, or longer if Spotify asks for more time; `H` requests a refresh without bypassing the cache or a Spotify Retry-After deadline. It is not a full listening-history archive and does not include podcasts.

In the signed-in test for v1.0.2, Spotify returned HTTP 429, so live account results could not be verified on this account. The app retains playback, labels the local fallback and retries after the server’s cooldown. History metadata and bearer tokens stay in memory; only the rate-limit deadline is saved across restarts.

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

To also build the installer, pass the path to the Inno Setup 7 x64 compiler with the build's Windows toolchain available.

```powershell
.\Build.ps1 -SetupCompiler 'C:\path\to\ISCC.exe'
```

For an existing portable build in `dist/`, run `.\scripts\Build-Setup.ps1 -Iscc 'C:\path\to\ISCC.exe'`. The setup executable is written to `dist/`.

## Security and release checks

`Test.ps1 -Security -PythonExecutable python` runs frontend tests/vet, the full selected Rust library suite, strict backend lint, govulncheck and the pinned Windows dependency audit. The audit preserves raw matches and fails on unassessed findings; assessment validity is bound to the Cargo lockfile and reviewed source hashes. See DEPENDENCY-ASSESSMENTS.json, DEPENDENCY-AUDIT.json and VEX.json.

`Sign-Windows.ps1` supports publisher certificates available through Windows and verifies SHA-256 signatures/timestamps. No public signing identity is bundled. Read SIGNING.md and SPOTIFY-DISTRIBUTION-REVIEW.md; this initial release is an unsigned preview and is not Spotify-approved.

The optional `scripts/PlaybackSoak.py` requires `--allow-audio`, a cached Premium profile, a Spotify album ID, and a separate state directory. It plays at 15% volume and writes a metadata-free process-memory report. Never point tests at the real profile's state directory or distribute private test state/catalog fixtures. Native tests are opt-in with NATIVE_LIVE_TEST=1 and a valid NATIVE_LIVE_TRACK_ID; run them with temporary AMTUI_CONFIG_DIR and XDG_STATE_HOME, and the existing profile's XDG_CONFIG_HOME/XDG_CACHE_HOME. The runtime data and credentials are never included in the ZIP.

## Credits and license

Lime TUI builds on [Pomme TUI](https://github.com/trashpenguin/pomme-tui), the MIT frontend from [applemusic-tui](https://github.com/k1y0miiii/applemusic-tui), and [spotatui](https://github.com/LargeModGames/spotatui). All upstream notices remain intact. See [NOTICE.md](NOTICE.md), [LICENSE](LICENSE) and the dependency notices. The backend source retains its upstream package/module versions; the Lime application release is v1.1.1.
