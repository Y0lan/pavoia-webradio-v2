import React from "react";
import { TrackProgress } from "./TrackProgress";

export function NowPlaying({ now, artistCard, player, onArtistClick, elapsed, duration, showPlaying }) {
  if (!now) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-[480px_1fr] gap-5 items-center">
        <div className="relative">
          <div className="rounded-2xl border border-[#263041] bg-[#0f1318] aspect-square animate-pulse" />
        </div>
        <div>
          <div className="h-7 w-2/3 rounded bg-[#1a2230] animate-pulse" />
          <div className="mt-2 h-4 w-1/2 rounded bg-[#141a24] animate-pulse" />
          <div className="mt-4 h-4 w-1/3 rounded bg-[#141a24] animate-pulse" />
        </div>
      </div>
    );
  }

  const { title, artist, album, year, cover_url } = now;
  const isLoading = player.status === "loading";

  return (
    <div className="grid grid-cols-1 md:grid-cols-[480px_1fr] gap-5 items-center">
      <div className="relative">
        {cover_url ? (
          <img
            src={cover_url}
            className="w-full aspect-square object-cover rounded-2xl border border-[#263041]"
            onError={(e) => (e.currentTarget.style.visibility = "hidden")}
            alt=""
          />
        ) : (
          <div className="w-full aspect-square rounded-2xl border border-[#263041] bg-[#0f1318] flex items-center justify-center text-slate-500">
            No cover
          </div>
        )}
        {isLoading && (
          <div className="absolute inset-0 bg-black/30 backdrop-blur-[1px] rounded-2xl flex items-center justify-center">
            <span className="inline-block w-8 h-8 rounded-full border-2 border-cyan-300/40 border-t-cyan-300 animate-spin" />
          </div>
        )}
        {showPlaying && (
          <div className="absolute bottom-2 right-2 px-2 py-1 rounded-full text-[11px] bg-cyan-500/20 text-cyan-200 border border-cyan-400/40">
            Playing
          </div>
        )}
      </div>

      <div>
        <h1 className="text-2xl md:text-4xl font-bold">{title || "Unknown title"}</h1>
        <div className="text-slate-300 mt-1 flex flex-wrap items-center gap-x-2 gap-y-1">
          {artistCard ? (
            <button
              className="font-semibold text-[#a78bfa] hover:underline"
              onClick={onArtistClick}
            >
              {artist || "Unknown artist"}
            </button>
          ) : (
            <span className="font-semibold text-slate-400 cursor-not-allowed">
              {artist || "Unknown artist"}
            </span>
          )}
          <span className="text-slate-500">•</span>
          <span>
            {album}
            {year ? <span className="text-slate-500"> ({year})</span> : null}
          </span>
        </div>
        <TrackProgress elapsed={elapsed} duration={duration} />
      </div>
    </div>
  );
}
