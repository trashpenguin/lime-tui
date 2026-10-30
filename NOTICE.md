# Attribution

Lime TUI is an independent Spotify terminal player, not an official Spotify product or an upstream spotatui release.

- Pomme TUI: https://github.com/trashpenguin/pomme-tui, commit `0d5e7b180b2170b67d64f62eb793d743fd82ace9`. Only its MIT-licensed Go frontend is used. Its GPL Apple Music backend, Sonora and Widevine components are not included.
- The frontend originates from applemusic-tui by k1y0miiii: https://github.com/k1y0miiii/applemusic-tui. Its MIT notice remains in `LICENSE-FRONTEND.txt`. Go module/import paths are retained for build compatibility.
- spotatui: https://github.com/LargeModGames/spotatui, commit `ae9524b961c66014e40ecc9600e27486886d2970`, MIT. Its MIT notice remains in `LICENSE-SPOTATUI.txt`.
- Native streaming uses spotatui's `spotatui-librespot-*` 0.8.6 packages and their third-party dependencies. Cargo.lock and go.sum identify the exact resolved dependencies. Dependency copyright/license notices are included separately.

Local changes: cover metadata fallback, bounded retry/normalization and browsing previews; JSON-lines Spotify engine/queue adapter, Spotify pairing flow, separate profile, Windows system media controls, green default palette and Spotify branding, smaller metadata cache, on-demand album tracks, static preview/help, tests and portable packaging. No credentials, proprietary CDM binaries or account profiles are included.

Lime branding and original lime/play icon were added locally. The editable original icon assets are released under the MIT license in LICENSE-FRONTEND.txt. This name has not undergone trademark clearance. Internal bridge/module names retain upstream Pomme identifiers for compatibility.

Security changes in v1.0.0: local spotatui-librespot-core source override for Windows DPAPI credential persistence and quick-xml 0.41 with bounded/fail-fast product parsing; block-buffer update; serde_norway parser replacement; protected optional session tokens and disconnect-history cleanup. The modified core source and LIME-PATCHES.md are included in the matching source ZIP; this is not an upstream core release.

Visualizer change in v1.0.1: optional PCM observer in a local spotatui-librespot-playback 0.8.6 override, plus bounded stereo FFT in Lime. The original playback license/source and LIME-PATCHES.md are retained.

Library interaction changes in v1.0.1: clickable action controls, in-context loading and acknowledged queue/playback feedback, retryable errors and duplicate-action prevention.

History changes in v1.0.2: bounded account Web API reads, independent background refresh, explicit source labels, local fallback, server cooldown and session-generation cleanup.

Playback changes in v1.0.3: retained decoder request identities in the streaming adapter; bounded premature-start recovery in the Lime bridge; explicit failures, pause/handoff handling, and redacted opt-in support reports.

v1.0.4 modifies the vendored playback loader to retain typed failure categories through the one-shot completion channel and Unavailable event. It adds categorized diagnostics and player messages, visualizer recovery and default Windows window sizing.

v1.0.5 adds a Lime-specific official Soloist child/WebSocket adapter and setup UI. Windows job containment follows the MIT-licensed spotatui supervisor pattern. Spotify Soloist remains separately downloaded from Spotify and is not part of this distribution.
