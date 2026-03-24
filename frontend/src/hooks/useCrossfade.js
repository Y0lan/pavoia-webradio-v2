import { useState, useRef, useEffect, useCallback } from "react";

// ---------------------------------------------------------------------------
// useCrossfade
// ---------------------------------------------------------------------------
// Central audio engine for the webradio app.  Manages two HTML5 <audio>
// elements (A and B) and crossfades between them via the Web Audio API.
//
// Architecture:
//   AudioContext
//     audioA  -->  MediaElementSourceNode A  -->  GainNode A  --\
//                                                                +--> destination
//     audioB  -->  MediaElementSourceNode B  -->  GainNode B  --/
//
// On each stream switch the *inactive* element loads the new URL while the
// *active* element keeps playing.  Once the new element fires `canplay` we
// ramp gains over CROSSFADE_DURATION using exponentialRampToValueAtTime,
// then pause the outgoing element to release its network connection.
// ---------------------------------------------------------------------------

const CROSSFADE_DURATION = 1.5;        // seconds
const GAIN_FLOOR        = 0.001;       // exponentialRamp cannot target 0
const CANPLAY_TIMEOUT   = 8_000;       // ms before we give up on a new stream
const HEALTH_INTERVAL   = 500;         // ms – live-stream health check
const LOADING_DEBOUNCE  = 150;         // ms – suppress quick loading flicker

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** True when the Web Audio API is available. */
function hasWebAudio() {
  return typeof AudioContext !== "undefined" || typeof webkitAudioContext !== "undefined";
}

/** Create an AudioContext (with webkit fallback). */
function createAudioContext() {
  const Ctor = window.AudioContext || window.webkitAudioContext;
  return new Ctor();
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export function useCrossfade() {
  // --- public state --------------------------------------------------------
  const [status, setStatus]             = useState("idle");       // idle | loading | playing | paused | error
  const [statusMsg, setStatusMsg]       = useState("");
  const [volume, setVolumeState]        = useState(() => {
    const saved = Number(localStorage.getItem("playerVolPct") || 85);
    return Number.isFinite(saved) ? Math.max(0, Math.min(100, saved)) : 85;
  });
  const [muted, setMutedState]          = useState(false);
  const [currentStreamId, setCurrentId] = useState(null);

  // --- refs for the two <audio> elements (caller renders them) -------------
  const audioRefA = useRef(null);
  const audioRefB = useRef(null);

  // --- internal refs (mutable across renders, never trigger re-render) -----
  const ctxRef        = useRef(null);   // AudioContext
  const sourceRefA    = useRef(null);   // MediaElementAudioSourceNode for A
  const sourceRefB    = useRef(null);   // MediaElementAudioSourceNode for B
  const gainRefA      = useRef(null);   // GainNode for A
  const gainRefB      = useRef(null);   // GainNode for B
  const activeSlot    = useRef("A");    // which slot is currently the "active" one
  const crossfadeId   = useRef(0);      // monotonic id to detect stale crossfades
  const volumeRef     = useRef(volume); // shadow of current volume (avoids stale closures)
  const mutedRef      = useRef(muted);
  const statusRef     = useRef(status); // shadow for use inside non-reactive callbacks
  const currentIdRef  = useRef(null);   // shadow of currentStreamId
  const loadingTimer  = useRef(null);   // debounce timer for "loading" status
  const healthTimer   = useRef(null);   // health-check interval
  const webAudioOk    = useRef(hasWebAudio());
  const ctxInitDone   = useRef(false);  // have we already wired up source nodes?
  const pauseTimer    = useRef(null);   // delayed pause of outgoing element

  // -----------------------------------------------------------------------
  // Helpers to update status with the 150ms debounce on "loading"
  // -----------------------------------------------------------------------
  const setPlayerStatus = useCallback((s, msg = "") => {
    clearTimeout(loadingTimer.current);
    if (s === "loading") {
      // Debounce: only show "loading" if we stay in it for 150ms
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

  // -----------------------------------------------------------------------
  // Keep shadow refs in sync
  // -----------------------------------------------------------------------
  useEffect(() => { volumeRef.current = volume; }, [volume]);
  useEffect(() => { mutedRef.current = muted; }, [muted]);

  // -----------------------------------------------------------------------
  // Volume / mute application
  // -----------------------------------------------------------------------
  const applyVolume = useCallback(() => {
    const v = mutedRef.current ? 0 : volumeRef.current / 100;

    if (webAudioOk.current && ctxRef.current) {
      // When using Web Audio, the gain nodes already control volume.
      // We scale the *target* gain of whichever slot is active.
      const activeGain = activeSlot.current === "A" ? gainRefA.current : gainRefB.current;
      if (activeGain) {
        try {
          activeGain.gain.cancelScheduledValues(ctxRef.current.currentTime);
          activeGain.gain.setValueAtTime(Math.max(v, GAIN_FLOOR), ctxRef.current.currentTime);
        } catch { /* context may be closed */ }
      }
      // Keep element volumes at 1 so the gain node has full-range signal.
      if (audioRefA.current) audioRefA.current.volume = 1;
      if (audioRefB.current) audioRefB.current.volume = 1;
    } else {
      // Fallback: control volume on the element directly.
      const el = activeSlot.current === "A" ? audioRefA.current : audioRefB.current;
      if (el) {
        el.volume = v;
        el.muted = mutedRef.current;
      }
    }
  }, []);

  const setVolume = useCallback((v) => {
    const clamped = Math.max(0, Math.min(100, Number(v) || 0));
    setVolumeState(clamped);
    volumeRef.current = clamped;
    localStorage.setItem("playerVolPct", String(clamped));
    applyVolume();
  }, [applyVolume]);

  const setMuted = useCallback((m) => {
    const val = typeof m === "function" ? m(mutedRef.current) : !!m;
    setMutedState(val);
    mutedRef.current = val;
    applyVolume();
  }, [applyVolume]);

  // -----------------------------------------------------------------------
  // AudioContext initialisation (deferred until first user interaction)
  // -----------------------------------------------------------------------
  const ensureAudioContext = useCallback(() => {
    if (!webAudioOk.current) return false;

    // Create context if needed
    if (!ctxRef.current || ctxRef.current.state === "closed") {
      try {
        ctxRef.current = createAudioContext();
      } catch {
        webAudioOk.current = false;
        return false;
      }
    }

    // Resume if suspended (Chrome autoplay policy)
    if (ctxRef.current.state === "suspended") {
      ctxRef.current.resume().catch(() => {});
    }

    // Wire up source + gain nodes once per context lifetime.
    // MediaElementAudioSourceNode can only be created once per element per
    // context, so we guard with ctxInitDone.
    if (!ctxInitDone.current) {
      const ctx = ctxRef.current;
      const a = audioRefA.current;
      const b = audioRefB.current;
      if (!a || !b) return false;

      try {
        // Create source nodes (permanent – never reassigned)
        sourceRefA.current = ctx.createMediaElementSource(a);
        sourceRefB.current = ctx.createMediaElementSource(b);

        // Create gain nodes
        gainRefA.current = ctx.createGain();
        gainRefB.current = ctx.createGain();

        // Initial gains: A at volume level, B silent
        gainRefA.current.gain.setValueAtTime(GAIN_FLOOR, ctx.currentTime);
        gainRefB.current.gain.setValueAtTime(GAIN_FLOOR, ctx.currentTime);

        // Connect: source -> gain -> destination
        sourceRefA.current.connect(gainRefA.current);
        gainRefA.current.connect(ctx.destination);
        sourceRefB.current.connect(gainRefB.current);
        gainRefB.current.connect(ctx.destination);

        ctxInitDone.current = true;
      } catch (err) {
        // If source nodes fail (e.g. element already captured by another
        // context), fall back to non-Web-Audio mode.
        console.warn("[useCrossfade] Web Audio init failed, falling back:", err);
        webAudioOk.current = false;
        return false;
      }
    }

    return true;
  }, []);

  // -----------------------------------------------------------------------
  // Tear down and recreate AudioContext (for mobile lifecycle)
  // -----------------------------------------------------------------------
  const teardownAndRecreateContext = useCallback(() => {
    // Disconnect old nodes
    try { sourceRefA.current?.disconnect(); } catch {}
    try { sourceRefB.current?.disconnect(); } catch {}
    try { gainRefA.current?.disconnect(); } catch {}
    try { gainRefB.current?.disconnect(); } catch {}
    try { ctxRef.current?.close(); } catch {}

    sourceRefA.current = null;
    sourceRefB.current = null;
    gainRefA.current = null;
    gainRefB.current = null;
    ctxRef.current = null;
    ctxInitDone.current = false;

    // Recreate on next user interaction
  }, []);

  // -----------------------------------------------------------------------
  // AudioContext lifecycle: onstatechange + visibilitychange
  // -----------------------------------------------------------------------
  useEffect(() => {
    // --- onstatechange: detect closed context (mobile browser reclaims) ---
    function onCtxStateChange() {
      if (ctxRef.current?.state === "closed") {
        teardownAndRecreateContext();
      }
    }

    // Attach listener if context already exists
    if (ctxRef.current) {
      ctxRef.current.onstatechange = onCtxStateChange;
    }

    // --- visibilitychange: resume suspended context when tab returns ------
    function onVisibility() {
      if (document.visibilityState !== "visible") return;
      const ctx = ctxRef.current;
      if (!ctx) return;

      if (ctx.state === "suspended") {
        ctx.resume().catch(() => {});
      }

      // Also prod the active audio element in case play() was rejected
      const el = activeSlot.current === "A" ? audioRefA.current : audioRefB.current;
      if (el && statusRef.current === "playing" && el.paused) {
        el.play().catch(() => {});
      }
    }

    document.addEventListener("visibilitychange", onVisibility);

    return () => {
      document.removeEventListener("visibilitychange", onVisibility);
      if (ctxRef.current) ctxRef.current.onstatechange = null;
    };
  }, [teardownAndRecreateContext]);

  // -----------------------------------------------------------------------
  // Health check interval — detect stalled live streams
  // -----------------------------------------------------------------------
  useEffect(() => {
    healthTimer.current = setInterval(() => {
      const el = activeSlot.current === "A" ? audioRefA.current : audioRefB.current;
      if (!el) return;

      if (!el.paused && (el.currentTime > 0 || el.readyState >= 3)) {
        // Stream is healthy and producing data
        if (statusRef.current === "loading") {
          setPlayerStatus("playing");
        }
      }
    }, HEALTH_INTERVAL);

    return () => clearInterval(healthTimer.current);
  }, [setPlayerStatus]);

  // -----------------------------------------------------------------------
  // Audio element event wiring
  // -----------------------------------------------------------------------
  useEffect(() => {
    const a = audioRefA.current;
    const b = audioRefB.current;
    if (!a || !b) return;

    // We only attach events that update the public status to the
    // *active* element.  Since the active slot can change, we use a
    // thin wrapper that checks which element is active before acting.

    function isActive(el) {
      return (activeSlot.current === "A" && el === a) ||
             (activeSlot.current === "B" && el === b);
    }

    // --- Shared handler factories -----------------------------------------
    function makeHandlers(el) {
      const onPlaying   = () => { if (isActive(el)) setPlayerStatus("playing"); };
      const onPause     = () => { if (isActive(el)) setPlayerStatus("paused"); };
      const onWaiting   = () => { if (isActive(el)) setPlayerStatus("loading", "Buffering\u2026"); };
      const onStalled   = () => { if (isActive(el)) setPlayerStatus("loading", "Buffering\u2026"); };
      const onTimeUpdate = () => {
        if (isActive(el) && !el.paused && el.currentTime > 0) {
          setPlayerStatus("playing");
        }
      };
      const onError     = () => {
        if (isActive(el)) {
          setPlayerStatus("error", "Playback error");
        }
      };

      return { onPlaying, onPause, onWaiting, onStalled, onTimeUpdate, onError };
    }

    const hA = makeHandlers(a);
    const hB = makeHandlers(b);

    const eventsA = [
      ["playing", hA.onPlaying], ["pause", hA.onPause],
      ["waiting", hA.onWaiting], ["stalled", hA.onStalled],
      ["timeupdate", hA.onTimeUpdate], ["error", hA.onError],
    ];
    const eventsB = [
      ["playing", hB.onPlaying], ["pause", hB.onPause],
      ["waiting", hB.onWaiting], ["stalled", hB.onStalled],
      ["timeupdate", hB.onTimeUpdate], ["error", hB.onError],
    ];

    eventsA.forEach(([e, h]) => a.addEventListener(e, h));
    eventsB.forEach(([e, h]) => b.addEventListener(e, h));

    return () => {
      eventsA.forEach(([e, h]) => a.removeEventListener(e, h));
      eventsB.forEach(([e, h]) => b.removeEventListener(e, h));
    };
  }, [setPlayerStatus]);

  // -----------------------------------------------------------------------
  // getElement / getGain for a given slot
  // -----------------------------------------------------------------------
  function elFor(slot) {
    return slot === "A" ? audioRefA.current : audioRefB.current;
  }
  function gainFor(slot) {
    return slot === "A" ? gainRefA.current : gainRefB.current;
  }
  function otherSlot(slot) {
    return slot === "A" ? "B" : "A";
  }

  // -----------------------------------------------------------------------
  // Core: crossfade to a new stream
  // -----------------------------------------------------------------------
  const switchStream = useCallback((url, streamId) => {
    if (!url) return;

    // Bump crossfade ID so any in-flight crossfade knows it is stale.
    const myId = ++crossfadeId.current;

    // Cancel any pending pause of a previous outgoing element.
    clearTimeout(pauseTimer.current);

    const oldSlot = activeSlot.current;
    const newSlot = otherSlot(oldSlot);
    const oldEl   = elFor(oldSlot);
    const newEl   = elFor(newSlot);
    if (!newEl) return;

    const isFirstPlay = statusRef.current === "idle" || statusRef.current === "paused";

    // Ensure AudioContext is ready (first user gesture creates it)
    const usingWebAudio = ensureAudioContext();

    // Reattach onstatechange in case we just created a new context
    if (usingWebAudio && ctxRef.current) {
      ctxRef.current.onstatechange = () => {
        if (ctxRef.current?.state === "closed") {
          teardownAndRecreateContext();
        }
      };
    }

    setPlayerStatus("loading", "Connecting\u2026");

    // --- Rapid switching: cancel any ongoing crossfade on the old outgoing
    //     element (the one that *was* fading out in a previous switch).
    //     Immediately silence it and pause.
    const prevOutgoing = elFor(newSlot); // newSlot was the outgoing from last time
    if (usingWebAudio) {
      const prevGain = gainFor(newSlot);
      if (prevGain && ctxRef.current) {
        const now = ctxRef.current.currentTime;
        prevGain.gain.cancelScheduledValues(now);
        prevGain.gain.setValueAtTime(GAIN_FLOOR, now);
      }
    }
    if (prevOutgoing && !prevOutgoing.paused && prevOutgoing !== oldEl) {
      prevOutgoing.pause();
    }

    // --- Load the new stream ------------------------------------------------
    newEl.src = url;
    newEl.load();

    // Ensure new element volume/muted is correct
    if (!usingWebAudio) {
      newEl.volume = mutedRef.current ? 0 : volumeRef.current / 100;
      newEl.muted = mutedRef.current;
    } else {
      newEl.volume = 1;
      newEl.muted = false;
    }

    // --- Wait for canplay (with 8-second timeout) ---------------------------
    const canplayPromise = new Promise((resolve, reject) => {
      let settled = false;
      const timeoutId = setTimeout(() => {
        if (!settled) { settled = true; reject(new Error("timeout")); }
      }, CANPLAY_TIMEOUT);

      function onCanPlay() {
        if (!settled) {
          settled = true;
          clearTimeout(timeoutId);
          newEl.removeEventListener("canplay", onCanPlay);
          newEl.removeEventListener("error", onError);
          resolve();
        }
      }
      function onError() {
        if (!settled) {
          settled = true;
          clearTimeout(timeoutId);
          newEl.removeEventListener("canplay", onCanPlay);
          newEl.removeEventListener("error", onError);
          reject(new Error("network"));
        }
      }

      newEl.addEventListener("canplay", onCanPlay);
      newEl.addEventListener("error", onError);
    });

    canplayPromise
      .then(() => {
        // Stale? Another switchStream was called in the meantime.
        if (crossfadeId.current !== myId) return;

        // Start playback on the new element
        return newEl.play().then(() => {
          if (crossfadeId.current !== myId) return;

          // Flip the active slot
          activeSlot.current = newSlot;
          currentIdRef.current = streamId;
          setCurrentId(streamId);

          // ----- Crossfade gain ramps (Web Audio path) -----
          if (usingWebAudio && ctxRef.current && gainFor(oldSlot) && gainFor(newSlot)) {
            const ctx  = ctxRef.current;
            const now  = ctx.currentTime;
            const end  = now + CROSSFADE_DURATION;
            const targetVol = Math.max(mutedRef.current ? GAIN_FLOOR : volumeRef.current / 100, GAIN_FLOOR);

            const outGain = gainFor(oldSlot);
            const inGain  = gainFor(newSlot);

            // Cancel any residual scheduled values
            outGain.gain.cancelScheduledValues(now);
            inGain.gain.cancelScheduledValues(now);

            if (isFirstPlay) {
              // First play: just fade in, no outgoing ramp needed.
              inGain.gain.setValueAtTime(GAIN_FLOOR, now);
              inGain.gain.exponentialRampToValueAtTime(targetVol, end);
            } else {
              // Full crossfade: ramp out old, ramp in new.
              // Snapshot current actual values before scheduling.
              const currentOutVal = Math.max(outGain.gain.value, GAIN_FLOOR);
              const currentInVal  = GAIN_FLOOR;

              outGain.gain.setValueAtTime(currentOutVal, now);
              outGain.gain.exponentialRampToValueAtTime(GAIN_FLOOR, end);

              inGain.gain.setValueAtTime(currentInVal, now);
              inGain.gain.exponentialRampToValueAtTime(targetVol, end);
            }

            // After crossfade completes, pause the outgoing element to free
            // its network connection and CPU.
            pauseTimer.current = setTimeout(() => {
              // Guard: only pause if this crossfade is still the latest
              if (crossfadeId.current !== myId) return;
              const oel = elFor(oldSlot);
              if (oel && !oel.paused) oel.pause();

              // Hard-silence the outgoing gain in case ramp didn't quite finish
              if (outGain && ctxRef.current) {
                const t = ctxRef.current.currentTime;
                outGain.gain.cancelScheduledValues(t);
                outGain.gain.setValueAtTime(GAIN_FLOOR, t);
              }
            }, CROSSFADE_DURATION * 1000 + 50);

          } else {
            // ----- Fallback: no Web Audio, instant switch -----
            if (oldEl && !oldEl.paused) oldEl.pause();
          }

          setPlayerStatus("playing");
        });
      })
      .catch((err) => {
        // Stale switch — ignore.
        if (crossfadeId.current !== myId) return;

        if (err?.message === "timeout") {
          // 8-second timeout: revert, keep current stream.
          newEl.pause();
          newEl.removeAttribute("src");
          newEl.load();
          setPlayerStatus("error", "Stream took too long to load");
        } else if (err?.message === "network") {
          // Network error: keep current stream.
          newEl.pause();
          newEl.removeAttribute("src");
          newEl.load();
          setPlayerStatus("error", "Could not connect to stream");
        } else {
          // play() rejection (e.g. autoplay blocked)
          setPlayerStatus("error", "Unable to start audio");
          console.warn("[useCrossfade] play() rejected:", err);
        }
      });
  }, [ensureAudioContext, teardownAndRecreateContext, setPlayerStatus, applyVolume]);

  // -----------------------------------------------------------------------
  // play / pause the current stream
  // -----------------------------------------------------------------------
  const play = useCallback(() => {
    const el = elFor(activeSlot.current);
    if (!el) return;

    // Ensure AudioContext is alive
    ensureAudioContext();

    el.play()
      .then(() => {
        setPlayerStatus("playing");

        // Restore gain to audible level
        if (webAudioOk.current && ctxRef.current) {
          const g = gainFor(activeSlot.current);
          if (g) {
            const targetVol = Math.max(mutedRef.current ? GAIN_FLOOR : volumeRef.current / 100, GAIN_FLOOR);
            const now = ctxRef.current.currentTime;
            g.gain.cancelScheduledValues(now);
            g.gain.setValueAtTime(targetVol, now);
          }
        }
      })
      .catch(() => {
        setPlayerStatus("error", "Unable to resume audio");
      });
  }, [ensureAudioContext, setPlayerStatus]);

  const pause = useCallback(() => {
    const el = elFor(activeSlot.current);
    if (!el) return;
    el.pause();
    setPlayerStatus("paused");
  }, [setPlayerStatus]);

  // -----------------------------------------------------------------------
  // Cleanup on unmount
  // -----------------------------------------------------------------------
  useEffect(() => {
    return () => {
      clearTimeout(loadingTimer.current);
      clearTimeout(pauseTimer.current);
      clearInterval(healthTimer.current);

      // Pause both elements
      try { audioRefA.current?.pause(); } catch {}
      try { audioRefB.current?.pause(); } catch {}

      // Close AudioContext
      try { ctxRef.current?.close(); } catch {}
    };
  }, []);

  // -----------------------------------------------------------------------
  // Public API
  // -----------------------------------------------------------------------
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
