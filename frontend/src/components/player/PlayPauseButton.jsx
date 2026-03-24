import React from "react";
import { EqualizerBars } from "./EqualizerBars";

export function PlayPauseButton({ status, onClick }) {
  const isLoading = status === "loading";
  const isPlaying = status === "playing";
  const isPaused = status === "paused" || status === "idle" || status === "error";

  return (
    <button
      onClick={onClick}
      disabled={isLoading}
      className={[
        "group relative inline-flex items-center gap-3 px-4 py-2 rounded-xl",
        "border transition shadow-sm",
        isPlaying
          ? "border-cyan-400/60 bg-[#0d1720] hover:border-cyan-300/80"
          : "border-[#2b3445] bg-[#171c24] hover:border-[#3b4760]",
        isLoading ? "opacity-80 cursor-wait" : "",
      ].join(" ")}
      aria-pressed={isPlaying}
      title={isPlaying ? "Pause" : isLoading ? "Connecting…" : "Play"}
    >
      <div className="relative w-7 h-7 flex items-center justify-center">
        {isLoading && (
          <span className="absolute inline-block w-5 h-5 rounded-full border-2 border-cyan-300/40 border-t-cyan-300 animate-spin" />
        )}
        {isPlaying && !isLoading && <EqualizerBars />}
        {isPaused && !isLoading && (
          <span className="relative block">
            <span className="ml-0.5 border-l-[12px] border-l-cyan-300 border-y-[7px] border-y-transparent block" />
            <span className="absolute inset-0 blur-sm opacity-40 bg-cyan-400/30 rounded-full -z-10" />
          </span>
        )}
      </div>
      <span className="text-sm font-semibold">
        {isLoading ? "Connecting…" : isPlaying ? "Pause" : "Play"}
      </span>
    </button>
  );
}
