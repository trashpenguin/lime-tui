# Spotify distribution review — 2026-10-29

**Decision: the current native streaming beta is not cleared for general public distribution.** The review is complete; Spotify permission has not been obtained. No application, quota extension or contract was submitted, and no agreement was accepted on the publisher's behalf. The publisher confirmed they have no Spotify developer app.

## What this build does

Code inspection confirms device pairing with Spotify's desktop client ID, internal catalog/lyrics routes and librespot native audio. Lime does not currently authenticate as its own registered application. Premium is required. This implementation detail is material to distribution; passing tests, storing credentials securely and signing EXEs would not establish service authorization.

## Findings against official requirements

- The [Developer Terms](https://developer.spotify.com/terms) require app-specific access credentials and restrict unauthorized reverse engineering/access. Using the existing desktop client identity is not evidence that Spotify authorized Lime.
- The [Developer Policy](https://developer.spotify.com/policy) covers permission for replacing core experiences, Premium streaming, commercial streaming restrictions, privacy/deletion, attribution and links to original content. Lime's native client, local listening metrics, optional metadata sharing and terminal artwork presentation need specific review before promotion. The current code is not certified compliant with these requirements.
- A public Web API integration also needs eligibility under the current [quota rules](https://developer.spotify.com/documentation/web-api/concepts/quota-modes). Registering an app alone does not establish approval for unrestricted consumers.

These are release findings and an implementation assessment, not a determination of legal rights in every jurisdiction. They must not be presented to users as Spotify certification.

## Concrete options for the owner

1. Request written permission for this native-client integration, describing the desktop-client identity, internal interfaces, pairing and audio implementation accurately. Do not submit an application claiming it uses only the public Web API.
2. Evaluate [Spotify Soloist](https://developer.spotify.com/documentation/soloist) as the first native-playback migration candidate. Spotify now documents Windows x64 support and a local control API. Each user must generate their own Soloist API key and obtain the binary from Spotify; its binary/archive must not be bundled in Lime. Builds expire after 90 days. A Soloist adapter would replace the current native pairing/audio path; it would not authorize the existing internal catalog/lyrics calls. This integration is not implemented or tested in v1.0.0, and no user-owned Soloist key was available. UI, metadata, privacy and attribution requirements still need review.
3. For public metadata/library access, plan a separate implementation with Lime's own registered application and [Authorization Code with PKCE](https://developer.spotify.com/documentation/web-api/tutorials/code-pkce-flow). Public API playback control would rely on a supported Spotify device/client; approved in-app streaming would need a supported playback integration and its requirements. That changes the current lightweight standalone-audio design and must be tested as a separate product change. A public client must not embed a client secret.
4. Supply the real publisher name, support/privacy contact and published privacy/EULA locations. Review/remove derived listening metrics and external metadata transfer where not permitted; implement the applicable branding and content-link requirements. Publish the privacy information before installation or sign-in.

## Draft description for review — not submitted

Lime TUI is a Windows terminal music interface with keyboard/mouse navigation, album artwork, library browsing, a local queue and timed lyrics. The present beta uses spotatui/librespot for native Spotify pairing/playback and internal service requests. It caches login credentials with current-user Windows DPAPI, stores preferences locally and removes local Spotify history on disconnect. LRCLIB is a fallback lyric service; optional Last.fm integration exists. There is no added advertising, payment, telemetry uploader or browser webview. Existing local metrics and terminal attribution need review. We seek guidance on an authorized distribution model and any necessary changes or written permissions.

Owner/contact fields remain unfilled because no verified publisher contact was provided. This document is the concrete review artifact; it grants no permission by itself.
