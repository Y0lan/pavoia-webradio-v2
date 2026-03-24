import React from "react";
import { fmtTime } from "../../utils/formatters";

export function TrackProgress({ elapsed, duration }) {
  const hasDuration = Number.isFinite(duration) && duration > 0;
  const pct = hasDuration ? Math.min(Math.max(elapsed / duration, 0), 1) : null;
  return (
    <div className="mt-3">
      {hasDuration && (
        <div className="h-2.5 rounded-full bg-[#141a24] border border-[#1f2533] overflow-hidden" role="progressbar" aria-valuemin={0} aria-valuemax={100} aria-valuenow={Math.round((pct || 0) * 100)} aria-label="Track progress">
          <div className="h-full bg-gradient-to-r from-cyan-300 via-fuchsia-300 to-emerald-300" style={{ width: `${((pct || 0) * 100).toFixed(2)}%` }} />
        </div>
      )}
      <div className="mt-2 text-sm text-slate-300 font-mono tabular-nums">
        {fmtTime(elapsed)}{hasDuration ? ` / ${fmtTime(duration)}` : ""}
      </div>
    </div>
  );
}
