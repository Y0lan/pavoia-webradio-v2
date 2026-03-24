import React, { useEffect, useMemo, useRef, useState, useCallback } from "react";
import { StreamsList } from "./components/sidebar/StreamsList";
import { StreamsDrawer } from "./components/mobile/StreamsDrawer";
import { PlayPauseButton } from "./components/player/PlayPauseButton";
import { NowPlaying } from "./components/player/NowPlaying";
import { ArtistDrawer } from "./components/drawers/ArtistDrawer";
import { InfoDialog } from "./components/dialogs/InfoDialog";
import { BusMysteryCard } from "./components/BusMysteryCard";
import { getMetaFor } from "./utils/streamMeta";
import { fmtTime } from "./utils/formatters";

export default function App() {
  // ─── Core state (Browse & Switch model) ────────────────────────────────────
  const [streams, setStreams] = useState([]);
  const [viewingStreamId, setViewingStreamId] = useState(null);   // what you SEE
  const [playingStreamId, setPlayingStreamId] = useState(null);   // what you HEAR
  const [now, setNow] = useState(null);           // metadata for VIEWING stream
  const [playingNow, setPlayingNow] = useState(null); // metadata for PLAYING stream
  const [artistCard, setArtistCard] = useState(null);
  const [loadingStreams, setLoadingStreams] = useState(true);
  const [err, setErr] = useState("");
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [mobileStreamsOpen, setMobileStreamsOpen] = useState(false);
  const [infoOpen, setInfoOpen] = useState(false);
  const [busCardOpen, setBusCardOpen] = useState(false);

  const audioStreamIdRef = useRef(null);

  // ─── Audio + player state ────────────────────────────────────────────────────
  const audioRef = useRef(null);
  const [player, setPlayer] = useState({ status: "idle", msg: "" });

  // Volume
  const [vol, setVol] = useState(() => {
    const saved = Number(localStorage.getItem("playerVolPct") || 85);
    return Number.isFinite(saved) ? Math.max(0, Math.min(100, saved)) : 85;
  });
  const [muted, setMuted] = useState(false);

  useEffect(() => {
    if (audioRef.current) {
      audioRef.current.volume = vol / 100;
      localStorage.setItem("playerVolPct", String(vol));
    }
  }, [vol]);

  useEffect(() => {
    if (audioRef.current) audioRef.current.muted = muted;
  }, [muted]);

  // ─── Streams bootstrap ───────────────────────────────────────────────────────
  useEffect(() => {
    let alive = true;
    (async () => {
      try {
        const r = await fetch("/api/streams", { cache: "no-store" });
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        const data = await r.json();
        if (!alive) return;
        setStreams(data);
        const last = localStorage.getItem("activeStreamId");
        const def = data.find((s) => s.id === last) || data[0];
        if (def) setViewingStreamId(def.id);
      } catch {
        setErr("Unable to load streams");
      } finally {
        setLoadingStreams(false);
      }
    })();
    return () => { alive = false; };
  }, []);

  // Sorted stream IDs for navigation
  const streamOrder = useMemo(() => {
    const order = {
      "gaende-favorites": 0, "ambiance-safe": 1, "bermuda-day": 2, "bermuda-night": 3,
      "palac-slow-hypno": 4, "palac-dance": 5, "fontanna-laputa": 6, "etage-0": 7, "closing": 8, "bus": 9,
    };
    return [...streams].sort((a, b) => (order[a.id] ?? 999) - (order[b.id] ?? 999)).map(s => s.id);
  }, [streams]);

  const viewingStream = useMemo(
    () => streams.find((s) => s.id === viewingStreamId) || null,
    [streams, viewingStreamId]
  );
  const playingStream = useMemo(
    () => streams.find((s) => s.id === playingStreamId) || null,
    [streams, playingStreamId]
  );

  const isExploring = viewingStreamId && playingStreamId && viewingStreamId !== playingStreamId;

  // Preconnect to stream origins
  useEffect(() => {
    if (!streams.length) return;
    const created = [];
    for (const s of streams) {
      const url = s.streamUrl || "";
      if (!url) continue;
      try {
        const u = new URL(url, window.location.origin);
        const link = document.createElement("link");
        link.rel = "preconnect";
        link.href = u.port ? `${u.protocol}//${u.hostname}:${u.port}` : `${u.protocol}//${u.hostname}`;
        link.crossOrigin = "anonymous";
        document.head.appendChild(link);
        created.push(link);
      } catch {}
    }
    return () => created.forEach((el) => el.remove());
  }, [streams]);

  // ─── Poll now-playing for VIEWING stream ──────────────────────────────────────
  const pollRef = useRef({ timer: null, delay: 5000 });
  useEffect(() => {
    if (!viewingStream) return;
    let alive = true;

    async function fetchNow() {
      try {
        const r = await fetch(`/api/streams/${viewingStream.id}/now`, { cache: "no-store" });
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        const data = await r.json();
        if (!alive) return;
        setNow(data.now);
        setArtistCard(data.artist || null);
        setErr("");
        pollRef.current.delay = 5000;
        // Only update media session if this is the playing stream
        if (viewingStream.id === playingStreamId) {
          updateMediaSession(data);
          setPlayingNow(data.now);
        }
      } catch {
        if (!alive) return;
        setErr("Connection lost. Retrying…");
        pollRef.current.delay = Math.min(20000, Math.round((pollRef.current.delay || 5000) * 1.5));
      } finally {
        schedule();
      }
    }

    function schedule() {
      clearTimeout(pollRef.current.timer);
      const jitter = Math.floor(Math.random() * 400);
      pollRef.current.timer = setTimeout(fetchNow, pollRef.current.delay + jitter);
    }

    fetchNow();
    return () => { alive = false; clearTimeout(pollRef.current.timer); };
  }, [viewingStream, playingStreamId]);

  // ─── Poll now-playing for PLAYING stream (when exploring a different one) ────
  const playingPollRef = useRef({ timer: null, delay: 5000 });
  useEffect(() => {
    if (!playingStreamId || !isExploring) return;
    let alive = true;

    async function fetchPlayingNow() {
      try {
        const r = await fetch(`/api/streams/${playingStreamId}/now`, { cache: "no-store" });
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        const data = await r.json();
        if (!alive) return;
        setPlayingNow(data.now);
        updateMediaSession(data);
      } catch {
        // silently retry
      } finally {
        clearTimeout(playingPollRef.current.timer);
        playingPollRef.current.timer = setTimeout(fetchPlayingNow, 5000);
      }
    }

    fetchPlayingNow();
    return () => { alive = false; clearTimeout(playingPollRef.current.timer); };
  }, [playingStreamId, isExploring]);

  // ─── Audio events → resilient player state ─────────────────────────────────
  useEffect(() => {
    const a = audioRef.current;
    if (!a) return;

    let waitingTimer = null;
    const clearWaiting = () => { if (waitingTimer) { clearTimeout(waitingTimer); waitingTimer = null; } };
    const showLoading = (msg) => {
      clearWaiting();
      waitingTimer = setTimeout(() => { if (!a.paused) setPlayer({ status: "loading", msg }); }, 150);
    };
    const setPlaying = () => {
      clearWaiting();
      setPlayer({ status: "playing", msg: "" });
      setPlayingStreamId(audioStreamIdRef.current || playingStreamId || null);
    };
    const setPaused = (m = "") => {
      clearWaiting();
      setPlayer({ status: "paused", msg: m });
    };

    const onLoadStart = () => showLoading("Connecting…");
    const onPlay = () => { if (!a.paused && (a.currentTime > 0 || a.readyState >= 2)) setPlaying(); else showLoading("Connecting…"); };
    const onWaiting = () => showLoading("Buffering…");
    const onTimeUpdate = () => { if (!a.paused && a.currentTime > 0) setPlaying(); };
    const onCanPlay = () => { if (!a.paused) setPlaying(); else setPaused(); };
    const onLoadedData = () => { if (!a.paused) setPlaying(); };
    const onLoadedMetadata = () => { if (!a.paused) setPlaying(); };
    const onProgress = () => { if (!a.paused && (a.currentTime > 0 || a.readyState >= 2)) setPlaying(); };
    const onPlaying = () => setPlaying();
    const onPause = () => setPaused();
    const onStalled = () => showLoading("Buffering…");
    const onEnded = () => setPaused("Ended");
    const onError = () => { setPlayer({ status: "error", msg: "Playback error" }); };

    const events = [
      ["loadstart", onLoadStart], ["play", onPlay], ["waiting", onWaiting],
      ["timeupdate", onTimeUpdate], ["canplay", onCanPlay], ["loadeddata", onLoadedData],
      ["loadedmetadata", onLoadedMetadata], ["progress", onProgress], ["playing", onPlaying],
      ["pause", onPause], ["stalled", onStalled], ["ended", onEnded], ["error", onError],
    ];
    events.forEach(([e, h]) => a.addEventListener(e, h));
    const health = setInterval(() => { if (!a.paused && (a.currentTime > 0 || a.readyState >= 3)) setPlaying(); }, 500);

    return () => {
      events.forEach(([e, h]) => a.removeEventListener(e, h));
      clearWaiting();
      clearInterval(health);
    };
  }, [playingStreamId]);

  // Save viewing stream to localStorage
  useEffect(() => {
    if (viewingStreamId) localStorage.setItem("activeStreamId", viewingStreamId);
  }, [viewingStreamId]);

  // ─── Playable URL ──────────────────────────────────────────────────────────
  function getPlayableUrl(s) {
    if (!s) return "";
    let url = s.streamUrl || "";
    try {
      if (url.startsWith("/")) return new URL(url, window.location.origin).toString();
      if (window.location.protocol === "https:" && url.startsWith("http://")) url = "https://" + url.slice("http://".length);
      return url;
    } catch { return url; }
  }

  // ─── Elapsed/duration tracking ─────────────────────────────────────────────
  const [elapsedSec, setElapsedSec] = useState(0);
  const [durationSec, setDurationSec] = useState(null);
  const baseStartRef = useRef(performance.now());

  useEffect(() => {
    if (!now) return;
    const serverElapsed = Number(now?.mpd?.elapsed || 0);
    const serverDuration = Number(now?.duration_sec) || null;
    setDurationSec(serverDuration || null);
    setElapsedSec(serverElapsed);
    baseStartRef.current = performance.now() - serverElapsed * 1000;
  }, [now?.file, viewingStreamId]);

  useEffect(() => {
    const id = setInterval(() => {
      if (player.status !== "playing") return;
      const ms = performance.now() - baseStartRef.current;
      const sec = ms / 1000;
      setElapsedSec(durationSec ? Math.min(sec, durationSec) : sec);
    }, 250);
    return () => clearInterval(id);
  }, [player.status, durationSec]);

  useEffect(() => {
    const a = audioRef.current;
    if (!a) return;
    const onMeta = () => { if (isFinite(a.duration) && a.duration > 0) setDurationSec(Math.floor(a.duration)); };
    a.addEventListener("loadedmetadata", onMeta);
    return () => a.removeEventListener("loadedmetadata", onMeta);
  }, []);

  // ─── Share helper ──────────────────────────────────────────────────────────
  async function shareApp() {
    const url = window.location.href;
    try {
      if (navigator.share) await navigator.share({ title: "PAVOIA WEBRADIO", text: "Tune in to PAVOIA WEBRADIO", url });
      else if (navigator.clipboard) { await navigator.clipboard.writeText(url); alert("Link copied to clipboard."); }
    } catch {}
  }

  // ─── Play stream (switches audio) ─────────────────────────────────────────
  async function switchToStream(targetStream) {
    const a = audioRef.current;
    const s = targetStream;
    if (!a || !s) return;

    let target = getPlayableUrl(s);
    if (!target || !/^https?:\/\//i.test(target)) target = (s.streamUrl || "").trim();
    if (!target) { console.error("No stream URL", s); return; }

    if (!a.src || a.src !== target) {
      a.src = target;
      a.load();
      audioStreamIdRef.current = s.id;
    }
    try {
      setPlayer({ status: "loading", msg: "Connecting…" });
      await a.play();
      setTimeout(() => { if (!a.paused) { setPlayer({ status: "playing", msg: "" }); setPlayingStreamId(s.id); } }, 50);
      setTimeout(() => { if (!a.paused && (a.currentTime >= 0 || a.readyState >= 2)) { setPlayer({ status: "playing", msg: "" }); setPlayingStreamId(s.id); } }, 1500);
    } catch (e) {
      setPlayer({ status: "error", msg: "Unable to start audio" });
      console.error("audio.play() failed", e);
    }
  }

  function togglePlay() {
    const a = audioRef.current;
    if (!a) return;

    if (a.paused) {
      // If nothing playing yet, play the viewing stream
      const s = playingStream || viewingStream;
      if (s) switchToStream(s);
    } else {
      a.pause();
    }
  }

  // ─── Browse: click stream in sidebar (view only, no audio change) ──────────
  const browseStream = useCallback((id, closeDrawer = false) => {
    const s = streams.find((x) => x.id === id);
    if (!s) return;

    const meta = getMetaFor(s);
    if (meta.disabled) {
      setBusCardOpen(true);
      return;
    }

    setViewingStreamId(id);
    if (closeDrawer) setMobileStreamsOpen(false);

    // If nothing is playing yet, also start playing
    if (!playingStreamId && player.status !== "playing" && player.status !== "loading") {
      switchToStream(s);
    }
  }, [streams, playingStreamId, player.status]);

  // ─── Switch: explicit switch to the viewed stream ──────────────────────────
  const switchToViewing = useCallback(() => {
    const s = viewingStream;
    if (!s) return;
    switchToStream(s);
  }, [viewingStream]);

  // Effective status for play/pause button
  const effectiveStatus = viewingStreamId === playingStreamId ? player.status : "paused";

  // ─── Keyboard shortcuts ────────────────────────────────────────────────────
  useEffect(() => {
    let lastKeyTime = 0;
    function onKeyDown(e) {
      const tag = document.activeElement?.tagName;
      if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT" || document.activeElement?.isContentEditable) return;

      const now = Date.now();
      if (now - lastKeyTime < 300 && (e.key === "ArrowLeft" || e.key === "ArrowRight")) return;
      lastKeyTime = now;

      const activeStreams = streamOrder.filter(id => !getMetaFor(streams.find(s => s.id === id))?.disabled);

      switch (e.key) {
        case " ":
          e.preventDefault();
          togglePlay();
          break;
        case "ArrowLeft": {
          const idx = activeStreams.indexOf(viewingStreamId);
          const prev = idx <= 0 ? activeStreams[activeStreams.length - 1] : activeStreams[idx - 1];
          if (prev) setViewingStreamId(prev);
          break;
        }
        case "ArrowRight": {
          const idx = activeStreams.indexOf(viewingStreamId);
          const next = idx >= activeStreams.length - 1 ? activeStreams[0] : activeStreams[idx + 1];
          if (next) setViewingStreamId(next);
          break;
        }
        case "Enter":
          if (isExploring) switchToViewing();
          break;
        case "m":
        case "M":
          setMuted(m => !m);
          break;
        case "ArrowUp":
          e.preventDefault();
          setVol(v => Math.min(100, v + 5));
          break;
        case "ArrowDown":
          e.preventDefault();
          setVol(v => Math.max(0, v - 5));
          break;
        default:
          if (e.key >= "1" && e.key <= "9") {
            const idx = parseInt(e.key) - 1;
            if (activeStreams[idx]) setViewingStreamId(activeStreams[idx]);
          }
      }
    }
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [streamOrder, viewingStreamId, playingStreamId, isExploring, streams]);

  // ─── Media Session ─────────────────────────────────────────────────────────
  function updateMediaSession(data) {
    if (!("mediaSession" in navigator) || !data?.now) return;
    const n = data.now;
    try {
      navigator.mediaSession.metadata = new window.MediaMetadata({
        title: n.title || "", artist: n.artist || "", album: n.album || "",
        artwork: n.cover_url ? [{ src: n.cover_url, sizes: "512x512", type: "image/png" }] : [],
      });
      navigator.mediaSession.setActionHandler("play", () => audioRef.current?.play());
      navigator.mediaSession.setActionHandler("pause", () => audioRef.current?.pause());
    } catch {}
  }

  // ─── Render ────────────────────────────────────────────────────────────────
  const viewingMeta = viewingStream ? getMetaFor(viewingStream) : null;
  const playingMeta = playingStream ? getMetaFor(playingStream) : null;

  return (
    <div className="min-h-screen text-white relative">
      {/* Gradient background — two stacked layers for GPU-composited opacity transition */}
      <div
        className={`fixed inset-0 bg-gradient-to-b ${viewingMeta?.bgGradient || "from-[#0b0d10] to-[#111520]"} transition-opacity duration-[800ms] ease-in-out`}
        style={{ zIndex: -2 }}
      />
      <div className="fixed inset-0 bg-[#0b0d10]" style={{ zIndex: -3 }} />

      {/* Now-playing strip (shown when exploring a different stream) */}
      {isExploring && playingMeta && (
        <div
          className="fixed top-0 left-0 right-0 z-50 h-12 flex items-center gap-3 px-4 cursor-pointer border-b"
          style={{ backgroundColor: "rgba(0,0,0,0.85)", borderColor: playingMeta.accentColor + "40" }}
          onClick={() => setViewingStreamId(playingStreamId)}
          title="Return to playing stream"
        >
          <div className="w-1 h-8 rounded-full" style={{ backgroundColor: playingMeta.accentColor }} />
          {playingNow?.cover_url && (
            <img src={playingNow.cover_url} alt="" className="w-8 h-8 rounded object-cover" />
          )}
          <span className="text-sm font-medium truncate">
            {playingMeta.icon} Now playing: {playingNow?.artist || ""} — {playingNow?.title || playingMeta.title}
          </span>
          <span className="ml-auto text-xs text-slate-400">tap to return</span>
        </div>
      )}

      {/* Mobile header */}
      <header
        className={`md:hidden sticky ${isExploring ? "top-12" : "top-0"} z-40 bg-black/80 backdrop-blur border-b border-white/10 px-4 py-3 grid grid-cols-[auto_1fr_auto] items-center`}
      >
        <button
          onClick={() => setMobileStreamsOpen(true)}
          className="px-3 py-2 rounded-lg border border-white/20 bg-white/5 text-sm"
          aria-label="Open streams"
        >
          ☰ Explore
        </button>
        <div className="px-2" aria-hidden="true" />
        <button
          onClick={() => setInfoOpen(true)}
          className="px-3 py-2 rounded-lg border border-white/20 bg-white/5 text-sm"
          aria-label="Info"
        >
          ℹ️
        </button>
      </header>

      <div className={`md:grid md:grid-cols-[320px_1fr] ${isExploring ? "md:pt-12" : ""}`}>
        {/* Sidebar (desktop) */}
        <aside className="hidden md:block border-r border-white/10 bg-black/60 backdrop-blur-sm p-4 overflow-auto min-h-screen">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-bold tracking-wide">Explore</h3>
            <button
              onClick={() => setInfoOpen(true)}
              className="px-2 py-1 rounded-lg border border-white/20 bg-white/5 hover:border-white/30 text-sm"
              title="About"
            >
              ℹ️
            </button>
          </div>
          {loadingStreams && <div className="text-sm text-slate-400">Loading streams…</div>}
          {!loadingStreams && streams.length === 0 && <div className="text-sm text-slate-400">No streams found.</div>}
          <StreamsList
            streams={streams}
            viewingId={viewingStreamId}
            playingId={playingStreamId}
            isPlaying={player.status === "playing"}
            onSelect={browseStream}
          />
          {err && <div className="mt-3 text-xs text-amber-300">{err}</div>}
        </aside>

        {/* Main */}
        <main className="p-4 md:p-5 flex items-center justify-center min-h-[80vh]">
          <div className="w-full max-w-3xl">
            {!viewingStream ? (
              <div className="text-slate-400 text-center py-20">
                <p className="text-lg mb-2">Pick a stage to begin</p>
                <p className="text-sm">Choose a stream from the sidebar to start listening</p>
              </div>
            ) : (
              <>
                <NowPlaying
                  now={now}
                  artistCard={artistCard}
                  player={player}
                  onArtistClick={() => artistCard && setDrawerOpen(true)}
                  elapsed={elapsedSec}
                  duration={durationSec}
                  showPlaying={viewingStreamId === playingStreamId && player.status === "playing"}
                  isExploring={isExploring}
                  viewingMeta={viewingMeta}
                />

                {/* Controls — same position for play/pause AND switch */}
                <div className="mt-5 flex md:flex-row flex-col items-stretch md:items-center gap-3">
                  <div className="flex gap-3">
                    {isExploring ? (
                      <button
                        onClick={switchToViewing}
                        className="group relative inline-flex items-center gap-3 px-5 py-2.5 rounded-xl border transition shadow-sm"
                        style={{
                          borderColor: viewingMeta?.accentColor || "#64748b",
                          backgroundColor: (viewingMeta?.accentColor || "#64748b") + "20",
                        }}
                      >
                        <span className="text-sm font-semibold">Switch to this stage</span>
                      </button>
                    ) : (
                      <PlayPauseButton status={effectiveStatus} onClick={togglePlay} />
                    )}
                    <button
                      onClick={() => setMuted((m) => !m)}
                      className="px-4 py-2 rounded-xl border border-white/20 bg-white/5 hover:border-white/30"
                    >
                      {muted || vol === 0 ? "🔇" : "🔊"}
                    </button>
                  </div>

                  <div className="flex items-center gap-3 flex-1 min-w-[200px]">
                    <input
                      type="range" min={0} max={100}
                      value={muted ? 0 : vol}
                      className="w-full accent-cyan-300"
                      onChange={(e) => {
                        const v = Math.max(0, Math.min(100, Number(e.target.value)));
                        setVol(v);
                        if (v > 0 && muted) setMuted(false);
                      }}
                    />
                    <span className="text-xs text-slate-300 w-10 text-right">{muted ? 0 : vol}%</span>
                  </div>
                </div>

                {/* Stream info */}
                {viewingMeta && (
                  <div className="mt-3 text-sm text-white/70 flex flex-wrap items-center gap-2">
                    <span className="font-semibold">{viewingMeta.icon} {viewingMeta.title}</span>
                    <span className="text-white/30">·</span>
                    <span className="text-white/50">{viewingMeta.desc}</span>
                  </div>
                )}

                {/* Hidden audio element */}
                <audio ref={audioRef} preload="none" playsInline className="hidden" crossOrigin="anonymous" />

                {/* Artist Drawer */}
                <ArtistDrawer open={drawerOpen} onClose={() => setDrawerOpen(false)} artist={artistCard} />
              </>
            )}
          </div>
        </main>
      </div>

      {/* Mobile Streams Drawer */}
      <StreamsDrawer
        open={mobileStreamsOpen}
        onClose={() => setMobileStreamsOpen(false)}
        streams={streams}
        viewingId={viewingStreamId}
        playingId={playingStreamId}
        isPlaying={player.status === "playing"}
        onSelect={(id) => browseStream(id, true)}
      />

      {/* Info Modal */}
      <InfoDialog open={infoOpen} onClose={() => setInfoOpen(false)} onShare={shareApp} />

      {/* Bus Mystery Card */}
      <BusMysteryCard open={busCardOpen} onClose={() => setBusCardOpen(false)} />
    </div>
  );
}
