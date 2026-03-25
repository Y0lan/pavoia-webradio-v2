# Pavoia Webradio — Changes (week of 2026-03-24)

## New Features

### Browse & Switch Model (Phase 1-2)
- **Viewing vs Playing separation**: Clicking a stream in the sidebar now *browses* it (changes visuals, shows metadata) without switching audio. A "Switch to this stage" button explicitly switches what you're listening to.
- **Now-playing strip**: When browsing a different stream than what's playing, a strip at the top shows the currently playing track with a tap-to-return action.
- **Per-stream gradient backgrounds**: Each stage has unique gradient colors and accent theme (e.g., purple for Gaende's Favorites, teal for Bermuda Day, dark for Etage 0).
- **Cold-start state**: "Pick a stage to begin" screen when nothing is loaded yet.

### Bus Mystery Card (Phase 7)
- **Bus easter egg**: The Bus entry appears in the sidebar. Clicking it opens a full-screen overlay: "Some things must be experienced in person. The Bus is out there, somewhere at Pavoia."

### Audio Crossfade Engine (Phase 3)
- **Smooth crossfade**: Switching between streams fades out the old one and fades in the new one over 1.5 seconds using volume ramping.
- **A/B audio elements**: Two `<audio>` elements alternate to allow seamless transitions.
- **Mobile-friendly**: Works without Web Audio API (which has CORS issues with cross-origin streams).

### Stream Preview on Hover (Phase 4)
- **Hover popover**: Hovering over a stream in the sidebar (500ms delay) shows a preview with cover art, artist, and track title.
- **Bulk metadata fetch**: All stream metadata is fetched every 30 seconds (pauses when tab is hidden).

### Memory & "Was Playing" Toast (Phase 5)
- **Recently playing memory**: Persists to localStorage what was playing on each stream.
- **"Was playing" toast**: When you return to a stream, a brief toast shows what was playing before you left.

### Mobile Swipe Navigation (Phase 6)
- **Swipe between streams**: Horizontal swipe left/right on mobile navigates between stages.
- **Dead zones and exclusions**: Interactive elements (buttons, sliders) are excluded from swipe detection.

### Keyboard Shortcuts
- **Space**: Play/pause
- **Left/Right arrows**: Browse previous/next stream
- **Up/Down arrows**: Volume control
- **Enter**: Switch to the stream you're browsing
- **1-9**: Jump to stream by number
- **M**: Toggle mute

## Bug Fixes
- Fixed duplicate React import in App.jsx
- Fixed stray `}` at end of App.jsx
- Fixed gradient colors being purged by Tailwind JIT (switched to inline styles)
- Fixed play/resume logic (resume existing stream vs switch to new)
- Fixed auto-play on first stream click
- Fixed null ref in stream hover preview (captured target before setTimeout)
- Removed `crossOrigin="anonymous"` from audio elements (was silently muting cross-origin streams)
- Rewrote crossfade from Web Audio API to volume-based ramping (Web Audio mutes cross-origin audio)
- Added virtual Bus stream entry to sidebar (was missing from API response)

## Infrastructure
- **GitHub Actions deploy**: Automatic build + rsync to Whatbox on push to master
- **Crontab entries**: `@reboot` for MPD streams and Node.js server
- **MPD rebuilt from source**: Fixed broken binary after Gentoo library upgrades (libfmt 11->12)
- **9 MPD stream instances**: Per-stream configs, control ports 6600-6608, HTTP ports 14000-14008
- **Plex cover proxy**: Server-side proxy for Plex cover/artist images

## Streams
| # | Stage | Vibe |
|---|-------|------|
| 1 | Gaende's Favorites | Personal selection, all genres |
| 2 | Ambiance / Safe Space | Chill, relaxing beats |
| 3 | Bermuda Day / Oaza | Sunlit grooves |
| 4 | Bermuda Night | Progressive & Indie, sunset lift |
| 5 | Palac Feel | Melodic, hypnotic, emotional |
| 6 | Palac Dance | High energy, darker side |
| 7 | Fontanna / Laputa | After-hour house, minimal, tech house |
| 8 | Etage 0 | Hard techno, euro dance, trance |
| 9 | Closing | Beautiful closing tracks, no categories |
| 10 | Bus | Mystery — find it at Pavoia |
