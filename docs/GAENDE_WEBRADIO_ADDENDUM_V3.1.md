# GAENDE Webradio — Architecture Addendum v3.1

> This addendum layers on top of the v3 blueprint.
> It reframes the product as a fully transparent, open experience where
> everyone sees everything — the radio, the digging, the obsession.
> Plus: the About page, Podman for seedbox, and the GAENDE visual identity.

---

## A. RADICAL TRANSPARENCY — EVERYBODY SEES EVERYTHING

### The Philosophy

This is the anti-algorithm made visible. The listener doesn't just hear the music — they see how it got there. They see the digging calendar, the artist graph, the statistics, the obsession quantified. There's no curtain between the DJ and the audience. The curation process IS the content.

This is what makes it unlike any other webradio: you're not just streaming music, you're streaming your taste in real-time, and everyone can explore the full depth of it. A listener can go from "oh this track is good" → click the artist → see the similarity graph → discover 5 new artists → see that you added this track last Tuesday at 11pm after a 14-track digging session → understand the rabbit hole that led here. That's a level of intimacy and transparency no algorithm will ever offer.

### What Everyone Sees

Every visitor, whether they're a friend, a PAVOIA attendee, a random SoundCloud follower, or a curious DJ colleague — they all get the full experience:

- 🏠 **Stages** — live streams, stage selector, Now Playing with visualizer
- ℹ️ **About** — who is GAENDE, the philosophy, SoundCloud, booking
- 📊 **Dashboard** — the DJ's home base, live overview, insights
- ⛏️ **Digging** — GitHub-style contribution calendar of music additions
- 📜 **History** — full play history with every filter imaginable
- ✨ **Discovery** — what tracks were added to Plex playlists and when
- 🕸️ **Artists** — force-directed similarity graph + full artist profiles
- 📈 **Stats** — every metric through every lens (country, decade, label, BPM, key)
- 🎁 **Wrapped** — year/month in review, shareable cards
- 🔍 **Search** — instant search across everything
- 📋 **Queue** — what's coming up next on each stage
- ↓ **Download** — listeners can download tracks they discover (your choice to enable)
- ▶ **Re-listen** — go back and listen to a track that played earlier

### The Only Protected Route

```
/admin  → System admin (stage control, Plex sync, enrichment queue)
         Protected by a simple token in .env — the only private page.
         Everything else is public.
```

### Why This Works

- It's authentic — you're not performing transparency, you ARE transparent
- It makes the radio a discovery tool for listeners too, not just passive listening
- Other DJs will respect it — showing your sources is the ultimate confidence move
- It turns the technical infrastructure (digging calendar, graph, stats) into content itself
- People share interesting stats and graphs — it's built-in viral potential
- It aligns perfectly with "Anti-Algorithm": the algorithm hides its process, you expose yours

---

## B. THE ABOUT PAGE — `/about`

This is the listener's introduction to you. It should feel like walking into the radio station and seeing the person behind the decks.

### Content & Layout

```
┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│  [Background: subtle animated mesh gradient, your accent colors  │
│   — violet, deep black, with slow-moving organic shapes.         │
│   Not distracting. Atmospheric. Like a club at 4am.]             │
│                                                                  │
│                        G A E N D E                               │
│                                                                  │
│              Clash Display, 64px, letter-spacing: 0.3em          │
│              Subtle glow in accent color                         │
│                                                                  │
│                      "Lion" in Wolof                             │
│                  Satoshi, 16px, --text-muted                     │
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
│  │  "Anti-Algorithm" selection that is fresh, unique,     │      │
│  │  and impossible to replicate.                          │      │
│  │                                                        │      │
│  │  His sets flow with a dynamic clash of deep,           │      │
│  │  introspective melancholia and raw, euphoric           │      │
│  │  empowerment. Expect a narrative journey through       │      │
│  │  Progressive Driving Melodic House & Techno, sparked   │      │
│  │  with elements of Downtempo, Indie Dance, and          │      │
│  │  Hard House.                                           │      │
│  │                                                        │      │
│  │  Satoshi, 18px, line-height 1.8, max-width 640px      │      │
│  │  centered, --text-secondary                            │      │
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
│  Buenos Aires · Dakar · Nouméa · Berlin · The Internet          │
│  JetBrains Mono, 13px, --text-muted, spaced                     │
│                                                                  │
│  ─── PAVOIA Festival ──────────────────────────────────────────  │
│                                                                  │
│  GAENDE is a core builder of the PAVOIA festival —               │
│  artist, promoter, and technical architect.                      │
│  [Link to PAVOIA website]                                        │
│                                                                  │
│  ─── Listen Now ─────────────────────────────────────────────── │
│                                                                  │
│  [Stage cards — horizontal scroll — inviting the listener        │
│   to pick a stage and start listening]                            │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### Design Notes for the About Page

- The page should feel **immersive and cinematic**, not like a typical "about" page
- The mesh gradient background should react subtly to the currently playing audio (if the listener has a stage active) — the gradient pulses with bass frequencies
- The GAENDE title should have a subtle animation on load: letter-by-letter reveal, then a soft glow pulse
- The bio text should fade in with staggered animation (Framer Motion)
- The "Anti-Algorithm" phrase in the bio should be slightly highlighted (accent color, or italic)
- The cities line at the bottom is a subtle nod to the multicultural journey — use `·` separators, monospaced, muted
- The SoundCloud button should be prominent — it's the CTA for listeners who want more
- On mobile, this page should feel like a full-screen story card (think Instagram stories but slower, more considered)

---

## C. DASHBOARD — `/dashboard`

The nerve center. For you it's self-knowledge; for your listeners it's a window into the mind of the curator. Everyone sees it.

### Dashboard Overview Layout

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
│  [████████ Colored by accent — green to deep violet ██████████] │
│  [████████ Each cell = tracks added that day ████████████████] │
│                                                                  │
│  327 tracks added in the last year                              │
│  Longest streak: 23 days (Feb 1 — Feb 23)                       │
│  Current streak: 7 days                                         │
│  Most active day: March 3 — 19 tracks added                     │
│                                                                  │
│  ── Today's Additions ─────────────────────────────────────────  │
│                                                                  │
│  ┌──────┐  Hyperion — Gesaffelstein              → Techno       │
│  ┌──────┐  Opus — Eric Prydz                      → Main        │
│  ┌──────┐  Innerbloom (Edu Imbernon Rmx) — RÜFÜS  → Deep        │
│  [View all today's additions →]                                  │
│                                                                  │
│  ── What's Playing Now (across stages) ────────────────────────  │
│                                                                  │
│  ● Main     Echoes of Horizon — Stephan Bodzin     23 👤        │
│  ● Techno   Acid Rain — M.I.T.A.                    8 👤        │
│  ● Ambient  Infinite — Nils Frahm                   4 👤        │
│  ● Indie    [offline]                                            │
│  ...                                                            │
│                                                                  │
│  ── Taste Snapshot ────────────────────────────────────────────  │
│                                                                  │
│  ┌─────────────────────────────────────────┐                    │
│  │  Your library through different lenses: │                    │
│  │                                         │                    │
│  │  By Country    🇩🇪 25%  🇬🇧 15%  🇫🇷 10% │  ← mini donut      │
│  │  By Decade     2020s 62%  2010s 28%     │  ← mini bar        │
│  │  By Label      Afterlife 12%  Kompakt 8%│  ← mini bar        │
│  │  By BPM        Peak: 122 BPM           │  ← sparkline       │
│  │  By Key        Dominant: Am (8A)        │  ← mini wheel      │
│  │                                         │                    │
│  │  [Explore full stats →]                 │                    │
│  └─────────────────────────────────────────┘                    │
│                                                                  │
│  ── Recent Insights ───────────────────────────────────────────  │
│                                                                  │
│  💡 "You've been digging more German artists this month (+8%)"   │
│  💡 "Your BPM is creeping up — from 120 avg to 124 avg"         │
│  💡 "3 new artists entered your top 10 most-played this week"    │
│  💡 "You haven't added anything to the Ambient stage in 12 days" │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### Automated Insights Engine

The bridge computes insights by comparing current period vs. previous period:

| Insight Type | Computation | Example |
|---|---|---|
| Country shift | Compare country % this month vs last month | "More German artists this month (+8%)" |
| BPM drift | Compare avg BPM this week vs last month | "Your BPM is creeping up" |
| New top artists | Track rank changes in play count | "3 new artists in your top 10" |
| Neglected stages | Days since last addition per stage | "No Ambient additions in 12 days" |
| Genre shift | Compare genre % this month vs last | "Indie Dance went from 8% to 15%" |
| Discovery pace | Tracks/week this month vs last | "Digging pace down 30% this month" |
| Diversity change | Shannon index trend | "Your taste is getting more diverse" |
| Replay behavior | Repeat plays this week vs discovery | "You're replaying more than discovering" |
| Label loyalty | New label appearances | "You discovered 3 new labels this month" |
| Decade shift | Decade distribution change | "More 90s tracks this month (+5%)" |

---

## D. DIGGING CALENDAR — `/dashboard/digging`

A full-page, GitHub-style contribution calendar showing GAENDE's music digging activity — visible to everyone. Listeners can see exactly how active the curation is, which days were heavy digging sessions, and drill into what was found on any given day. For you, it's a mirror of your obsession. For listeners, it's proof that a human is behind every track.

### What Counts as a "Contribution"

Each cell = **tracks added to Plex playlists** (synced to the webradio) on that day. This is your "digging" — the act of finding, evaluating, and curating music.

### Layout

```
┌──────────────────────────────────────────────────────────────────┐
│  Digging Calendar                                    2026 ▼     │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  Jan    Feb    Mar    Apr    May    Jun    Jul    ...     │   │
│  │  ░░█░░  ░█░░█  ██░█░  ░░░░░  ░░░░░  ░░░░░  ░░░░░       │   │
│  │  ░█░░░  ░░░█░  █░░░█  ░░░░░  ░░░░░  ░░░░░  ░░░░░       │   │
│  │  ░░░█░  █░░░░  ░█░█░  ░░░░░  ░░░░░  ░░░░░  ░░░░░       │   │
│  │  █░░░░  ░░█░░  ░░░██  ░░░░░  ░░░░░  ░░░░░  ░░░░░       │   │
│  │  ░░░█░  ░█░░░  █░░░░  ░░░░░  ░░░░░  ░░░░░  ░░░░░       │   │
│  │  ░░░░█  ░░░░█  ░████  ░░░░░  ░░░░░  ░░░░░  ░░░░░       │   │
│  │  ░█░░░  ░░█░░  ░░░░█  ░░░░░  ░░░░░  ░░░░░  ░░░░░       │   │
│  │                                                          │   │
│  │  ░ 0   ▒ 1-2   ▓ 3-5   █ 6-10   █ 11+                  │   │
│  │  Color: accent gradient (light → saturated)              │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ── 2026 Summary ──────────────────────────────────────────────  │
│                                                                  │
│  327 tracks added · 89 new artists · 34 new labels              │
│  Current streak: 7 days · Longest streak: 23 days               │
│  Most active day: March 3 (19 tracks)                           │
│  Average: 4.2 tracks/day on active days                         │
│  Active days: 78 of 84 (93%)                                    │
│                                                                  │
│  ── Click a Day ───────────────────────────────────────────────  │
│                                                                  │
│  [Clicking any cell expands to show what was added that day]     │
│                                                                  │
│  March 3, 2026 — 19 tracks added                                │
│                                                                  │
│  ┌──────┐  Hyperion — Gesaffelstein → Techno Bunker             │
│  ┌──────┐  Acid Rain — M.I.T.A. → Techno Bunker                 │
│  ┌──────┐  Opus — Eric Prydz → Main Stage                       │
│  ┌──────┐  ... (16 more)                                        │
│  [View all 19 tracks →]                                         │
│                                                                  │
│  ── Digging Patterns ──────────────────────────────────────────  │
│                                                                  │
│  Day of week:                                                    │
│  Mon ██████████░░░░░░░░░░  22%                                  │
│  Tue ████████░░░░░░░░░░░░  18%                                  │
│  Wed ██████░░░░░░░░░░░░░░  14%                                  │
│  Thu ████████░░░░░░░░░░░░  16%                                  │
│  Fri ██████████░░░░░░░░░░  20%                                  │
│  Sat ████░░░░░░░░░░░░░░░░   6%  ← you dig less on weekends     │
│  Sun ██░░░░░░░░░░░░░░░░░░   4%                                  │
│                                                                  │
│  Time of day:                                                    │
│  Morning (6-12):   ██████░░░░░░░░░░░░░░  15%                    │
│  Afternoon (12-18): ████████████░░░░░░░░  30%                    │
│  Evening (18-24):   ████████████████░░░░  40%  ← peak digging   │
│  Night (0-6):       ██████░░░░░░░░░░░░░░  15%                    │
│                                                                  │
│  ── Digging by Stage ──────────────────────────────────────────  │
│                                                                  │
│  [Same calendar, but color = stage accent instead of green]      │
│  Toggle: "Color by: Volume / Stage / Genre / Decade"             │
│                                                                  │
│  When colored by stage, you can see which stages you feed most   │
│  and when. Maybe you dig Techno on Fridays and Ambient on        │
│  Sunday mornings.                                                │
│                                                                  │
│  ── Year Selector ─────────────────────────────────────────────  │
│                                                                  │
│  [2026 ▼]  Switch between years to see your digging history      │
│  Multiple years side-by-side comparison view                    │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### Calendar Color Modes

| Mode | Cell Color Represents | Best For |
|---|---|---|
| **Volume** (default) | Track count (light→dark accent gradient) | Overall digging intensity |
| **Stage** | Dominant stage that day (stage accent color) | Where you're feeding |
| **Genre** | Dominant genre added that day | Taste patterns over time |
| **Decade** | Dominant decade of tracks added | Vintage vs. modern digging |
| **Country** | Dominant country of artists added | Geographic patterns |

### Digging Streaks

Like GitHub streaks, but for music curation:
- **Current streak**: Days in a row with at least 1 track added
- **Longest streak**: All-time record
- **Best week**: Most tracks added in a 7-day window
- **Best month**: Most tracks added in a calendar month
- **Dry spells**: Longest gap without adding anything (with humor: "Your longest drought was 8 days in July. Were you on vacation or just listening to what you had?")

---

## E. ANALYTICS — EVERY LENS

The stats dashboard is public — anyone can explore the library through multiple dimensions. Each dimension has its own visualization and drill-down. For you it's self-knowledge; for listeners it's a fascinating deep-dive into a curated music collection.

### Lens: By Country

```
┌──────────────────────────────────────────────────────────────┐
│  Your Library by Country                                     │
│                                                              │
│  [Interactive world map — countries colored by artist count]  │
│  [Click a country → filter everything to that country]       │
│                                                              │
│  Top countries:                                              │
│  🇩🇪 Germany        87 artists   642 tracks   25.1%          │
│  🇬🇧 United Kingdom 52 artists   389 tracks   15.0%          │
│  🇫🇷 France         34 artists   241 tracks    9.8%          │
│  🇮🇹 Italy          28 artists   198 tracks    8.1%          │
│  🇺🇸 USA            25 artists   167 tracks    7.2%          │
│  🇳🇱 Netherlands    19 artists   134 tracks    5.5%          │
│  🇧🇪 Belgium        14 artists    98 tracks    4.0%          │
│  🇦🇷 Argentina       8 artists    56 tracks    2.3%          │
│  🇸🇳 Senegal         3 artists    21 tracks    0.9%          │
│  ...                                                        │
│                                                              │
│  Insight: "78% European, 12% North American, 4% African,    │
│  3% South American, 3% other"                               │
│                                                              │
│  Trend: "Argentina +3 artists this month (Buenos Aires       │
│  connection showing in your digging?)"                       │
└──────────────────────────────────────────────────────────────┘
```

### Lens: By Decade / Year

```
┌──────────────────────────────────────────────────────────────┐
│  Your Library by Decade                                      │
│                                                              │
│  [Stacked bar chart: decades × track count]                  │
│                                                              │
│  2020s ████████████████████████████████████░░  62% (1,761)   │
│  2010s ████████████████░░░░░░░░░░░░░░░░░░░░░░  28%   (795)  │
│  2000s ████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░   6%   (170)  │
│  1990s ██░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░   3%    (85)  │
│  1980s ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░   1%    (30)  │
│                                                              │
│  [Click a decade → expand to year-by-year breakdown]         │
│                                                              │
│  1990s deep dive:                                            │
│  1991: 3 tracks (Plastikman — Sheet One)                     │
│  1993: 7 tracks (Aphex Twin, Autechre, early Warp)           │
│  1995: 12 tracks (Basic Channel, Chain Reaction era)         │
│  1997: 18 tracks (Daft Punk era, French touch)               │
│  1999: 8 tracks (Underworld, Sasha)                          │
│                                                              │
│  "Your 90s digging focuses on early German techno and        │
│  French touch — the forgotten vinyls the bio promises."      │
└──────────────────────────────────────────────────────────────┘
```

### Lens: By Label

```
┌──────────────────────────────────────────────────────────────┐
│  Your Library by Label                                       │
│                                                              │
│  [Treemap visualization: box size = track count]             │
│                                                              │
│  Top labels:                                                 │
│  Afterlife           48 tracks  12 artists   1.7%            │
│  Kompakt             36 tracks   9 artists   1.3%            │
│  Innervisions        31 tracks   7 artists   1.1%            │
│  Diynamic            28 tracks   8 artists   1.0%            │
│  Drumcode            24 tracks   6 artists   0.8%            │
│  Life and Death      22 tracks   5 artists   0.8%            │
│  Anjunadeep          20 tracks   7 artists   0.7%            │
│  Herzblut            18 tracks   2 artists   0.6%            │
│  ARTS                16 tracks   4 artists   0.6%            │
│  ...                                                        │
│                                                              │
│  New labels this month: Stroboscopic Artefacts, L.I.E.S.,    │
│  Semantica (your taste is getting darker?)                   │
│                                                              │
│  Label loyalty: 68% of your library is on labels you've      │
│  collected 5+ tracks from. You're a label digger.            │
└──────────────────────────────────────────────────────────────┘
```

### Lens: By Genre (Tag Cloud + Evolution)

```
┌──────────────────────────────────────────────────────────────┐
│  Your Library by Genre                                       │
│                                                              │
│  [Weighted tag cloud — size = track count, color = stage]    │
│                                                              │
│       melodic techno          techno                         │
│    progressive house     deep house   minimal                │
│       indie dance    ambient    organic house                 │
│    downtempo   acid    electro   dub techno                  │
│     breakbeat   UK garage   afro house                       │
│                                                              │
│  [Genre evolution: stacked area chart over time]             │
│  Shows how your genre proportions shift month to month       │
│                                                              │
│  "In January, melodic techno was 45% of your adds.           │
│  By March, it's down to 32% — indie dance and acid           │
│  are taking over."                                           │
└──────────────────────────────────────────────────────────────┘
```

### Lens: Crossover Analysis

**Which artists appear on multiple stages?** This reveals the versatility of certain artists in your collection:

```
Artists on 3+ stages:
  Stephan Bodzin    → Main (15) · Techno (5) · Deep (3)
  Âme               → Main (12) · Deep (5) · Chill (2)
  Bicep             → Main (8) · Indie (5) · Bass (2)

Artists on exactly 1 stage (specialists):
  89 artists only appear on Techno
  34 artists only appear on Ambient
  ...

"Your most versatile artist is Stephan Bodzin —
he spans 3 stages. Your most stage-loyal genre is
Drum & Bass (100% on Bass Cave)."
```

---

## F. PODMAN — NOT DOCKER

Since the seedbox has no root access, all container orchestration uses **Podman** (rootless, daemonless, drop-in Docker replacement).

### Key Differences

| Docker | Podman |
|---|---|
| `docker` | `podman` |
| `docker-compose` | `podman-compose` (or `podman compose` with podman 4+) |
| Requires daemon (root) | Daemonless (rootless) |
| `docker.sock` | No socket needed |
| Docker Hub default | Same registries |
| `Dockerfile` | Same (compatible) |
| `docker-compose.yml` | Same (compatible) |

### Podman Compose File

The `docker-compose.yml` from v3 works as-is with `podman-compose`. The only changes:

```yaml
# podman-compose.yml (or just use docker-compose.yml — it's compatible)

# Key difference: no 'version' field needed with podman-compose v1.1+
# Just remove the version line

services:
  # All services identical to v3 docker-compose.yml
  # ...

  # For Podman on a seedbox without root:
  # - All ports must be > 1024 (no privileged ports)
  # - Use podman's --userns=keep-id for volume permissions
  # - No need for Caddy if the seedbox already has a reverse proxy

  web:
    build: ./apps/web
    ports: ["3000:3000"]  # > 1024, OK
    # ...

  bridge:
    build: ./apps/bridge
    ports: ["3001:3001"]  # > 1024, OK
    # ...

  postgres:
    image: docker.io/library/postgres:16-alpine  # Full registry path for Podman
    ports: ["15432:5432"]  # Remap to avoid conflicts with system postgres
    # ...

  redis:
    image: docker.io/library/redis:7-alpine
    ports: ["16379:6379"]
    # ...
```

### Running

```bash
# Install podman-compose (if not available)
pip install --user podman-compose

# Build and start everything
podman-compose up -d --build

# Or with Podman 4+ built-in compose
podman compose up -d --build

# View logs
podman-compose logs -f bridge

# Restart a single service
podman-compose restart bridge

# Status
podman-compose ps
```

### Seedbox-Specific Notes

- **No privileged ports**: All services bind to ports > 1024
- **Volume permissions**: Use `podman unshare chown` if needed for volume mounts
- **Systemd user units**: Generate with `podman generate systemd --new --name gaende-web` for auto-restart on reboot
- **Reverse proxy**: If the seedbox has Nginx/Apache already, add proxy_pass rules instead of running Caddy
- **Storage**: Use the seedbox's large storage for the music directory — mount as read-only volumes
- **Resource limits**: Set `--memory` and `--cpus` per container to be a good citizen on shared hosting

### Systemd User Service (auto-start without root)

```bash
# Generate systemd unit files for all containers
mkdir -p ~/.config/systemd/user

# For each container:
podman generate systemd --new --name gaende-bridge > ~/.config/systemd/user/gaende-bridge.service

# Enable auto-start
systemctl --user enable gaende-bridge.service
systemctl --user enable gaende-web.service
# etc.

# Allow user services to run after logout
loginctl enable-linger $USER
```

---

## G. VISUAL IDENTITY — THE GAENDE BRAND

### Design Philosophy

The bio gives us clear design cues:

> *"intersection of emotion and code"* → The UI should feel emotional (warm, organic gradients, music-reactive elements) but built with engineering precision (clean grids, monospace metadata, sharp interactions)

> *"Anti-Algorithm"* → The design should feel **handcrafted**, not template-generated. Reject the generic AI aesthetic. No rounded-everything, no pastel gradients, no cookie-cutter cards. Instead: asymmetric layouts, unexpected typography choices, subtle textures.

> *"deep, introspective melancholia and raw, euphoric empowerment"* → The color palette should oscillate between **dark, contemplative tones** (deep violets, midnight blues, near-blacks) and **bursts of saturated energy** (hot accent colors, glowing elements, pulsing visualizers).

> *"Progressive Driving Melodic House & Techno"* → The UI should feel like it's **moving forward**. Transitions should feel directional (left-to-right, upward momentum). No stasis. The visualizer should pulse like a heartbeat.

### Color Palette

```css
:root {
  /* The GAENDE palette — not generic dark mode, but intentionally moody */

  /* Base: near-black with a violet undertone (not pure neutral) */
  --bg-primary: #08070d;      /* Barely perceptible violet-black */
  --bg-secondary: #0f0e17;    /* Slightly warmer, like a club at 3am */
  --bg-elevated: #161525;     /* Cards, panels — hint of indigo */
  --bg-card: #1a1929;         /* Elevated cards */
  --bg-card-hover: #222137;   /* Hover state */

  /* Text: warm whites, not clinical */
  --text-primary: #e8e6f0;    /* Slightly lavender white */
  --text-secondary: #9b97b0;  /* Muted, but warm */
  --text-muted: #5a5672;      /* Very subdued */

  /* Accent: deep violet → electric */
  --accent: #a855f7;          /* Primary accent */
  --accent-hot: #d946ef;      /* High-energy moments */
  --accent-cold: #6366f1;     /* Contemplative moments */

  /* The "Anti-Algorithm" red — used sparingly for emphasis */
  --anti: #ff3366;            /* For "Anti-Algorithm" badge, error states, hot moments */

  /* Fonts — intentionally NOT the usual suspects */
  --font-display: 'Clash Display', sans-serif;     /* Headers: geometric, bold, modern */
  --font-body: 'Satoshi', sans-serif;               /* Body: clean but characterful */
  --font-mono: 'JetBrains Mono', monospace;         /* Data: BPM, keys, timestamps */
  --font-editorial: 'Instrument Serif', serif;      /* Quotes, bio text: unexpected warmth */
}
```

### Typography Pairing Rationale

- **Clash Display**: Geometric, confident, modern — used for GAENDE, stage names, big numbers. Says "I built this."
- **Satoshi**: Clean but not sterile — used for track titles, navigation, UI text. Feels designed, not defaulted to.
- **JetBrains Mono**: The coder's font — used for BPM, Camelot keys, timestamps, metadata. Nods to the "engineer" in the bio.
- **Instrument Serif**: An unexpected serif for the bio text and editorial moments. Creates warmth and humanity against the techy mono. The "emotion" in "emotion and code."

### Micro-Details That Set the Tone

- **"ANTI-ALGORITHM"** badge: A small pill/badge that appears on certain UI elements (the about page, the discovery feed) in `--anti` red with a subtle glitch animation. It's the brand's signature mark.
- **Loading states**: Instead of generic spinners, use a pulsing waveform animation (like a heartbeat flatline that comes alive).
- **Empty states**: When a filter returns no results, show a message like *"Nothing here. Time to dig."* with a shovel emoji.
- **404 page**: *"This frequency doesn't exist. Try tuning to a different stage."*
- **First visit**: A brief animation on the homepage: the GAENDE letters assemble one by one, then the stages cards fade in from the bottom. Not long — 1.5 seconds. Just enough to feel intentional.
- **Scrollbar**: Custom thin scrollbar in accent color, barely visible until hovered.
- **Selection color**: Text selection in accent color with opacity.
- **Focus rings**: Accent-colored, slightly glowing, not the default blue.

### The "Anti-Algorithm" Badge

A recurring micro-element throughout the UI:

```css
.anti-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 4px; /* intentionally NOT rounded-full — angular, deliberate */
  background: var(--anti);
  color: white;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.15em;
  text-transform: uppercase;
  animation: glitch 4s infinite;
}

@keyframes glitch {
  0%, 95%, 100% { transform: translate(0); opacity: 1; }
  96% { transform: translate(-2px, 1px); opacity: 0.8; }
  97% { transform: translate(2px, -1px); opacity: 0.9; }
  98% { transform: translate(-1px, 0); opacity: 1; }
}
```

This badge appears:
- On the About page, next to the bio paragraph about anti-algorithm
- On the Discovery Feed header ("Anti-Algorithm discoveries")
- On the Digging Calendar ("Your anti-algorithm contributions")
- As a subtle watermark on shareable Wrapped cards

---

## H. UNIFIED NAVIGATION — EVERYONE GETS THE FULL EXPERIENCE

### Desktop Sidebar

```
┌──────────────────┐
│  G A E N D E     │
│  ───────────     │
│                  │
│  🏠 Stages       │  ← Stage selector + Now Playing
│  ℹ️ About        │  ← Bio, SoundCloud, booking
│                  │
│  ── The Crate ── │  ← Section label: your collection
│                  │
│  📊 Dashboard    │  ← Overview: pulse, insights, today
│  ⛏️ Digging      │  ← GitHub contribution calendar
│  📜 History      │  ← Full play history
│  ✨ Discovery    │  ← Plex additions feed
│                  │
│  ── The Map ──── │  ← Section label: understanding
│                  │
│  🕸️ Artists      │  ← Graph + list + search
│  📈 Stats        │  ← Every lens, every metric
│  🎁 Wrapped      │  ← Year/month in review
│                  │
│  ── Controls ─── │
│                  │
│  📋 Queue        │  ← Up next per stage
│  🔍 Search       │  ← Global search (also ⌘K)
│  ⚙️ Settings     │  ← Audio, visual, preferences
│                  │
│  ─────────────── │
│  🔊 23 listening │  ← Total listener count
│  🟢 9/9 stages   │  ← Stage health
└──────────────────┘
```

### Mobile Bottom Bar

```
┌─────────────────────────────────────┐
│  🏠     ⛏️      🕸️     📈     ℹ️   │
│ Stages  Dig   Artists Stats About  │
└─────────────────────────────────────┘

+ Hamburger menu (≡) in header for:
  Dashboard, History, Discovery, Queue,
  Wrapped, Search, Settings
```

### Route Structure (all public)

```
/                      → Stage selector (home)
/stage/[id]            → Full Now Playing view for a stage
/about                 → GAENDE bio, philosophy, links
/dashboard             → DJ dashboard overview
/digging               → GitHub-style digging calendar
/history               → Full play history (tabs: Played / Added)
/history/played        → Chronological play history
/history/added         → Plex additions (discovery feed)
/artists               → Artist graph + list toggle
/artists/[id]          → Full artist detail (tabs: Tracks, Albums, Stats, History, Similar, Timeline)
/stats                 → Full statistics dashboard (every lens)
/wrapped               → Latest wrapped
/wrapped/[year]        → Specific year
/search                → Search results
/queue                 → Queue view
/settings              → Preferences (audio, visual, crossfade)
/admin                 → System admin (ONLY protected route — token auth)
```

### The Experience for a First-Time Visitor

1. Lands on `/` — sees stage cards with live indicators, picks one, music starts
2. Notices the sidebar: "Digging? Artists? Stats? What is this?"
3. Clicks About — reads the bio, understands the anti-algorithm philosophy
4. Goes back to listening, clicks an artist name → full artist page with bio, graph connections, play history, albums
5. Curiosity pulls them to the Digging Calendar — "wait, this person added 19 tracks on March 3? Let me see which ones"
6. Falls into the Artist Graph — zooms, drags, discovers clusters of artists they've never heard of
7. Checks Stats — "this library is 25% German artists, peaks at 122 BPM, dominant key is Am"
8. Shares the Wrapped card on Instagram — "look at this DJ's year in curation"

The transparency IS the hook. The curation process IS the content.

---

## I. WHAT MAKES THIS DIFFERENT FROM EVERY OTHER WEBRADIO

| Other Webradios | GAENDE Radio |
|---|---|
| Press play, listen, leave | Press play, then fall into a rabbit hole of curation data |
| No idea who curated this | Full bio, philosophy, SoundCloud, booking — the human is front and center |
| "Now Playing: Track — Artist" | Now Playing + BPM + key + label + year + artist bio + similar artists graph |
| History? Maybe last 5 tracks | Full history with filters by stage/date/genre/BPM/key, re-listen, download |
| No idea how the playlist was built | Digging Calendar: see exactly when every track was found and added |
| Artist = just a name on screen | Artist = full page with bio, photo, albums, stage presence, play heatmap, similar graph, timeline |
| Stats? What stats? | 20+ visualizations: BPM histogram, Camelot wheel, country map, genre evolution, Sankey flows |
| "Discover new music" = an algorithm | "Discover new music" = follow a human's taste through every lens imaginable |
| Share a link to the stream | Share a Wrapped card showing a year of curation in one beautiful image |

---

---

## K. ADDITIONAL STATS IDEAS

### For the Digging Calendar specifically:

- **"Git blame" for stages**: For any stage's current playlist, show when each track was added, creating a timeline of how that stage's rotation was built over time
- **Additions per source**: If you later add more sources (Bandcamp purchases, Beatport, manual adds), track which source each track came from
- **Digging sessions**: Cluster additions that happened within the same hour/day into "sessions" — "March 3 evening session: 14 tracks in 2 hours, mostly Techno, triggered by a Stephan Bodzin rabbit hole"

### Curator insights (visible to everyone — part of the transparency):

- **Set compatibility**: "The Main Stage rotation has 78% harmonic compatibility (adjacent Camelot keys) — great for smooth mixing"
- **Energy curve**: Plot the BPM/energy of a stage's playlist over time to see if it has a natural arc or is random
- **Freshness score**: What % of each stage's rotation was added in the last 30 days? "Techno is 34% fresh, Ambient is only 8% — time to dig for ambient?"
- **Artist concentration risk**: "Main Stage has 15% Stephan Bodzin. If diversifying, here are similar artists with fewer tracks..."
- **Cross-pollination score**: How much overlap is there between stages? "Main and Deep share 23 artists. Techno and Ambient share 0 — completely separate worlds."
- **Genre coherence per stage**: "The Indie Floor is 82% indie dance/nu-disco — very focused. Main Stage is more eclectic at 45% melodic techno + 20% progressive + 15% indie dance."

---

*Architecture Addendum v3.1 (Open Edition) by GAENDE × Claude — March 2026*
*Everybody sees everything. The curation process IS the content.*
