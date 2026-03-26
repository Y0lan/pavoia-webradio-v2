"use client";

import { useQuery } from "@tanstack/react-query";
import { useWSStore } from "@/lib/ws";
import { BRIDGE_URL, STAGES } from "@/lib/stages";

export default function DashboardPage() {
  const { data } = useQuery({
    queryKey: ["stats-overview"],
    queryFn: () => fetch(`${BRIDGE_URL}/api/stats/overview`).then((r) => r.json()),
  });
  const stages = useWSStore((s) => s.stages);

  const cards = [
    { label: "TOTAL TRACKS", value: data?.total_tracks ?? "—", delta: data?.week_added ? `▲ ${data.week_added} this week` : null },
    { label: "ARTISTS", value: data?.total_artists ?? "—", delta: null },
    { label: "TOTAL PLAYS", value: data?.total_plays ?? "—", delta: data?.week_plays ? `▲ ${data.week_plays} this week` : null },
    { label: "HOURS STREAMED", value: data?.total_hours ? Math.round(data.total_hours) : "—", delta: null },
  ];

  return (
    <main className="p-6">
      {/* Section: Quick Pulse */}
      <div className="flex items-center gap-2 mb-6">
        <span className="font-[family-name:var(--font-terminal)] text-[11px] tracking-[0.25em] uppercase"
          style={{ color: "var(--color-accent)" }}>// QUICK PULSE</span>
        <div className="flex-1 h-px" style={{ background: "linear-gradient(90deg, var(--color-accent-dim), transparent)" }} />
      </div>

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-10">
        {cards.map((card) => (
          <div key={card.label} className="clip-card p-5"
            style={{ background: "var(--color-bg-card)", border: "1px solid var(--border-subtle)" }}>
            <div className="font-[family-name:var(--font-display)] text-[32px] font-extrabold"
              style={{ color: "var(--color-text-primary)" }}>
              {typeof card.value === "number" ? card.value.toLocaleString() : card.value}
            </div>
            <div className="font-[family-name:var(--font-terminal)] text-[10px] tracking-[0.1em] uppercase"
              style={{ color: "var(--color-text-muted)" }}>
              {card.label}
            </div>
            {card.delta && (
              <div className="font-[family-name:var(--font-terminal)] text-[11px] mt-1"
                style={{ color: "var(--color-accent)" }}>
                {card.delta}
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Section: What's Playing Now */}
      <div className="flex items-center gap-2 mb-6">
        <span className="font-[family-name:var(--font-terminal)] text-[11px] tracking-[0.25em] uppercase"
          style={{ color: "var(--color-accent)" }}>// WHAT&apos;S PLAYING NOW</span>
        <div className="flex-1 h-px" style={{ background: "linear-gradient(90deg, var(--color-accent-dim), transparent)" }} />
      </div>

      <div className="flex flex-col gap-2">
        {STAGES.map((s) => {
          const ws = stages.get(s.id);
          const np = ws?.nowPlaying;
          return (
            <div key={s.id} className="flex items-center gap-3 py-2 px-3"
              style={{ borderLeft: `2px solid ${s.color}`, background: "var(--color-bg-card)" }}>
              <div className="w-2 h-2 shrink-0" style={{
                background: np ? "var(--color-live)" : "var(--color-text-muted)",
                boxShadow: np ? "0 0 6px var(--color-live)" : "none",
              }} />
              <span className="font-[family-name:var(--font-terminal)] text-[10px] tracking-[0.08em] uppercase w-28 shrink-0"
                style={{ color: s.color }}>{s.name}</span>
              {np ? (
                <span className="font-[family-name:var(--font-mono)] text-[12px] truncate"
                  style={{ color: "var(--color-text-secondary)" }}>
                  {np.artist} — {np.title}
                </span>
              ) : (
                <span className="font-[family-name:var(--font-terminal)] text-[10px] uppercase"
                  style={{ color: "var(--color-text-muted)" }}>// WAITING</span>
              )}
            </div>
          );
        })}
      </div>
    </main>
  );
}
