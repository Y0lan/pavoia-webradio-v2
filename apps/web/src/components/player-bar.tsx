"use client";

import { useAudioStore } from "@/lib/audio-engine";
import { useWSStore } from "@/lib/ws";
import { useEffect, useRef } from "react";

export function PlayerBar() {
  const { playing, stageId, volume, muted, stop, setVolume, toggleMute, analyserNode } = useAudioStore();
  const stages = useWSStore((s) => s.stages);
  const canvasRef = useRef<HTMLCanvasElement>(null);

  const currentStage = stageId ? stages.get(stageId) : null;
  const np = currentStage?.nowPlaying;

  // Mini visualizer
  useEffect(() => {
    if (!analyserNode || !canvasRef.current || !playing) return;

    const canvas = canvasRef.current;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    const bufferLength = analyserNode.frequencyBinCount;
    const dataArray = new Uint8Array(bufferLength);
    let animId: number;

    function draw() {
      animId = requestAnimationFrame(draw);
      analyserNode!.getByteFrequencyData(dataArray);

      ctx!.clearRect(0, 0, canvas.width, canvas.height);
      const barCount = 16;
      const barWidth = canvas.width / barCount;
      const step = Math.floor(bufferLength / barCount);

      for (let i = 0; i < barCount; i++) {
        const value = dataArray[i * step] / 255;
        const barHeight = value * canvas.height;
        ctx!.fillStyle = "var(--color-accent)";
        ctx!.fillRect(i * barWidth, canvas.height - barHeight, barWidth - 1, barHeight);
      }
    }

    draw();
    return () => cancelAnimationFrame(animId);
  }, [analyserNode, playing]);

  if (!playing && !stageId) {
    return (
      <div className="fixed bottom-0 left-0 right-0 z-[100] h-16 flex items-center justify-center"
        style={{
          background: "rgba(2, 2, 4, 0.92)",
          backdropFilter: "blur(24px)",
          borderTop: "1px solid var(--border-subtle)",
        }}
      >
        <span className="font-[family-name:var(--font-terminal)] text-[11px] tracking-[0.25em] uppercase"
          style={{ color: "var(--color-text-muted)" }}
        >
          // SELECT A STAGE
        </span>
      </div>
    );
  }

  return (
    <div
      className="fixed bottom-0 left-0 right-0 z-[100] h-[72px] flex items-center gap-4 px-4"
      style={{
        background: "rgba(2, 2, 4, 0.92)",
        backdropFilter: "blur(24px)",
        borderTop: "1px solid var(--border-subtle)",
      }}
    >
      {/* Top gradient line */}
      <div
        className="absolute top-0 left-0 right-0 h-px"
        style={{
          background: "linear-gradient(90deg, transparent, var(--color-accent-dim), transparent)",
        }}
      />

      {/* Album art placeholder with scanning laser */}
      <div className="relative w-12 h-12 shrink-0 overflow-hidden"
        style={{ background: "var(--color-bg-card)", border: "1px solid var(--border-subtle)" }}
      >
        <div className="w-full h-full flex items-center justify-center">
          <span className="font-[family-name:var(--font-terminal)] text-[8px]"
            style={{ color: "var(--color-text-muted)" }}
          >
            ♫
          </span>
        </div>
        {playing && (
          <div
            className="absolute left-0 right-0 h-[2px] opacity-40"
            style={{
              background: "linear-gradient(90deg, transparent, var(--color-accent), transparent)",
              animation: "scan 4s linear infinite",
            }}
          />
        )}
      </div>

      {/* Track info */}
      <div className="flex flex-col min-w-0 flex-1">
        <span className="font-[family-name:var(--font-mono)] text-[13px] font-medium truncate"
          style={{ color: "var(--color-text-primary)" }}
        >
          {np?.title || "// LOADING..."}
        </span>
        <span className="font-[family-name:var(--font-mono)] text-[12px] truncate"
          style={{ color: "var(--color-text-secondary)" }}
        >
          {np?.artist || ""}
        </span>
      </div>

      {/* Mini visualizer */}
      <canvas
        ref={canvasRef}
        width={64}
        height={32}
        className="shrink-0"
        style={{ opacity: playing ? 1 : 0.2 }}
      />

      {/* Controls */}
      <div className="flex items-center gap-3">
        {/* Play/Stop */}
        <button
          onClick={() => (playing ? stop() : null)}
          className="w-8 h-8 flex items-center justify-center clip-octagon"
          style={{
            background: playing ? "var(--color-accent)" : "var(--color-bg-elevated)",
            color: playing ? "var(--color-bg-void)" : "var(--color-accent)",
            boxShadow: playing ? "0 0 12px var(--color-accent-dim)" : "none",
          }}
        >
          {playing ? "■" : "▶"}
        </button>

        {/* Volume */}
        <button
          onClick={toggleMute}
          className="font-[family-name:var(--font-terminal)] text-[10px]"
          style={{ color: muted ? "var(--color-text-muted)" : "var(--color-accent)" }}
        >
          {muted ? "MUTE" : "VOL"}
        </button>
        <input
          type="range"
          min={0}
          max={1}
          step={0.01}
          value={muted ? 0 : volume}
          onChange={(e) => setVolume(parseFloat(e.target.value))}
          className="w-16 h-[3px] appearance-none rounded-none"
          style={{
            background: `linear-gradient(to right, var(--color-accent) ${(muted ? 0 : volume) * 100}%, var(--color-bg-elevated) ${(muted ? 0 : volume) * 100}%)`,
          }}
        />
      </div>

      {/* Stage indicator */}
      {stageId && (
        <div className="font-[family-name:var(--font-terminal)] text-[9px] tracking-[0.1em] uppercase shrink-0"
          style={{ color: "var(--color-text-muted)" }}
        >
          {stageId}
        </div>
      )}
    </div>
  );
}
