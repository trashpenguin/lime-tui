# Local Lime patches to spotatui-librespot-core 0.8.6

Original upstream source and license are preserved. This directory is a local Cargo path override, not an upstream published release.

- Upgrade quick-xml to 0.41.0. Preserve prior XML text normalization explicitly. Product-info parsing has a 64 KiB bound and returns immediately on errors instead of repeatedly logging a malformed document.
- On Windows, credential cache reads/writes use current-user DPAPI and application-purpose entropy. New data is encrypted before any disk write, flushed and atomically replaced, then verified. Legacy JSON is removed only after protected storage succeeds. Corrupt protected data never falls back to JSON. Credential operations use an exclusive file handle to serialize local processes. Plaintext buffers owned by the storage implementation are zeroed after use; live session memory still necessarily contains credentials.
- Add Windows API bindings and zeroize dependencies for that storage implementation. Other operating systems retain upstream credential persistence; Lime's binary distribution targets Windows x64.

Downstream tests in src/lime_security_tests.rs exercise migration, ciphertext replacement, tampering, bounded reads, failed migration, concurrent writes, logout and XML parsing. Do not remove this override without reapplying or replacing the security fixes and running the dependency/security checks.
