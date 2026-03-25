# GAENDE Webradio — Ultimate Architecture Blueprint v3.1

> The definitive, exhaustive specification for a world-class webradio
> where the curation process IS the content. Multi-stage crossfading,
> deep music intelligence, Plex sync, artist graph, GitHub-style digging
> calendar, and radical transparency — everybody sees everything.
> Built for a seedbox with Podman. ~250+ features across 35 sections.

---

## PHILOSOPHY — RADICAL TRANSPARENCY

This is not a typical webradio where you press play and leave. This is the
**anti-algorithm made visible**: the listener doesn't just hear the music,
they see how it got there. They can explore the digging calendar, the artist
graph, the statistics, the curation obsession quantified. There's no curtain
between the DJ and the audience.

A listener can go from *"oh this track is good"* → click the artist → see the
similarity graph → discover 5 new artists → see that you added this track
last Tuesday at 11pm after a 14-track digging session → understand the
rabbit hole that led here.

**Every page is public.** The only protected route is `/admin` (system control).
Everything else — dashboard, digging calendar, history, discovery feed, artist
graph, stats, wrapped, downloads — is open to the world.

> *"GAENDE operates at the rare intersection of emotion and code."*

---

## TABLE OF CONTENTS

1. [High-Level Architecture](#1)
2. [Tech Stack](#2)
3. [Uninterrupted Audio — The #1 Rule](#3)
4. [About Page — The Introduction](#4)
5. [Dashboard — The Nerve Center](#5)
6. [Audio Engine](#6)
7. [Stage System](#7)
8. [Stage Transitions — Audio + Visual Sync](#8)
9. [Now Playing View](#9)
10. [Player Bar — The Persistent Anchor](#10)
11. [Visualizer System](#11)
12. [Track History System](#12)
13. [Plex Sync & Discovery Feed](#13)
14. [Digging Calendar — GitHub for Music Curation](#14)
15. [Artist Intelligence System](#15)
16. [Artist Graph Visualization](#16)
17. [Artist Detail View](#17)
18. [Statistics & Analytics — Every Lens](#18)
19. [GAENDE Wrapped — Year in Review](#19)
20. [Search — Global Unified Search](#20)
21. [Queue & Up Next](#21)
22. [Notifications & Alerts](#22)
23. [Settings & Preferences](#23)
24. [PWA & Native Feel](#24)
25. [Mobile-Specific UX](#25)
26. [Keyboard Shortcuts & Accessibility](#26)
27. [Admin Panel (only protected route)](#27)
28. [Social & Sharing](#28)
29. [Performance & Technical Details](#29)
30. [Visual Identity — The GAENDE Brand](#30)
31. [Navigation — Unified, All Public](#31)
32. [Bridge API — Complete Endpoints](#32)
33. [Database Schema — Complete](#33)
34. [Podman Compose — Full Production Stack](#34)
35. [Project Structure — Complete](#35)
36. [Feature Checklist — Exhaustive](#36)
37. [Key Decisions & Rationale](#37)

---

## 1. High-Level Architecture

```
┌──────────────────────────────────────────────────────────────────────────┐
│                       VISITORS (PWA) — everybody sees everything         │
│                                                                          │
│  Next.js 15 (App Router) + React 19 + Tailwind 4 + Framer Motion 11     │
│                                                                          │
│  ┌──────────────────────────────────────────────────────────────────┐    │
│  │ AUDIO LAYER (root layout — NEVER unmounts)                       │    │
│  │ AudioContext → 2× MediaElementSource → 2× GainNode              │    │
│  │            → 2× AnalyserNode → MasterGain → Destination          │    │
│  │ Zustand store: volume, stage, playing, crossfadeProgress         │    │
│  └──────────────────────────────────────────────────────────────────┘    │
│                                                                          │
│  ┌──────────────────────────────────────────────────────────────────┐    │
│  │ UI LAYER (swappable pages — music never stops)                    │    │
│  │ /stages  /about  /dashboard  /digging  /history  /discovery       │    │
│  │ /artists  /artists/[id]  /stats  /wrapped  /search  /queue        │    │
│  └──────────────────────────────────────────────────────────────────┘    │
│                                                                          │
│  ┌──────────────────────────────────────────────────────────────────┐    │
│  │ PLAYER BAR (fixed bottom — NEVER unmounts)                        │    │
│  └──────────────────────────────────────────────────────────────────┘    │
│                                                                          │
│  WebSocket (now-playing, events) ◄────────────┐                          │
│  SSE (listeners, status) ◄────────────────────┤                          │
│  HTTP Streams (audio) ◄──────────────────────┤                          │
└──────────────────────────────────────────────┤──────────────────────────┘
                                               │
┌──────────────────────────────────────────────┤──────────────────────────┐
│                    SERVER (Seedbox, Podman)    │                          │
│                                              │                          │
│  ┌─────────────────────────────────────────┐ │                          │
│  │  MPD Instances ×9 (ports 14001-14009)   │ │  + hidden bus stages     │
│  │  + bus stages ×N (ports 14010+)         │ │    (preview, testing)    │
│  └──────────────────┬──────────────────────┘ │                          │
│                     │                        │                          │
│  ┌──────────────────▼──────────────────────┐ │                          │
│  │          Bridge API (Go)                │ │                          │
│  │  • MPD TCP pool (idle watchers)         │ │                          │
│  │  • WebSocket hub (broadcast per stage)  │ │                          │
│  │  • REST API (all data)                  │ │                          │
│  │  • SSE (listener counts, alerts)        │ │                          │
│  │  • Plex sync worker (cron, 5min)        │ │                          │
│  │  • Artist enrichment pipeline (queue)   │ │                          │
│  │  • File server (re-listen, download)    │ │                          │
│  │  • Insights engine (automated trends)   │ │                          │
│  └─────────────────────────────────────────┘ │                          │
│                                              │                          │
│  ┌────────────┐ ┌────────────┐ ┌───────────┐│                          │
│  │ PostgreSQL │ │   Redis    │ │Meilisearch││                          │
│  │ 16         │ │   7        │ │ (search)  ││                          │
│  └────────────┘ └────────────┘ └───────────┘│                          │
│                                              │                          │
│  ┌─────────────────────────────────────────┐ │                          │
│  │  Icecast2 / MPD httpd (HTTP streams)    │ │                          │
│  └─────────────────────────────────────────┘ │                          │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Tech Stack

| Layer | Technology | Why |
|---|---|---|
| Framework | Next.js 15 (App Router) | SSR, nested layouts for persistent audio, PWA native |
| UI | React 19 + Tailwind CSS 4 + shadcn/ui | Dark-first, consistent, accessible |
| Animation | Framer Motion 11 | Stage transitions synced with crossfade, layout animations |
| Audio | Raw Web Audio API | Dual GainNode crossfade, AnalyserNode, zero deps |
| Visualization | AnalyserNode + Canvas 2D / Three.js (r128) | Real-time frequency, waveform, particles |
| Graph | react-force-graph-2d (d3-force) | Artist network, Canvas/WebGL, zoom/pan/drag |
| Charts | Recharts + custom Canvas | Stats dashboard, histograms, trends |
| Heatmap | @uiw/react-heat-map or react-activity-calendar | GitHub-style digging calendar |
| State | Zustand 5 | Lightweight, persistent audio + UI state |
| Data Fetching | TanStack Query (React Query) v5 | Cache, refetch, optimistic updates |
| Bridge API | Go 1.22+ (gorilla/websocket) | 9+ MPD conns, goroutines, Plex sync, enrichment |
| Database | PostgreSQL 16 | Relational: tracks, plays, artists, relations, sync log |
| Cache | Redis 7 | Now-playing, listener counts, API response cache |
| Search | Meilisearch | Instant fuzzy search across tracks, artists, albums |
| Streaming | Icecast2 or MPD httpd | HTTP audio streams per stage |
| PWA | Next.js native manifest.ts + Serwist | Installable, offline shell, Media Session API |
| Artist Data | Last.fm + MusicBrainz + Discogs + Wikidata + TheAudioDB | Bio, similar, tags, photos, country, labels, banners |
| Containers | Podman + podman-compose | Rootless, daemonless — runs on seedbox without root |
| Reverse Proxy | Caddy or seedbox's existing Nginx | Auto HTTPS if Caddy, or proxy_pass rules |

---

## 3. Uninterrupted Audio — The #1 Rule

The audio engine, WebSocket connection, and player bar live in the **root layout** which Next.js App Router guarantees will never unmount during navigation.

```tsx
// app/layout.tsx — NEVER unmounts
export default function RootLayout({ children }) {
  return (
    <html lang="en" className="dark">
      <body className="bg-[--bg-primary] text-[--text-primary]">
        <AudioProvider>
          <WebSocketProvider>
            <MediaSessionBridge />
            <div className="flex h-dvh flex-col">
              <Header />
              <div className="flex flex-1 overflow-hidden">
                <Sidebar className="hidden lg:flex" />
                <main className="flex-1 overflow-y-auto scroll-smooth
                                 pb-[var(--player-height)]">
                  <AnimatePresence mode="wait">
                    {children}   {/* ← ONLY this changes on navigation */}
                  </AnimatePresence>
                </main>
              </div>
              <PlayerBar />
              <MobileNav className="lg:hidden" />
            </div>
          </WebSocketProvider>
        </AudioProvider>
      </body>
    </html>
  );
}
```

**What never unmounts:** AudioContext, both HTMLAudioElement refs, GainNode/AnalyserNode graph, WebSocket, PlayerBar, Zustand stores, MediaSessionBridge.

**What swaps:** Only the `{children}` inside `<main>` — wrapped in Framer Motion `<AnimatePresence>` for page transitions.

---

## 4. About Page — `/about`

The listener's introduction. Cinematic, immersive, not a typical "about" page.

```
┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│  [Animated mesh gradient background — violet/black organic       │
│   shapes, subtly reactive to the currently playing audio         │
│   if a stage is active. Pulses with bass frequencies.]           │
│                                                                  │
│                        G A E N D E                               │
│                                                                  │
│              Clash Display, 64px, letter-spacing: 0.3em          │
│              Subtle glow in accent color                         │
│              Letter-by-letter reveal animation on load           │
│                                                                  │
│                      "Lion" in Wolof                             │
│                  Instrument Serif, 16px, italic, --text-muted    │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐      │
│  │                                                        │      │
│  │  GAENDE operates at the rare intersection of emotion   │      │
│  │  and code. Producing since age 13 and an active        │      │
│  │  builder of the underground music scene, he is known   │      │
│  │  for weaponizing his background as a software engineer │      │
│  │  to redefine music discovery.                          │      │
│  │                                                        │      │
│  │  He bypasses commercial charts to scour global         │      │
│  │  databases for what the algorithms miss: hidden        │      │
│  │  B-sides, forgotten 90s techno & house vinyls, and     │      │
│  │  obscure modern tracks. The result is a curated,       │      │
│  │  [ANTI-ALGORITHM] selection that is fresh, unique,     │      │
│  │  and impossible to replicate.                          │      │
│  │                                                        │      │
│  │  His sets flow with a dynamic clash of deep,           │      │
│  │  introspective melancholia and raw, euphoric           │      │
│  │  empowerment. Expect a narrative journey through       │      │
│  │  Progressive Driving Melodic House & Techno, sparked   │      │
│  │  with elements of Downtempo, Indie Dance, and          │      │
│  │  Hard House.                                           │      │
│  │                                                        │      │
│  │  Instrument Serif, 18px, line-height 1.8, max-w 640px │      │
│  │  [ANTI-ALGORITHM] = glitching red badge inline         │      │
│  │                                                        │      │
│  └────────────────────────────────────────────────────────┘      │
│                                                                  │
│          ┌─────────────────────────────────────┐                 │
│          │  🎧  Listen on SoundCloud            │                 │
│          │  soundcloud.com/gaende               │                 │
│          │  [Large, accent-colored button]       │                 │
│          └─────────────────────────────────────┘                 │
│                                                                  │
│          ┌─────────────────────────────────────┐                 │
│          │  ✉️  Booking & Love Letters           │                 │
│          │  gaende.ofc@gmail.com                │                 │
│          │  [Secondary button, outlined]         │                 │
│          └─────────────────────────────────────┘                 │
│                                                                  │
│  ─── The Radio ──────────────────────────────────────────────── │
│                                                                  │
│  This webradio is a window into GAENDE's personal music          │
│  library — 9 stages streaming 24/7, each a different facet       │
│  of the anti-algorithm philosophy. What you hear has been        │
│  hand-picked, not generated. Every track was found, evaluated,   │
│  and placed by a human who cares.                                │
│                                                                  │
│  Everything is open. Explore the digging calendar to see when    │
│  tracks were discovered. Dive into the artist graph to map the   │
│  connections. Check the stats to understand the taste. The       │
│  curation process is the content.                                │
│                                                                  │
│  Buenos Aires · Dakar · Nouméa · Berlin · The Internet          │
│  JetBrains Mono, 13px, --text-muted, letter-spacing 0.15em      │
│                                                                  │
│  ─── PAVOIA Festival ──────────────────────────────────────────  │
│                                                                  │
│  GAENDE is a core builder of the PAVOIA festival —               │
│  artist, promoter, and technical architect.                      │
│  [Link to PAVOIA website]                                        │
│                                                                  │
│  ─── Start Listening ────────────────────────────────────────── │
│                                                                  │
│  [Stage cards — horizontal scroll — pick a stage, start now]     │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

Design details:
- Mesh gradient reacts to bass frequencies if a stage is playing
- GAENDE title: letter-by-letter reveal, then soft glow pulse
- Bio text fades in with staggered animation (Framer Motion)
- "Anti-Algorithm" inline badge: `--anti` red (#ff3366) with subtle glitch animation
- Cities line: monospace, muted — nod to the multicultural journey
- On mobile: full-screen story-card feel (think Instagram stories, but slower)

---

## 5. Dashboard — `/dashboard`

The nerve center. For GAENDE it's self-knowledge; for listeners it's a window into the mind of the curator. Everyone sees it.

```
┌──────────────────────────────────────────────────────────────────┐
│  GAENDE's Crate                                March 25, 2026   │
│                                                                  │
│  ── Quick Pulse ──────────────────────────────────────────────── │
│                                                                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐           │
│  │ 2,841    │ │  347     │ │  14,293  │ │  196h    │           │
│  │ tracks   │ │ artists  │ │ plays    │ │ streamed │           │
│  │ +12 /wk  │ │ +3 /wk   │ │ +847/wk  │ │ +11h/wk  │           │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘           │
│                                                                  │
│  ── Digging Activity (last 12 months) ────────────────────────  │
│                                                                  │
│  [████████ GITHUB-STYLE CONTRIBUTION CALENDAR ████████████████] │
│  [████████ Colored by accent — light to saturated ████████████] │
│  [████████ Each cell = tracks added that day ████████████████] │
│                                                                  │
│  327 tracks added · Longest streak: 23 days · Current: 7 days   │
│  Most active day: March 3 — 19 tracks                           │
│                                                                  │
│  ── Today's Additions ─────────────────────────────────────────  │
│                                                                  │
│  ┌──────┐  Hyperion — Gesaffelstein              → Techno       │
│  ┌──────┐  Opus — Eric Prydz                      → Main        │
│  ┌──────┐  Innerbloom (Edu Imbernon Rmx) — RÜFÜS  → Deep        │
│  [View all →]                                                    │
│                                                                  │
│  ── What's Playing Now ────────────────────────────────────────  │
│                                                                  │
│  ● Main     Echoes of Horizon — Stephan Bodzin     23 👤        │
│  ● Techno   Acid Rain — M.I.T.A.                    8 👤        │
│  ● Ambient  Infinite — Nils Frahm                   4 👤        │
│  ● Indie    [offline]                                            │
│  ...                                                            │
│                                                                  │
│  ── Taste Snapshot ────────────────────────────────────────────  │
│                                                                  │
│  By Country   🇩🇪 25%  🇬🇧 15%  🇫🇷 10%      ← mini donut       │
│  By Decade    2020s 62%  2010s 28%              ← mini bar       │
│  By Label     Afterlife 12%  Kompakt 8%         ← mini bar       │
│  By BPM       Peak: 122 BPM                    ← sparkline      │
│  By Key       Dominant: Am (8A)                 ← mini wheel     │
│  [Explore full stats →]                                          │
│                                                                  │
│  ── Insights ──────────────────────────────────────────────────  │
│                                                                  │
│  💡 "More German artists this month (+8%)"                       │
│  💡 "BPM creeping up — from 120 avg to 124 avg"                  │
│  💡 "3 new artists entered the top 10 most-played this week"     │
│  💡 "No additions to Ambient stage in 12 days"                   │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### Automated Insights Engine

The bridge computes insights by comparing current period vs. previous:

| Insight Type | Example |
|---|---|
| Country shift | "More German artists this month (+8%)" |
| BPM drift | "BPM creeping up — from 120 avg to 124 avg" |
| New top artists | "3 new artists in your top 10" |
| Neglected stages | "No Ambient additions in 12 days" |
| Genre shift | "Indie Dance went from 8% to 15%" |
| Discovery pace | "Digging pace down 30% this month" |
| Diversity change | "Taste is getting more diverse" |
| Replay behavior | "Replaying more than discovering this week" |
| Label loyalty | "3 new labels discovered this month" |
| Decade shift | "More 90s tracks this month (+5%)" |

---

## 6. Audio Engine

### Why Not Howler.js

- `html5: true` (required for streams) bypasses Web Audio API → no `GainNode`, no `AnalyserNode`
- Can't run two streams through a shared audio graph
- No equal-power crossfade — only linear volume property
- For a multi-stage crossfader, raw Web Audio API is simpler and more powerful

### Complete Engine Class

```typescript
// lib/audio-engine.ts

type CrossfadeCurve = 'equal-power' | 'linear' | 'cut' | 's-curve' | 'exponential';

interface StageSlot {
  id: string;
  audio: HTMLAudioElement;
  source: MediaElementAudioSourceNode;
  gain: GainNode;
  analyser: AnalyserNode;
  isActive: boolean;
  streamUrl: string;
}

interface CrossfadeOptions {
  curve: CrossfadeCurve;
  durationSec: number;
  prebufferMs: number;
}

class AudioEngine {
  private ctx: AudioContext | null = null;
  private slots: [StageSlot | null, StageSlot | null] = [null, null];
  private activeSlot: 0 | 1 = 0;
  private masterGain: GainNode | null = null;
  private isMuted: boolean = false;
  private savedVolume: number = 1;
  public crossfadeProgress = { active: false, progress: 0, fromStage: '', toStage: '' };

  async init() {
    this.ctx = new AudioContext();
    this.masterGain = this.ctx.createGain();
    this.masterGain.connect(this.ctx.destination);
  }

  async resume() {
    if (this.ctx?.state === 'suspended') await this.ctx.resume();
  }

  private createSlot(stageId: string, streamUrl: string, slotIndex: 0 | 1): StageSlot {
    const prev = this.slots[slotIndex];
    if (prev) {
      prev.audio.pause();
      prev.audio.removeAttribute('src');
      prev.audio.load();
      prev.source.disconnect();
      prev.gain.disconnect();
      prev.analyser.disconnect();
    }

    const audio = new Audio();
    audio.crossOrigin = 'anonymous';
    audio.src = streamUrl;
    audio.preload = 'none';

    const source = this.ctx!.createMediaElementSource(audio);
    const gain = this.ctx!.createGain();
    const analyser = this.ctx!.createAnalyser();
    analyser.fftSize = 2048;
    analyser.smoothingTimeConstant = 0.82;
    analyser.minDecibels = -90;
    analyser.maxDecibels = -10;

    source.connect(gain);
    gain.connect(analyser);
    analyser.connect(this.masterGain!);
    gain.gain.value = 0;

    const slot: StageSlot = { id: stageId, audio, source, gain, analyser, isActive: false, streamUrl };
    this.slots[slotIndex] = slot;
    return slot;
  }

  private generateCurve(type: CrossfadeCurve, steps = 256): [Float32Array, Float32Array] {
    const fadeOut = new Float32Array(steps);
    const fadeIn = new Float32Array(steps);
    for (let i = 0; i < steps; i++) {
      const t = i / (steps - 1);
      switch (type) {
        case 'equal-power':
          fadeOut[i] = Math.cos(t * Math.PI / 2);
          fadeIn[i] = Math.sin(t * Math.PI / 2);
          break;
        case 's-curve':
          const s = t * t * (3 - 2 * t);
          fadeOut[i] = 1 - s;
          fadeIn[i] = s;
          break;
        case 'exponential':
          fadeOut[i] = Math.pow(1 - t, 2);
          fadeIn[i] = Math.pow(t, 2);
          break;
        case 'linear':
          fadeOut[i] = 1 - t;
          fadeIn[i] = t;
          break;
        default:
          fadeOut[i] = i === 0 ? 1 : 0;
          fadeIn[i] = i === 0 ? 0 : 1;
      }
    }
    return [fadeOut, fadeIn];
  }

  async switchStage(stageId: string, streamUrl: string, options: Partial<CrossfadeOptions> = {}) {
    const { curve = 'equal-power', durationSec = 3, prebufferMs = 500 } = options;
    await this.resume();

    const fromSlot = this.activeSlot;
    const toSlot: 0 | 1 = fromSlot === 0 ? 1 : 0;
    const from = this.slots[fromSlot];
    const to = this.createSlot(stageId, streamUrl, toSlot);

    this.crossfadeProgress = { active: true, progress: 0, fromStage: from?.id ?? '', toStage: stageId };
    to.gain.gain.value = 0;
    await to.audio.play();
    if (prebufferMs > 0) await new Promise(r => setTimeout(r, prebufferMs));

    const now = this.ctx!.currentTime;

    if (curve === 'cut') {
      if (from) { from.gain.gain.setValueAtTime(0, now); from.audio.pause(); from.isActive = false; }
      to.gain.gain.setValueAtTime(1, now);
      this.crossfadeProgress = { active: false, progress: 1, fromStage: '', toStage: stageId };
    } else {
      const [fadeOutCurve, fadeInCurve] = this.generateCurve(curve);
      if (from) from.gain.gain.setValueCurveAtTime(fadeOutCurve, now, durationSec);
      to.gain.gain.setValueCurveAtTime(fadeInCurve, now, durationSec);

      const startTime = Date.now();
      const interval = setInterval(() => {
        const progress = Math.min((Date.now() - startTime) / 1000 / durationSec, 1);
        this.crossfadeProgress = { active: progress < 1, progress, fromStage: from?.id ?? '', toStage: stageId };
        if (progress >= 1) clearInterval(interval);
      }, 50);

      if (from) setTimeout(() => { from.audio.pause(); from.isActive = false; }, durationSec * 1000);
    }

    to.isActive = true;
    this.activeSlot = toSlot;
  }

  setVolume(v: number) { this.savedVolume = v; if (this.masterGain && !this.isMuted) this.masterGain.gain.setTargetAtTime(v, this.ctx!.currentTime, 0.015); }
  toggleMute(): boolean { this.isMuted = !this.isMuted; if (this.masterGain) this.masterGain.gain.setTargetAtTime(this.isMuted ? 0 : this.savedVolume, this.ctx!.currentTime, 0.015); return this.isMuted; }
  play() { this.slots[this.activeSlot]?.audio.play(); }
  pause() { this.slots[this.activeSlot]?.audio.pause(); }
  togglePlayPause() { const s = this.slots[this.activeSlot]; if (s) s.audio.paused ? s.audio.play() : s.audio.pause(); }
  getActiveAnalyser(): AnalyserNode | null { return this.slots[this.activeSlot]?.analyser ?? null; }
  getBothAnalysers(): [AnalyserNode | null, AnalyserNode | null] { return [this.slots[0]?.analyser ?? null, this.slots[1]?.analyser ?? null]; }
  getActiveStageId(): string | null { return this.slots[this.activeSlot]?.id ?? null; }
  isPlaying(): boolean { return this.slots[this.activeSlot] ? !this.slots[this.activeSlot]!.audio.paused : false; }
  destroy() { this.slots.forEach(s => { if (s) { s.audio.pause(); s.source.disconnect(); } }); this.ctx?.close(); }
}
```

### Crossfade Curves

| Curve | Math | Feel | Duration | Best For |
|---|---|---|---|---|
| **Equal-power** | cos/sin | Constant loudness, smooth | 2-5s | Default, melodic transitions |
| **S-curve** | smoothstep | Cinematic, slow start/end | 3-8s | Dramatic stage reveals |
| **Exponential** | t² / (1-t)² | Fast attack, slow tail | 2-4s | Energetic transitions |
| **Linear** | t / 1-t | Volume dip mid-fade | 1-3s | Quick switches |
| **Cut** | instant | DJ hard-cut | 0s | Same-BPM, hard drops |

### Audio Quality Options

| Quality | Bitrate | Codec | Mountpoint | Use Case |
|---|---|---|---|---|
| High | 320 kbps | MP3 | `/stage-techno-320` | WiFi, desktop |
| Medium | 192 kbps | MP3 | `/stage-techno-192` | Mobile data |
| Low | 128 kbps | MP3 | `/stage-techno-128` | Poor connection |
| Lossless | FLAC | OGG/FLAC | `/stage-techno-flac` | Audiophile |

Auto-switches based on `navigator.connection.effectiveType` if "Auto" is selected.

---

## 7. Stage System

### Configuration (config.yaml — 9 stages + hidden bus)

```yaml
stages:
  - id: main
    name: "Main Stage"
    description: "Progressive melodic techno, the heart of PAVOIA"
    port: 14001
    stream_port: 18001
    genre: "Progressive Melodic Techno"
    subgenres: ["melodic techno", "progressive house", "indie dance"]
    color: "#a855f7"
    color_secondary: "#7c3aed"
    gradient: "from-violet-950 via-violet-950/50 to-black"
    icon: "🟣"
    bpm_range: [118, 128]
    visible: true
    order: 1

  - id: techno
    name: "Techno Bunker"
    description: "Raw, industrial, uncompromising"
    port: 14002
    stream_port: 18002
    genre: "Techno"
    subgenres: ["industrial techno", "hard techno", "acid"]
    color: "#ef4444"
    color_secondary: "#dc2626"
    gradient: "from-red-950 via-zinc-950 to-black"
    icon: "🔴"
    bpm_range: [128, 145]
    visible: true
    order: 2

  - id: ambient
    name: "Ambient Garden"
    description: "Downtempo, ambient, electronica. Breathe."
    port: 14003
    genre: "Ambient / Downtempo"
    color: "#14b8a6"
    gradient: "from-teal-950 via-cyan-950/30 to-black"
    bpm_range: [70, 110]
    visible: true
    order: 3

  - id: indie
    name: "Indie Floor"
    description: "Indie dance, nu-disco, feel-good grooves"
    port: 14004
    genre: "Indie Dance / Nu-Disco"
    color: "#f59e0b"
    gradient: "from-amber-950 via-orange-950/30 to-black"
    bpm_range: [110, 125]
    visible: true
    order: 4

  - id: bass
    name: "Bass Cave"
    description: "Deep bass, UK garage, breakbeat"
    port: 14005
    genre: "Bass / UK Garage"
    color: "#d946ef"
    gradient: "from-fuchsia-950 via-purple-950/30 to-black"
    bpm_range: [130, 174]
    visible: true
    order: 5

  - id: live
    name: "Live Stage"
    description: "Live sets, DJ mixes, uninterrupted long-form"
    port: 14006
    genre: "Live / DJ Sets"
    color: "#10b981"
    gradient: "from-emerald-950 via-green-950/30 to-black"
    bpm_range: [0, 999]
    visible: true
    order: 6

  - id: chill
    name: "Chill Terrace"
    description: "Organic house, afro house, balearic"
    port: 14007
    genre: "Organic House / Afro"
    color: "#0ea5e9"
    gradient: "from-sky-950 via-blue-950/30 to-black"
    bpm_range: [115, 126]
    visible: true
    order: 7

  - id: deep
    name: "Deep Room"
    description: "Deep house, minimal, micro house"
    port: 14008
    genre: "Deep House / Minimal"
    color: "#6366f1"
    gradient: "from-indigo-950 via-slate-950/30 to-black"
    bpm_range: [118, 128]
    visible: true
    order: 8

  - id: world
    name: "World Sounds"
    description: "Global beats, Middle Eastern, African, Asian electronica"
    port: 14009
    genre: "World / Global Bass"
    color: "#f43f5e"
    gradient: "from-rose-950 via-pink-950/30 to-black"
    bpm_range: [100, 140]
    visible: true
    order: 9

  # Hidden bus stages
  - id: bus-preview
    name: "Preview Bus"
    port: 14010
    visible: false

  - id: bus-recording
    name: "Recording Bus"
    port: 14011
    visible: false
```

### Stage Card Elements

Each card shows: stage art (rounded-xl), name (Clash Display), genre (Satoshi, muted), 🟢 LIVE pulsing dot, listener count badge, current track artist — title (truncated), mini waveform preview (4-5 bars animating), colored border matching accent, hover: scale(1.02) + "Switch to this stage" tooltip, active: glowing ring + "NOW PLAYING" badge.

### Stage Selector Layouts
- **Grid** (desktop default): 3-column
- **Horizontal scroll** (mobile default): single row, swipeable
- **List** (compact): minimal rows
- Toggle via small icon in header

---

## 8. Stage Transitions — Audio + Visual Sync

### Visual Transitions (synced with audio crossfade)

1. **Background gradient morph** — CSS custom property animation between stage gradients
2. **Album art crossfade** — hero art fades/blurs out while new fades/sharpens in
3. **Blurred backdrop transition** — 80px blur, 30% opacity album art morphs between covers
4. **Accent color sweep** — all `var(--accent)` elements transition (buttons, glows, scrollbar)
5. **Visualizer blend** — both analysers' data blended (weighted by crossfade progress)
6. **Stage indicator animation** — sidebar active dot slides with spring animation
7. **Player bar metadata** — track info crossfades with text slide-up

### Crossfade Progress Indicator

Subtle horizontal bar at viewport top showing crossfade progress, accent color transitioning from old to new stage. Only visible during active crossfade.

---

## 9. Now Playing View — `/stage/[id]`

### Desktop Layout

Hero album art (360×360, rounded-2xl, shadow-2xl), blurred backdrop (80px blur, 25% opacity), track title (Clash Display, 28px), artist name (Satoshi, clickable → `/artists/[id]`), album · year, metadata badges in pills (BPM, Camelot key, musical key, genre, label — JetBrains Mono), full-width visualizer (120px tall), 🟢 LIVE indicator + listener count, "Recently on this stage" horizontal scroll of last ~15 tracks, queue preview (next 3-5 tracks), stage info (genre, BPM range, track count, unique artists, total hours of content), fullscreen toggle.

### Per-Track Metadata Badges

| Badge | Style | Example |
|---|---|---|
| BPM | `bg-white/5 font-mono` | `122 BPM` |
| Camelot Key | `bg-white/5 font-mono` | `8A` |
| Musical Key | parentheses | `(Am)` |
| Genre | `bg-[--accent]/10 text-[--accent]` | `melodic techno` |
| Label | `bg-white/5` | `Afterlife` |
| Year | plain, muted | `2017` |
| Duration | plain, muted | `7:42` |

---

## 10. Player Bar — The Persistent Anchor

### Desktop (80px, glass morphism)

Elements left to right: album art thumbnail (56×56, rounded-lg, click → Now Playing), track info (title, artist — album, stage with colored dot), mini visualizer (80×40px), transport controls (◀◀ play/pause ▶▶), volume slider (horizontal, logarithmic, with mute icon toggle), additional controls on hover (crossfade curve selector, queue toggle, fullscreen).

### Mobile Mini (64px)

Art (40px), track + artist + stage, play/pause, volume. Tap anywhere → expand to full-screen. Swipe left/right → switch stage with haptic. Crossfade progress bar at bottom edge.

### Mobile Full-Screen (swipe up from mini)

Full album art (85vw), large controls, volume slider, visualizer, stage chips (horizontal scroll), 🟢 LIVE + listener count. Swipe down to collapse.

---

## 11. Visualizer System

### 8 Types Available

| Type | Render | Best For |
|---|---|---|
| Frequency Bars | Canvas 2D | Default, works everywhere |
| Mirrored Bars | Canvas 2D | Player bar, compact |
| Circular Waveform | Canvas 2D | Now Playing hero, vinyl-like |
| Waveform Line | Canvas 2D | Minimal, ambient stages |
| Mesh Gradient | CSS/WebGL | Background atmosphere |
| Particle Field | Three.js | Fullscreen immersive |
| Spectrum Ring | Canvas 2D | Compact, elegant |
| Blob | SVG/Canvas | Ambient, organic |

Auto-selected per stage by default; user-configurable. Crossfade blending merges both analysers' data during transitions. Canvas pauses when tab is hidden.

### Visualizer Settings

Type, color (stage accent / custom / rainbow), sensitivity, smoothing, bar count, show/hide, fullscreen mode, react to (all / bass only / mids / highs).

---

## 12. Track History System

### Database

```sql
CREATE TABLE track_plays (
  id            BIGSERIAL PRIMARY KEY,
  stage_id      TEXT NOT NULL,
  title         TEXT NOT NULL,
  artist        TEXT NOT NULL,
  album         TEXT,
  album_artist  TEXT,
  bpm           SMALLINT,
  camelot_key   TEXT,
  musical_key   TEXT,
  genre         TEXT,
  label         TEXT,
  year          SMALLINT,
  duration_sec  INT,
  cover_url     TEXT,
  file_path     TEXT,
  file_format   TEXT,
  played_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  ended_at      TIMESTAMPTZ,
  artist_id     BIGINT REFERENCES artists(id),
  day_of_week   SMALLINT GENERATED ALWAYS AS (EXTRACT(DOW FROM played_at)) STORED,
  hour_of_day   SMALLINT GENERATED ALWAYS AS (EXTRACT(HOUR FROM played_at)) STORED
);
```

### History View Features

**Filters:** Stage, time range (presets + custom date picker), artist (type-ahead), genre, BPM range (slider), Camelot key (wheel/dropdown), free text search.

**Sort:** Newest/oldest, by artist, by BPM, by key, most played.

**View modes:** Timeline (grouped by day), Grid (album art cards), Table (sortable columns), Calendar heatmap (plays per day).

**Per-track actions:** ▶ Re-listen (overlay mini-player, doesn't interrupt live stream), ↓ Download, 🔗 Copy link, 👤 Go to artist, ℹ️ Track details modal.

**History stats sidebar:** Total tracks/hours (today/week/month), top track of the period, current streak, stage breakdown.

---

## 13. Plex Sync & Discovery Feed

### Sync Mechanism

Background Go worker polls Plex every 5 minutes, diffs playlists against last snapshot, inserts new tracks into `plex_sync_log`, syncs to MPD playlists.

### Database

```sql
CREATE TABLE plex_sync_log (
  id                BIGSERIAL PRIMARY KEY,
  plex_playlist_id  TEXT NOT NULL,
  plex_playlist_name TEXT NOT NULL,
  stage_id          TEXT NOT NULL,
  title             TEXT NOT NULL,
  artist            TEXT NOT NULL,
  album             TEXT,
  plex_rating_key   TEXT,
  bpm               SMALLINT,
  camelot_key       TEXT,
  genre             TEXT,
  year              SMALLINT,
  duration_sec      INT,
  file_path         TEXT,
  cover_url         TEXT,
  added_to_plex_at  TIMESTAMPTZ,
  synced_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  synced_to_mpd     BOOLEAN DEFAULT false,
  artist_id         BIGINT REFERENCES artists(id)
);
```

### Discovery Feed (`/history/added`)

Chronological feed grouped by day. Each entry: album art, track title, artist, album, which Plex playlist → which stage, relative time. Filters: stage/playlist, time range, artist, genre. Stats: tracks added this week/month/year, discovery velocity sparkline, most-added artist, new artists (first appearance), most active playlist. Actions per track: ▶ Listen, ↓ Download, 👤 Go to artist, 🔍 Find on stage, 🎵 Show play history for this track. Sync status indicators: ✅ Synced, ⏳ Pending, ❌ Failed, 🆕 New artist.

---

## 14. Digging Calendar — GitHub for Music Curation

A full-page GitHub-style contribution calendar showing music digging activity — visible to everyone. Each cell = tracks added to Plex playlists that day. For you it's a mirror of your obsession. For listeners it's proof that a human is behind every track.

### Layout

```
┌──────────────────────────────────────────────────────────────────┐
│  Digging Calendar                                    2026 ▼     │
│                                                                  │
│  [████████████████ CONTRIBUTION CALENDAR ████████████████████████]│
│  [Each cell = tracks added that day, colored light→saturated]    │
│                                                                  │
│  ░ 0   ▒ 1-2   ▓ 3-5   █ 6-10   █ 11+                         │
│                                                                  │
│  ── Summary ──────────────────────────────────────────────────── │
│  327 tracks added · 89 new artists · 34 new labels              │
│  Current streak: 7 days · Longest: 23 days                      │
│  Most active day: March 3 (19 tracks)                           │
│  Average: 4.2 tracks/day on active days                         │
│  Active days: 78 of 84 (93%)                                    │
│                                                                  │
│  ── Click Any Day ─────────────────────────────────────────────  │
│                                                                  │
│  March 3, 2026 — 19 tracks added                                │
│  ┌──────┐  Hyperion — Gesaffelstein → Techno Bunker             │
│  ┌──────┐  Acid Rain — M.I.T.A. → Techno Bunker                 │
│  ┌──────┐  Opus — Eric Prydz → Main Stage                       │
│  ... (16 more) [View all →]                                     │
│                                                                  │
│  ── Digging Patterns ──────────────────────────────────────────  │
│                                                                  │
│  Day of week:                                                    │
│  Mon ██████████░░  22%   Fri ██████████░░  20%                  │
│  Sat ████░░░░░░░░   6%   Sun ██░░░░░░░░░░   4% ← less weekends │
│                                                                  │
│  Time of day:                                                    │
│  Evening (18-24) ████████████████░░░░  40%  ← peak digging      │
│  Afternoon (12-18) ████████████░░░░░░  30%                      │
│  Morning (6-12) ██████░░░░░░░░░░░░░░  15%                       │
│  Night (0-6) ██████░░░░░░░░░░░░░░░░  15%                        │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### Calendar Color Modes

| Mode | Cell Color = | Best For |
|---|---|---|
| **Volume** (default) | Track count (accent gradient) | Overall intensity |
| **Stage** | Dominant stage that day | Where you're feeding |
| **Genre** | Dominant genre added | Taste patterns |
| **Decade** | Dominant decade of tracks | Vintage vs. modern |
| **Country** | Dominant artist country | Geographic patterns |

### Streaks & Records

Current streak, longest streak, best week, best month, dry spells (with humor: "Your longest drought was 8 days in July. Vacation or just listening?"), year selector for multi-year comparison.

### Digging Sessions

Cluster additions within the same hour/day into sessions: "March 3 evening: 14 tracks in 2 hours, mostly Techno, triggered by a Stephan Bodzin rabbit hole."

---

## 15. Artist Intelligence System

### 5-Source Enrichment Pipeline

| Source | Data | Auth | Rate Limit |
|---|---|---|---|
| **Last.fm** | Bio, tags, similar (with match score), listener stats | API key (free) | 5/sec |
| **MusicBrainz** | MBID, country, type, aliases, relationships, URL rels | User-Agent | 1/sec |
| **Discogs** | Profile/bio, images, labels, styles | Token (free) | 60/min |
| **Wikidata** | Wikipedia URL, image, birth/death, website, genres | None | Generous |
| **TheAudioDB** | Artist thumb/banner/logo/fanart, biography | Free tier | Generous |

### Merge Strategy

Bio: longest non-empty (Last.fm > Discogs > TheAudioDB > Wikidata). Image: TheAudioDB thumb > Discogs > Wikidata Commons. Banner: TheAudioDB (for detail page header). Tags: union of all sources. Country: MusicBrainz > Wikidata > Discogs. Links: aggregate from MusicBrainz url-rels + Discogs sites.

### Database (complete artist schema)

```sql
CREATE TABLE artists (
  id                BIGSERIAL PRIMARY KEY,
  name              TEXT NOT NULL UNIQUE,
  name_sort         TEXT,
  mbid              UUID UNIQUE,
  discogs_id        INT,
  lastfm_url        TEXT,
  bio_short         TEXT,
  bio_full          TEXT,
  bio_source        TEXT,
  country           TEXT,
  country_name      TEXT,
  artist_type       TEXT,
  formed_year       SMALLINT,
  ended_year        SMALLINT,
  disambiguation    TEXT,
  aliases           TEXT[],
  members           JSONB,
  member_of         TEXT[],
  image_thumb       TEXT,
  image_banner      TEXT,
  image_logo        TEXT,
  image_fanart      TEXT,
  image_gallery     TEXT[],
  tags              TEXT[],
  genres            TEXT[],
  moods             TEXT[],
  lastfm_listeners  INT,
  lastfm_playcount  INT,
  labels            TEXT[],
  labels_current    TEXT[],
  website           TEXT,
  wikipedia_url     TEXT,
  spotify_url       TEXT,
  bandcamp_url      TEXT,
  soundcloud_url    TEXT,
  instagram_url     TEXT,
  ra_url            TEXT,
  track_count       INT DEFAULT 0,
  album_count       INT DEFAULT 0,
  play_count        INT DEFAULT 0,
  first_played_at   TIMESTAMPTZ,
  last_played_at    TIMESTAMPTZ,
  dominant_stage    TEXT,
  enriched_at       TIMESTAMPTZ,
  enrichment_status TEXT DEFAULT 'pending',
  created_at        TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE artist_relations (
  id              BIGSERIAL PRIMARY KEY,
  artist_id       BIGINT NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
  related_id      BIGINT REFERENCES artists(id) ON DELETE CASCADE,
  related_name    TEXT NOT NULL,
  relation_type   TEXT NOT NULL,
  weight          REAL DEFAULT 1.0,
  source          TEXT NOT NULL,
  in_library      BOOLEAN DEFAULT false,
  UNIQUE(artist_id, related_name, relation_type)
);
```

---

## 16. Artist Graph Visualization

### Library: react-force-graph-2d

Canvas/WebGL rendering, d3-force physics. Node size = `sqrt(trackCount)`, color = dominant stage accent, circular artist photos as nodes. Labels appear at zoom > 1.5×. Link directional particles for ambient animation.

### Interactions

Click node → artist detail. Hover → highlight neighborhood + tooltip (name, tracks, stage, genre). Right-click → context menu. Drag → reposition. Double-click → zoom to ego graph. Scroll/pinch → zoom.

### Filters & Controls

Filter by: stage, genre, country, label, min track count (slider), min similarity (slider), relation type. Color by: dominant stage / country / genre / play count / decade formed. Size by: track count / play count / album count / connections. Layout: force-directed / radial / clustered by genre. Search within graph: type to highlight + zoom. Reset view button.

### Graph Stats Panel

Total nodes/edges, average connections, most connected artist (degree centrality), best "bridge" artist (betweenness centrality), number of clusters, isolated artists.

---

## 17. Artist Detail View — `/artists/[id]`

### Layout

Banner image (TheAudioDB, or stage gradient fallback, 240px). Artist photo (120×120, rounded-full). Name (Clash Display), metadata (type, country, since year, active/ended). Tags as pills. Labels. Last.fm listeners/scrobbles. External links row (website, Spotify, Bandcamp, SoundCloud, Instagram, RA). Expandable bio (Instrument Serif, 18px).

### 6 Tabs

**Tracks** — All tracks in library by this artist. Sort: alphabetical, BPM, key, year, most played, recently added/played. Filter: stage. Each row: art, title, album, duration, BPM, key, play count, last played, ▶ and ↓ buttons.

**Albums** — Albums grouped by release. Cover, title, year, label, track count in library (of total), total plays. ▶ Play all.

**Stats** — Stage presence bars (stage name + bar + count). Total plays, listen time, most played track, most played month, avg plays/track, first/last appeared. BPM range visualization. Key distribution. Play heatmap (7×24 grid: when do you listen to this artist?). Monthly play timeline sparkline.

**History** — Same as main History view, pre-filtered to this artist.

**Similar** — Artists in your library with similarity scores. Artists NOT in library (recommendations). Match percentage, track count, dominant stage.

**Timeline** — Chronological events: first track added, more tracks added, first play, added to new stage, most-played day, album added, milestones.

---

## 18. Statistics & Analytics — Every Lens

### `/stats` — Full Dashboard (public)

#### Overview Cards (top row)

Total Tracks (+this month), Total Artists (+this month), Total Albums, Total Listen Time, Total Plays, Unique Genres, Stages Active.

#### Top Lists (multiple ranking modes)

**Top Artists** by: track count, play count, play frequency (plays/track), recent additions, connection count (graph). **Top Tracks** by: most played (all time / month / week / per stage), longest, highest/lowest BPM. **Top Albums** by total play count. **Top Labels** by track count.

Display: numbered list with rank, photo, name, count, spark mini-bar, change indicator (↑↓NEW—).

#### By Stage — Distribution

Interactive bar chart or treemap: each stage as colored segment. Track count, % of total, unique artist count. Click stage → filter all stats.

#### By BPM — Histogram

X: 5-BPM bins. Y: track count. Colored by dominant stage per bin. Vertical line at average. Hover bar → example tracks. Filter by stage.

#### By Key — Camelot Wheel

Interactive circular viz: 24 segments (12 major + 12 minor). Size = track count. Click key → filter, highlight compatible keys. Harmonic mixing paths shown.

#### By Country — Geographic Spread

World map or ranked list with flags. Sized/colored by artist count. Click country → filter. Insight: "78% European, 12% North American..."

#### By Decade / Year

Stacked bar chart: decades × track count. Click decade → expand to year-by-year breakdown with specific tracks. "Your 90s digging focuses on early German techno and French touch — the forgotten vinyls the bio promises."

#### By Label

Treemap: box size = track count. Top labels ranked with artist count. New labels this month. Label loyalty metric ("68% of your library is on labels you've collected 5+ tracks from").

#### By Genre — Tag Cloud + Evolution

Weighted tag cloud (size = track count, color = stage). Stacked area chart over time showing genre proportion shifts month to month. "In January, melodic techno was 45%. By March, indie dance grew to 25%."

#### Listening Calendar Heatmap

GitHub-style 52×7 grid for the year. Color intensity = plays that day. Click cell → jump to history. Streaks: current, longest, most plays in single day.

#### Time-of-Day Heatmap

7×24 grid (day × hour). Color = play count. "Peak listening: Friday 22:00-02:00." Filter by stage: "Techno peaks Saturday midnight, Ambient peaks Sunday morning."

#### Discovery Velocity

Line chart: tracks added per week (last 52 weeks). Trend line, annotations for spikes, quarter comparison, weekly average.

#### Genre Evolution Over Time

Stacked area chart: genre proportions per month. Shows taste shifts.

#### Track Length Distribution

Histogram: duration bins (1-3min, 3-5, 5-7, 7-10, 10-15, 15+). "Average: 6:42. 34% over 7 minutes."

#### Diversity Index

Shannon diversity index on play distribution. Higher = broader listening. Trend over time. "Your diversity score is 0.82 (high)."

#### Replay Value

Most-replayed tracks (play count / days since first play). One-hit wonders (played only once).

#### Stage Transitions — Sankey Diagram

Flow diagram: stage → stage transitions. "Main → Techno: 34%, Techno → Ambient: 12%." Average time on a stage before switching.

#### Crossover Analysis

Artists on 3+ stages (versatile). Artists on exactly 1 stage (specialists). "Most versatile: Stephan Bodzin (3 stages). Most loyal genre: D&B (100% Bass Cave)."

#### Curator Insights (computed, visible to everyone)

Set compatibility (harmonic % per stage), energy curve (BPM arc of playlist), freshness score (% rotation added in last 30 days), artist concentration risk, cross-pollination score (stage overlap), genre coherence per stage.

#### Filters (apply to entire dashboard)

Time range (all time / year / month / week / custom). Stage. Genre (multi-select). Country (multi-select).

---

## 19. GAENDE Wrapped — Year in Review

### Annual Wrapped (December)

Animated card sequence (Framer Motion): total listening time, top 5 tracks, top 5 artists, top stage, stage journey (animated race chart), BPM personality, key signature, discovery stats, artist graph snapshot, listening calendar heatmap, peak moment, new artists discovered, shareable summary card.

### Monthly Recap (1st of each month)

Top track, top artist, top stage. New additions count. Listening hours. One insight.

### Shareable

Beautiful PNG card generation for social sharing. ANTI-ALGORITHM watermark badge.

---

## 20. Search — Global Unified Search

Meilisearch-powered. **⌘K / Ctrl+K** to open. Instant, fuzzy, typo-tolerant. Searches: tracks (title, artist, album, genre, label), artists (name, aliases, tags, country, bio), albums, labels. Category tabs. Recent searches. Click track → play, click artist → navigate.

---

## 21. Queue & Up Next

Shows upcoming tracks for current stage (from MPD playlist). Now playing + up next list. Queue stats (count, total duration, avg BPM). Drag to reorder, swipe/click to remove, click to skip to track.

---

## 22. Notifications & Alerts

| Event | Display |
|---|---|
| New track on active stage | Player bar update (standard) |
| New Plex additions | Badge on Discovery tab + toast |
| New artist in library | "🆕 New artist: [name] → [stage]" toast |
| Listener milestone | "🎉 Main Stage: 100 listeners!" toast |
| Stream error | "⚠️ Reconnecting..." with auto-retry |
| Stage online/offline | Stage card status update |

Configurable: enable/disable each type, toast position/duration, desktop push (PWA), "Do Not Disturb" mode.

---

## 23. Settings & Preferences

**Audio:** Crossfade duration (0-10s), curve, prebuffer, stream quality (auto/manual), volume normalization, resume last stage on load.

**Visualizer:** Type, color, sensitivity, smoothing, bar count, show/hide, fullscreen bg, frequency band.

**Appearance:** Theme (Dark / OLED Black / Midnight Blue), accent (per-stage auto / custom), sidebar mode, stage selector layout, album art shape, reduce motion, font size, compact mode.

**Notifications:** Per-type toggles, toast settings, desktop push, sound.

**Data:** Export history (CSV), export artists (JSON), export stats (PDF report).

**Advanced:** WebSocket reconnect interval, audio context latency hint, debug mode (graph inspector, FPS, WS log).

---

## 24. PWA & Native Feel

```typescript
// app/manifest.ts
export default function manifest(): MetadataRoute.Manifest {
  return {
    name: 'GAENDE Radio — Anti-Algorithm Webradio',
    short_name: 'GAENDE',
    description: 'Multi-stage live webradio with full curation transparency',
    start_url: '/',
    display: 'standalone',
    background_color: '#08070d',
    theme_color: '#a855f7',
    orientation: 'any',
    categories: ['music', 'entertainment'],
    icons: [
      { src: '/icons/icon-192.png', sizes: '192x192', type: 'image/png' },
      { src: '/icons/icon-512.png', sizes: '512x512', type: 'image/png' },
      { src: '/icons/maskable-512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
    ],
    shortcuts: [
      { name: 'Main Stage', url: '/stage/main' },
      { name: 'Digging Calendar', url: '/digging' },
      { name: 'Artist Graph', url: '/artists' },
      { name: 'Stats', url: '/stats' },
    ],
  };
}
```

Media Session API: lock screen controls (play/pause, prev/next stage), album art, track metadata. Background audio on all platforms.

---

## 25. Mobile-Specific UX

Gestures: swipe up/down on player (expand/collapse), swipe left/right (switch stage with haptic), long press album art (save/share), pull to refresh, pinch on graph. Haptic feedback on stage switch, play/pause, volume endpoints. Adaptive layout: bottom tab bar < 1024px, mini player 64px, album art 85vw.

---

## 26. Keyboard Shortcuts

| Key | Action |
|---|---|
| `Space` | Play / Pause |
| `←` / `→` | Previous / Next stage |
| `↑` / `↓` | Volume ±5% |
| `M` | Mute |
| `F` | Fullscreen |
| `⌘K` / `Ctrl+K` | Search |
| `1-9` | Jump to stage |
| `Esc` | Close modal/search |
| `?` | Shortcut cheat sheet |

Accessibility: ARIA labels, focus styles (accent ring), screen reader track announcements, reduced motion mode, semantic HTML, WCAG AA contrast.

---

## 27. Admin Panel — `/admin` (only protected route)

Token auth from `.env`. Stage management (start/stop MPD, toggle visibility, edit metadata). Queue management. Plex sync (force sync, error log, playlist mapping). Enrichment (queue status, re-enrich, merge duplicates). System monitoring (WS clients, active streams, CPU/memory/disk, PG stats, Redis memory, Meilisearch index).

---

## 28. Social & Sharing

Share current track (deep link with OG preview card showing album art + track + GAENDE branding). Share stage. Share stats/Wrapped card (PNG). Copy track info to clipboard. Open Graph meta tags for rich social previews.

---

## 29. Performance

**Frontend:** Virtual scrolling (TanStack Virtual), lazy images with blur placeholders, multi-size album art (56/128/256/512px), optimistic UI, code splitting, service worker caching, prefetch on visible links, canvas pauses when tab hidden, WebSocket reconnection with exponential backoff.

**Backend:** Persistent TCP to all MPD instances, MPD idle loop (zero CPU when nothing changes), PostgreSQL connection pool (pgx), Redis for all hot data, Meilisearch indexed on insert/update, rate-limited enrichment worker.

---

## 30. Visual Identity — The GAENDE Brand

### Design Philosophy

> *"intersection of emotion and code"* → Warm organic gradients + engineering precision
> *"Anti-Algorithm"* → Handcrafted feel, not template-generated. Asymmetric layouts, unexpected typography.
> *"deep melancholia and raw empowerment"* → Dark contemplative tones + bursts of saturated energy
> *"Progressive Driving"* → UI feels directional, forward-moving. Transitions have momentum.

### Color Palette

```css
:root {
  /* Base: near-black with violet undertone */
  --bg-primary: #08070d;
  --bg-secondary: #0f0e17;
  --bg-elevated: #161525;
  --bg-card: #1a1929;
  --bg-card-hover: #222137;
  --bg-glass: rgba(10, 10, 10, 0.85);

  /* Text: warm, not clinical */
  --text-primary: #e8e6f0;
  --text-secondary: #9b97b0;
  --text-muted: #5a5672;

  /* Accent: dynamic per stage */
  --accent: #a855f7;
  --accent-hover: #9333ea;
  --accent-glow: #a855f720;

  /* The Anti-Algorithm red */
  --anti: #ff3366;

  /* Borders */
  --border-subtle: rgba(255, 255, 255, 0.06);
  --border-default: rgba(255, 255, 255, 0.1);

  /* Status */
  --live: #22c55e;
  --error: #ef4444;

  /* Glass */
  --glass-blur: 20px;
  --glass-border: rgba(255, 255, 255, 0.06);
}
```

### Typography

| Usage | Font | Weight | Size |
|---|---|---|---|
| Title / GAENDE | Clash Display | 700 | 36-64px |
| Section headers | Clash Display | 600 | 24px |
| Track titles | Satoshi | 500 | 16-28px |
| Body / artist names | Satoshi | 400 | 14-20px |
| Bio / editorial | Instrument Serif | 400 | 18px |
| Metadata (BPM, key) | JetBrains Mono | 400 | 12-13px |
| Badges / pills | JetBrains Mono | 500 | 10-12px |

### The ANTI-ALGORITHM Badge

Recurring micro-element: `font-mono, 10px, uppercase, letter-spacing 0.15em, bg: --anti, rounded-[4px] (angular, not rounded-full), subtle glitch animation every 4s`. Appears on: About page, Discovery feed header, Digging Calendar, Wrapped cards.

### Micro-Details

- Loading: pulsing waveform animation (heartbeat that comes alive)
- Empty states: *"Nothing here. Time to dig."*
- 404: *"This frequency doesn't exist. Try a different stage."*
- First visit: GAENDE letters assemble one-by-one, then stage cards fade in (1.5s total)
- Scrollbar: thin, accent-colored, barely visible until hovered
- Selection: accent color with opacity
- Focus rings: accent-colored, slightly glowing

---

## 31. Navigation — Unified, All Public

### Desktop Sidebar

```
┌──────────────────┐
│  G A E N D E     │
│  ───────────     │
│                  │
│  🏠 Stages       │
│  ℹ️ About        │
│                  │
│  ── The Crate ── │
│  📊 Dashboard    │
│  ⛏️ Digging      │
│  📜 History      │
│  ✨ Discovery    │
│                  │
│  ── The Map ──── │
│  🕸️ Artists      │
│  📈 Stats        │
│  🎁 Wrapped      │
│                  │
│  ── Controls ─── │
│  📋 Queue        │
│  🔍 Search       │
│  ⚙️ Settings     │
│                  │
│  ─────────────── │
│  🔊 23 listening │
│  🟢 9/9 online   │
└──────────────────┘
```

### Mobile Bottom Bar

```
┌─────────────────────────────────────┐
│  🏠     ⛏️      🕸️     📈     ℹ️   │
│ Stages  Dig   Artists Stats About  │
└─────────────────────────────────────┘
+ hamburger for: Dashboard, History, Discovery,
  Queue, Wrapped, Search, Settings
```

### All Routes (public except /admin)

```
/                    → Stage selector
/stage/[id]          → Now Playing
/about               → Bio, SoundCloud, booking
/dashboard           → Nerve center
/digging             → Contribution calendar
/history             → Tabs: Played / Added
/artists             → Graph + list
/artists/[id]        → Detail (6 tabs)
/stats               → Full dashboard
/wrapped             → Year in review
/wrapped/[year]      → Specific year
/search              → Results
/queue               → Up next
/settings            → Preferences
/admin               → PROTECTED: system control
```

---

## 32. Bridge API — Complete Endpoints

### Stages
```
GET  /api/stages                          → All visible stages + now-playing
GET  /api/stages/:id                      → Single stage
GET  /api/stages/:id/now-playing          → Current track
GET  /api/stages/:id/queue                → Upcoming tracks
GET  /api/stages/:id/art                  → Album art (proxied)
GET  /api/stages/:id/history?page=&per_page=  → Recent plays
GET  /api/stages/:id/stats                → Stage-specific stats
```

### History
```
GET  /api/history?stage=&from=&to=&artist=&artist_id=&genre=&bpm_min=&bpm_max=&key=&search=&sort=&page=&per_page=&view=
GET  /api/history/:id
GET  /api/history/calendar?year=
GET  /api/history/heatmap?stage=
```

### Tracks
```
GET  /api/tracks/:id/stream       → Range-request streaming
GET  /api/tracks/:id/download     → Content-Disposition: attachment
GET  /api/tracks/:id/details      → Full metadata
```

### Discovery
```
GET  /api/discovery?stage=&playlist=&from=&to=&artist=&search=&sort=&page=&per_page=
GET  /api/discovery/stats
GET  /api/discovery/new-artists
GET  /api/discovery/sync-status
POST /api/discovery/sync          → Force sync (admin)
```

### Digging
```
GET  /api/digging/calendar?year=                   → Calendar heatmap data
GET  /api/digging/calendar/:date                   → Tracks added on specific date
GET  /api/digging/streaks                          → Current/longest/records
GET  /api/digging/patterns                         → Day-of-week, time-of-day distributions
GET  /api/digging/sessions?date=                   → Clustered digging sessions
GET  /api/digging/calendar?year=&color_by=stage    → Calendar with stage coloring
```

### Artists
```
GET  /api/artists?sort=&country=&genre=&label=&stage=&min_tracks=&search=&page=&per_page=
GET  /api/artists/:id
GET  /api/artists/:id/tracks?sort=&stage=
GET  /api/artists/:id/albums
GET  /api/artists/:id/history
GET  /api/artists/:id/similar?in_library=
GET  /api/artists/:id/timeline
GET  /api/artists/:id/stats
POST /api/artists/:id/enrich      → Force re-enrich (admin)
```

### Graph
```
GET  /api/graph?min_tracks=&min_weight=&stage=&genre=&relation_type=
GET  /api/graph/stats             → Centrality, clusters, bridges
```

### Statistics
```
GET  /api/stats/overview
GET  /api/stats/top-artists?by=&limit=&stage=&period=
GET  /api/stats/top-tracks?by=&limit=&stage=&period=
GET  /api/stats/top-albums
GET  /api/stats/top-labels
GET  /api/stats/stages
GET  /api/stats/bpm?stage=&bins=
GET  /api/stats/keys?stage=
GET  /api/stats/geography
GET  /api/stats/genres
GET  /api/stats/labels
GET  /api/stats/decades?stage=
GET  /api/stats/listening-heatmap?stage=
GET  /api/stats/calendar-heatmap?year=
GET  /api/stats/discovery-velocity
GET  /api/stats/genre-evolution
GET  /api/stats/diversity
GET  /api/stats/track-length
GET  /api/stats/replay-value
GET  /api/stats/stage-transitions
GET  /api/stats/streaks
GET  /api/stats/crossover-artists
GET  /api/stats/curator-insights
```

### Wrapped
```
GET  /api/wrapped/:year
GET  /api/wrapped/monthly/:yearmonth
GET  /api/wrapped/:year/card        → PNG image
```

### Search
```
GET  /api/search?q=&type=all|tracks|artists|albums|labels&limit=
```

### WebSocket
```
WS   /ws
     Client → { type: "subscribe", stages: ["main","techno"] }
     Server → { type: "now_playing", stage, data }
     Server → { type: "listeners", data }
     Server → { type: "stage_status", stage, status }
     Server → { type: "new_track_added", data }
```

### SSE
```
GET  /api/events
     event: listeners     data: { "main": 23, "techno": 8 }
     event: sync_update   data: { stage, new_tracks }
     event: enrichment    data: { artist, status }
```

---

## 33. Database Schema — Additional Tables

(See §12, §13, §15 for track_plays, plex_sync_log, artists, artist_relations.)

```sql
CREATE TABLE library_tracks (
  id              BIGSERIAL PRIMARY KEY,
  file_path       TEXT NOT NULL UNIQUE,
  title           TEXT NOT NULL,
  artist          TEXT NOT NULL,
  album           TEXT,
  genre           TEXT,
  label           TEXT,
  year            SMALLINT,
  bpm             SMALLINT,
  camelot_key     TEXT,
  duration_sec    INT,
  file_format     TEXT,
  stage_id        TEXT,
  artist_id       BIGINT REFERENCES artists(id),
  added_at        TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE plex_playlist_snapshots (
  id                BIGSERIAL PRIMARY KEY,
  plex_playlist_id  TEXT NOT NULL,
  track_keys        TEXT[] NOT NULL,
  snapshot_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE wrapped_data (
  id              BIGSERIAL PRIMARY KEY,
  period          TEXT NOT NULL,
  data            JSONB NOT NULL,
  generated_at    TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE user_preferences (
  id    BIGSERIAL PRIMARY KEY,
  key   TEXT NOT NULL UNIQUE,
  value JSONB NOT NULL
);
```

---

## 34. Podman Compose — Full Production Stack

Since the seedbox has no root access, all containers run with **Podman** (rootless, daemonless, drop-in Docker replacement).

```yaml
# podman-compose.yml (or docker-compose.yml — both work)

services:
  mpd-main:
    image: docker.io/vimagick/mpd
    volumes: [./music/main:/music:ro, ./mpd/main.conf:/etc/mpd.conf:ro]
    ports: ["14001:6600", "18001:8000"]
    restart: unless-stopped

  mpd-techno:
    image: docker.io/vimagick/mpd
    volumes: [./music/techno:/music:ro, ./mpd/techno.conf:/etc/mpd.conf:ro]
    ports: ["14002:6600", "18002:8000"]
    restart: unless-stopped

  mpd-ambient:
    image: docker.io/vimagick/mpd
    volumes: [./music/ambient:/music:ro, ./mpd/ambient.conf:/etc/mpd.conf:ro]
    ports: ["14003:6600", "18003:8000"]
    restart: unless-stopped

  # ... repeat for indie, bass, live, chill, deep, world (×9 total)

  icecast:
    image: docker.io/infiniteproject/icecast
    ports: ["8000:8000"]
    volumes: [./icecast.xml:/etc/icecast.xml:ro]
    restart: unless-stopped

  postgres:
    image: docker.io/library/postgres:16-alpine
    environment:
      POSTGRES_DB: gaende
      POSTGRES_USER: gaende
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes: [pgdata:/var/lib/postgresql/data]
    ports: ["15432:5432"]
    restart: unless-stopped

  redis:
    image: docker.io/library/redis:7-alpine
    ports: ["16379:6379"]
    volumes: [redisdata:/data]
    restart: unless-stopped

  meilisearch:
    image: docker.io/getmeili/meilisearch:v1.7
    environment:
      MEILI_MASTER_KEY: ${MEILI_KEY}
      MEILI_ENV: production
    volumes: [meilidata:/meili_data]
    ports: ["7700:7700"]
    restart: unless-stopped

  bridge:
    build: ./apps/bridge
    ports: ["3001:3001"]
    environment:
      - DATABASE_URL=postgres://gaende:${DB_PASSWORD}@postgres:15432/gaende
      - REDIS_URL=redis://redis:16379
      - MEILI_URL=http://meilisearch:7700
      - MEILI_KEY=${MEILI_KEY}
      - LASTFM_API_KEY=${LASTFM_API_KEY}
      - DISCOGS_TOKEN=${DISCOGS_TOKEN}
      - PLEX_URL=${PLEX_URL}
      - PLEX_TOKEN=${PLEX_TOKEN}
      - ADMIN_TOKEN=${ADMIN_TOKEN}
      - MUSIC_BASE_PATH=/music
    volumes:
      - ./apps/bridge/config.yaml:/config/config.yaml:ro
      - ./music:/music:ro
    depends_on: [postgres, redis, meilisearch]
    restart: unless-stopped

  web:
    build: ./apps/web
    ports: ["3000:3000"]
    environment:
      - NEXT_PUBLIC_BRIDGE_WS=wss://${DOMAIN}/ws
      - NEXT_PUBLIC_BRIDGE_API=https://${DOMAIN}/api
      - NEXT_PUBLIC_STREAM_BASE=https://${DOMAIN}/stream
    depends_on: [bridge]
    restart: unless-stopped

volumes:
  pgdata:
  redisdata:
  meilidata:
```

### Podman-Specific Notes

- **No privileged ports**: All > 1024
- **Full registry paths**: `docker.io/library/postgres:16-alpine` (Podman requires explicit registry)
- **Running**: `podman-compose up -d --build` or `podman compose up -d --build` (Podman 4+)
- **Volume permissions**: `podman unshare chown` if needed
- **Auto-start without root**: Generate systemd user units:
  ```bash
  podman generate systemd --new --name gaende-bridge > ~/.config/systemd/user/gaende-bridge.service
  systemctl --user enable gaende-bridge.service
  loginctl enable-linger $USER
  ```
- **Reverse proxy**: Use seedbox's existing Nginx with `proxy_pass` rules, or add Caddy container
- **Storage**: Mount seedbox's large disk for music — read-only volumes

---

## 35. Project Structure — Complete

```
gaende-radio/
├── apps/
│   ├── web/                              # Next.js 15
│   │   ├── app/
│   │   │   ├── layout.tsx                # Root: Audio + WS + PlayerBar (NEVER unmounts)
│   │   │   ├── page.tsx                  # / — Stage selector
│   │   │   ├── about/page.tsx            # About GAENDE
│   │   │   ├── dashboard/page.tsx        # Nerve center
│   │   │   ├── digging/page.tsx          # Contribution calendar
│   │   │   ├── stage/[id]/page.tsx       # Now Playing
│   │   │   ├── history/
│   │   │   │   ├── layout.tsx            # Tabs container
│   │   │   │   ├── played/page.tsx       # Play history
│   │   │   │   └── added/page.tsx        # Discovery feed
│   │   │   ├── artists/
│   │   │   │   ├── page.tsx              # Graph + list
│   │   │   │   └── [id]/page.tsx         # Artist detail (6 tabs)
│   │   │   ├── stats/page.tsx            # Full dashboard
│   │   │   ├── wrapped/
│   │   │   │   ├── page.tsx
│   │   │   │   └── [year]/page.tsx
│   │   │   ├── search/page.tsx
│   │   │   ├── queue/page.tsx
│   │   │   ├── settings/page.tsx
│   │   │   ├── admin/page.tsx            # Protected
│   │   │   └── manifest.ts
│   │   ├── components/
│   │   │   ├── audio/                    # AudioProvider, AudioEngine, MediaSession
│   │   │   ├── player/                   # PlayerBar, MobileFullPlayer, Volume, PlayPause
│   │   │   ├── stage/                    # StageSelector, StageCard, StageTransition
│   │   │   ├── visualizer/              # 8 types + Mini + Fullscreen
│   │   │   ├── now-playing/             # NowPlaying, AlbumArt, Backdrop, Meta, Listeners
│   │   │   ├── about/                   # AboutHero, Bio, Links, StagePreview
│   │   │   ├── dashboard/              # QuickPulse, DiggingMini, TasteSnapshot, Insights
│   │   │   ├── digging/                # DiggingCalendar, DayDetail, Streaks, Patterns
│   │   │   ├── history/                # PlayHistory, DiscoveryFeed, TrackRow, Filters
│   │   │   ├── artists/                # Graph, GraphControls, ArtistDetail, 6 tab components
│   │   │   ├── stats/                  # 20+ chart/viz components
│   │   │   ├── wrapped/               # WrappedStory, Cards, ShareCard
│   │   │   ├── search/                # SearchModal, Results, Suggestions
│   │   │   ├── queue/                 # QueueView, QueueItem, QueueStats
│   │   │   ├── notifications/         # ToastProvider, Toasts
│   │   │   ├── layout/               # Sidebar, Header, MobileNav, TabSwitcher
│   │   │   └── shared/               # ContextMenu, Tooltip, Modal, Slider, Badge, etc.
│   │   ├── lib/                       # audio-engine, crossfade, ws-client, api-client, etc.
│   │   ├── hooks/                     # useAudioEngine, useMediaSession, useNowPlaying, etc.
│   │   ├── stores/                    # player-store, ui-store, settings-store (Zustand)
│   │   ├── types/                     # TypeScript interfaces
│   │   └── styles/                    # globals.css, fonts.css
│   │
│   └── bridge/                        # Go bridge
│       ├── main.go
│       ├── config/
│       ├── mpd/                       # client, pool, watcher
│       ├── plex/                      # client, sync worker
│       ├── enrichment/               # pipeline, lastfm, musicbrainz, discogs, wikidata, theaudiodb
│       ├── insights/                 # automated trend detection
│       ├── handlers/                 # ws, stages, history, artists, discovery, digging, stats, etc.
│       ├── search/                   # Meilisearch indexing
│       ├── db/migrations/
│       ├── models/
│       └── config.yaml
│
├── db/init.sql
├── mpd/                              # 9 stage configs
├── podman-compose.yml
├── .env.example
├── Makefile
└── README.md
```

---

## 36. Feature Checklist — Exhaustive (~260 features)

### Core Audio (10)
- [ ] Web Audio API engine, dual slots, 5 crossfade curves
- [ ] Equal-power, S-curve, exponential, linear, cut
- [ ] Master GainNode volume, mute toggle
- [ ] AudioContext resume on user gesture
- [ ] Pre-buffering before crossfade
- [ ] Crossfade progress tracking (exposed to UI)
- [ ] Audio quality auto-selection by connection type
- [ ] Stream reconnection on error
- [ ] Multiple quality mountpoints (128/192/320/FLAC)
- [ ] Volume normalization option

### Uninterrupted Playback (5)
- [ ] Audio engine in root layout
- [ ] PlayerBar in root layout
- [ ] WebSocket in root layout
- [ ] Zustand state persists across navigation
- [ ] AnimatePresence on {children} only

### About Page (6)
- [ ] Animated mesh gradient background (audio-reactive)
- [ ] GAENDE title with letter-reveal animation
- [ ] Bio with Instrument Serif, staggered fade-in
- [ ] ANTI-ALGORITHM inline badge with glitch animation
- [ ] SoundCloud CTA + booking email
- [ ] PAVOIA festival link + cities line

### Dashboard (8)
- [ ] Quick Pulse cards (tracks, artists, plays, hours + deltas)
- [ ] Mini digging calendar (12 months)
- [ ] Today's additions feed
- [ ] All stages now-playing overview
- [ ] Taste snapshot (mini charts: country, decade, label, BPM, key)
- [ ] Automated insights (10 types)
- [ ] Link to full stats
- [ ] Active listeners total

### Stage System (8)
- [ ] 9 visible stages + hidden bus stages
- [ ] Full config per stage (name, genre, subgenres, color, BPM range, cover)
- [ ] Stage cards with live preview, listener count, mini waveform
- [ ] 3 layout modes (grid, list, horizontal scroll)
- [ ] Active stage indicator (sidebar + player bar)
- [ ] Stage info panel (track count, artist count, duration)
- [ ] Stage ordering
- [ ] Online/offline status

### Stage Transitions (7)
- [ ] Audio crossfade synced with visual
- [ ] Background gradient morph
- [ ] Album art + blurred backdrop crossfade
- [ ] Accent color CSS animation
- [ ] Visualizer blend during transition
- [ ] Crossfade progress indicator bar
- [ ] Stage indicator slide animation

### Now Playing (12)
- [ ] Hero album art + blurred backdrop
- [ ] Track/artist/album metadata
- [ ] BPM, key, genre, label badges
- [ ] Full visualizer
- [ ] Live indicator + listener count
- [ ] Recently played scroll
- [ ] Queue preview
- [ ] Stage info
- [ ] Fullscreen toggle
- [ ] Artist name clickable → artist page
- [ ] Album art hover: scale(1.02)
- [ ] Back to stages nav

### Player Bar (14)
- [ ] Art thumbnail, track info, stage indicator
- [ ] Mini visualizer
- [ ] Play/Pause, Prev/Next stage
- [ ] Volume slider (log), mute toggle
- [ ] Crossfade curve selector
- [ ] Queue toggle, fullscreen toggle
- [ ] Glass morphism
- [ ] Desktop 80px, mobile mini 64px, mobile full (swipe-up)
- [ ] Swipe left/right to switch stage (mobile)
- [ ] Double-click to expand
- [ ] Right-click track → context menu
- [ ] Hover truncated title → tooltip
- [ ] Long-press prev/next → stage picker
- [ ] Scroll on volume area → fine-tune

### Visualizer (14)
- [ ] 8 types (bars, mirrored, circular, line, gradient, particles, ring, blob)
- [ ] Per-stage auto-selection
- [ ] User-configurable type, color, sensitivity
- [ ] Mini visualizer in player bar
- [ ] Crossfade blend between analysers
- [ ] Pause when tab hidden
- [ ] Fullscreen mode
- [ ] Frequency band selection (all/bass/mids/highs)

### Track History (16)
- [ ] Auto-log every play to PostgreSQL
- [ ] 4 view modes (timeline, grid, table, calendar)
- [ ] 7 filter types (stage, time, artist, genre, BPM, key, search)
- [ ] 6 sort options
- [ ] Re-listen (overlay, doesn't interrupt stream)
- [ ] Download original file
- [ ] Copy link, go to artist, track details
- [ ] History stats sidebar

### Plex Sync & Discovery (12)
- [ ] Background sync worker (5min)
- [ ] Playlist diffing
- [ ] MPD playlist sync
- [ ] Addition logging
- [ ] Discovery feed (chronological, grouped)
- [ ] Filters (stage, time, artist)
- [ ] Discovery velocity stats
- [ ] New artist badges
- [ ] Sync status indicators
- [ ] Most-added artist
- [ ] Actions per track (listen, download, go to artist)
- [ ] Force sync (admin)

### Digging Calendar (12)
- [ ] GitHub-style contribution calendar (@uiw/react-heat-map)
- [ ] 5 color modes (volume, stage, genre, decade, country)
- [ ] Click day → expand to show tracks added
- [ ] Streaks (current, longest, best week/month)
- [ ] Digging patterns (day-of-week, time-of-day)
- [ ] Digging sessions (clustered additions)
- [ ] Year selector + multi-year comparison
- [ ] Summary stats (total, averages, active days %)
- [ ] Dry spell tracking with humor
- [ ] Stage-specific calendar view
- [ ] Drill-down to individual tracks
- [ ] Annual overview (327 tracks · 89 artists · 34 labels)

### Artist Intelligence (12)
- [ ] 5-source enrichment (Last.fm, MB, Discogs, Wikidata, TheAudioDB)
- [ ] Merge strategy (best bio, best image, union tags)
- [ ] Similar artist edges with weight
- [ ] Redis cache (24h TTL)
- [ ] Background enrichment worker
- [ ] Stale re-enrichment (>30 days)
- [ ] Meilisearch indexing
- [ ] External links aggregation
- [ ] Banner, thumb, logo, fanart images
- [ ] Members/groups relationships
- [ ] Enrichment queue status
- [ ] Manual merge (admin)

### Artist Graph (12)
- [ ] react-force-graph-2d
- [ ] Node size/color/image
- [ ] Labels on zoom threshold
- [ ] Link particles
- [ ] Click/hover/drag/zoom interactions
- [ ] 5 filter types + 3 color-by modes + 3 size-by modes
- [ ] Layout modes (force/radial/clustered)
- [ ] Search within graph
- [ ] Graph stats panel
- [ ] Ego graph on double-click
- [ ] Context menu on right-click
- [ ] Reset view

### Artist Detail (10)
- [ ] Banner + photo + metadata + links
- [ ] Expandable bio (Instrument Serif)
- [ ] 6 tabs (Tracks, Albums, Stats, History, Similar, Timeline)
- [ ] Per-artist BPM range + key distribution
- [ ] Play heatmap (when do you listen to this artist?)
- [ ] Stage presence bars
- [ ] Similar artists in library + recommendations
- [ ] Chronological timeline
- [ ] Play all / play random
- [ ] External links (Spotify, BC, SC, IG, RA, etc.)

### Statistics Dashboard (22)
- [ ] Overview cards
- [ ] Top lists (artists, tracks, albums, labels — multi-mode)
- [ ] Stage distribution
- [ ] BPM histogram
- [ ] Camelot key wheel
- [ ] Geographic spread
- [ ] Decade breakdown (with deep-dive)
- [ ] Label treemap
- [ ] Genre tag cloud + evolution
- [ ] Listening calendar heatmap
- [ ] Time-of-day heatmap
- [ ] Discovery velocity
- [ ] Track length distribution
- [ ] Diversity index
- [ ] Replay value
- [ ] Stage transitions (Sankey)
- [ ] Crossover analysis
- [ ] Curator insights (6 types)
- [ ] All filterable by time/stage/genre/country
- [ ] Streaks & milestones
- [ ] Genre evolution over time
- [ ] Label distribution

### Wrapped (5)
- [ ] Annual (December) — animated card sequence
- [ ] Monthly recap
- [ ] Shareable PNG card
- [ ] Archive (past years)
- [ ] ANTI-ALGORITHM watermark

### Search (6)
- [ ] Meilisearch instant fuzzy search
- [ ] ⌘K shortcut
- [ ] Tracks, artists, albums, labels
- [ ] Category tabs
- [ ] Recent searches
- [ ] Click actions (play, navigate)

### Queue (5), Notifications (7), Settings (15), PWA (6), Mobile (6), Keyboard (8), Admin (6), Social (5), Performance (8)

*(See sections 21-29 for full breakdowns)*

**Total: ~260+ features**

---

## 37. Key Decisions & Rationale

| Decision | Chosen | Rejected | Why |
|---|---|---|---|
| Access model | **Everything public** | Split listener/DJ views | Radical transparency IS the product. The curation process is the content. |
| Audio | Raw Web Audio API | Howler.js, Tone.js | Dual GainNode crossfade + AnalyserNode |
| Uninterrupted audio | Root layout pattern | SPA single page | Next.js layouts never unmount |
| State | Zustand | Redux, Jotai | Minimal boilerplate, fast audio updates |
| Data fetching | TanStack Query | SWR | Better cache, devtools |
| Bridge | Go | Node.js | Goroutines for 9+ MPD conns + workers |
| Database | PostgreSQL | SQLite, MongoDB | Relational, GIN indexes for tags |
| Cache | Redis | In-memory | Shared, TTL, pub/sub |
| Search | Meilisearch | Elasticsearch, PG FTS | Instant, fuzzy, lightweight, self-hosted |
| Graph | react-force-graph-2d | D3 raw, vis.js | Canvas/WebGL, React-native, maintained |
| Charts | Recharts | Chart.js, Nivo | React-native, declarative |
| Digging calendar | @uiw/react-heat-map | Custom SVG | SVG, customizable, lightweight |
| Artist data | 5 sources (LFM+MB+DC+WD+TADB) | Single source | No one API has everything |
| Containers | **Podman** | Docker | Rootless, no daemon — seedbox has no root |
| Styling | Tailwind + CSS custom properties | Styled Components | Dynamic accent per stage |
| Animation | Framer Motion | GSAP, CSS | AnimatePresence, gesture support |
| PWA | Next.js native + Serwist | next-pwa | No extra deps |
| Fonts | Clash Display + Satoshi + JetBrains Mono + Instrument Serif | Inter, system | Distinctive, each font has a role |
| Brand color | Violet-undertone blacks (#08070d) | Neutral dark (#0a0a0a) | Matches the melancholic + euphoric brand |
| Bio typography | Instrument Serif | Same as body | Unexpected warmth — the "emotion" in "emotion and code" |

---

*GAENDE Radio — Ultimate Architecture v3.1*
*~260 features · 37 sections · all public · the curation process IS the content*
*Built for a seedbox with Podman · Buenos Aires · Dakar · Nouméa · Berlin · The Internet*
*Architecture by GAENDE × Claude — March 2026*
