# Lime TUI beta: data and privacy

Lime is an independent local terminal player. It is not endorsed by Spotify. This file describes this build; it is not a published operator privacy policy.

## Local data

Lime stores settings, search history, listening statistics, queue and playback position in your profile. New installs use `%LOCALAPPDATA%\LimeTUI`. Existing `%LOCALAPPDATA%\PommeSpotifyTUI` profiles are reused in place if no Lime profile exists. `LIME_DATA_DIR` overrides this location. No profile files are bundled in the release.

The Rust engine stores reusable Spotify credentials in `cache/spotatui/streaming_cache/credentials.dpapi`, protected with Windows DPAPI for the current Windows user and an application-specific purpose. Legacy JSON is migrated only after encrypted storage is written and verified. Corrupt protected credentials are rejected, with no fallback to a remaining plaintext copy. Optional Last.fm session tokens use `settings/lastfm.dpapi`. Manually supplied configuration secrets are not automatically encrypted.

Encryption protects data at rest against other Windows accounts; malware or an administrator acting as your logged-in user may still access credentials or process memory. Keep your account/profile private. Do not share cached credentials or pairing codes. Moving a profile to another account/machine can require signing in again. Removing a legacy file does not erase copies retained in filesystem backups or snapshots.

The account menu removes the cached Spotify login and clears the queue and local Spotify search/listening history. Lime closes after successful sign-out so pending work cannot restore the old account's data. Theme/preferences and independently configured Last.fm settings remain. To erase all local settings, quit Lime and remove the profile directory. Spotify account access can also be managed through Spotify's own account settings.

## Network services

Spotify pairing, catalog, library, artwork, timed lyrics and playback contact Spotify and its infrastructure. LRCLIB is a fallback for lyrics; lyric searches can send the current track title and artist to LRCLIB. Artwork requests go to the URL returned by catalog metadata. Optional Last.fm features inherited from the frontend can contact Last.fm if configured. Your IP address and requests are visible to contacted services under their own policies. Internet access and Spotify Premium are required for playback.

No analytics/telemetry uploader, advertising service or automatic crash-report uploader was added. Diagnostics are local. The --diagnose-system report contains binary hashes, platform/toolchain information and storage-presence flags; it does not read credential contents or include account names, catalog, history, tokens or profile paths. The visualizer analyzes small, bounded buffers of Lime’s outgoing audio in memory. It does not record a microphone, capture other applications or save audio. Recently Played can request up to 50 recent tracks from Spotify’s Web API using the existing account session. Account results stay in memory and are labeled SPOTIFY; when unavailable, the panel shows locally observed plays labeled LOCAL. Imported account results are not added to local listening statistics or written as a history archive. Only a rate-limit retry deadline is saved in state/spotatui/spotify-history-backoff.json. It contains no account name, track metadata or token. Disconnect clears the in-memory feed and invalidates pending refreshes. This is not a full import of Spotify history.

## Before public release

A release owner must provide a contact address, publish an appropriate privacy policy and review account disconnection/data deletion behavior against Spotify requirements. Credential-protection changes have local automated and signed-in migration tests. Independent security testing and Spotify/public-distribution approval have not been established. See SPOTIFY-DISTRIBUTION-REVIEW.md for unresolved requirements.

## Optional playback support report

F12 explicitly writes Lime-playback-report.json beside the executable. This adds up to 64 in-memory playback event types, internal numeric request counters, relative elapsed times, positions and connection/error-presence flags to the existing system report. It excludes song/artist/account identifiers, credentials, tokens, profile paths and raw native logs. The rolling trace is not written into the queue/profile and is cleared at process exit or account disconnect. Only pressing F12 exports it; no automatic uploader is added. The report overwrites the previous report of the same name. Delete the JSON file to remove an exported report.

F12 playback events can also contain a fixed load-stage label and a numeric librespot error-kind code. Neither contains the original error message, account data, identifiers, stream URLs or audio keys. Reports still require an explicit F12 export and are never uploaded automatically.
