import React from "react";

export function InfoDialog({ open, onClose, onShare }) {
  if (!open) return null; // don't render when closed
  return (
    <div className={["fixed inset-0 z-50 transition", "pointer-events-auto"].join(" ")}
      aria-hidden={!open}
    >
      <div
        className={["absolute inset-0 bg-black/50 backdrop-blur-sm transition-opacity", "opacity-100"].join(" ")}
        onClick={onClose}
      />
      <div
        className={[
          "absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-[94vw] max-w-[720px]",
          "max-h-[85vh] overflow-y-auto",
          "bg-[#0e1118] border border-[#1f2430] rounded-2xl p-4 md:p-6 shadow-2xl",
          "transition-transform scale-100",
        ].join(" ")}
      >
        <div className="relative">
          <button
            type="button"
            onClick={onClose}
            className="absolute right-0 top-0 px-3 py-1 rounded-lg border border-[#2b3445] bg-[#171c24] hover:border-[#3b4760] z-10"
            aria-label="Close"
          >
            ✕
          </button>
        </div>

        {/* Centered brand heading (GIF) */}
        <div className="text-center px-3 mt-1">
          <img
            src="https://pavoia.com/wp-content/uploads/2022/10/PavoiaKV_Homebutton_300px.gif"
            alt="PAVOIA animated logo"
            className="mx-auto w-[120px] md:w-[260px] h-auto"
          />
          <div className="mt-1 md:mt-2 text-slate-400 text-sm md:text-base">
            Pavoia Webradio
          </div>
        </div>

        <div className="mt-3 md:mt-5 text-slate-300 space-y-2 md:space-y-3 text-sm leading-relaxed">
          <p>
            This collection started years ago, built from artists heard at Pavoia and countless hours of digging for new sounds. Every time I fall for a track, I picture which stage it belongs to. What began as a personal obsession slowly grew into something worth sharing. Today, these nine streams are open to everyone, and the playlists keep growing with fresh discoveries.
          </p>
          <p>
            All streams are in <strong>high quality audio</strong>, pure and unprocessed, just as the artists intended.
          </p>
          <p>
            <strong>On your phone?</strong> Most browsers keep playing in the background while the tab stays open. For the best experience, add this page to your home screen and enjoy the mini-player.
          </p>
          <div className="pt-2 border-t border-[#1f2430] grid grid-cols-1 sm:grid-cols-2 gap-2">
            <a href="https://instagram.com/wearepavoia" target="_blank" rel="noopener noreferrer" className="px-3 py-2 rounded-lg border border-[#2b3445] bg-[#171c24] hover:border-[#3b4760]">
              Instagram <span className="text-slate-400">@wearepavoia</span>
            </a>
            <a href="https://soundcloud.com/pavoia" target="_blank" rel="noopener noreferrer" className="px-3 py-2 rounded-lg border border-[#2b3445] bg-[#171c24] hover:border-[#3b4760]">
              SoundCloud <span className="text-slate-400">@pavoia</span>
            </a>
            <button onClick={onShare} className="px-3 py-2 rounded-lg border border-[#2b3445] bg-[#171c24] hover:border-[#3b4760] sm:col-span-2">
              Share this station
            </button>
          </div>
          <div className="pt-2 text-center text-slate-500 text-xs">
            made with ♥ by <a href="https://instagram.com/gaende_music" target="_blank" rel="noopener noreferrer" className="text-slate-400 hover:text-slate-300">Gaende</a>
          </div>
        </div>
      </div>
    </div>
  );
}
