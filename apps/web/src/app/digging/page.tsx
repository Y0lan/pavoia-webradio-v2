"use client";

import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { BRIDGE_URL } from "@/lib/stages";

function cellColor(count: number): string {
  if (count === 0) return "var(--color-bg-elevated)";
  if (count <= 2) return "rgba(0, 255, 200, 0.2)";
  if (count <= 5) return "rgba(0, 255, 200, 0.4)";
  if (count <= 10) return "rgba(0, 255, 200, 0.7)";
  return "var(--color-accent)";
}

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
const DAYS = ["", "Mon", "", "Wed", "", "Fri", ""];

export default function DiggingPage() {
  const [year, setYear] = useState(new Date().getFullYear());

  const { data: calendar } = useQuery({
    queryKey: ["digging-calendar", year],
    queryFn: () => fetch(`${BRIDGE_URL}/api/digging/calendar?year=${year}`).then((r) => r.json()),
  });

  const { data: streaks } = useQuery({
    queryKey: ["digging-streaks"],
    queryFn: () => fetch(`${BRIDGE_URL}/api/digging/streaks`).then((r) => r.json()),
  });

  // Build 52x7 grid from calendar data
  const dayMap = new Map<string, number>();
  if (calendar?.days) {
    for (const d of calendar.days) {
      dayMap.set(d.date, d.count);
    }
  }

  // Generate weeks
  const startDate = new Date(year, 0, 1);
  const startDay = startDate.getDay(); // 0=Sun
  const weeks: { date: string; count: number }[][] = [];
  let currentWeek: { date: string; count: number }[] = [];

  // Pad first week
  for (let i = 0; i < startDay; i++) {
    currentWeek.push({ date: "", count: -1 });
  }

  for (let d = 0; d < 366; d++) {
    const date = new Date(year, 0, 1 + d);
    if (date.getFullYear() !== year) break;
    const dateStr = date.toISOString().split("T")[0];
    currentWeek.push({ date: dateStr, count: dayMap.get(dateStr) || 0 });
    if (currentWeek.length === 7) {
      weeks.push(currentWeek);
      currentWeek = [];
    }
  }
  if (currentWeek.length > 0) {
    while (currentWeek.length < 7) currentWeek.push({ date: "", count: -1 });
    weeks.push(currentWeek);
  }

  return (
    <main className="p-6">
      {/* Header */}
      <div className="flex items-center gap-2 mb-6">
        <span className="font-[family-name:var(--font-terminal)] text-[11px] tracking-[0.25em] uppercase"
          style={{ color: "var(--color-accent)" }}>// DIGGING CALENDAR</span>
        <div className="flex-1 h-px" style={{ background: "linear-gradient(90deg, var(--color-accent-dim), transparent)" }} />
      </div>

      {/* Year selector */}
      <div className="flex gap-2 mb-6">
        {[year - 1, year, year + 1].map((y) => (
          <button key={y} onClick={() => setYear(y)}
            className="font-[family-name:var(--font-terminal)] text-[11px] px-3 py-1"
            style={{
              color: y === year ? "var(--color-accent)" : "var(--color-text-muted)",
              background: y === year ? "var(--color-accent-glow)" : "transparent",
              border: `1px solid ${y === year ? "var(--color-accent-dim)" : "var(--border-subtle)"}`,
            }}>
            {y}
          </button>
        ))}
      </div>

      {/* Calendar grid */}
      <div className="overflow-x-auto mb-10">
        {/* Month labels */}
        <div className="flex gap-0 mb-1 ml-8">
          {MONTHS.map((m, i) => (
            <div key={m} className="font-[family-name:var(--font-terminal)] text-[9px]"
              style={{ color: "var(--color-text-muted)", width: `${(weeks.length / 12) * 14}px` }}>
              {m}
            </div>
          ))}
        </div>

        <div className="flex gap-0">
          {/* Day labels */}
          <div className="flex flex-col gap-[2px] mr-1">
            {DAYS.map((d, i) => (
              <div key={i} className="h-[12px] flex items-center font-[family-name:var(--font-terminal)] text-[9px]"
                style={{ color: "var(--color-text-muted)", width: "28px" }}>
                {d}
              </div>
            ))}
          </div>

          {/* Weeks */}
          <div className="flex gap-[2px]">
            {weeks.map((week, wi) => (
              <div key={wi} className="flex flex-col gap-[2px]">
                {week.map((day, di) => (
                  <div key={`${wi}-${di}`}
                    className="w-[12px] h-[12px] transition-all"
                    title={day.date ? `${day.date}: ${day.count} tracks` : ""}
                    style={{
                      background: day.count < 0 ? "transparent" : cellColor(day.count),
                      outline: "none",
                    }}
                    onMouseEnter={(e) => {
                      if (day.count >= 0) (e.target as HTMLElement).style.outline = "1px solid var(--color-accent)";
                    }}
                    onMouseLeave={(e) => {
                      (e.target as HTMLElement).style.outline = "none";
                    }}
                  />
                ))}
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Streaks */}
      <div className="flex items-center gap-2 mb-6">
        <span className="font-[family-name:var(--font-terminal)] text-[11px] tracking-[0.25em] uppercase"
          style={{ color: "var(--color-accent)" }}>// STREAKS</span>
        <div className="flex-1 h-px" style={{ background: "linear-gradient(90deg, var(--color-accent-dim), transparent)" }} />
      </div>

      <div className="grid grid-cols-3 gap-4">
        {[
          { label: "CURRENT STREAK", value: streaks?.current ?? "—", unit: "days" },
          { label: "LONGEST STREAK", value: streaks?.longest ?? "—", unit: "days" },
          { label: "BEST WEEK", value: streaks?.best_week ?? "—", unit: "tracks" },
        ].map((s) => (
          <div key={s.label} className="clip-card p-5"
            style={{ background: "var(--color-bg-card)", border: "1px solid var(--border-subtle)" }}>
            <div className="font-[family-name:var(--font-display)] text-[32px] font-extrabold"
              style={{ color: "var(--color-text-primary)" }}>
              {s.value}
            </div>
            <div className="font-[family-name:var(--font-terminal)] text-[10px] tracking-[0.1em] uppercase"
              style={{ color: "var(--color-text-muted)" }}>{s.label}</div>
            <div className="font-[family-name:var(--font-terminal)] text-[9px]"
              style={{ color: "var(--color-text-ghost)" }}>{s.unit}</div>
          </div>
        ))}
      </div>
    </main>
  );
}
