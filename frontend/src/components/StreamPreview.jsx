import React from "react";

export function StreamPreview({ streamId, metadata, position, onClose }) {
  return (
    <div
      className="absolute z-50 animate-fade-in"
      style={{ top: position?.top, left: position?.left }}
    >
      <div
        className="flex items-center gap-3 px-4 py-3 rounded-xl border border-white/15
                   bg-black/70 backdrop-blur-xl shadow-lg min-w-[200px] max-w-[280px]"
        onMouseLeave={onClose}
      >
        {metadata ? (
          <>
            {metadata.cover_url ? (
              <img
                src={metadata.cover_url}
                alt=""
                className="w-10 h-10 rounded-lg object-cover flex-shrink-0"
              />
            ) : (
              <div className="w-10 h-10 rounded-lg bg-white/10 flex-shrink-0" />
            )}
            <div className="min-w-0">
              <div className="text-sm font-medium text-white truncate">
                {metadata.title || "Unknown track"}
              </div>
              <div className="text-xs text-white/50 truncate">
                {metadata.artist || "Unknown artist"}
              </div>
            </div>
          </>
        ) : (
          <span className="text-xs text-white/40">No data available</span>
        )}
      </div>
    </div>
  );
}
