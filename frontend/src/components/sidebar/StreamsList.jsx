import React from "react";
import { getMetaFor } from "../../utils/streamMeta";
import { EqualizerBars } from "../player/EqualizerBars";

export function StreamsList({ streams, viewingId, playingId, isPlaying, onSelect, onHover, onHoverEnd }) {
  const order = {
    "gaende-favorites": 0, "ambiance-safe": 1, "bermuda-day": 2, "bermuda-night": 3,
    "palac-slow-hypno": 4, "palac-dance": 5, "fontanna-laputa": 6, "etage-0": 7,
    "closing": 8, "bus": 9,
  };
  const items = [...streams].sort((a, b) => (order[a.id] ?? 999) - (order[b.id] ?? 999));

  return (
    <div className="space-y-2">
      {items.map((s) => {
        const isViewing = viewingId === s.id;
        const isNowPlaying = isPlaying && playingId === s.id;
        const meta = getMetaFor(s);
        const isBus = meta.disabled;
        return (
          <button
            key={s.id}
            onClick={() => onSelect(s.id)}
            onMouseEnter={onHover ? (e) => onHover(s.id, e) : undefined}
            onMouseLeave={onHoverEnd || undefined}
            className={[
              "w-full text-left p-3 rounded-xl border transition flex items-start gap-2",
              isViewing ? "bg-white/5" : "border-white/10 hover:border-white/20 bg-white/[0.02]",
              isBus ? "opacity-70 italic" : "",
            ].join(" ")}
            style={isViewing ? { borderColor: meta.accentColor + "60" } : undefined}
          >
            <span className="mt-0.5 w-5 text-lg" aria-hidden>{meta.icon}</span>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <div className="font-semibold line-clamp-2 leading-snug">{meta.title}</div>
                {isNowPlaying && <span className="ml-auto flex-shrink-0"><EqualizerBars /></span>}
              </div>
              <div className="text-xs text-white/40 truncate mt-0.5">{meta.desc}</div>
            </div>
          </button>
        );
      })}
    </div>
  );
}
