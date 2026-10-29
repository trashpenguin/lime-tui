# Lime TUI 1.0.3 release readiness

v1.0.3 hardens premature stream-end handling and adds F12 support reports. A stream fault is no longer silently treated as a normal song completion. Controlled real-player failure injection passed; spontaneous intermittent stops were not reproduced in the sampled seven-second starts. General fresh-machine/hardware qualification remains incomplete. See VALIDATION.md.


v1.0.2 adds conditional account-history sync and a clearly labeled local fallback. The live history request returned HTTP 429; actual account contents could not be verified. Local protocol/UI tests cover success and failure handling, and Retry-After survives restarts. This does not resolve Spotify authorization or quota eligibility. See VALIDATION.md.


v1.0.1 adds explicit library action buttons and confirmed progress/error feedback,
and replaces the decorative visualizer with output-paced stereo FFT analysis.
See VALIDATION.md for the patch's tests and updated dependency audit. Historical
v1.0.0 evidence below remains identified by version. Signing, Spotify permission
and broader hardware qualification remain unresolved.

Reviewed 2026-10-29. **Initial GitHub preview; general consumer qualification and Spotify distribution authorization remain incomplete.** The first GitHub source/preview upload is v1.0.0. No certificate/trust setting was created, and no Spotify application or agreement was submitted.

## Completed technical work

| Area | Result |
| --- | --- |
| Dependency remediation | quick-xml 0.41.0, block-buffer 0.12.1, serde_norway with maintained libyaml fork. Previously patched HTTP/2 and certificate/browser dependencies retained. |
| Dependency assessment | 350 Windows normal/build package versions queried. Two raw matches remain, both assessed not affected for this exact build. Zero unassessed findings. |
| Credential protection | Windows current-user DPAPI for cached Spotify credentials and optional Last.fm session tokens; encrypted atomic writes, verification, bounded reads, tamper rejection, serialized native writes and safe legacy migration. |
| Existing login | Actual cached login migrated successfully, protected file present, legacy plaintext file removed, account still usable. User's saved queue hash unchanged. |
| Disconnect | Removes both cached Spotify formats, queue and local search/listening history. Shared history is cleared before shutdown so it cannot be resurrected. Preferences remain. Successful sign-out closes Lime. |
| Support diagnostics | --diagnose-system writes local platform/build/storage flags and binary hashes without reading credential contents or including account/catalog/history/profile paths. |
| Signing preparation | Sign-Windows.ps1 validates an installed publisher certificate, uses SHA-256 file/RFC3161 timestamp digests, verifies signatures and refreshes checksums. Check-only path tested. Actual signing requires a real certificate. |
| Spotify review | SPOTIFY-DISTRIBUTION-REVIEW.md records the current implementation, official requirements and concrete owner options. Review completed; permission not obtained. |

## Security evidence and limits

The maintainer's GHSA-qwgh-2vcv-g2f7 affected range excludes block-buffer 0.10.4; the affected 0.12.0 copy was upgraded. rsa 0.9.10 remains in the graph, but the selected application path uses only public-key verification of Spotify's server signature, with no RSA private-key operations. This is a source/reachability assessment rather than an upstream RSA fix. DEPENDENCY-ASSESSMENTS.json and VEX.json retain the rationale and references. Assessment acceptance is bound to the lockfile and relevant source hashes, and must be repeated when they change. Raw findings are retained in DEPENDENCY-AUDIT.json.

The local core override and its modifications are included under backend/vendor/spotatui-librespot-core in matching source. See LIME-PATCHES.md. The credential boundary uses Windows DPAPI, not an app-embedded encryption key. It does not protect against malicious code running as the same logged-in user, compromise of that user's Windows account, or old filesystem backups/snapshots. Manual configuration files are not encrypted. This is not an independent penetration-test certificate.

## Tests performed

- Full selected Rust library suite: **1,478 passed, zero failed, eight ignored**. Includes eight new Windows security/parser tests. Ignored cases were not counted as passed.
- Full Go frontend/package suite and Go vet passed. Includes optional-token encryption/migration, disconnect erasure/shutdown regression, diagnostics privacy, cover/theme/focus and library navigation tests.
- Strict backend Clippy with -D warnings passed. Govulncheck 1.9.0 on Go 1.27.3 reported no vulnerabilities found.
- Final optimized engine passed 26 offline protocol requests, queue persistence, oversized-input rejection and EOF shutdown with a disposable unauthenticated profile.
- Real signed-in tests passed native playback/progress, automatic next-track playback, repeat-track restart, repeat-off stop at the queue end, pause/resume, seek, volume and paused reconnect recovery.
- Actual Go model/engine integration rendered 256x256 artwork and 46 synchronized lyric lines and left initialization correctly. LRCLIB returned a temporary 503 during an earlier probe; checks use the production Spotify-first path rather than treating a fallback outage as a playback failure.
- Read-only library checks returned 50 albums and 50 playlists; the fixtures resolved to 12 and 87 tracks. Album and initially missing playlist artwork resolved successfully.
- A **600.66-second native playback soak** passed with 2 automatic track transitions and the saved user queue unchanged. Peak native working set: **34.60 MiB**; final sample: **33.32 MiB**. This excludes frontend/process-host memory and is not a whole-app leak proof or a matched comparison with spotatui. Audio state/progress was verified; human listening quality was not certified.
- The preceding real Windows Terminal diagnostic verified RGB cover foreground/background and theme changes with inherited NO_COLOR=1 and TERM=dumb. Final v1.0.0 executable/package checks are recorded separately during packaging.

## Remaining owner/external gates

1. **Trusted signing:** neither user nor machine certificate stores contained a usable signing certificate, and the publisher confirmed they have none. Both EXEs remain unsigned. Obtain a verified publisher signing identity and use the prepared workflow; no self-signed workaround was used. Actual certificate-backed signing cannot be tested without that identity. See SIGNING.md.
2. **Spotify permission and publisher information:** the native implementation uses a desktop client identity and internal interfaces. A new developer app alone would not clear that native integration. Obtain appropriate permission or choose an authorized integration design (including evaluation of the official Windows Soloist option documented in the review), then provide real publisher/contact/privacy/EULA information and address the review's feature/branding findings. See SPOTIFY-DISTRIBUTION-REVIEW.md. No approval can be generated by a code change.
3. **Fresh-machine and hardware qualification:** this host had no available Windows Sandbox/VM runner. A clean Windows 10/11 installation, another Windows user for cross-account DPAPI behavior, additional terminals/audio devices, sleep/resume, real network/device handoff, accessibility and longer full-app memory/audio testing remain unverified. The current Windows build still requires Microsoft's Visual C++ runtime where absent. Do not claim those environments passed.

Use this preview for local evaluation. Keep the profile private, use the new build for protected storage, and verify the exact signed artifacts after the external gates are resolved. No automatic updater, installer qualification or general consumer-ready certification is claimed.

## Final artifact checks

v1.0.0 Windows version/product metadata and the full nine-size embedded icon group were verified. The actual v1.0.0 executable passed the real Windows Terminal RGB/theme diagnostic. Its local support report confirmed DPAPI storage, no legacy plaintext login, correct binary hashes and no network access. Microsoft's Defender custom scan completed on the two release EXEs with antivirus enabled, installed definition version 1.463.16.0 and zero matching detections. That is one local scanner result, not a guarantee against every threat. The dependency audit accepted the reviewed source and correctly failed when its lockfile assessment hash was invalidated.
