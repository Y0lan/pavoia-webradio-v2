import React from "react";

export function EqualizerBars() {
  return (
    <div className="flex items-end gap-0.5 h-5" aria-hidden>
      {[0, 1, 2, 3].map((i) => (
        <span
          key={i}
          className="w-[3px] bg-cyan-300 rounded-sm animate-pulse"
          style={{ animationDelay: `${i * 120}ms`, height: `${6 + ((i % 3) * 4)}px` }}
        />
      ))}
    </div>
  );
}
