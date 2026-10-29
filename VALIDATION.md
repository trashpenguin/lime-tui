# Lime TUI 1.0.0 validation — 2026-10-29

See RELEASE-READINESS.md for the current complete test results, dependency assessments and unresolved external/hardware gates. The source used for this first GitHub release includes verified Windows DPAPI credential migration, disconnect-history erasure, full Rust/Go checks, real Spotify playback/UI/repeat/end tests and a ten-minute native playback soak. The actual user queue remained unchanged. It is unsigned and not cleared for public Spotify distribution.

v1.0.0 artifact metadata/icon, actual terminal colors/themes and a local Defender custom scan passed. Package integrity, source correspondence and credential-file exclusions are checked during final packaging.


The first public version is v1.0.0. Internal development labels were never previous GitHub releases. The native helper is unchanged from the tested hardened build; version metadata, documentation and the lyric HTTP user-agent were updated for v1.0.0. Frontend/version/package checks are repeated before upload. See RELEASE-READINESS.md for the 1,478-test Rust result, full frontend checks, live playback/repeat/end checks, ten-minute native soak and remaining qualification limits.
