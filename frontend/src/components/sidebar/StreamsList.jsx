import React from "react";
import { getMetaFor } from "../../utils/streamMeta";

export function StreamsList({ streams, activeId, playingId, isPlaying, onSelect }) {
  const order = {
    "gaende-favorites": 0,
    "ambiance-safe": 1,
    "bermuda-day": 2,          // Before 18:00 / Oaza
    "bermuda-night": 3,        // 18:00–00:00
    "palac-slow-hypno": 4,     // Palac Feel
    "palac-dance": 5,
    "fontanna-laputa": 6,
    "etage-0": 7,
    "closing": 8,
    "bus": 9,
  };
  const items = [...streams].sort((a, b) => (order[a.id] ?? 999) - (order[b.id] ?? 999));

  return (
    <div className="space-y-2">
      {items.map((s) => {
        const isActive = activeId === s.id;
        const isNowPlaying = isPlaying && playingId === s.id;
        const meta = getMetaFor(s);
        return (
          <button
            key={s.id}
            onClick={() => { if (!meta.disabled) onSelect(s.id); }}
            className={[
              "w-full text-left p-3 rounded-xl border transition flex items-start gap-2",
              isActive
                ? "border-cyan-300/60 outline outline-2 outline-cyan-300/30 bg-[#0f1318]"
                : "border-[#202632] hover:border-[#2a3444] bg-[#0f1318]",
              meta.disabled ? "opacity-60 cursor-not-allowed" : ""
            ].join(" ")}
            aria-disabled={meta.disabled || undefined}
          >
            <span className="mt-0.5 w-5 text-lg" aria-hidden>{meta.icon}</span>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <div className="font-semibold line-clamp-2 leading-snug">{meta.title}</div>
                {isNowPlaying ? (
                  <span className="ml-auto inline-block w-2 h-2 rounded-full bg-cyan-300 animate-pulse" title="Now playing" />
                ) : null}
              </div>
              <div className="text-xs text-slate-400 truncate mt-0.5">{meta.desc}</div>
            </div>
          </button>
        );
      })}
    </div>
  );
}
