# Lime TUI 1.0.2 validation — 2026-10-30

The history update passed the full Go suite/vet, 1,486 Rust library tests (nine ignored), and strict Clippy. Tests cover authenticated HTTP requests, Spotify order and metadata mapping, empty results, errors/local fallback, long Retry-After seconds and HTTP dates, persistent cooldown without tokens/history, malformed/oversized responses, stale session results, focused selection and disconnect cleanup. Cache entries carry their session generation so a busy cache cannot retain the previous account across logout/re-pairing. The opt-in real-history test was run separately and failed because Spotify returned HTTP 429; it is not counted as a successful account import. The app must display LOCAL until a real history request succeeds.

The following sections document earlier v1.0.1/v1.0.0 checks.


Library controls passed the full Go suite and vet, including button hit targets,
duplicate-request prevention, acknowledgement-only queue success, persistent
failure/retry, stale completion rejection and layout checks at 70x22, 100x34 and
140x45. Real signed-in model/engine tests confirmed Play All on a 12-track album
and 87-track playlist; Add to Queue then produced 24 and 174 tracks respectively.
These tests used isolated queue state, not the saved user queue.

The visualizer patch passed all Go tests/vet, 1,481 selected Rust library tests
(eight ignored), strict Clippy, and an output-consumption observer test. A real
signed-in playback test measured 60 changing spectrum frames, ten silent frames
after pause, and resumed spectrum after seeking. Unit tests verify a 1 kHz
stereo signal, opposite-phase channels, silence, bounded buffers and stale output.
The Windows dependency audit checked 356 package versions with zero unassessed
findings. The two existing advisory assessments were reviewed and bound to the
new lockfile/source hashes, including the playback override. No system audio or
microphone is captured. Device/terminal buffering still adds latency.

The evidence below is historical v1.0.0 qualification; its ten-minute soak was
not repeated for this visualizer patch.

See RELEASE-READINESS.md for the current complete test results, dependency assessments and unresolved external/hardware gates. The source used for this first GitHub release includes verified Windows DPAPI credential migration, disconnect-history erasure, full Rust/Go checks, real Spotify playback/UI/repeat/end tests and a ten-minute native playback soak. The actual user queue remained unchanged. It is unsigned and not cleared for public Spotify distribution.

v1.0.0 artifact metadata/icon, actual terminal colors/themes and a local Defender custom scan passed. Package integrity, source correspondence and credential-file exclusions are checked during final packaging.


The first public version is v1.0.0. Internal development labels were never previous GitHub releases. The native helper is unchanged from the tested hardened build; version metadata, documentation and the lyric HTTP user-agent were updated for v1.0.0. Frontend/version/package checks are repeated before upload. See RELEASE-READINESS.md for the 1,478-test Rust result, full frontend checks, live playback/repeat/end checks, ten-minute native soak and remaining qualification limits.
