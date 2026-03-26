"use client";

import { useParams } from "next/navigation";
import { useEffect, useRef } from "react";
import { useAudioStore } from "@/lib/audio-engine";
import { useWSStore } from "@/lib/ws";
import { getStage, STAGES } from "@/lib/stages";

export default function StagePage() {
  const { id } = useParams<{ id: string }>();
  const stage = getStage(id);
  const { playing, stageId, play, switchStage, analyserNode } = useAudioStore();
  const { subscribe, stages } = useWSStore();
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const color = stage?.color || "#00ffc8";
  const wsStage = stages.get(id);
  const np = wsStage?.nowPlaying;

  useEffect(() => {
    subscribe([id]);
  }, [id, subscribe]);

  useEffect(() => {
    if (!id) return;
    if (playing && stageId !== id) switchStage(id);
    else if (!playing) play(id);
  }, [id]); // eslint-disable-line react-hooks/exhaustive-deps

  // Visualizer
  useEffect(() => {
    if (!analyserNode || !canvasRef.current || !playing) return;
    const canvas = canvasRef.current;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;
    const buf = new Uint8Array(analyserNode.frequencyBinCount);
    let animId: number;
    function draw() {
      animId = requestAnimationFrame(draw);
      analyserNode!.getByteFrequencyData(buf);
      ctx!.clearRect(0, 0, canvas.width, canvas.height);
      const bars = 64;
      const w = canvas.width / bars;
      const step = Math.floor(buf.length / bars);
      for (let i = 0; i < bars; i++) {
        const v = buf[i * step] / 255;
        const h = v * canvas.height;
        ctx!.fillStyle = color;
        ctx!.shadowColor = color;
        ctx!.shadowBlur = 4;
        ctx!.fillRect(i * w, canvas.height - h, w - 1, h);
      }
    }
    draw();
    return () => cancelAnimationFrame(animId);
  }, [analyserNode, playing, color]);

  if (!stage) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <span className="font-[family-name:var(--font-terminal)] text-[11px] uppercase tracking-[0.25em]"
          style={{ color: "var(--color-text-muted)" }}>// STAGE NOT FOUND</span>
      </div>
    );
  }

  return (
    <div className="relative min-h-screen overflow-hidden">
      {/* Blurred backdrop */}
      <div className="absolute inset-0 opacity-15" style={{
        background: `radial-gradient(circle at 50% 30%, ${color}, transparent 70%)`,
        filter: "blur(100px)",
      }} />

      <div className="relative z-10 flex flex-col items-center pt-12 px-6">
        {/* Stage name */}
        <div className="font-[family-name:var(--font-terminal)] text-[11px] tracking-[0.25em] uppercase mb-8"
          style={{ color }}>
          {stage.name}
        </div>

        {/* Album art */}
        <div className="relative w-[280px] h-[280px] clip-card mb-6 overflow-hidden"
          style={{ background: "var(--color-bg-card)", border: `1px solid ${color}20` }}>
          <div className="w-full h-full flex items-center justify-center">
            <span className="font-[family-name:var(--font-display)] text-6xl opacity-10" style={{ color }}>♫</span>
          </div>
          {playing && (
            <div className="absolute left-0 right-0 h-[2px] opacity-40"
              style={{ background: `linear-gradient(90deg, transparent, ${color}, transparent)`, animation: "scan 4s linear infinite" }} />
          )}
        </div>

        {/* Track info */}
        <h1 className="font-[family-name:var(--font-display)] text-[28px] font-bold text-center mb-1"
          style={{ color: "var(--color-text-primary)" }}>
          {np?.title || "// LOADING..."}
        </h1>
        <p className="font-[family-name:var(--font-mono)] text-[15px] mb-1" style={{ color }}>
          {np?.artist || ""}
        </p>
        <p className="font-[family-name:var(--font-mono)] text-[13px]" style={{ color: "var(--color-text-secondary)" }}>
          {np?.album || ""}
        </p>

        {/* Description */}
        <p className="font-[family-name:var(--font-mono)] text-[12px] mt-4 max-w-md text-center"
          style={{ color: "var(--color-text-secondary)" }}>
          {stage.icon} {stage.desc}
        </p>

        {/* Visualizer */}
        <canvas ref={canvasRef} width={800} height={120}
          className="w-full max-w-[800px] mt-8" style={{ opacity: playing ? 1 : 0.2 }} />
      </div>
    </div>
  );
}
