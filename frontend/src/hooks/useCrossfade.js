import { useState, useRef, useEffect, useCallback } from "react";

// ---------------------------------------------------------------------------
// useCrossfade — volume-based crossfade (no Web Audio API, no CORS issues)
// ---------------------------------------------------------------------------
// Two <audio> elements alternate. Crossfade is done by ramping element.volume
// via requestAnimationFrame. Works with cross-origin streams.
// ---------------------------------------------------------------------------

const CROSSFADE_MS = 1500;
const HEALTH_INTERVAL = 500;
const LOADING_DEBOUNCE = 150;

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
  const activeSlot = useRef("A");
  const crossfadeId = useRef(0);
  const volumeRef = useRef(volume);
  const mutedRef = useRef(muted);
  const statusRef = useRef(status);
  const loadingTimer = useRef(null);
  const healthTimer = useRef(null);
  const fadeFrame = useRef(null);
  const pauseTimer = useRef(null);

  // --- Status with loading debounce ---
  const setPlayerStatus = useCallback((s, msg = "") => {
    clearTimeout(loadingTimer.current);
    if (s === "loading") {
      loadingTimer.current = setTimeout(() => {
        statusRef.current = "loading";
        setStatus("loading");
        setStatusMsg(msg);
      }, LOADING_DEBOUNCE);
    } else {
      statusRef.current = s;
      setStatus(s);
      setStatusMsg(msg);
    }
  }, []);

  // Keep refs in sync
  useEffect(() => { volumeRef.current = volume; }, [volume]);
  useEffect(() => { mutedRef.current = muted; }, [muted]);

  // --- Helpers ---
  function elFor(slot) { return slot === "A" ? audioRefA.current : audioRefB.current; }
  function otherSlot(slot) { return slot === "A" ? "B" : "A"; }

  function effectiveVol() {
    return mutedRef.current ? 0 : volumeRef.current / 100;
  }

  // --- Volume / mute ---
  const applyVolume = useCallback(() => {
    const v = effectiveVol();
    const el = elFor(activeSlot.current);
    if (el && !el.paused) el.volume = v;
  }, []);

  const setVolume = useCallback((v) => {
    const clamped = Math.max(0, Math.min(100, Number(v) || 0));
    setVolumeState(clamped);
    volumeRef.current = clamped;
    localStorage.setItem("playerVolPct", String(clamped));
    const vol = mutedRef.current ? 0 : clamped / 100;
    const el = elFor(activeSlot.current);
    if (el) el.volume = vol;
  }, []);

  const setMuted = useCallback((m) => {
    const val = typeof m === "function" ? m(mutedRef.current) : !!m;
    setMutedState(val);
    mutedRef.current = val;
    const el = elFor(activeSlot.current);
    if (el) el.volume = val ? 0 : volumeRef.current / 100;
  }, []);

  // --- Health check ---
  useEffect(() => {
    healthTimer.current = setInterval(() => {
      const el = elFor(activeSlot.current);
      if (!el) return;
      if (!el.paused && (el.currentTime > 0 || el.readyState >= 3)) {
        if (statusRef.current === "loading") setPlayerStatus("playing");
      }
    }, HEALTH_INTERVAL);
    return () => clearInterval(healthTimer.current);
  }, [setPlayerStatus]);

  // --- Audio event wiring ---
  useEffect(() => {
    const a = audioRefA.current;
    const b = audioRefB.current;
    if (!a || !b) return;

    function isActive(el) {
      return (activeSlot.current === "A" && el === a) || (activeSlot.current === "B" && el === b);
    }

    function makeHandlers(el) {
      return {
        onPlaying: () => { if (isActive(el)) setPlayerStatus("playing"); },
        onPause: () => { if (isActive(el)) setPlayerStatus("paused"); },
        onWaiting: () => { if (isActive(el)) setPlayerStatus("loading", "Buffering\u2026"); },
        onStalled: () => { if (isActive(el)) setPlayerStatus("loading", "Buffering\u2026"); },
        onTimeUpdate: () => { if (isActive(el) && !el.paused && el.currentTime > 0) setPlayerStatus("playing"); },
        onError: () => { if (isActive(el)) setPlayerStatus("error", "Playback error"); },
      };
    }

    const hA = makeHandlers(a);
    const hB = makeHandlers(b);
    const pairs = [
      [a, [["playing", hA.onPlaying], ["pause", hA.onPause], ["waiting", hA.onWaiting], ["stalled", hA.onStalled], ["timeupdate", hA.onTimeUpdate], ["error", hA.onError]]],
      [b, [["playing", hB.onPlaying], ["pause", hB.onPause], ["waiting", hB.onWaiting], ["stalled", hB.onStalled], ["timeupdate", hB.onTimeUpdate], ["error", hB.onError]]],
    ];
    pairs.forEach(([el, evts]) => evts.forEach(([e, h]) => el.addEventListener(e, h)));
    return () => pairs.forEach(([el, evts]) => evts.forEach(([e, h]) => el.removeEventListener(e, h)));
  }, [setPlayerStatus]);

  // --- Visibility change: resume if paused by OS ---
  useEffect(() => {
    function onVis() {
      if (document.visibilityState !== "visible") return;
      const el = elFor(activeSlot.current);
      if (el && statusRef.current === "playing" && el.paused) {
        el.play().catch(() => {});
      }
    }
    document.addEventListener("visibilitychange", onVis);
    return () => document.removeEventListener("visibilitychange", onVis);
  }, []);

  // --- Volume ramp via requestAnimationFrame ---
  function rampVolume(el, from, to, durationMs, onDone) {
    cancelAnimationFrame(fadeFrame.current);
    const start = performance.now();
    el.volume = from;

    function tick(now) {
      const elapsed = now - start;
      const t = Math.min(elapsed / durationMs, 1);
      // Ease-in-out curve for smooth perceived transition
      const eased = t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2;
      el.volume = from + (to - from) * eased;
      if (t < 1) {
        fadeFrame.current = requestAnimationFrame(tick);
      } else {
        el.volume = to;
        if (onDone) onDone();
      }
    }
    fadeFrame.current = requestAnimationFrame(tick);
  }

  // --- Core: crossfade to a new stream ---
  const switchStream = useCallback((url, streamId) => {
    if (!url) return;

    const myId = ++crossfadeId.current;
    clearTimeout(pauseTimer.current);
    cancelAnimationFrame(fadeFrame.current);

    const oldSlot = activeSlot.current;
    const newSlot = otherSlot(oldSlot);
    const oldEl = elFor(oldSlot);
    const newEl = elFor(newSlot);
    if (!newEl) return;

    const isFirstPlay = statusRef.current === "idle" || statusRef.current === "paused" || !oldEl || oldEl.paused;

    setPlayerStatus("loading", "Connecting\u2026");

    // Silence and pause any leftover on the new slot
    if (newEl && !newEl.paused) {
      newEl.volume = 0;
      newEl.pause();
    }

    newEl.src = url;
    newEl.load();
    newEl.volume = 0;

    // Try to play new stream. If browser allows concurrent audio, we get
    // a smooth crossfade. If it rejects, we pause old first and retry.
    function startNew() {
      return newEl.play().then(() => {
        if (crossfadeId.current !== myId) return;
        activeSlot.current = newSlot;
        setCurrentId(streamId);

        const targetVol = effectiveVol();
        rampVolume(newEl, 0, targetVol, CROSSFADE_MS, null);

        // Crossfade: fade out old element
        if (oldEl && !oldEl.paused) {
          const oldVol = oldEl.volume;
          const fadeStart = performance.now();
          let oldFrame = null;
          function tickOld(now) {
            const t = Math.min((now - fadeStart) / CROSSFADE_MS, 1);
            const eased = t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2;
            oldEl.volume = oldVol * (1 - eased);
            if (t < 1) { oldFrame = requestAnimationFrame(tickOld); }
            else { oldEl.volume = 0; oldEl.pause(); }
          }
          oldFrame = requestAnimationFrame(tickOld);
          pauseTimer.current = setTimeout(() => {
            cancelAnimationFrame(oldFrame);
            if (oldEl && !oldEl.paused) { oldEl.volume = 0; oldEl.pause(); }
          }, CROSSFADE_MS + 100);
        }

        setPlayerStatus("playing");
      });
    }

    startNew().catch(() => {
      if (crossfadeId.current !== myId) return;
      // Browser rejected concurrent audio — pause old and retry
      if (oldEl && !oldEl.paused) { oldEl.volume = 0; oldEl.pause(); }
      newEl.load();
      newEl.volume = 0;
      return newEl.play().then(() => {
        if (crossfadeId.current !== myId) return;
        activeSlot.current = newSlot;
        setCurrentId(streamId);
        rampVolume(newEl, 0, effectiveVol(), CROSSFADE_MS, null);
        setPlayerStatus("playing");
      });
    }).catch(() => {
      if (crossfadeId.current !== myId) return;
      newEl.pause();
      newEl.removeAttribute("src");
      newEl.load();
      setPlayerStatus("error", "Unable to start audio");
    });
  }, [setPlayerStatus]);

  // --- Play / pause ---
  const play = useCallback(() => {
    const el = elFor(activeSlot.current);
    if (!el) return;
    el.play()
      .then(() => {
        el.volume = effectiveVol();
        setPlayerStatus("playing");
      })
      .catch(() => setPlayerStatus("error", "Unable to resume audio"));
  }, [setPlayerStatus]);

  const pause = useCallback(() => {
    const el = elFor(activeSlot.current);
    if (!el) return;
    el.pause();
    setPlayerStatus("paused");
  }, [setPlayerStatus]);

  // --- Cleanup ---
  useEffect(() => {
    return () => {
      clearTimeout(loadingTimer.current);
      clearTimeout(pauseTimer.current);
      cancelAnimationFrame(fadeFrame.current);
      clearInterval(healthTimer.current);
      try { audioRefA.current?.pause(); } catch {}
      try { audioRefB.current?.pause(); } catch {}
    };
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
    audioElements: [audioRefA, audioRefB],
  };
}
