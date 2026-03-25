# GAENDE Radio — Design System Prompt v2



> Use this prompt as a reference when building ANY page, component, or view

> in the GAENDE webradio. Every element must follow these rules. No exceptions.

> The goal: a hacker's control room for a music obsessive. Not a Spotify clone.



---



## IDENTITY



GAENDE Radio is an anti-algorithm webradio built by a DJ/software engineer. The UI should feel like you hacked into a radio station's control room at 4am — futuristic, slightly glitchy, data-dense, but emotionally warm underneath the technical surface. It's the intersection of emotion and code made visual.



**Keywords:** cyber-brutalist, terminal aesthetic, glitch, data-dense, hacker, futuristic, void-dark, electric, angular, alive



**NOT:** corporate, rounded, pastel, gradient-heavy, generic dark mode, Spotify clone, Material Design, shadcn defaults



---



## COLOR PALETTE



```css

:root {

  /* ═══ VOID — the darkness ═══ */

  --bg-void: #020204;          /* True void, deepest black with blue-black undertone */

  --bg-primary: #06060c;       /* Page backgrounds */

  --bg-secondary: #0a0a14;     /* Sections, alternating areas */

  --bg-elevated: #0e0e1a;      /* Elevated surfaces, inputs */

  --bg-card: #101020;          /* Cards, panels */

  --bg-card-hover: #16162a;    /* Card hover state */



  /* ═══ TEXT — cool, not warm ═══ */

  --text-primary: #d4f0ff;     /* Main text — icy blue-white, NOT warm white */

  --text-secondary: #6b8a99;   /* Secondary — desaturated cyan-gray */

  --text-muted: #2e4450;       /* Muted labels, timestamps */

  --text-ghost: #1a2a33;       /* Barely visible — watermarks, version numbers */



  /* ═══ ACCENT — electric cyan-green ═══ */

  --accent: #00ffc8;           /* Primary interactive color. Buttons, links, active states, visualizers */

  --accent-dim: #00ffc830;     /* Accent at 19% opacity — glows, shadows */

  --accent-glow: #00ffc815;    /* Accent at 8% — subtle bg tints */

  --accent-hot: #00ffaa;       /* Brighter variant for emphasis moments */



  /* ═══ SIGNAL — magenta/pink for alerts, secondary highlights ═══ */

  --signal: #ff0066;           /* Secondary accent. Genre badges, warnings, live indicators, Anti-Algorithm */

  --signal-dim: #ff006630;     /* Signal at 19% */



  /* ═══ DATA — amber for insights, metrics, changes ═══ */

  --data: #ffaa00;             /* Tertiary accent. Stats deltas, insight highlights, data points */

  --data-dim: #ffaa0025;       /* Data at 15% */



  /* ═══ ANTI — the brand red ═══ */

  --anti: #ff2244;             /* The Anti-Algorithm badge color. Rarely used elsewhere. */



  /* ═══ STATUS ═══ */

  --live: #00ff88;             /* Live/online indicators */

  --error: #ff3344;            /* Errors, offline, failed */

  --warning: var(--data);      /* Warnings use data amber */



  /* ═══ BORDERS ═══ */

  --border-subtle: rgba(0, 255, 200, 0.04);  /* Default borders — barely visible, cyan-tinted */

  --border-default: rgba(0, 255, 200, 0.08); /* Slightly more visible */

  --border-glow: rgba(0, 255, 200, 0.15);    /* Hover/active borders */



  /* ═══ STAGE COLORS (override --accent per stage) ═══ */

  --stage-main: #00ffc8;      /* Cyan-green (same as accent) */

  --stage-techno: #ff0066;    /* Signal magenta */

  --stage-ambient: #00ddff;   /* Ice blue */

  --stage-indie: #ffaa00;     /* Amber */

  --stage-bass: #ff44ff;      /* Hot magenta */

  --stage-live: #00ff88;      /* Neon green */

  --stage-chill: #44ddff;     /* Sky cyan */

  --stage-deep: #7b7bff;      /* Soft indigo */

  --stage-world: #ff4466;     /* Coral red */

}

```



### Color Rules

- **NEVER use warm whites.** Text is always cool/icy (`#d4f0ff`, not `#e5e5e5` or `#ffffff`).

- **NEVER use neutral grays.** All grays are tinted toward cyan or blue.

- **Background should feel like a void** — almost black with a deep blue/indigo undertone, never neutral gray.

- **Accent (`#00ffc8`) is the hero color.** It appears on: active sidebar items, links, buttons, visualizer bars, badges, progress bars, scrollbar thumb, volume fills, crossfade indicators.

- **Signal (`#ff0066`) is the secondary.** It appears on: genre/tag badges, the Anti-Algorithm badge, alert toasts, certain stage colors.

- **Data (`#ffaa00`) is for metrics.** It appears on: stat deltas ("+12 this week"), insight highlights, chart annotations.

- **Each stage has its own color.** When a stage is active, its color replaces `--accent` for that context (visualizer, player bar dot, active ring). The sidebar and global UI keep the default `--accent`.



---



## TYPOGRAPHY



```css

:root {

  --font-display: 'Syne', sans-serif;           /* Titles, big numbers, stage names. Bold, geometric, confident. */

  --font-mono: 'JetBrains Mono', monospace;      /* Body text, track titles, artist names, navigation. The "code" in emotion+code. */

  --font-terminal: 'Space Mono', monospace;       /* Labels, badges, timestamps, section headers, system text. Raw, terminal feel. */

  --font-editorial: 'Instrument Serif', serif;    /* Bio text, long-form descriptions. The "emotion" in emotion+code. Unexpected warmth. */

}

```



### Typography Rules



| Element | Font | Weight | Size | Color | Letter-Spacing | Extra |

|---|---|---|---|---|---|---|

| GAENDE logo | `--font-display` | 800 | 14-16px in sidebar, 64-96px hero | `--accent` in sidebar, `--text-primary` in hero | 0.3-0.4em | Text-shadow glow |

| Page/section titles | `--font-display` | 700 | 28-36px | `--text-primary` | 0.02em | — |

| Section labels | `--font-terminal` | 700 | 11px | `--accent` | 0.25em | Uppercase. Prefix with `//`. Followed by gradient line. |

| Stat numbers | `--font-display` | 800 | 32-48px | `--text-primary` | 0 | — |

| Stat labels | `--font-terminal` | 400 | 10-11px | `--text-muted` | 0.1em | Uppercase |

| Track titles | `--font-mono` | 500 | 13-15px | `--text-primary` | 0 | — |

| Artist names | `--font-mono` | 400 | 13-15px | `--text-secondary` or `--accent` as link | 0 | Links: underline on hover with glow |

| Navigation items | `--font-mono` | 400 | 12px | `--text-secondary`, `--accent` when active | 0 | — |

| Badges/pills | `--font-terminal` | 700 | 10-11px | varies | 0.08-0.1em | Uppercase when appropriate |

| Timestamps | `--font-terminal` | 400 | 10-11px | `--text-muted` | 0.05em | — |

| Bio / editorial text | `--font-editorial` | 400 | 17-18px | `--text-secondary` | 0 | Line-height 1.85-1.9, italic for subtitles |

| Code/data readouts | `--font-terminal` | 400 | 9-11px | `--text-muted` or `--accent` at low opacity | 0.05-0.1em | — |

| Button labels | `--font-mono` | 600 | 12-13px | `--accent` or white | 0.08em | Uppercase |



### Typography Anti-Patterns

- **NEVER** use sans-serif body text (no DM Sans, Inter, system-ui). The main body font IS monospace (`JetBrains Mono`). This is what makes it feel like a terminal.

- **NEVER** use large font sizes for body text. Keep everything compact and data-dense. Max body text: 14px. Max editorial: 18px.

- **NEVER** center-align body text (except hero section and Now Playing view).

- Section labels ALWAYS follow this pattern: `// LABEL NAME` + horizontal gradient line extending to the right.



---



## GEOMETRY & SHAPES



### The Rule: Zero Softness



- **border-radius: 0** on everything. No rounded corners. No rounded pills. No circles (except live dots and artist photos in graph).

- **Exceptions:** Live status dots (small squares actually — 0 radius, but so tiny they appear as dots), artist photos in the graph view (circular as data visualization convention only).

- **clip-path is the design tool**, not border-radius. Every card, button, and panel uses angular clip-paths:



```css

/* Standard card — top-right corner cut */

.card {

  clip-path: polygon(0 0, calc(100% - 12px) 0, 100% 12px, 100% 100%, 0 100%);

}



/* Reversed — bottom-left corner cut */

.card-reverse {

  clip-path: polygon(0 0, 100% 0, 100% 100%, 12px 100%, 0 calc(100% - 12px));

}



/* Both corners cut */

.card-both {

  clip-path: polygon(0 0, calc(100% - 12px) 0, 100% 12px, 100% 100%, 12px 100%, 0 calc(100% - 12px));

}



/* Octagonal button */

.btn-octagon {

  clip-path: polygon(4px 0, calc(100% - 4px) 0, 100% 4px, 100% calc(100% - 4px), calc(100% - 4px) 100%, 4px 100%, 0 calc(100% - 4px), 0 4px);

}



/* Parallelogram badge (the Anti-Algorithm badge shape) */

.badge-skew {

  clip-path: polygon(4px 0, 100% 0, calc(100% - 4px) 100%, 0 100%);

}

```



### Borders

- Default: `1px solid var(--border-subtle)` — barely visible, cyan-tinted

- Hover: transition to `var(--border-glow)` — brighter, visible

- Active: `var(--accent)` with `box-shadow: 0 0 N var(--accent-dim)`

- **Left-border accents** on list items: `border-left: 2px solid var(--accent)` (or stage color)

- **Top-border accents** on cards: `2px solid var(--stage-color)` with `box-shadow: 0 0 8px`



### Shadows

- Cards: `box-shadow: none` by default. On hover: `0 8px 24px rgba(0,0,0,0.3)`

- Accent elements: `box-shadow: 0 0 12px var(--accent-dim)`

- Player bar play button: `box-shadow: 0 0 12px var(--accent-dim)`

- Glowing text: `text-shadow: 0 0 12px var(--accent-dim)`

- **NEVER** use large soft shadows. Shadows are either absent or sharp/glowing.



---



## GLOBAL OVERLAYS & EFFECTS



These MUST be present on every page, applied at the `body` or root level:



### 1. Scanlines

```css

body::before {

  content: '';

  position: fixed;

  inset: 0;

  z-index: 10000;

  pointer-events: none;

  background: repeating-linear-gradient(

    0deg, transparent, transparent 2px,

    rgba(0, 0, 0, 0.03) 2px, rgba(0, 0, 0, 0.03) 4px

  );

}

```



### 2. CRT Vignette

```css

body::after {

  content: '';

  position: fixed;

  inset: 0;

  z-index: 9999;

  pointer-events: none;

  background: radial-gradient(ellipse at center, transparent 60%, rgba(0,0,0,0.5) 100%);

}

```



### 3. Noise Texture (animated)

```css

.noise {

  position: fixed;

  inset: 0;

  z-index: 9998;

  pointer-events: none;

  opacity: 0.035;

  background-image: url("data:image/svg+xml,..."); /* fractalNoise SVG */

  background-size: 128px;

  animation: noiseShift 0.5s steps(3) infinite;

}

```



### 4. Matrix Rain (optional, hero/about pages)

Faint vertical columns of katakana/symbols falling at ~3% opacity behind content. Creates depth. Use sparingly — hero, about page, maybe the graph page background.



### 5. Grid Background (hero/about only)

```css

.grid-bg {

  background-image:

    linear-gradient(var(--border-subtle) 1px, transparent 1px),

    linear-gradient(90deg, var(--border-subtle) 1px, transparent 1px);

  background-size: 60px 60px;

  mask-image: radial-gradient(ellipse at center, rgba(0,0,0,0.4) 0%, transparent 70%);

}

```



---



## ANIMATION RULES



### Glitch Effects



The GAENDE title uses a **dual-layer glitch**:

- `::before` clone in `--accent` color, clipped to top third, jitters independently

- `::after` clone in `--signal` color, clipped to bottom third, jitters independently

- Both have `clip-path: polygon(...)` and offset transforms every ~3-5 seconds

- Keep glitch intervals LONG (3-6 seconds between glitch bursts). Constant glitching is annoying.



### Flicker

System text (coordinates, version numbers) can have a subtle flicker:

```css

animation: flicker 4s infinite;

@keyframes flicker {

  0%, 97%, 100% { opacity: 1; }

  98% { opacity: 0.4; }

  99% { opacity: 0.8; }

}

```



### Anti-Algorithm Badge Glitch

The badge does a skewX glitch every ~6 seconds:

```css

animation: badgeGlitch 6s infinite;

@keyframes badgeGlitch {

  0%, 93%, 100% { transform: translate(0) skewX(0); }

  94% { transform: translate(-3px, 1px) skewX(-2deg); }

  95% { transform: translate(4px, -1px) skewX(3deg); }

  96% { transform: translate(-1px, 0) skewX(-1deg); }

}

```



### Scanning Laser

Album art and player bar thumbnails have a scanning line:

```css

&::after {

  content: '';

  position: absolute;

  left: 0; right: 0;

  height: 2px;

  background: linear-gradient(90deg, transparent, var(--accent), transparent);

  opacity: 0.4;

  animation: scan 4s linear infinite;

}

@keyframes scan { 0% { top: -2px; } 100% { top: 100%; } }

```



### Floating Orbs

Hero/about backgrounds use 2-3 blurred gradient orbs drifting slowly:

```css

.orb {

  position: absolute;

  border-radius: 50%;

  filter: blur(80px);

  animation: orbFloat 20s ease-in-out infinite alternate;

}

```



### Page Transitions (Framer Motion)

```tsx

initial={{ opacity: 0, y: 12, filter: 'blur(4px)' }}

animate={{ opacity: 1, y: 0, filter: 'blur(0)' }}

exit={{ opacity: 0, y: -8, filter: 'blur(4px)' }}

transition={{ duration: 0.3, ease: [0.25, 0, 0.2, 1] }}

```



### Hover States

- Cards: `translateY(-1px)` or `translateY(-2px)`, border brightens to `--border-glow`

- Buttons: `box-shadow` glow intensifies, `inset` glow appears

- Links: `text-shadow: 0 0 8px var(--accent-dim)`, border-bottom appears

- **NEVER** scale on hover beyond 1.06. Subtle movement only.



### Loading States

- Use a pulsing horizontal line (accent color, 2px height) scanning left to right

- Or: a row of visualizer-like bars pulsing (like the mini viz in the player bar)

- **NEVER** use a spinner. Spinners are generic.



### Empty States

- Message in `--font-terminal`, `--text-muted`, 11px, uppercase, centered

- Example: `// NO DATA — TIME TO DIG`

- Subtle grid background behind the message



---



## COMPONENT PATTERNS



### Cards

```css

.card {

  background: var(--bg-card);

  border: 1px solid var(--border-subtle);

  padding: 20px;

  clip-path: polygon(0 0, calc(100% - 12px) 0, 100% 12px, 100% 100%, 0 100%);

  transition: all 0.25s ease;

}

.card:hover {

  background: var(--bg-card-hover);

  border-color: var(--border-glow);

  transform: translateY(-1px);

}

```



### Section Labels

```

// SECTION NAME ─────────────────────────

```

Always: `--font-terminal`, 11px, `--accent`, uppercase, 0.25em letter-spacing, `//` prefix, flex with gradient line `::after`.



### Stat Cards

Angular corner cut. Big number in `--font-display` 800. Label in `--font-terminal` uppercase muted. Delta in `--accent` with `▲` prefix. Right edge has a 1px gradient line (accent → transparent).



### Badges

```css

.badge {

  font-family: var(--font-terminal);

  font-size: 10px;

  font-weight: 700;

  padding: 4px 10px;

  border-radius: 0;

  letter-spacing: 0.08em;

  border: 1px solid var(--border-subtle);

}

.badge-accent { background: var(--accent-glow); color: var(--accent); border-color: var(--accent-dim); }

.badge-signal { background: rgba(255,0,102,0.06); color: var(--signal); border-color: var(--signal-dim); }

.badge-data { background: var(--data-dim); color: var(--data); border-color: rgba(255,170,0,0.15); }

```



### Buttons

```css

/* Primary: transparent with accent border, parallelogram clip-path, glow on hover */

.btn-primary {

  background: transparent;

  color: var(--accent);

  border: 1px solid var(--accent);

  font-family: var(--font-mono);

  font-weight: 600;

  font-size: 13px;

  letter-spacing: 0.08em;

  padding: 12px 28px;

  clip-path: polygon(0 0, calc(100% - 12px) 0, 100% 12px, 100% 100%, 12px 100%, 0 calc(100% - 12px));

  cursor: pointer;

  transition: all 0.3s ease;

}

.btn-primary:hover {

  background: var(--accent-glow);

  box-shadow: 0 0 30px var(--accent-dim), inset 0 0 30px var(--accent-glow);

}



/* Secondary: signal variant */

.btn-secondary {

  /* Same as primary but with --signal instead of --accent */

}



/* Play button: octagonal, filled accent, dark icon */

.btn-play {

  background: var(--accent);

  clip-path: polygon(4px 0, calc(100% - 4px) 0, 100% 4px, 100% calc(100% - 4px), calc(100% - 4px) 100%, 4px 100%, 0 calc(100% - 4px), 0 4px);

  box-shadow: 0 0 12px var(--accent-dim);

}

```



### Sidebar Items

```css

.sidebar-item {

  font-family: var(--font-mono);

  font-size: 12px;

  padding: 7px 14px;

  border-radius: 0;

  color: var(--text-secondary);

  position: relative;

}

.sidebar-item.active {

  color: var(--accent);

  background: var(--accent-glow);

}

.sidebar-item.active::before {

  /* Left accent bar: 2px wide, accent color, with glow */

  content: '';

  position: absolute;

  left: 0; top: 4px; bottom: 4px;

  width: 2px;

  background: var(--accent);

  box-shadow: 0 0 6px var(--accent);

}

```



### Insight Cards

```css

.insight {

  background: var(--bg-card);

  border-left: 2px solid var(--data); /* or --accent, --signal — alternate */

  padding: 14px 18px;

  font-family: var(--font-mono);

  font-size: 13px;

  color: var(--text-secondary);

}

```

Prefix each insight with a signal code: `SIG.01`, `SIG.02`, etc. in `--font-terminal`, `--text-muted`.



### Tables

```css

table {

  width: 100%;

  border-collapse: collapse;

  font-family: var(--font-mono);

  font-size: 12px;

}

th {

  font-family: var(--font-terminal);

  font-size: 10px;

  letter-spacing: 0.1em;

  text-transform: uppercase;

  color: var(--text-muted);

  text-align: left;

  padding: 8px 12px;

  border-bottom: 1px solid var(--border-default);

}

td {

  padding: 10px 12px;

  border-bottom: 1px solid var(--border-subtle);

  color: var(--text-secondary);

}

tr:hover td {

  background: var(--bg-card);

  color: var(--text-primary);

}

```



### Inputs / Selects

```css

input, select {

  background: var(--bg-elevated);

  border: 1px solid var(--border-subtle);

  border-radius: 0;

  color: var(--text-primary);

  font-family: var(--font-mono);

  font-size: 13px;

  padding: 8px 12px;

}

input:focus, select:focus {

  border-color: var(--accent);

  outline: none;

  box-shadow: 0 0 8px var(--accent-dim);

}

```



### Modals

```css

.modal-backdrop {

  background: rgba(2, 2, 4, 0.85);

  backdrop-filter: blur(8px);

}

.modal {

  background: var(--bg-secondary);

  border: 1px solid var(--border-default);

  clip-path: polygon(0 0, calc(100% - 20px) 0, 100% 20px, 100% 100%, 20px 100%, 0 calc(100% - 20px));

}

```



### Tooltips

```css

.tooltip {

  background: var(--bg-card);

  border: 1px solid var(--border-default);

  font-family: var(--font-terminal);

  font-size: 11px;

  color: var(--text-primary);

  padding: 6px 10px;

  border-radius: 0;

  box-shadow: 0 4px 12px rgba(0,0,0,0.4);

}

```



### Player Bar

Glass morphism with `rgba(2, 2, 4, 0.92)` + `backdrop-filter: blur(24px)`. Top edge has a subtle gradient line (`linear-gradient(90deg, transparent, var(--accent-dim), transparent)`). All controls use `--accent` for active states. Volume bar: 3px height, no border-radius, fill is `--accent` with glow.



### Visualizer Bars

```css

.viz-bar {

  width: 2-4px;

  border-radius: 0; /* Square tops, not rounded */

  background: var(--accent); /* Or current stage color */

  box-shadow: 0 0 4px var(--accent-dim);

}

```



### Contribution Calendar Cells

```css

.cell { border-radius: 1px; } /* Almost square, tiny rounding */

.cell:hover { outline: 1px solid var(--accent); outline-offset: 1px; }

/* Levels use accent at increasing opacity: 8%, 20%, 37%, 100% */

```



---



## SCROLLBAR



```css

::-webkit-scrollbar { width: 4px; }

::-webkit-scrollbar-track { background: transparent; }

::-webkit-scrollbar-thumb { background: var(--accent-dim); border-radius: 0; }

::-webkit-scrollbar-thumb:hover { background: var(--accent); }

```



## SELECTION



```css

::selection { background: var(--accent-dim); color: var(--accent); }

```



## CURSOR



```css

html { cursor: crosshair; }

a, button, [role="button"] { cursor: pointer; }

/* Consider: custom cursor SVG with a small crosshair in accent color */

```



---



## RESPONSIVE RULES



- Sidebar: hidden below 900px, replaced by bottom tab bar

- Bottom tab bar: `--bg-primary`, top border `--border-subtle`, icons + `--font-terminal` 9px labels

- Cards: stack to single column on mobile

- Stat grid: 2 columns on mobile, 4 on desktop

- Album art: 85vw on mobile, 280-320px on desktop

- Player bar: 72px on mobile, can expand to full-screen on swipe-up

- All clip-paths: reduce corner cut size on mobile (12px → 8px)

- Contribution calendar: horizontally scrollable on mobile with smaller cells



---



## CHARTS & DATA VISUALIZATION (Recharts / Custom)



- Chart backgrounds: transparent (the card bg shows through)

- Grid lines: `var(--border-subtle)`, 1px, dashed

- Axis labels: `--font-terminal`, 9px, `--text-muted`

- Bars: `--accent` with `--accent-dim` glow, no border-radius

- Lines: `--accent`, 2px stroke, no fill (or fill with `--accent-glow`)

- Area fills: gradient from `var(--accent)` at 15% to transparent

- Tooltips: same as component tooltips above

- Legends: `--font-terminal`, 10px, inline, no background box

- Pie/donut: use stage colors, no strokes, subtle gap between segments

- Heatmaps: `--accent` gradient (dark → full)



---



## WHAT MAKES THIS UNIQUE (checklist for every page)



- [ ] Scanlines visible

- [ ] CRT vignette visible

- [ ] Noise texture animated

- [ ] No rounded corners anywhere (except graph node photos)

- [ ] At least one angular clip-path element visible

- [ ] Section labels use `// NAME` + gradient line pattern

- [ ] All monospace text (not serif body)

- [ ] Accent color is cyan-green (`#00ffc8`), not purple/violet

- [ ] Backgrounds are void-dark with blue undertone, not neutral gray

- [ ] Text is icy blue-white, not warm white

- [ ] At least one glowing element (box-shadow or text-shadow with accent-dim)

- [ ] Crosshair cursor on page

- [ ] Badges are square/angular, not rounded pills

- [ ] Player bar has glass morphism + top gradient line

- [ ] All interactive elements have `--accent` focus/active states

- [ ] No generic loading spinners — use pulsing bars or scanning lines

- [ ] Empty states use `// NO DATA` terminal style



---



## PAGE-SPECIFIC NOTES



### Hero / About

- Grid background, floating orbs, matrix rain

- GAENDE title with dual-layer glitch

- Bio in Instrument Serif (the ONLY place serif is used extensively)

- Anti-Algorithm badge inline with skew glitch

- Coordinates above title (GPS + city name, flickering)

- Cities at bottom in `--font-terminal`, separated by `◆`



### Dashboard

- Stat cards with corner cuts

- Mini contribution calendar

- Insights with signal codes (`SIG.01`) and alternating left-border colors

- "What's Playing Now" as a system status readout



### Digging Calendar

- Full-width calendar, square cells, accent gradient levels

- Click to expand day → track list slides in

- Color mode toggle (volume/stage/genre/decade/country)

- Streak counters prominently displayed



### History

- Dense table or timeline view

- Filter bar with angular inputs

- Each row has a scanning line on hover (like album art scan)

- Re-listen button glows on hover



### Artist Graph

- Dark void background (no card container — the graph IS the page)

- Nodes glow with stage colors

- Links are subtle white at 3-6% opacity, brighten on hover

- Sidebar filter panel with angular inputs

- Graph stats in a floating panel (bottom-left or top-right)



### Stats Dashboard

- Dense grid of chart cards, each with angular corners

- Charts use accent color exclusively (or stage colors for stage-specific charts)

- Camelot wheel: custom SVG, segments glow on hover

- Heatmaps: square cells, no border-radius



### Wrapped

- Full-screen card sequence, each card fills viewport

- Dramatic reveal animations (scale, blur, translate)

- Big numbers in `--font-display` 800 at 64px+

- Shareable card has scanlines baked into the design



---



*GAENDE Radio Design System v2 — Cyber-Brutalist Terminal Aesthetic*

*"The intersection of emotion and code, made visible."*
