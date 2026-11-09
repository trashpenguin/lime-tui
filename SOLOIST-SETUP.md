# Official Spotify Soloist playback

Use this mode when Lime signs in but tracks fail with `audio_key` / error 14.

If you run Lime's Windows installer, choose **Install and configure Spotify Soloist**. Setup downloads it directly from Spotify and checks its publisher. Use the dashboard link in setup to generate your own key. In Lime's prompt, press **Enter** to use the installed copy and enter your key. Then follow steps 4 and 5 below.

For a portable install, follow all five steps.

1. Download and extract the Windows x64 build from [Spotify](https://developer.spotify.com/documentation/soloist/reference/downloads-and-updates). Keep the extracted folder in place. Lime does not redistribute Soloist.
2. Generate your own key at [Spotify for Developers](https://developer.spotify.com/dashboard/soloist). Each user needs their own key. Keep it private.
3. Run **Setup Soloist.cmd** beside Lime.exe, or press **F8** on the player screen. Enter the full path to `soloist.exe`, then paste the key into the hidden prompt. Windows encrypts the local settings for the current user.
4. Start Lime. Open Spotify on a phone or computer on the same network and choose **Lime Soloist** in its device menu once. Allow Soloist through Windows Firewall if prompted. After pairing, Soloist can play headlessly without keeping the desktop app open.
5. Select tracks, albums or playlists in Lime. The existing queue, lyrics, artwork and transport controls use Soloist. Playback is confirmed from its events.

The key is excluded from support reports and logs. Spotify requires it on the Soloist command line while running, so other programs running as your Windows user may see it.

## Preview limitations

- Lime v1.0.8 uses Windows process audio capture for Soloist on build 20348 or newer. If capture is unavailable, Lime shows **SOLOIST · UNAVAILABLE**.
- Soloist builds expire after 90 days. Rerun Lime's Windows installer with Soloist selected to refresh the installed copy. For portable installs, replace the executable from Spotify and rerun configuration if its path changed.
- Startup and pairing state were tested with the official Windows build. Controlled tests cover queue and transport events. Playback on the affected account still requires its owner to pair and verify it.
- Device handoff cancels Lime's playback intent. Select a queue track explicitly to take control again.

To switch back, run setup and enter **native** at the path prompt. This removes the encrypted key/settings while keeping the Soloist pairing. Signing out of Lime also clears its local Soloist session/cache and Spotify history.
