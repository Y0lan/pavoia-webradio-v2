import { create } from "zustand";
import { BRIDGE_URL } from "./stages";

export type CrossfadeCurve = "equal-power" | "s-curve" | "exponential" | "linear" | "cut";

export interface AudioState {
  // Playback
  playing: boolean;
  stageId: string | null;
  volume: number;
  muted: boolean;

  // Crossfade
  crossfading: boolean;
  crossfadeDuration: number; // seconds
  crossfadeCurve: CrossfadeCurve;

  // Analyser data (updated per frame by the visualizer)
  analyserNode: AnalyserNode | null;

  // Actions
  play: (stageId: string) => void;
  stop: () => void;
  setVolume: (v: number) => void;
  toggleMute: () => void;
  switchStage: (stageId: string) => void;
  setCrossfadeCurve: (curve: CrossfadeCurve) => void;
  setCrossfadeDuration: (seconds: number) => void;
}

// Dual-slot audio engine
//
// Two <audio> elements alternate as source and target.
// When switching stages, the target fades in while the source fades out.
//
// ┌──────────┐     ┌───────────┐     ┌───────────┐     ┌─────────┐
// │ Audio[0] │────►│ GainNode0 │────►│           │     │         │
// └──────────┘     └───────────┘     │ Analyser  │────►│  Dest   │
// ┌──────────┐     ┌───────────┐     │           │     │         │
// │ Audio[1] │────►│ GainNode1 │────►│           │     │         │
// └──────────┘     └───────────┘     └───────────┘     └─────────┘

let audioCtx: AudioContext | null = null;
let audioElements: [HTMLAudioElement | null, HTMLAudioElement | null] = [null, null];
let gainNodes: [GainNode | null, GainNode | null] = [null, null];
let sourceNodes: [MediaElementAudioSourceNode | null, MediaElementAudioSourceNode | null] = [null, null];
let analyser: AnalyserNode | null = null;
let activeSlot: 0 | 1 = 0;

function getAudioContext(): AudioContext {
  if (!audioCtx) {
    audioCtx = new AudioContext();
    analyser = audioCtx.createAnalyser();
    analyser.fftSize = 256;
    analyser.connect(audioCtx.destination);
  }
  return audioCtx;
}

function getOrCreateSlot(slot: 0 | 1): HTMLAudioElement {
  if (!audioElements[slot]) {
    const el = new Audio();
    el.crossOrigin = "anonymous";
    audioElements[slot] = el;

    const ctx = getAudioContext();
    const source = ctx.createMediaElementSource(el);
    const gain = ctx.createGain();
    gain.gain.value = 0;
    source.connect(gain);
    gain.connect(analyser!);

    sourceNodes[slot] = source;
    gainNodes[slot] = gain;
  }
  return audioElements[slot]!;
}

function streamUrl(stageId: string): string {
  return `${BRIDGE_URL}/api/stream/${stageId}`;
}

// Compute gain curve values at a given progress (0 → 1)
function curveValue(progress: number, curve: CrossfadeCurve): [number, number] {
  switch (curve) {
    case "equal-power": {
      const out = Math.cos(progress * Math.PI * 0.5);
      const inn = Math.sin(progress * Math.PI * 0.5);
      return [out, inn];
    }
    case "s-curve": {
      const s = progress * progress * (3 - 2 * progress); // smoothstep
      return [1 - s, s];
    }
    case "exponential": {
      const out = Math.pow(1 - progress, 2);
      const inn = Math.pow(progress, 2);
      return [out, inn];
    }
    case "linear":
      return [1 - progress, progress];
    case "cut":
      return [progress < 0.5 ? 1 : 0, progress < 0.5 ? 0 : 1];
  }
}

export const useAudioStore = create<AudioState>((set, get) => ({
  playing: false,
  stageId: null,
  volume: 0.8,
  muted: false,
  crossfading: false,
  crossfadeDuration: 3,
  crossfadeCurve: "equal-power" as CrossfadeCurve,
  analyserNode: null,

  play: (stageId: string) => {
    const ctx = getAudioContext();
    if (ctx.state === "suspended") ctx.resume();

    const el = getOrCreateSlot(activeSlot);
    el.src = streamUrl(stageId);
    el.play().catch(() => {});

    const gain = gainNodes[activeSlot]!;
    const vol = get().muted ? 0 : get().volume;
    gain.gain.setValueAtTime(vol, ctx.currentTime);

    set({ playing: true, stageId, analyserNode: analyser });
  },

  stop: () => {
    for (const el of audioElements) {
      if (el) {
        el.pause();
        el.src = "";
      }
    }
    for (const g of gainNodes) {
      if (g) g.gain.value = 0;
    }
    set({ playing: false, stageId: null });
  },

  setVolume: (v: number) => {
    const vol = Math.max(0, Math.min(1, v));
    set({ volume: vol, muted: false });
    const gain = gainNodes[activeSlot];
    if (gain && !get().muted) {
      gain.gain.setValueAtTime(vol, getAudioContext().currentTime);
    }
  },

  toggleMute: () => {
    const muted = !get().muted;
    set({ muted });
    const gain = gainNodes[activeSlot];
    if (gain) {
      const ctx = getAudioContext();
      gain.gain.setValueAtTime(muted ? 0 : get().volume, ctx.currentTime);
    }
  },

  switchStage: (stageId: string) => {
    if (stageId === get().stageId) return;
    if (!get().playing) {
      get().play(stageId);
      return;
    }

    const state = get();
    set({ crossfading: true });

    const ctx = getAudioContext();
    const sourceSlot = activeSlot;
    const targetSlot: 0 | 1 = activeSlot === 0 ? 1 : 0;

    // Set up target
    const targetEl = getOrCreateSlot(targetSlot);
    targetEl.src = streamUrl(stageId);
    targetEl.play().catch(() => {});

    const sourceGain = gainNodes[sourceSlot]!;
    const targetGain = gainNodes[targetSlot]!;
    const vol = state.muted ? 0 : state.volume;
    const duration = state.crossfadeDuration;
    const curve = state.crossfadeCurve;

    // Animate crossfade
    const startTime = ctx.currentTime;
    targetGain.gain.setValueAtTime(0, startTime);

    const steps = Math.ceil(duration * 30); // 30 updates/sec
    const interval = (duration / steps) * 1000;
    let step = 0;

    const timer = setInterval(() => {
      step++;
      const progress = Math.min(step / steps, 1);
      const [fadeOut, fadeIn] = curveValue(progress, curve);
      sourceGain.gain.setValueAtTime(fadeOut * vol, ctx.currentTime);
      targetGain.gain.setValueAtTime(fadeIn * vol, ctx.currentTime);

      if (progress >= 1) {
        clearInterval(timer);
        // Clean up source
        const sourceEl = audioElements[sourceSlot];
        if (sourceEl) {
          sourceEl.pause();
          sourceEl.src = "";
        }
        activeSlot = targetSlot;
        set({ crossfading: false, stageId });
      }
    }, interval);

    set({ stageId });
  },

  setCrossfadeCurve: (curve) => set({ crossfadeCurve: curve }),
  setCrossfadeDuration: (seconds) => set({ crossfadeDuration: Math.max(0.5, Math.min(10, seconds)) }),
}));
