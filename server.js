// server.js — CommonJS, single-file API + UI for 9 MPD streams using HTTPS subdomains via baseUrl
// Plays audio client-side only; shows now-playing (cover, title, artist, album, year); artist page from artists.json

const express = require("express");
const cors = require("cors");
const fs = require("fs/promises");
const fssync = require("fs");
const path = require("path");
const os = require("os");
const mpd = require("mpd");
const https = require("https");

// ───────────────── PLEX COVER PROXY ──────────────
const PLEX_LOCAL = "https://127.0.0.1:31711";
// Read token from Plex Preferences.xml
let PLEX_TOKEN = "";
try {
  const prefsPath = path.join(os.homedir(), "Library", "Application Support", "Plex Media Server", "Preferences.xml");
  const prefs = fssync.readFileSync(prefsPath, "utf8");
  const m = prefs.match(/PlexOnlineToken="([^"]+)"/);
  if (m) PLEX_TOKEN = m[1];
  console.log("Plex token loaded from Preferences.xml");
} catch (e) {
  console.warn("Could not read Plex token:", e.message);
}

// ───────────────── CONFIG ─────────────────
const MPD_HOST = process.env.MPD_HOST || "127.0.0.1";

// Path to artists.json (default to your provided path)
const ARTISTS_JSON_PATH =
  process.env.ARTISTS_JSON || path.join(os.homedir(), "files", "Webradio", "artists.json");

const DEFAULT_BASE = path.join(os.homedir(), ".local", "share", "mpd");

// 1) streams config (your baseUrl mapping)
const streams = [
  { id: "gaende-favorites", name: "gaende's favorites (all genres)", controlPort: 6600, musicDir: path.join(DEFAULT_BASE, "gaende-favorites", "music"), baseUrl: "https://mpd-gaende-favorites.nicemouth.box.ca/" },
  { id: "ambiance-safe",    name: "Ambiance / Safe Chillroom",       controlPort: 6601, musicDir: path.join(DEFAULT_BASE, "ambiance-safe", "music"), baseUrl: "https://mpd-ambiance-safe.nicemouth.box.ca/" },
  { id: "etage-0",          name: "ETAGE 0",                         controlPort: 6602, musicDir: path.join(DEFAULT_BASE, "etage-0", "music"),       baseUrl: "https://mpd-etage-0.nicemouth.box.ca/" },
  { id: "fontanna-laputa",  name: "FONTANNA / LAPUTA",               controlPort: 6603, musicDir: path.join(DEFAULT_BASE, "fontanna-laputa", "music"), baseUrl: "https://mpd-fontanna-laputa.nicemouth.box.ca/" },
  { id: "palac-dance",      name: "PALAC – DANCE",                   controlPort: 6604, musicDir: path.join(DEFAULT_BASE, "palac-dance", "music"),     baseUrl: "https://mpd-palac-dance.nicemouth.box.ca/" },
  { id: "palac-slow-hypno", name: "PALAC – SLOW / HYPNOTIQUE",       controlPort: 6605, musicDir: path.join(DEFAULT_BASE, "palac-slow-hypno", "music"), baseUrl: "https://mpd-palac-slow-hypno.nicemouth.box.ca/" },
  { id: "bermuda-night",    name: "BERMUDA (18:00–00:00)",           controlPort: 6606, musicDir: path.join(DEFAULT_BASE, "bermuda-night", "music"),   baseUrl: "https://mpd-bermuda-night.nicemouth.box.ca/" },
  { id: "bermuda-day",      name: "BERMUDA (Before 18:00) / OAZA",   controlPort: 6607, musicDir: path.join(DEFAULT_BASE, "bermuda-day", "music"),     baseUrl: "https://mpd-bermuda-day.nicemouth.box.ca/" },
  { id: "closing",          name: "CLOSING",                         controlPort: 6608, musicDir: path.join(DEFAULT_BASE, "closing", "music"),         baseUrl: "https://mpd-closing.nicemouth.box.ca/" },
];

// ──────────────── ARTISTS INDEX ───────────
let artistsIndex = new Map();
let artistsRaw = { artists: [] };

async function loadArtists() {
  try {
    const raw = await fs.readFile(ARTISTS_JSON_PATH, "utf8");
    artistsRaw = JSON.parse(raw);
    artistsIndex = new Map(
      (artistsRaw.artists || []).map((a) => [String(a.name || "").toLowerCase(), a])
    );
    console.log(`Loaded ${artistsIndex.size} artists from ${ARTISTS_JSON_PATH}`);
  } catch (e) {
    console.warn(`artists.json not loaded (${ARTISTS_JSON_PATH}): ${e.message}`);
  }
}
loadArtists();
if (fssync.existsSync(ARTISTS_JSON_PATH)) {
  fssync.watch(ARTISTS_JSON_PATH, { persistent: false }, () => loadArtists());
}

// ──────────────── MPD HELPERS ─────────────
function parseKV(msg) {
  const out = {};
  String(msg || "")
    .split("\n")
    .forEach((line) => {
      const i = line.indexOf(": ");
      if (i > -1) {
        const k = line.slice(0, i);
        const v = line.slice(i + 2);
        if (out[k] === undefined) out[k] = v;
        else if (Array.isArray(out[k])) out[k].push(v);
        else out[k] = [out[k], v];
      }
    });
  return out;
}

function mpdCommand({ host, port }, command, args = []) {
  return new Promise((resolve, reject) => {
    const client = mpd.connect({ host, port });
    const done = (err, res) => {
      try { client.socket.end(); } catch (_) {}
      err ? reject(err) : resolve(res);
    };
    client.on("ready", () => {
      client.sendCommand(mpd.cmd(command, args), (err, msg) => done(err, parseKV(msg)));
    });
    client.on("error", (err) => done(err));
  });
}

// Per-track JSON next to the file (".mp3.json" or ".json")
async function readTrackJson(musicDir, relFile) {
  if (!relFile) return null;
  const candidates = [
    path.join(musicDir, relFile + ".json"),
    path.join(musicDir, relFile.replace(/\.[^.]+$/, "") + ".json"),
  ];
  for (const p of candidates) {
    try {
      const raw = await fs.readFile(p, "utf8");
      return JSON.parse(raw);
    } catch (_) {}
  }
  return null;
}

// Build browser stream URL (prefer explicit baseUrl)
function streamUrlFor(stream) {
  return stream.baseUrl || "";
}

// Rewrite Plex URLs to go through local proxy
function proxyUrl(url) {
  if (!url || typeof url !== "string") return url;
  if (url.includes("plex")) return "/api/cover-proxy?url=" + encodeURIComponent(url);
  return url;
}

function proxyArtist(card) {
  if (!card) return null;
  const out = { ...card };
  if (out.thumb_url) out.thumb_url = proxyUrl(out.thumb_url);
  if (out.art_url) out.art_url = proxyUrl(out.art_url);
  if (out.albums) {
    const proxiedAlbums = {};
    for (const [k, v] of Object.entries(out.albums)) {
      proxiedAlbums[k] = { ...v };
      if (proxiedAlbums[k].cover_url) proxiedAlbums[k].cover_url = proxyUrl(proxiedAlbums[k].cover_url);
    }
    out.albums = proxiedAlbums;
  }
  return out;
}

async function getNowPlaying(stream) {
  const [song, status] = await Promise.all([
    mpdCommand({ host: MPD_HOST, port: stream.controlPort }, "currentsong"),
    mpdCommand({ host: MPD_HOST, port: stream.controlPort }, "status"),
  ]);

  const relFile = song.file || song.File || "";
  const j = await readTrackJson(stream.musicDir, relFile);

  const jTrack = j?.track || {};
  const jAlbum = j?.album || {};
  const jArtist = j?.artist || {};

  const title  = jTrack.title  || song.Title || path.basename(relFile);
  const artist = jTrack.artist || song.Artist || jArtist.name || "";
  const album  = jTrack.album  || song.Album || "";
  const year   = jTrack.year || jAlbum.year || (song.Date ? String(song.Date).slice(0, 4) : null);

  const cover_url = proxyUrl(jAlbum.cover_url || jTrack.cover_url || null);
  const duration_sec =
    (status.duration ? Number(status.duration) : null) ||
    (jTrack.duration_ms ? Math.round(jTrack.duration_ms / 1000) : null);

  const key = String(artist || "").toLowerCase();
  const artistCard = key && artistsIndex.has(key) ? artistsIndex.get(key) : null;

  return {
    stream: { id: stream.id, name: stream.name, streamUrl: streamUrlFor(stream) },
    now: {
      file: relFile,
      title, artist, album, year,
      cover_url, duration_sec,
      mpd: { elapsed: status.elapsed ? Number(status.elapsed) : null },
    },
    artist: proxyArtist(artistCard),
  };
}

// ──────────────── APP ─────────────────────
const app = express();
app.use(cors());
app.use(express.json());

// Quiet the favicon 404s
// Serve React production build
const distPath = path.join(__dirname, "frontend", "dist");
if (fssync.existsSync(distPath)) {
  app.use(express.static(distPath));
}
app.get("/favicon.ico", (_req, res) => res.status(204).end());

// API: list streams
app.get("/api/streams", (_req, res) => {
  res.json(
    streams.map((s) => ({
      id: s.id,
      name: s.name,
      controlPort: s.controlPort,
      streamUrl: streamUrlFor(s),
    }))
  );
});

// API: now playing (merged)
app.get("/api/streams/:id/now", async (req, res) => {
  const s = streams.find((x) => x.id === req.params.id);
  if (!s) return res.status(404).json({ error: "Unknown stream id" });
  try {
    res.json(await getNowPlaying(s));
  } catch (e) {
    res.status(500).json({ error: e.message });
  }
});

// API: artist by name (case-insensitive)
app.get("/api/artists/:name", (req, res) => {
  const key = decodeURIComponent(req.params.name || "").toLowerCase();
  if (!key || !artistsIndex.has(key)) return res.status(404).json({ error: "Artist not found" });
  res.json(proxyArtist(artistsIndex.get(key)));
});

// API: proxy Plex cover art
app.get("/api/cover-proxy", (req, res) => {
  const rawUrl = req.query.url;
  if (!rawUrl) return res.status(400).json({ error: "Missing url param" });
  try {
    const parsed = new URL(rawUrl);
    const localUrl = PLEX_LOCAL + parsed.pathname + "?X-Plex-Token=" + PLEX_TOKEN;
    https.get(localUrl, { rejectUnauthorized: false }, (upstream) => {
      if (upstream.statusCode !== 200) {
        return res.status(upstream.statusCode || 502).end();
      }
      res.set("Content-Type", upstream.headers["content-type"] || "image/jpeg");
      res.set("Cache-Control", "public, max-age=86400");
      upstream.pipe(res);
    }).on("error", () => res.status(502).end());
  } catch {
    res.status(400).json({ error: "Invalid url" });
  }
});

// FRONTEND (single-page app)
app.get("/", (_req, res) => {
  res.type("html").send(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Webradio · Now Playing</title>
<style>
  :root { color-scheme: dark; --bg:#0b0d10; --panel:#13171c; --muted:#98a2b3; --text:#e6edf3; --accent:#7dd3fc; --accent2:#a78bfa; }
  *{box-sizing:border-box} body{margin:0;background:var(--bg);color:var(--text);font:16px/1.5 system-ui,Inter,Roboto,Arial}
  a{color:var(--accent)} .container{display:grid;grid-template-columns:280px 1fr;min-height:100svh}
  .sidebar{border-right:1px solid #1f2430;background:var(--panel);padding:16px;overflow:auto}
  .brand{font-weight:700;letter-spacing:.5px;margin:0 0 12px}
  .stream{padding:10px 12px;border-radius:12px;cursor:pointer;margin-bottom:8px;background:#0f1318;border:1px solid #202632}
  .stream.active{outline:2px solid var(--accent)}
  .main{padding:20px;display:flex;align-items:center;justify-content:center}
  .card{max-width:980px;width:100%;background:linear-gradient(180deg,#111520,#0e1118);border:1px solid #22293a;border-radius:20px;box-shadow:0 30px 60px rgba(0,0,0,.3);padding:22px}
  .now{display:grid;grid-template-columns:220px 1fr;gap:20px;align-items:center}
  .cover{width:100%;aspect-ratio:1;object-fit:cover;border-radius:16px;border:1px solid #263041}
  .title{font-size:24px;font-weight:700;margin:0}
  .meta{color:var(--muted);margin-top:4px}
  .artist{font-weight:600;cursor:pointer;color:var(--accent2)}
  .row{display:flex;align-items:center;gap:16px;flex-wrap:wrap;margin-top:16px}
  .vol{flex:1}
  input[type=range]{width:100%}
  .audio{display:flex;align-items:center;gap:10px}
  .chip{display:inline-flex;padding:4px 10px;border-radius:9999px;background:#0f141b;border:1px solid #202632;color:#cbd5e1;font-size:12px}
  .muted{color:#9aa4b2}
  button{background:#171c24;border:1px solid #2b3445;color:#e6edf3;border-radius:12px;padding:10px 14px;cursor:pointer}
  button:hover{border-color:#3b4760}
</style>
</head>
<body>
<div class="container">
  <aside class="sidebar">
    <h3 class="brand">MPD Streams</h3>
    <div id="streams"></div>
    <div style="margin-top:12px;font-size:12px;color:var(--muted)">Click a stream, then press “Play”.</div>
  </aside>
  <main class="main">
    <div class="card"><div id="view"></div></div>
  </main>
</div>

<script>
const $ = s => document.querySelector(s);
const $$ = s => Array.from(document.querySelectorAll(s));
let STREAMS = [];
let current = null;
let pollTimer = null;

const audioEl = new Audio();
audioEl.setAttribute('playsinline','');
audioEl.preload = "none";   // no autoplay (blocked by browsers)

// Persisted local-only volume
let savedVol = Number(localStorage.getItem("playerVolPct") || "85");
if (!Number.isFinite(savedVol) || savedVol < 0 || savedVol > 100) savedVol = 85;
audioEl.volume = savedVol / 100;

audioEl.addEventListener("error", () => {
  const code = (audioEl.error && audioEl.error.code) || 0;
  const map = {1:"ABORTED",2:"NETWORK",3:"DECODE",4:"SRC_NOT_SUPPORTED"};
  console.warn("audio error", map[code]||code);
});

function fmtTime(s){ if(!Number.isFinite(s)) return "—:—"; s = Math.max(0, Math.floor(s)); const m = Math.floor(s/60), ss = String(s%60).padStart(2,"0"); return m+":"+ss; }

async function loadStreams() {
  const res = await fetch('/api/streams');
  STREAMS = await res.json();
  $('#streams').innerHTML = STREAMS.map(s =>
    \`<div class="stream" data-id="\${s.id}">
       <div style="font-weight:600">\${s.name}</div>
       <div class="muted" style="font-size:12px">\${s.streamUrl}</div>
     </div>\`
  ).join("");
  $$('#streams .stream').forEach(el => el.onclick = () => selectStream(el.dataset.id));
  if (STREAMS[0]) selectStream(STREAMS[0].id);
}

async function selectStream(id) {
  if (current?.id === id) return;
  current = STREAMS.find(s => s.id === id);
  $$('#streams .stream').forEach(el => el.classList.toggle('active', el.dataset.id === id));
  renderNow({ loading:true });
  if (pollTimer) clearInterval(pollTimer);
  await refreshNow();
  pollTimer = setInterval(refreshNow, 5000);
}

async function refreshNow() {
  if (!current) return;
  try {
    const res = await fetch(\`/api/streams/\${current.id}/now\`);
    const data = await res.json();
    renderNow({ data });
  } catch {
    renderNow({ error:true });
  }
}

function renderNow({ data, loading, error }) {
  if (loading) { $('#view').innerHTML = '<div class="muted">Loading…</div>'; return; }
  if (error || !data) { $('#view').innerHTML = '<div class="muted">Unable to load now playing.</div>'; return; }

  const it = data.now || {};
  const artist = it.artist || "Unknown artist";
  const cover = it.cover_url || "";
  const album = it.album || "";
  const year = it.year ? \` (\${it.year})\` : "";
  const dur = it.duration_sec;
  const elapsed = (it.mpd && it.mpd.elapsed) || null;

  $('#view').innerHTML = \`
    <div class="now">
      <img class="cover" src="\${cover}" onerror="this.style.visibility='hidden'">
      <div>
        <h1 class="title">\${it.title || 'Unknown title'}</h1>
        <div class="meta">
          <span class="artist" id="artistLink">\${artist}</span>
          <span class="muted"> • </span>
          <span>\${album}\${year}</span>
          <span class="muted"> • </span>
          <span>\${fmtTime(elapsed)} / \${fmtTime(dur)}</span>
        </div>
        <div class="row">
          <div class="audio">
            <button id="playpause">▶ Play</button>
            <button id="mute">🔇 Mute</button>
            <span class="chip">\${data.stream.name}</span>
          </div>
          <div class="vol">
            <input id="vol" type="range" min="0" max="100" value="\${Math.round(audioEl.volume*100)}">
          </div>
          <div class="chip">Local volume</div>
        </div>
      </div>
    </div>
  \`;

  $('#artistLink').onclick = () => openArtist(artist);
  $('#mute').onclick = () => { audioEl.muted = !audioEl.muted; };
  $('#playpause').onclick = async () => {
    try {
      if (!audioEl.src || audioEl.src !== data.stream.streamUrl) {
        audioEl.src = data.stream.streamUrl;
        audioEl.load();
      }
      if (audioEl.paused) {
        await audioEl.play(); // user gesture
        $('#playpause').textContent = '⏸ Pause';
      } else {
        audioEl.pause();
        $('#playpause').textContent = '▶ Play';
      }
    } catch (e) {
      alert('Unable to start audio. Check HTTPS reachability of the stream URL.');
    }
  };

  const volInput = $('#vol');
  if (volInput) {
    volInput.oninput = () => {
      const v = Math.max(0, Math.min(100, Number(volInput.value)));
      audioEl.volume = v / 100;
      localStorage.setItem("playerVolPct", String(v));
    };
  }
}

async function openArtist(name) {
  if (!name) return;
  try {
    const res = await fetch('/api/artists/' + encodeURIComponent(name));
    if (!res.ok) return;
    const a = await res.json();
    $('#view').innerHTML = \`
      <div style="margin-bottom:14px"><button onclick="refreshNow()">← Back to Now Playing</button></div>
      <div style="display:grid;grid-template-columns:220px 1fr;gap:18px">
        <img class="cover" src="\${a.art_url||a.thumb_url||''}" onerror="this.style.display='none'">
        <div>
          <h2 style="margin:0 0 8px">\${a.name}</h2>
          <div class="muted" style="white-space:pre-wrap;max-height:220px;overflow:auto">\${(a.bio||'').slice(0,1500)}</div>
        </div>
      </div>\`;
  } catch (_) {}
}

loadStreams();
</script>
</body>
</html>`);
});

// ─────────────── START ────────────────────
const PORT = process.env.PORT || 20000;
app.listen(PORT, () => console.log(`✓ Webradio UI/API on http://localhost:${PORT}`));

