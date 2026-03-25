import { useState, useRef, useEffect, useCallback } from "react";

// ---------------------------------------------------------------------------
// usePlayer — single audio element, direct src swap, volume fade-in
// ---------------------------------------------------------------------------

const FADE_MS = 800;
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

  const audioRef = useRef(null);
  const switchId = useRef(0);
  const volumeRef = useRef(volume);
  const mutedRef = useRef(muted);
  const statusRef = useRef(status);
  const fadeFrame = useRef(null);

  function el() { return audioRef.current; }
  function effectiveVol() { return mutedRef.current ? 0 : volumeRef.current / 100; }

  // Keep refs in sync
  useEffect(() => { volumeRef.current = volume; }, [volume]);
  useEffect(() => { mutedRef.current = muted; }, [muted]);

  // --- Status helper ---
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
    if (el()) el().volume = mutedRef.current ? 0 : clamped / 100;
  }, []);

  const setMuted = useCallback((m) => {
    const val = typeof m === "function" ? m(mutedRef.current) : !!m;
    setMutedState(val);
    mutedRef.current = val;
    if (el()) el().volume = val ? 0 : volumeRef.current / 100;
  }, []);

  // --- Health check: catch playing state from buffering ---
  useEffect(() => {
    const timer = setInterval(() => {
      const a = el();
      if (!a) return;
      if (!a.paused && a.currentTime > 0 && statusRef.current === "loading") {
        setPlayerStatus("playing");
      }
    }, HEALTH_INTERVAL);
    return () => clearInterval(timer);
  }, [setPlayerStatus]);

  // --- Audio events ---
  useEffect(() => {
    const a = el();
    if (!a) return;
    const h = {
      playing: () => setPlayerStatus("playing"),
      pause: () => { if (statusRef.current !== "loading") setPlayerStatus("paused"); },
      waiting: () => setPlayerStatus("loading", "Buffering\u2026"),
      error: () => setPlayerStatus("error", "Playback error"),
    };
    Object.entries(h).forEach(([e, fn]) => a.addEventListener(e, fn));
    return () => Object.entries(h).forEach(([e, fn]) => a.removeEventListener(e, fn));
  }, [setPlayerStatus]);

  // --- Volume fade-in ---
  function fadeIn(audio, targetVol) {
    cancelAnimationFrame(fadeFrame.current);
    audio.volume = 0;
    const start = performance.now();
    function tick(now) {
      const t = Math.min((now - start) / FADE_MS, 1);
      audio.volume = targetVol * t;
      if (t < 1) fadeFrame.current = requestAnimationFrame(tick);
    }
    fadeFrame.current = requestAnimationFrame(tick);
  }

  // --- Switch stream: just swap src and play ---
  const switchStream = useCallback((url, streamId) => {
    if (!url) return;
    const a = el();
    if (!a) return;

    const myId = ++switchId.current;
    cancelAnimationFrame(fadeFrame.current);
    setPlayerStatus("loading", "Connecting\u2026");
    setCurrentId(streamId);

    // Stop current playback
    a.pause();
    a.src = url;
    a.load();
    a.volume = 0;

    a.play()
      .then(() => {
        if (switchId.current !== myId) return;
        fadeIn(a, effectiveVol());
        setPlayerStatus("playing");
      })
      .catch(() => {
        if (switchId.current !== myId) return;
        setPlayerStatus("error", "Unable to start audio");
      });
  }, [setPlayerStatus]);

  // --- Play / pause ---
  const play = useCallback(() => {
    const a = el();
    if (!a) return;
    a.play()
      .then(() => { a.volume = effectiveVol(); setPlayerStatus("playing"); })
      .catch(() => setPlayerStatus("error", "Unable to resume"));
  }, [setPlayerStatus]);

  const pause = useCallback(() => {
    const a = el();
    if (!a) return;
    a.pause();
    setPlayerStatus("paused");
  }, [setPlayerStatus]);

  // --- Cleanup ---
  useEffect(() => {
    return () => { cancelAnimationFrame(fadeFrame.current); try { el()?.pause(); } catch {} };
  }, []);

  return {
    play,
    pause,
    switchStream,
    status,
    statusMsg,
    volume,
    setVolume,
    muted,
    setMuted,
    currentStreamId,
    audioElements: [audioRef],
  };
}
