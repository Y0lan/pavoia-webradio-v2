import { useState, useRef, useEffect, useCallback } from "react";

// ---------------------------------------------------------------------------
// usePlayer — two audio elements, zero-silence switching for live streams
// ---------------------------------------------------------------------------
// RULE: never stop the playing element until the new one is confirmed playing.
// Element A keeps playing while B buffers. Once B fires 'playing', fade A→B.
// ---------------------------------------------------------------------------

const FADE_MS = 1200;
const HEALTH_INTERVAL = 500;

export function useCrossfade() {
  const [status, setStatus] = useState("idle");
  const [statusMsg, setStatusMsg] = useState("");
  const [volume, setVolumeState] = useState(() => {
    const saved = Number(localStorage.getItem("playerVolPct") || 85);
    return Number.isFinite(saved) ? Math.max(0, Math.min(100, saved)) : 85;
  });
  const [muted, setMutedState] = useState(false);
  const [currentStreamId, setCurrentId] = useState(null);

  const audioRefA = useRef(null);
  const audioRefB = useRef(null);
  const activeRef = useRef("A"); // which element is currently the "live" one
  const switchId = useRef(0);
  const volumeRef = useRef(volume);
  const mutedRef = useRef(muted);
  const statusRef = useRef(status);
  const fadeFrameA = useRef(null);
  const fadeFrameB = useRef(null);

  function getEl(which) { return which === "A" ? audioRefA.current : audioRefB.current; }
  function activeEl() { return getEl(activeRef.current); }
  function standbyEl() { return getEl(activeRef.current === "A" ? "B" : "A"); }
  function effectiveVol() { return mutedRef.current ? 0 : volumeRef.current / 100; }

  useEffect(() => { volumeRef.current = volume; }, [volume]);
  useEffect(() => { mutedRef.current = muted; }, [muted]);

  const setPlayerStatus = useCallback((s, msg = "") => {
    statusRef.current = s;
    setStatus(s);
    setStatusMsg(msg);
  }, []);

  // --- Volume / mute ---
  const setVolume = useCallback((v) => {
    const clamped = Math.max(0, Math.min(100, Number(v) || 0));
    setVolumeState(clamped);
    volumeRef.current = clamped;
    localStorage.setItem("playerVolPct", String(clamped));
    const a = activeEl();
    if (a && !a.paused) a.volume = mutedRef.current ? 0 : clamped / 100;
  }, []);

  const setMuted = useCallback((m) => {
    const val = typeof m === "function" ? m(mutedRef.current) : !!m;
    setMutedState(val);
    mutedRef.current = val;
    const a = activeEl();
    if (a) a.volume = val ? 0 : volumeRef.current / 100;
  }, []);

  // --- Health check ---
  useEffect(() => {
    const timer = setInterval(() => {
      const a = activeEl();
      if (a && !a.paused && a.currentTime > 0 && statusRef.current === "loading") {
        setPlayerStatus("playing");
      }
    }, HEALTH_INTERVAL);
    return () => clearInterval(timer);
  }, [setPlayerStatus]);

  // --- Volume ramp ---
  function ramp(el, from, to, ms, frameRef, onDone) {
    cancelAnimationFrame(frameRef.current);
    el.volume = from;
    const start = performance.now();
    function tick(now) {
      const t = Math.min((now - start) / ms, 1);
      el.volume = from + (to - from) * t;
      if (t < 1) frameRef.current = requestAnimationFrame(tick);
      else { el.volume = to; if (onDone) onDone(); }
    }
    frameRef.current = requestAnimationFrame(tick);
  }

  // --- Switch stream: keep old playing until new is confirmed ---
  const switchStream = useCallback((url, streamId) => {
    if (!url) return;

    const myId = ++switchId.current;
    const oldEl = activeEl();
    const newEl = standbyEl();
    if (!newEl) return;

    // Don't touch the old element — it keeps playing
    setPlayerStatus("loading", "Connecting\u2026");

    // Prepare the standby element
    newEl.pause();
    newEl.src = url;
    newEl.load();
    newEl.volume = 0;

    // Listen for the new element to actually produce audio
    function onPlaying() {
      newEl.removeEventListener("playing", onPlaying);
      newEl.removeEventListener("error", onError);
      if (switchId.current !== myId) { newEl.pause(); return; }

      // New stream is confirmed playing — now crossfade
      const newSlot = activeRef.current === "A" ? "B" : "A";
      activeRef.current = newSlot;
      setCurrentId(streamId);

      const targetVol = effectiveVol();

      // Fade in new
      ramp(newEl, 0, targetVol, FADE_MS, fadeFrameB, null);

      // Fade out old (if it was playing)
      if (oldEl && !oldEl.paused) {
        const oldVol = oldEl.volume;
        ramp(oldEl, oldVol, 0, FADE_MS, fadeFrameA, () => {
          oldEl.pause();
          oldEl.removeAttribute("src");
          oldEl.load(); // reset for next use
        });
      }

      setPlayerStatus("playing");
    }

    function onError() {
      newEl.removeEventListener("playing", onPlaying);
      newEl.removeEventListener("error", onError);
      if (switchId.current !== myId) return;
      // Failed to load — old element is still playing, no harm done
      newEl.pause();
      newEl.removeAttribute("src");
      setPlayerStatus("error", "Could not connect to stream");
    }

    newEl.addEventListener("playing", onPlaying);
    newEl.addEventListener("error", onError);

    // Kick off playback — the 'playing' event fires when audio is flowing
    newEl.play().catch(() => {
      // play() rejected on standby element (autoplay policy) —
      // fall back: pause old element to free the audio context, then retry
      if (switchId.current !== myId) return;
      if (oldEl && !oldEl.paused) { oldEl.pause(); }
      newEl.play().catch(() => {
        newEl.removeEventListener("playing", onPlaying);
        newEl.removeEventListener("error", onError);
        if (switchId.current !== myId) return;
        newEl.pause();
        setPlayerStatus("error", "Unable to start audio — tap to retry");
      });
    });
  }, [setPlayerStatus]);

  // --- Play / pause ---
  const play = useCallback(() => {
    const a = activeEl();
    if (!a) return;
    a.play()
      .then(() => { a.volume = effectiveVol(); setPlayerStatus("playing"); })
      .catch(() => setPlayerStatus("error", "Unable to resume"));
  }, [setPlayerStatus]);

  const pause = useCallback(() => {
    const a = activeEl();
    if (!a) return;
    a.pause();
    setPlayerStatus("paused");
  }, [setPlayerStatus]);

  // --- Cleanup ---
  useEffect(() => {
    return () => {
      cancelAnimationFrame(fadeFrameA.current);
      cancelAnimationFrame(fadeFrameB.current);
      try { audioRefA.current?.pause(); } catch {}
      try { audioRefB.current?.pause(); } catch {}
    };
  }, []);

  return {
    play, pause, switchStream,
    status, statusMsg, volume, setVolume, muted, setMuted,
    currentStreamId,
    audioElements: [audioRefA, audioRefB],
  };
}
