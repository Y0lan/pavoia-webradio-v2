"use client";

import { useAudioStore } from "@/lib/audio-engine";
import { useWSStore } from "@/lib/ws";
import { STAGES } from "@/lib/stages";
import { useEffect } from "react";
import Link from "next/link";

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
            // 9 STAGES
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

          return (
            <button
              key={stage.id}
              onClick={() => handleStageClick(stage.id)}
              className="clip-card text-left p-5 transition-all duration-250"
              style={{
                background: isActive ? "var(--color-bg-card-hover)" : "var(--color-bg-card)",
                border: `1px solid ${isActive ? stage.color + "40" : "var(--border-subtle)"}`,
                borderTop: `2px solid ${stage.color}`,
                boxShadow: isActive ? `0 0 12px ${stage.color}30` : "none",
              }}
            >
              {/* Stage name */}
              <div className="flex items-center gap-2 mb-2">
                {isActive && (
                  <div
                    className="w-2 h-2"
                    style={{ background: stage.color, boxShadow: `0 0 6px ${stage.color}` }}
                  />
                )}
                <span
                  className="font-[family-name:var(--font-display)] text-[15px] font-bold"
                  style={{ color: "var(--color-text-primary)" }}
                >
                  {stage.name}
                </span>
              </div>

              {/* Icon + description */}
              <div className="font-[family-name:var(--font-mono)] text-[11px] leading-relaxed"
                style={{ color: "var(--color-text-secondary)" }}>
                <span className="mr-1">{stage.icon}</span>
                {stage.desc}
              </div>

              {/* Now playing */}
              <div className="mt-3">
                {np ? (
                  <>
                    <div
                      className="font-[family-name:var(--font-mono)] text-[12px] truncate"
                      style={{ color: "var(--color-text-primary)" }}
                    >
                      {np.title}
                    </div>
                    <div
                      className="font-[family-name:var(--font-mono)] text-[11px] truncate"
                      style={{ color: "var(--color-text-secondary)" }}
                    >
                      {np.artist}
                    </div>
                  </>
                ) : (
                  <div
                    className="font-[family-name:var(--font-terminal)] text-[10px] tracking-[0.1em] uppercase"
                    style={{ color: "var(--color-text-muted)" }}
                  >
                    // WAITING FOR DATA
                  </div>
                )}
              </div>
            </button>
          );
        })}
      </div>
    </main>
  );
}
