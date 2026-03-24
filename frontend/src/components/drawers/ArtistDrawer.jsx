import React, { useState, useEffect } from "react";

export function ArtistDrawer({ open, onClose, artist }) {
  const [current, setCurrent] = useState(artist || null);
  const [clickableMap, setClickableMap] = useState({});

  useEffect(() => {
    if (open) setCurrent(artist || null);
  }, [open, artist]);

  // Probe which similar artists exist
  useEffect(() => {
    let cancel = false;
    async function probe() {
      if (!open || !current || !Array.isArray(current.similar)) return;
      const names = current.similar;
      const results = {};
      await Promise.all(
        names.map(async (n) => {
          try {
            const r = await fetch("/api/artists/" + encodeURIComponent(n));
            results[n] = r.ok;
          } catch {
            results[n] = false;
          }
        })
      );
      if (!cancel) setClickableMap(results);
    }
    probe();
    return () => {
      cancel = true;
    };
  }, [open, current]);

  async function openSimilar(name) {
    try {
      const r = await fetch("/api/artists/" + encodeURIComponent(name));
      if (!r.ok) return;
      const a = await r.json();
      setCurrent(a);
    } catch {}
  }

  return (
    <div
      className={[
        "fixed inset-0 z-40 transition",
        open ? "pointer-events-auto" : "pointer-events-none",
      ].join(" ")}
      aria-hidden={!open}
    >
      <div
        className={[
          "absolute inset-0 bg-black/40 backdrop-blur-sm transition-opacity",
          open ? "opacity-100" : "opacity-0",
        ].join(" ")}
        onClick={onClose}
      />
      <div
        className={[
          "absolute right-0 top-0 h-full w-full md:w-[520px] max-w-[100vw] bg-[#0e1118] border-l border-[#1f2430] p-6 overflow-auto",
          "transition-transform",
          open ? "translate-x-0" : "translate-x-full",
        ].join(" ")}
      >
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-semibold">About the artist</h2>
          <button
            onClick={onClose}
            className="px-3 py-1 rounded-lg border border-[#2b3445] bg-[#171c24] hover:border-[#3b4760]"
          >
            ✕
          </button>
        </div>

        {!current ? (
          <div className="text-slate-400 text-sm">No artist data.</div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-[180px_1fr] gap-4">
            <img
              src={current.art_url || current.thumb_url || ""}
              className="w-full aspect-square object-cover rounded-xl border border-[#263041]"
              onError={(e) => (e.currentTarget.style.display = "none")}
              alt=""
            />
            <div>
              <div className="text-lg font-semibold">{current.name}</div>
              <div className="text-slate-400 whitespace-pre-wrap max-h-72 overflow-auto mt-1">
                {current.bio || "No bio available."}
              </div>

              {Array.isArray(current.similar) && current.similar.length > 0 && (
                <div className="mt-3">
                  <div className="text-slate-400 text-sm mb-1">Similar</div>
                  <div className="flex flex-wrap gap-2">
                    {current.similar.slice(0, 14).map((n) => {
                      const clickable = !!clickableMap[n];
                      return clickable ? (
                        <button
                          key={n}
                          onClick={() => openSimilar(n)}
                          className="px-2 py-1 rounded-full text-xs border border-cyan-400/50 bg-cyan-500/10 text-cyan-200 hover:border-cyan-300 hover:bg-cyan-400/20"
                          title="Open artist"
                        >
                          {n}
                        </button>
                      ) : (
                        <span
                          key={n}
                          className="px-2 py-1 rounded-full text-xs border border-[#202632] bg-[#0f141b] text-slate-400"
                          title="No page available"
                        >
                          {n}
                        </span>
                      );
                    })}
                  </div>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
