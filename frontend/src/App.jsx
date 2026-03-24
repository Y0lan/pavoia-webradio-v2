import React, { useEffect, useMemo, useRef, useState } from "react";
import { StreamsList } from "./components/sidebar/StreamsList";
import { StreamsDrawer } from "./components/mobile/StreamsDrawer";
import { PlayPauseButton } from "./components/player/PlayPauseButton";
import { NowPlaying } from "./components/player/NowPlaying";
import { ArtistDrawer } from "./components/drawers/ArtistDrawer";
import { InfoDialog } from "./components/dialogs/InfoDialog";
import { getMetaFor } from "./utils/streamMeta";
import { fmtTime } from "./utils/formatters";


export default function App() {
  // ─── Core state ──────────────────────────────────────────────────────────────
  const [streams, setStreams] = useState([]);
  const [activeId, setActiveId] = useState(null);
  const [now, setNow] = useState(null);
  const [artistCard, setArtistCard] = useState(null);
  const [loadingStreams, setLoadingStreams] = useState(true);
  const [err, setErr] = useState("");
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [mobileStreamsOpen, setMobileStreamsOpen] = useState(false);
  const [infoOpen, setInfoOpen] = useState(false);

  // Which stream is truly playing (can differ from activeId while browsing)
  const [playingStreamId, setPlayingStreamId] = useState(null);
  // The stream id that matches the current audio.src
  const audioStreamIdRef = useRef(null);

  // ─── Audio + player state ────────────────────────────────────────────────────
  const audioRef = useRef(null);
  const [player, setPlayer] = useState({ status: "idle", msg: "" }); // idle | loading | playing | paused | error

  // Volume (local only)
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
        if (def) setActiveId(def.id);
      } catch {
        setErr("Unable to load streams");
      } finally {
        setLoadingStreams(false);
      }
    })();
    return () => {
      alive = false;
    };
  }, []);

  const activeStream = useMemo(
    () => streams.find((s) => s.id === activeId) || null,
    [streams, activeId]
  );

  // Preconnect to stream origins for faster start
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

  // ─── Poll now-playing with backoff ──────────────────────────────────────────
  const pollRef = useRef({ timer: null, delay: 5000 });
  useEffect(() => {
    if (!activeStream) return;
    let alive = true;

    async function fetchNow() {
      try {
        const r = await fetch(`/api/streams/${activeStream.id}/now`, {
          cache: "no-store",
        });
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        const data = await r.json();
        if (!alive) return;
        setNow(data.now);
        setArtistCard(data.artist || null);
        setErr("");
        pollRef.current.delay = 5000;
        updateMediaSession(data);
      } catch {
        if (!alive) return;
        setErr("Connection lost. Retrying…");
        pollRef.current.delay = Math.min(
          20000,
          Math.round((pollRef.current.delay || 5000) * 1.5)
        );
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
    return () => {
      alive = false;
      clearTimeout(pollRef.current.timer);
    };
  }, [activeStream]);

  // ─── Audio events → resilient player state ───────────────────────────────────
  useEffect(() => {
    const a = audioRef.current;
    if (!a) return;

    let waitingTimer = null;
    const clearWaiting = () => {
      if (waitingTimer) {
        clearTimeout(waitingTimer);
        waitingTimer = null;
      }
    };
    const showLoading = (msg) => {
      clearWaiting();
      waitingTimer = setTimeout(() => {
        if (!a.paused) setPlayer({ status: "loading", msg });
      }, 150);
    };
    const setPlaying = () => {
      clearWaiting();
      setPlayer({ status: "playing", msg: "" });
      setPlayingStreamId(audioStreamIdRef.current || playingStreamId || null);
    };
    const setPaused = (m = "") => {
      clearWaiting();
      setPlayer({ status: "paused", msg: m });
      setPlayingStreamId(null);
    };

    const onLoadStart = () => showLoading("Connecting…");
    const onPlay = () => {
      if (!a.paused && (a.currentTime > 0 || a.readyState >= 2)) setPlaying();
      else showLoading("Connecting…");
    };
    const onWaiting = () => showLoading("Buffering…");
    const onTimeUpdate = () => {
      if (!a.paused && a.currentTime > 0) setPlaying();
    };
    const onCanPlay = () => {
      if (!a.paused) setPlaying();
      else setPaused();
    };
    const onLoadedData = () => {
      if (!a.paused) setPlaying();
    };
    const onLoadedMetadata = () => {
      if (!a.paused) setPlaying();
    };
    const onProgress = () => {
      if (!a.paused && (a.currentTime > 0 || a.readyState >= 2)) setPlaying();
    };
    const onPlaying = () => setPlaying();
    const onPause = () => setPaused();
    const onStalled = () => showLoading("Buffering…");
    const onEnded = () => setPaused("Ended");
    const onError = () => {
      setPlayer({ status: "error", msg: "Playback error" });
      setPlayingStreamId(null);
    };

    a.addEventListener("loadstart", onLoadStart);
    a.addEventListener("play", onPlay);
    a.addEventListener("waiting", onWaiting);
    a.addEventListener("timeupdate", onTimeUpdate);
    a.addEventListener("canplay", onCanPlay);
    a.addEventListener("loadeddata", onLoadedData);
    a.addEventListener("loadedmetadata", onLoadedMetadata);
    a.addEventListener("progress", onProgress);
    a.addEventListener("playing", onPlaying);
    a.addEventListener("pause", onPause);
    a.addEventListener("stalled", onStalled);
    a.addEventListener("ended", onEnded);
    a.addEventListener("error", onError);

    // Health check for live streams that miss events
    const health = setInterval(() => {
      if (!a.paused && (a.currentTime > 0 || a.readyState >= 3)) setPlaying();
    }, 500);

    return () => {
      a.removeEventListener("loadstart", onLoadStart);
      a.removeEventListener("play", onPlay);
      a.removeEventListener("waiting", onWaiting);
      a.removeEventListener("timeupdate", onTimeUpdate);
      a.removeEventListener("canplay", onCanPlay);
      a.removeEventListener("loadeddata", onLoadedData);
      a.removeEventListener("loadedmetadata", onLoadedMetadata);
      a.removeEventListener("progress", onProgress);
      a.removeEventListener("playing", onPlaying);
      a.removeEventListener("pause", onPause);
      a.removeEventListener("stalled", onStalled);
      a.removeEventListener("ended", onEnded);
      a.removeEventListener("error", onError);
      clearWaiting();
      clearInterval(health);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [playingStreamId]);

  // Reset any stale loading when switching stream
  useEffect(() => {
    if (!activeStream) return;
    localStorage.setItem("activeStreamId", activeStream.id);
    setPlayer((p) => (p.status === "loading" ? { status: "paused", msg: "" } : p));
  }, [activeStream]);

  // ─── Building the playable URL (fallback if API returns /s/...) ─────────────
  function getPlayableUrl(s) {
    if (!s) return "";
    let url = s.streamUrl || "";
    try {
      // If API gives a relative path like "/s/ambiance.mp3", resolve against current origin
      if (url.startsWith("/")) {
        return new URL(url, window.location.origin).toString();
      }
      // If the app runs on HTTPS and the stream is HTTP, try an HTTPS upgrade
      if (window.location.protocol === "https:" && url.startsWith("http://")) {
        url = "https://" + url.slice("http://".length);
      }
      return url;
    } catch {
      return url;
    }
  }

  // ─── Local elapsed/duration tracking ────────────────────────────────────────
  const [elapsedSec, setElapsedSec] = useState(0);
  const [durationSec, setDurationSec] = useState(null);
  const baseStartRef = useRef(performance.now());

  // When track changes, rebase timer using server's elapsed/duration once
  useEffect(() => {
    if (!now) return;
    const serverElapsed = Number(now?.mpd?.elapsed || 0);
    const serverDuration = Number(now?.duration_sec) || null;
    setDurationSec(serverDuration || null);
    setElapsedSec(serverElapsed);
    baseStartRef.current = performance.now() - serverElapsed * 1000;
  }, [now?.file, activeStream?.id]); // rebase only on track/stream change

  // While playing, tick elapsed locally
  useEffect(() => {
    const id = setInterval(() => {
      if (!(player.status === "playing")) return;
      const ms = performance.now() - baseStartRef.current;
      const sec = ms / 1000;
      setElapsedSec(durationSec ? Math.min(sec, durationSec) : sec);
    }, 250);
    return () => clearInterval(id);
  }, [player.status, durationSec]);

  // Also update duration from actual media if present
  useEffect(() => {
    const a = audioRef.current;
    if (!a) return;
    const onLoadedMetadata = () => {
      if (isFinite(a.duration) && a.duration > 0) {
        setDurationSec(Math.floor(a.duration));
      }
    };
    a.addEventListener("loadedmetadata", onLoadedMetadata);
    return () => a.removeEventListener("loadedmetadata", onLoadedMetadata);
  }, []);

  // Do not rebase on resume; baseline is set on real track/stream change for stability.
// (This avoids resetting to 0 when pressing Play.)

  // ─── Share helper ───────────────────────────────────────────────────────────
  async function shareApp() {
    const url = window.location.href;
    try {
      if (navigator.share) {
        await navigator.share({ title: "PAVOIA WEBRADIO", text: "Tune in to PAVOIA WEBRADIO", url });
      } else if (navigator.clipboard) {
        await navigator.clipboard.writeText(url);
        alert("Link copied to clipboard.");
      }
    } catch {}
  }

  // ─── Play/Pause ─────────────────────────────────────────────────────────────
  async function togglePlay(targetStream) {
    const a = audioRef.current;
    const s = targetStream || activeStream;
    if (!a || !s) return;

    // Easter egg: BUS cannot be streamed
    const activeMeta = getMetaFor(s);
    if (activeMeta.disabled) {
      alert("🚌 This one can’t be streamed — it must be experienced IRL!\nFind the real bus and hop on.");
      return;
    }

    let target = getPlayableUrl(s);
    // Fallbacks + validation
    if (!target || !/^https?:\/\//i.test(target)) {
      target = (s.streamUrl || "").trim();
    }
    if (!target) {
      console.error("No stream URL found for active stream", s);
      alert("No stream URL found for this station.");
      return;
    }

    // When we switch source, remember which stream this audio belongs to
    if (!a.src || a.src !== target) {
      a.src = target;
      a.load();
      audioStreamIdRef.current = s.id;
    }
    try {
      if (a.paused) {
        setPlayer({ status: "loading", msg: "Connecting…" });
        await a.play();
        // If play() resolves we assume audio will flow
        setTimeout(() => {
          if (!a.paused) {
            setPlayer({ status: "playing", msg: "" });
            setPlayingStreamId(audioStreamIdRef.current || s.id);
          }
        }, 50);
        setTimeout(() => {
          if (!a.paused && (a.currentTime >= 0 || a.readyState >= 2)) {
            setPlayer({ status: "playing", msg: "" });
            setPlayingStreamId(audioStreamIdRef.current || s.id);
          }
        }, 1500);
      } else {
        a.pause();
        setPlayingStreamId(null);
      }
    } catch (e) {
      setPlayer({ status: "error", msg: "Unable to start audio" });
      setPlayingStreamId(null);
      console.error("audio.play() failed", { error: e?.name || e, message: e?.message, src: target });
      alert("Unable to start audio. Check HTTPS/reachability of the stream URL.");
    }
  }

  // Quick-select: choose stream and autoplay
  const playStreamById = (id, closeDrawer = false) => {
    const s = streams.find((x) => x.id === id);
    setActiveId(id);
    if (closeDrawer) setMobileStreamsOpen(false);
    if (s) {
      togglePlay(s);
    } else {
      togglePlay();
    }
  };

  // Effective status for the visible stream only
  const effectiveStatus =
    activeStream?.id === playingStreamId ? player.status : "paused";

  // ─── Media Session ──────────────────────────────────────────────────────────
  function updateMediaSession(data) {
    if (!("mediaSession" in navigator) || !data?.now) return;
    const n = data.now;
    try {
      navigator.mediaSession.metadata = new window.MediaMetadata({
        title: n.title || "",
        artist: n.artist || "",
        album: n.album || "",
        artwork: n.cover_url
          ? [{ src: n.cover_url, sizes: "512x512", type: "image/png" }]
          : [],
      });
      navigator.mediaSession.setActionHandler("play", () => audioRef.current?.play());
      navigator.mediaSession.setActionHandler("pause", () => audioRef.current?.pause());
    } catch {}
  }

  // ─── Render ─────────────────────────────────────────────────────────────────
  const activeMeta = activeStream ? getMetaFor(activeStream) : null;
  // Mobile header: show selected stream (or playing stream if different)
  const mobileHeaderMeta = useMemo(() => {
    const playing = streams.find((x) => x.id === playingStreamId);
    const selected = streams.find((x) => x.id === activeId);
    const s = playing || selected || null;
    return s ? getMetaFor(s) : null;
  }, [streams, playingStreamId, activeId]);

  return (
    <div className="min-h-screen bg-[#0b0d10] text-[#e6edf3]">
      {/* Mobile header */}
      <header className="md:hidden sticky top-0 z-40 bg-[#13171c]/95 backdrop-blur border-b border-[#1f2430] px-4 py-3 grid grid-cols-[auto_1fr_auto] items-center">
        <button
          onClick={() => setMobileStreamsOpen(true)}
          className="px-3 py-2 rounded-lg border border-[#2b3445] bg-[#171c24] text-sm"
          aria-label="Open streams"
        >
          ☰ Explore
        </button>
        <div className="px-2" aria-hidden="true"></div>
        <button
          onClick={() => setInfoOpen(true)}
          className="px-3 py-2 rounded-lg border border-[#2b3445] bg-[#171c24] text-sm"
          aria-label="Info about stream quality"
          title="About stream quality"
        >
          ℹ️
        </button>
      </header>

      <div className="md:grid md:grid-cols-[320px_1fr]">
        {/* Sidebar (desktop) */}
        <aside className="hidden md:block border-r border-[#1f2430] bg-[#13171c] p-4 overflow-auto min-h-screen">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-bold tracking-wide">Explore</h3>
            <button
              onClick={() => setInfoOpen(true)}
              className="px-2 py-1 rounded-lg border border-[#2b3445] bg-[#171c24] hover:border-[#3b4760] text-sm"
              title="About stream quality"
            >
              ℹ️
            </button>
          </div>
          {loadingStreams && (
            <div className="text-sm text-slate-400">Loading streams…</div>
          )}
          {!loadingStreams && streams.length === 0 && (
            <div className="text-sm text-slate-400">No streams found.</div>
          )}
          <StreamsList
            streams={streams}
            activeId={activeId}
            playingId={playingStreamId}
            isPlaying={player.status === "playing"}
            onSelect={setActiveId}
          />
          {err && <div className="mt-3 text-xs text-amber-300">{err}</div>}
        </aside>

        {/* Main */}
        <main className="p-4 md:p-5 flex items-center justify-center">
          <div className="w-full max-w-6xl bg-gradient-to-b from-[#111520] to-[#0e1118] border border-[#22293a] rounded-2xl shadow-2xl p-4 md:p-8 relative">
            {!activeStream ? (
              <div className="text-slate-400">
                Select a station from <span className="md:inline hidden">the Explore panel</span>
                <span className="md:hidden inline"> Explore</span>.
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
                  showPlaying={activeStream?.id === playingStreamId && player.status === "playing"}
                />

                {/* Controls */}
                <div className="mt-5 flex md:flex-row flex-col items-stretch md:items-center gap-3">
                  <div className="flex gap-3">
                    <PlayPauseButton status={effectiveStatus} onClick={() => togglePlay()} />
                    <button
                      onClick={() => setMuted((m) => !m)}
                      className="px-4 py-2 rounded-xl border border-[#2b3445] bg-[#171c24] hover:border-[#3b4760]"
                    >
                      {muted || vol === 0 ? "🔇 Mute" : "🔊 Mute"}
                    </button>
                  </div>

                  {/* Simple volume slider */}
                  <div className="flex items-center gap-3 flex-1 min-w-[240px]">
                    <span className="text-xs text-slate-400">Volume</span>
                    <input
                      id="vol"
                      type="range"
                      min={0}
                      max={100}
                      value={muted ? 0 : vol}
                      className="w-full"
                      onChange={(e) => {
                        const v = Math.max(0, Math.min(100, Number(e.target.value)));
                        setVol(v);
                        if (v > 0 && muted) setMuted(false);
                      }}
                    />
                    <span className="text-xs text-slate-300 w-10 text-right">
                      {muted ? 0 : vol}%
                    </span>
                  </div>
                </div>

                {/* Stream info below slider (moved from badge) */}
                {activeMeta && (
                  <div className="mt-2 text-sm text-slate-300 flex flex-wrap items-center gap-2">
                    <span className="font-semibold">{activeMeta.icon} {activeMeta.title}</span>
                    <span className="text-slate-500">—</span>
                    <span className="text-slate-400">{activeMeta.desc}</span>
                  </div>
                )}

                {/* Hidden audio element */}
                <audio ref={audioRef} preload="none" playsInline className="hidden" crossOrigin="anonymous" />

                {/* Artist Drawer */}
                <ArtistDrawer
                  open={drawerOpen}
                  onClose={() => setDrawerOpen(false)}
                  artist={artistCard}
                />
              </>
            )}
          </div>
        </main>
      </div>

      {/* Streams Drawer (mobile) */}
      <StreamsDrawer
        open={mobileStreamsOpen}
        onClose={() => setMobileStreamsOpen(false)}
        streams={streams}
        activeId={activeId}
        playingId={playingStreamId}
        isPlaying={player.status === "playing"}
        onSelect={(id) => { setActiveId(id); setMobileStreamsOpen(false); }}
      />

      {/* Info Modal */}
      <InfoDialog open={infoOpen} onClose={() => setInfoOpen(false)} onShare={shareApp} />
    </div>
  );
}

// ─── Stream metadata mapping ──────────────────────────────────────────────────
