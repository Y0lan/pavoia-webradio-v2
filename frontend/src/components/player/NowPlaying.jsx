import React from "react";
import { TrackProgress } from "./TrackProgress";

export function NowPlaying({ now, artistCard, player, onArtistClick, elapsed, duration, showPlaying, isExploring, viewingMeta }) {
  if (!now) {
    return (
      <div className="flex flex-col md:flex-row gap-5 items-center">
        <div className="w-full max-w-[400px] md:max-w-[400px] max-w-[80vw]">
          <div className="rounded-xl border border-white/10 bg-black/30 aspect-square animate-pulse" />
        </div>
        <div className="flex-1 w-full">
          <div className="bg-black/40 backdrop-blur-sm rounded-lg p-4">
            <div className="h-7 w-2/3 rounded bg-white/10 animate-pulse" />
            <div className="mt-2 h-4 w-1/2 rounded bg-white/5 animate-pulse" />
          </div>
        </div>
      </div>
    );
  }

  const { title, artist, album, year, cover_url } = now;
  const isLoading = player.status === "loading";

  return (
    <div className="flex flex-col md:flex-row gap-5 items-center">
      {/* Cover art */}
      <div className="w-full max-w-[400px] md:w-[400px] flex-shrink-0">
        <div className="relative">
          {cover_url ? (
            <img
              src={cover_url}
              className="w-full aspect-square object-cover rounded-xl shadow-lg border border-white/10"
              onError={(e) => (e.currentTarget.style.visibility = "hidden")}
              alt=""
            />
          ) : (
            <div className="w-full aspect-square rounded-xl border border-white/10 bg-black/30 flex items-center justify-center text-white/30 text-4xl">
              {viewingMeta?.icon || "📻"}
            </div>
          )}
          {isLoading && (
            <div className="absolute inset-0 bg-black/30 backdrop-blur-[1px] rounded-xl flex items-center justify-center">
              <span className="inline-block w-8 h-8 rounded-full border-2 border-cyan-300/40 border-t-cyan-300 animate-spin" />
            </div>
          )}
          {showPlaying && (
            <div className="absolute bottom-2 right-2 px-2 py-1 rounded-full text-[11px] bg-cyan-500/20 text-cyan-200 border border-cyan-400/40">
              Playing
            </div>
          )}
        </div>
      </div>

      {/* Track info with dark overlay for contrast */}
      <div className="flex-1 w-full">
        <div className="bg-black/40 backdrop-blur-sm rounded-lg p-4">
          <h1 className="text-2xl md:text-4xl font-bold text-white">{title || "Unknown title"}</h1>
          <div className="text-white/80 mt-1 flex flex-wrap items-center gap-x-2 gap-y-1">
            {artistCard ? (
              <button className="font-semibold text-[#a78bfa] hover:underline" onClick={onArtistClick}>
                {artist || "Unknown artist"}
              </button>
            ) : (
              <span className="font-semibold text-white/60">{artist || "Unknown artist"}</span>
            )}
            <span className="text-white/30">•</span>
            <span>{album}{year ? <span className="text-white/40"> ({year})</span> : null}</span>
          </div>
          <TrackProgress elapsed={elapsed} duration={duration} />
        </div>
      </div>
    </div>
  );
}
