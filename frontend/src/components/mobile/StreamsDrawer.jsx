import React from "react";
import { StreamsList } from "../sidebar/StreamsList";

export function StreamsDrawer({ open, onClose, streams, viewingId, playingId, isPlaying, onSelect }) {
  return (
    <div
      className={["fixed inset-0 z-50 md:hidden transition", open ? "pointer-events-auto" : "pointer-events-none"].join(" ")}
      aria-hidden={!open}
    >
      <div
        className={["absolute inset-0 bg-black/40 backdrop-blur-sm transition-opacity", open ? "opacity-100" : "opacity-0"].join(" ")}
        onClick={onClose}
      />
      <div
        className={[
          "absolute left-0 top-0 h-full w-[92vw] max-w-[420px] bg-[#0e1118] border-r border-white/10 p-4 overflow-auto",
          "transition-transform", open ? "translate-x-0" : "-translate-x-full",
        ].join(" ")}
      >
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-lg font-semibold">Explore</h2>
          <button onClick={onClose} className="px-3 py-1 rounded-lg border border-white/20 bg-white/5 hover:border-white/30">✕</button>
        </div>
        <StreamsList
          streams={streams}
          viewingId={viewingId}
          playingId={playingId}
          isPlaying={isPlaying}
          onSelect={onSelect}
        />
      </div>
    </div>
  );
}
