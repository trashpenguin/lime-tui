# Spotatui features used and useful next steps

The app stays a terminal interface with a focused Spotify backend to keep the memory goal practical.

## Used in this build

- Native Spotify authentication, catalog and playback from spotatui's maintained librespot integration.
- Spotify timed lyrics via its color-lyrics reader, plus the existing LRCLIB fallback.
- A full-screen lyric view, timing offset controls, explicit focused panels and mouse navigation, following the interaction patterns in spotatui's terminal frontend.
- Album/playlist track resolution, a visible editable queue, saved library browsing and Windows media controls.

Reference source at the inspected upstream commit:

- [Lyrics screen](https://github.com/LargeModGames/spotatui/blob/ae9524b961c66014e40ecc9600e27486886d2970/src/tui/ui/lyrics.rs)
- [Mouse handling](https://github.com/LargeModGames/spotatui/blob/ae9524b961c66014e40ecc9600e27486886d2970/src/tui/handlers/mouse.rs)
- [Spotify lyric reader](https://github.com/LargeModGames/spotatui/blob/ae9524b961c66014e40ecc9600e27486886d2970/src/infra/spotify/lyrics.rs)
- [Playlist API mapping](https://github.com/LargeModGames/spotatui/blob/ae9524b961c66014e40ecc9600e27486886d2970/src/infra/spotify/playlists.rs)

## Best candidates for another release

1. A Spotify Connect device picker with an explicit local/remote playback indication.
2. Album/artist detail pages with a large cover, track counts and visible Play All / Queue actions.
3. In-app settings for text size/layout density, artwork, keybindings and reduced animation.
4. Implemented in v1.0.1: a real audio-reactive visualizer using a bounded stereo FFT at native output consumption. It uses no system or microphone capture. Live-source failure shows UNAVAILABLE instead of synthetic motion.

Those candidates are not claimed as implemented. Before a broad public launch, validate a fresh Windows machine, longer playback/reconnection sessions and total process memory, then arrange signed distribution.
