import React, { useEffect, useMemo, useRef, useState, useCallback } from "react";
import { StreamsList } from "./components/sidebar/StreamsList";
import { StreamsDrawer } from "./components/mobile/StreamsDrawer";
import { PlayPauseButton } from "./components/player/PlayPauseButton";
import { NowPlaying } from "./components/player/NowPlaying";
import { ArtistDrawer } from "./components/drawers/ArtistDrawer";
import { InfoDialog } from "./components/dialogs/InfoDialog";
import { BusMysteryCard } from "./components/BusMysteryCard";
import { StreamPreview } from "./components/StreamPreview";
import { useCrossfade } from "./hooks/useCrossfade";
import { useSwipeNavigation } from "./hooks/useSwipeNavigation";
import { getMetaFor } from "./utils/streamMeta";
import { fmtTime } from "./utils/formatters";

export default function App() {
  // ─── Core state (Browse & Switch model) ────────────────────────────────────
  const [streams, setStreams] = useState([]);
  const [viewingStreamId, setViewingStreamId] = useState(null);
  const [now, setNow] = useState(null);
  const [playingNow, setPlayingNow] = useState(null);
  const [artistCard, setArtistCard] = useState(null);
  const [loadingStreams, setLoadingStreams] = useState(true);
  const [err, setErr] = useState("");
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [mobileStreamsOpen, setMobileStreamsOpen] = useState(false);
  const [infoOpen, setInfoOpen] = useState(false);
  const [busCardOpen, setBusCardOpen] = useState(false);

  // Stream preview state
  const [previewStreamId, setPreviewStreamId] = useState(null);
  const [previewPos, setPreviewPos] = useState(null);
  const [allStreamsMeta, setAllStreamsMeta] = useState({});

  // "Recently playing" memory
  const [lastSeen, setLastSeen] = useState(() => {
    try { return JSON.parse(localStorage.getItem("lastSeenStreams") || "{}"); }
    catch { return {}; }
  });
  const [wasPlayingToast, setWasPlayingToast] = useState(null);

  // ─── Crossfade audio engine ────────────────────────────────────────────────
  const crossfade = useCrossfade();
  const { status: playerStatus, statusMsg, currentStreamId: playingStreamId,
    play, pause, switchStream, volume: vol, setVolume: setVol,
    muted, setMuted, audioElements } = crossfade;

  const player = useMemo(() => ({ status: playerStatus, msg: statusMsg }), [playerStatus, statusMsg]);

  // ─── Streams bootstrap ───────────────────────────────────────────────────────
  useEffect(() => {
    let alive = true;
    (async () => {
      try {
        const r = await fetch("/api/streams", { cache: "no-store" });
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        const data = await r.json();
        if (!alive) return;
        if (!data.find(s => s.id === "bus")) data.push({ id: "bus", name: "Bus" });
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

  const activeStreams = useMemo(
    () => streamOrder.filter(id => { const s = streams.find(x => x.id === id); return s && !getMetaFor(s).disabled; }),
    [streamOrder, streams]
  );

  const viewingStream = useMemo(() => streams.find((s) => s.id === viewingStreamId) || null, [streams, viewingStreamId]);
  const playingStream = useMemo(() => streams.find((s) => s.id === playingStreamId) || null, [streams, playingStreamId]);
  const isExploring = !!(viewingStreamId && playingStreamId && viewingStreamId !== playingStreamId);

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

  // ─── Poll now-playing for VIEWING stream ──────────────────────────────────
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
      pollRef.current.timer = setTimeout(fetchNow, pollRef.current.delay + Math.floor(Math.random() * 400));
    }
    fetchNow();
    return () => { alive = false; clearTimeout(pollRef.current.timer); };
  }, [viewingStream, playingStreamId]);

  // ─── Poll now-playing for PLAYING stream (when exploring different one) ────
  const playingPollRef = useRef({ timer: null });
  useEffect(() => {
    if (!playingStreamId || !isExploring) return;
    let alive = true;
    async function fetchPlayingNow() {
      try {
        const r = await fetch(`/api/streams/${playingStreamId}/now`, { cache: "no-store" });
        if (!r.ok) throw new Error();
        const data = await r.json();
        if (!alive) return;
        setPlayingNow(data.now);
        updateMediaSession(data);
      } catch {}
      finally { clearTimeout(playingPollRef.current.timer); playingPollRef.current.timer = setTimeout(fetchPlayingNow, 5000); }
    }
    fetchPlayingNow();
    return () => { alive = false; clearTimeout(playingPollRef.current.timer); };
  }, [playingStreamId, isExploring]);

  // ─── Bulk fetch all streams metadata for previews (every 30s, pause when hidden) ──
  useEffect(() => {
    if (!streams.length) return;
    let alive = true;
    let timer = null;

    async function fetchAll() {
      if (document.visibilityState === "hidden") return;
      try {
        const results = await Promise.allSettled(
          streams.filter(s => !getMetaFor(s).disabled).map(async (s) => {
            const r = await fetch(`/api/streams/${s.id}/now`, { cache: "no-store" });
            if (!r.ok) return null;
            const data = await r.json();
            return { id: s.id, now: data.now };
          })
        );
        if (!alive) return;
        const meta = {};
        results.forEach(r => { if (r.status === "fulfilled" && r.value) meta[r.value.id] = r.value.now; });
        setAllStreamsMeta(meta);
      } catch {}
      finally { timer = setTimeout(fetchAll, 30000); }
    }

    fetchAll();
    const onVis = () => { if (document.visibilityState === "visible") { clearTimeout(timer); setTimeout(fetchAll, 2000); } };
    document.addEventListener("visibilitychange", onVis);
    return () => { alive = false; clearTimeout(timer); document.removeEventListener("visibilitychange", onVis); };
  }, [streams]);

  // Save viewing stream to localStorage
  useEffect(() => { if (viewingStreamId) localStorage.setItem("activeStreamId", viewingStreamId); }, [viewingStreamId]);

  // ─── Playable URL helper ───────────────────────────────────────────────────
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
    setDurationSec(Number(now?.duration_sec) || null);
    setElapsedSec(serverElapsed);
    baseStartRef.current = performance.now() - serverElapsed * 1000;
  }, [now?.file, viewingStreamId]);

  useEffect(() => {
    const id = setInterval(() => {
      if (playerStatus !== "playing") return;
      const sec = (performance.now() - baseStartRef.current) / 1000;
      setElapsedSec(durationSec ? Math.min(sec, durationSec) : sec);
    }, 250);
    return () => clearInterval(id);
  }, [playerStatus, durationSec]);

  // ─── Share ─────────────────────────────────────────────────────────────────
  async function shareApp() {
    const url = window.location.href;
    try {
      if (navigator.share) await navigator.share({ title: "PAVOIA WEBRADIO", text: "Tune in to PAVOIA WEBRADIO", url });
      else if (navigator.clipboard) { await navigator.clipboard.writeText(url); alert("Link copied to clipboard."); }
    } catch {}
  }

  // ─── Switch: play a specific stream via crossfade ──────────────────────────
  const doSwitchToStream = useCallback((s) => {
    if (!s) return;
    const url = getPlayableUrl(s);
    if (!url) return;
    switchStream(url, s.id);
  }, [switchStream]);

  function togglePlay() {
    if (playerStatus === "playing") {
      pause();
    } else if (playerStatus === "paused" && playingStreamId) {
      // Resume the same stream
      play();
    } else {
      // Nothing loaded yet — switch to viewing stream
      const s = playingStream || viewingStream;
      if (s) doSwitchToStream(s);
    }
  }

  const switchToViewing = useCallback(() => {
    if (viewingStream) doSwitchToStream(viewingStream);
  }, [viewingStream, doSwitchToStream]);

  // ─── Browse: click stream in sidebar ───────────────────────────────────────
  const prevViewingRef = useRef(null);
  const browseStream = useCallback((id, closeDrawer = false) => {
    const s = streams.find((x) => x.id === id);
    if (!s) return;
    const meta = getMetaFor(s);
    if (meta.disabled) { setBusCardOpen(true); return; }

    // Save "last seen" for the stream we're leaving
    const prevId = prevViewingRef.current;
    if (prevId && prevId === playingStreamId && now) {
      setLastSeen(prev => {
        const updated = { ...prev, [prevId]: { title: now.title, artist: now.artist, cover_url: now.cover_url, timestamp: Date.now() } };
        try { localStorage.setItem("lastSeenStreams", JSON.stringify(updated)); } catch {}
        return updated;
      });
    }

    setViewingStreamId(id);
    prevViewingRef.current = id;
    if (closeDrawer) setMobileStreamsOpen(false);

    // If nothing playing yet, auto-start playing the first stream clicked
    if (!playingStreamId) {
      doSwitchToStream(s);
    }
  }, [streams, playingStreamId, playerStatus, now, doSwitchToStream]);

  // ─── "Was playing" toast ───────────────────────────────────────────────────
  useEffect(() => {
    if (!viewingStreamId || !lastSeen[viewingStreamId]) return;
    // Only show if this stream was previously the playing stream
    const saved = lastSeen[viewingStreamId];
    if (!saved || !saved.title) return;
    // Check if the current track differs from what we last saw
    if (now && saved.title === now.title && saved.artist === now.artist) return;

    setWasPlayingToast(saved);
    const timer = setTimeout(() => setWasPlayingToast(null), 3000);
    return () => clearTimeout(timer);
  }, [viewingStreamId]);

  // ─── Keyboard shortcuts ────────────────────────────────────────────────────
  useEffect(() => {
    let lastKeyTime = 0;
    function onKeyDown(e) {
      const tag = document.activeElement?.tagName;
      if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT" || document.activeElement?.isContentEditable) return;
      const t = Date.now();
      if (t - lastKeyTime < 300 && (e.key === "ArrowLeft" || e.key === "ArrowRight")) return;
      lastKeyTime = t;

      switch (e.key) {
        case " ": e.preventDefault(); togglePlay(); break;
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
        case "Enter": if (isExploring) switchToViewing(); break;
        case "m": case "M": setMuted(m => !m); break;
        case "ArrowUp": e.preventDefault(); setVol(Math.min(100, vol + 5)); break;
        case "ArrowDown": e.preventDefault(); setVol(Math.max(0, vol - 5)); break;
        default:
          if (e.key >= "1" && e.key <= "9") {
            const idx = parseInt(e.key) - 1;
            if (activeStreams[idx]) setViewingStreamId(activeStreams[idx]);
          }
      }
    }
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [activeStreams, viewingStreamId, playingStreamId, isExploring, vol]);

  // ─── Swipe navigation ─────────────────────────────────────────────────────
  const mainRef = useRef(null);
  useSwipeNavigation(mainRef, {
    onPrev: useCallback(() => {
      const idx = activeStreams.indexOf(viewingStreamId);
      const prev = idx <= 0 ? activeStreams[activeStreams.length - 1] : activeStreams[idx - 1];
      if (prev) setViewingStreamId(prev);
    }, [activeStreams, viewingStreamId]),
    onNext: useCallback(() => {
      const idx = activeStreams.indexOf(viewingStreamId);
      const next = idx >= activeStreams.length - 1 ? activeStreams[0] : activeStreams[idx + 1];
      if (next) setViewingStreamId(next);
    }, [activeStreams, viewingStreamId]),
  });

  // ─── Stream preview hover ─────────────────────────────────────────────────
  const hoverTimerRef = useRef(null);
  const handleStreamHover = useCallback((id, e) => {
    clearTimeout(hoverTimerRef.current);
    const target = e.currentTarget;
    hoverTimerRef.current = setTimeout(() => {
      if (!target) return;
      const rect = target.getBoundingClientRect();
      setPreviewStreamId(id);
      setPreviewPos({ top: rect.top, left: rect.right + 8 });
    }, 500);
  }, []);
  const handleStreamHoverEnd = useCallback(() => {
    clearTimeout(hoverTimerRef.current);
    setPreviewStreamId(null);
  }, []);

  // ─── Media Session ─────────────────────────────────────────────────────────
  function updateMediaSession(data) {
    if (!("mediaSession" in navigator) || !data?.now) return;
    const n = data.now;
    try {
      navigator.mediaSession.metadata = new window.MediaMetadata({
        title: n.title || "", artist: n.artist || "", album: n.album || "",
        artwork: n.cover_url ? [{ src: n.cover_url, sizes: "512x512", type: "image/png" }] : [],
      });
      navigator.mediaSession.setActionHandler("play", play);
      navigator.mediaSession.setActionHandler("pause", pause);
    } catch {}
  }

  // ─── Render ────────────────────────────────────────────────────────────────
  const viewingMeta = viewingStream ? getMetaFor(viewingStream) : null;
  const playingMeta = playingStream ? getMetaFor(playingStream) : null;
  const effectiveStatus = viewingStreamId === playingStreamId ? playerStatus : "paused";

  return (
    <div className="min-h-screen text-white relative">
      {/* Gradient background layers (inline styles — Tailwind can't detect dynamic gradient classes) */}
      <div
        className="fixed inset-0 transition-all duration-[800ms] ease-in-out"
        style={{
          zIndex: -2,
          background: viewingMeta ? `linear-gradient(to bottom, ${viewingMeta.gradientFrom || '#0b0d10'}, ${viewingMeta.gradientVia || '#111520'}, ${viewingMeta.gradientTo || '#0b0d10'})` : 'linear-gradient(to bottom, #0b0d10, #111520)',
        }}
      />
      <div className="fixed inset-0 bg-[#0b0d10]" style={{ zIndex: -3 }} />

      {/* Hidden audio elements (owned by useCrossfade) */}
      <audio ref={audioElements[0]} preload="none" playsInline className="hidden"  />
      <audio ref={audioElements[1]} preload="none" playsInline className="hidden"  />

      {/* Now-playing strip (exploring a different stream) */}
      {isExploring && playingMeta && (
        <div
          className="fixed top-0 left-0 right-0 z-50 h-12 flex items-center gap-3 px-4 cursor-pointer border-b"
          style={{ backgroundColor: "rgba(0,0,0,0.85)", borderColor: playingMeta.accentColor + "40" }}
          onClick={() => setViewingStreamId(playingStreamId)}
          title="Return to playing stream"
        >
          <div className="w-1 h-8 rounded-full" style={{ backgroundColor: playingMeta.accentColor }} />
          {playingNow?.cover_url && <img src={playingNow.cover_url} alt="" className="w-8 h-8 rounded object-cover" />}
          <span className="text-sm font-medium truncate">
            {playingMeta.icon} {playingMeta.title}: {playingNow?.artist || ""} — {playingNow?.title || "Loading…"}
          </span>
          <span className="ml-auto text-xs text-slate-400 hidden sm:inline">tap to return</span>
        </div>
      )}

      {/* "Was playing" toast */}
      {wasPlayingToast && (
        <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 bg-black/80 backdrop-blur-sm rounded-lg px-4 py-2 text-sm text-white/70 animate-fade-in">
          Was playing: {wasPlayingToast.artist} — {wasPlayingToast.title}
        </div>
      )}

      {/* Mobile header */}
      <header className={`md:hidden sticky ${isExploring ? "top-12" : "top-0"} z-40 bg-black/80 backdrop-blur border-b border-white/10 px-4 py-3 grid grid-cols-[auto_1fr_auto] items-center`}>
        <button onClick={() => setMobileStreamsOpen(true)} className="px-3 py-2 rounded-lg border border-white/20 bg-white/5 text-sm" aria-label="Open streams">
          ☰ Explore
        </button>
        <div className="px-2" aria-hidden="true" />
        <button onClick={() => setInfoOpen(true)} className="px-3 py-2 rounded-lg border border-white/20 bg-white/5 text-sm" aria-label="Info">ℹ️</button>
      </header>

      <div className={`md:grid md:grid-cols-[320px_1fr] ${isExploring ? "md:pt-12" : ""}`}>
        {/* Sidebar (desktop) */}
        <aside className="hidden md:block border-r border-white/10 bg-black/60 backdrop-blur-sm p-4 overflow-auto min-h-screen relative">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-bold tracking-wide">Explore</h3>
            <button onClick={() => setInfoOpen(true)} className="px-2 py-1 rounded-lg border border-white/20 bg-white/5 hover:border-white/30 text-sm" title="About">ℹ️</button>
          </div>
          {loadingStreams && <div className="text-sm text-slate-400">Loading streams…</div>}
          {!loadingStreams && streams.length === 0 && <div className="text-sm text-slate-400">No streams found.</div>}
          <StreamsList
            streams={streams}
            viewingId={viewingStreamId}
            playingId={playingStreamId}
            isPlaying={playerStatus === "playing"}
            onSelect={browseStream}
            onHover={handleStreamHover}
            onHoverEnd={handleStreamHoverEnd}
          />
          {err && <div className="mt-3 text-xs text-amber-300">{err}</div>}

          {/* Stream preview popover */}
          {previewStreamId && previewPos && (
            <StreamPreview
              streamId={previewStreamId}
              metadata={allStreamsMeta[previewStreamId] || null}
              position={previewPos}
              onClose={() => setPreviewStreamId(null)}
            />
          )}
        </aside>

        {/* Main */}
        <main ref={mainRef} className="p-4 md:p-5 flex items-center justify-center min-h-[80vh]">
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
                  showPlaying={viewingStreamId === playingStreamId && playerStatus === "playing"}
                  isExploring={isExploring}
                  viewingMeta={viewingMeta}
                />

                {/* Controls */}
                <div className="mt-5 flex md:flex-row flex-col items-stretch md:items-center gap-3">
                  <div className="flex gap-3">
                    {isExploring ? (
                      <button
                        onClick={switchToViewing}
                        disabled={playerStatus === "loading"}
                        className="group relative inline-flex items-center gap-3 px-5 py-2.5 rounded-xl border transition shadow-sm hover:brightness-110 disabled:opacity-70"
                        style={{ borderColor: viewingMeta?.accentColor || "#64748b", backgroundColor: (viewingMeta?.accentColor || "#64748b") + "20" }}
                      >
                        {playerStatus === "loading" ? (
                          <span className="text-sm font-semibold animate-pulse">Switching stage…</span>
                        ) : (
                          <span className="text-sm font-semibold">Switch to this stage</span>
                        )}
                      </button>
                    ) : (
                      <PlayPauseButton status={effectiveStatus} onClick={togglePlay} />
                    )}
                    <button onClick={() => setMuted(m => !m)} className="px-4 py-2 rounded-xl border border-white/20 bg-white/5 hover:border-white/30">
                      {muted || vol === 0 ? "🔇" : "🔊"}
                    </button>
                  </div>
                  <div className="flex items-center gap-3 flex-1 min-w-[200px]">
                    <input type="range" min={0} max={100} value={muted ? 0 : vol} className="w-full accent-cyan-300"
                      onChange={(e) => { const v = Math.max(0, Math.min(100, Number(e.target.value))); setVol(v); if (v > 0 && muted) setMuted(false); }} />
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

                <ArtistDrawer open={drawerOpen} onClose={() => setDrawerOpen(false)} artist={artistCard} />
              </>
            )}
          </div>
        </main>
      </div>

      {/* Mobile Streams Drawer */}
      <StreamsDrawer
        open={mobileStreamsOpen} onClose={() => setMobileStreamsOpen(false)}
        streams={streams} viewingId={viewingStreamId} playingId={playingStreamId}
        isPlaying={playerStatus === "playing"} onSelect={(id) => browseStream(id, true)}
      />

      <InfoDialog open={infoOpen} onClose={() => setInfoOpen(false)} onShare={shareApp} />
      <BusMysteryCard open={busCardOpen} onClose={() => setBusCardOpen(false)} />
    </div>
  );
}
