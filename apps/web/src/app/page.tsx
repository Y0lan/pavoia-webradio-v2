"use client";

import { useAudioStore } from "@/lib/audio-engine";
import { useWSStore } from "@/lib/ws";
import { STAGES } from "@/lib/stages";
import { useEffect } from "react";

// Parse "123.4" / "" → seconds (the bridge emits strings).
function toSeconds(v: string | undefined | null): number {
  if (!v) return 0;
  const n = parseFloat(v);
  return Number.isFinite(n) ? n : 0;
}

function formatTime(s: number): string {
  if (!s || s < 0) return "";
  const m = Math.floor(s / 60);
  const r = Math.floor(s % 60);
  return `${m}:${String(r).padStart(2, "0")}`;
}

export default function Home() {
  const { playing, stageId, switchStage, play } = useAudioStore();
  const { subscribe, stages } = useWSStore();

  useEffect(() => {
    subscribe(STAGES.map((s) => s.id));
  }, [subscribe]);

  const handleStageClick = (id: string) => {
    if (playing) {
      switchStage(id);
    } else {
      play(id);
    }
  };

  return (
    <main className="min-h-screen p-6">
      {/* Header */}
      <div className="mb-8">
        <h1
          className="font-[family-name:var(--font-display)] text-3xl font-bold tracking-[0.02em]"
          style={{ color: "var(--color-text-primary)" }}
        >
          Pavoia Webradio
        </h1>
        <div className="flex items-center gap-2 mt-1">
          <span
            className="font-[family-name:var(--font-terminal)] text-[11px] tracking-[0.25em] uppercase"
            style={{ color: "var(--color-accent)" }}
          >
            // 9 STAGES · 24/7
          </span>
          <div
            className="flex-1 h-px"
            style={{
              background: "linear-gradient(90deg, var(--color-accent-dim), transparent)",
            }}
          />
        </div>
      </div>

      {/* 3x3 Stage Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {STAGES.map((stage) => {
          const isActive = stageId === stage.id;
          const wsStage = stages.get(stage.id);
          const np = wsStage?.nowPlaying;
          const alive = wsStage?.alive ?? false;

          // Elapsed/duration drive a live progress bar at the card bottom —
          // cheap dopamine per design system "each page should feel alive."
          const elapsed = toSeconds(np?.elapsed);
          const duration = toSeconds(np?.duration);
          const progress = duration > 0 ? Math.min(1, elapsed / duration) : 0;

          return (
            <button
              key={stage.id}
              onClick={() => handleStageClick(stage.id)}
              className="clip-card text-left p-5 pb-6 transition-all duration-250 relative overflow-hidden"
              style={{
                background: isActive ? "var(--color-bg-card-hover)" : "var(--color-bg-card)",
                border: `1px solid ${isActive ? stage.color + "40" : "var(--border-subtle)"}`,
                borderTop: `2px solid ${stage.color}`,
                boxShadow: isActive ? `0 0 18px ${stage.color}40` : "none",
              }}
            >
              {/* Scanning laser — only when the stream is live. Uses the
                  stage's own color, so it reads like a stage-themed pulse,
                  not a generic accent flourish. */}
              {alive && (
                <div
                  className="absolute left-0 right-0 h-[2px] pointer-events-none"
                  style={{
                    background: `linear-gradient(90deg, transparent, ${stage.color}, transparent)`,
                    opacity: 0.35,
                    animation: "scan 4s linear infinite",
                  }}
                />
              )}

              {/* Stage name + live LED */}
              <div className="flex items-center gap-2 mb-2">
                {/* LED: solid pulsing square when alive, dim otherwise.
                    Uses stage color so a glance tells you which stage is live. */}
                <span
                  className="w-2 h-2 shrink-0"
                  style={{
                    background: alive ? stage.color : "var(--color-text-ghost)",
                    boxShadow: alive ? `0 0 8px ${stage.color}` : "none",
                    animation: alive ? "ledPulse 2.6s ease-in-out infinite" : "none",
                  }}
                />
                <span
                  className="font-[family-name:var(--font-display)] text-[15px] font-bold truncate"
                  style={{ color: "var(--color-text-primary)" }}
                >
                  {stage.name}
                </span>
              </div>

              {/* Icon + description */}
              <div
                className="font-[family-name:var(--font-mono)] text-[11px] leading-relaxed"
                style={{ color: "var(--color-text-secondary)" }}
              >
                <span className="mr-1">{stage.icon}</span>
                {stage.desc}
              </div>

              {/* Now playing: live title/artist + elapsed readout */}
              <div className="mt-3 min-h-[44px]">
                {np && np.title ? (
                  <>
                    <div
                      className="font-[family-name:var(--font-mono)] text-[12px] truncate"
                      style={{ color: "var(--color-text-primary)" }}
                    >
                      {np.title}
                    </div>
                    <div className="flex items-center justify-between gap-2 mt-0.5">
                      <div
                        className="font-[family-name:var(--font-mono)] text-[11px] truncate"
                        style={{ color: "var(--color-text-secondary)" }}
                      >
                        {np.artist}
                      </div>
                      {duration > 0 && (
                        <div
                          className="font-[family-name:var(--font-terminal)] text-[9px] shrink-0 tabular-nums"
                          style={{ color: "var(--color-text-muted)" }}
                        >
                          {formatTime(elapsed)} / {formatTime(duration)}
                        </div>
                      )}
                    </div>
                  </>
                ) : (
                  <div
                    className="font-[family-name:var(--font-terminal)] text-[10px] tracking-[0.1em] uppercase"
                    style={{ color: "var(--color-text-muted)" }}
                  >
                    {alive ? "// LOADING NOW-PLAYING…" : "// OFFLINE"}
                  </div>
                )}
              </div>

              {/* Live progress bar (stage-colored); absent when no duration
                  to avoid pretending we know a track length we don't have. */}
              {np && duration > 0 && (
                <div
                  className="absolute left-0 right-0 bottom-0 h-[2px] pointer-events-none"
                  style={{ background: "var(--color-bg-elevated)" }}
                >
                  <div
                    className="h-full"
                    style={{
                      width: `${progress * 100}%`,
                      background: stage.color,
                      boxShadow: `0 0 6px ${stage.color}60`,
                      transition: "width 1s linear",
                    }}
                  />
                </div>
              )}
            </button>
          );
        })}
      </div>
    </main>
  );
}
