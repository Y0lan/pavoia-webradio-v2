"use client";

import { useQuery } from "@tanstack/react-query";
import { BRIDGE_URL, STAGES } from "@/lib/stages";

function SectionLabel({ label }: { label: string }) {
  return (
    <div className="flex items-center gap-2 mb-6 mt-10 first:mt-0">
      <span className="font-[family-name:var(--font-terminal)] text-[11px] tracking-[0.25em] uppercase"
        style={{ color: "var(--color-accent)" }}>{label}</span>
      <div className="flex-1 h-px" style={{ background: "linear-gradient(90deg, var(--color-accent-dim), transparent)" }} />
    </div>
  );
}

export default function StatsPage() {
  const { data: topArtists } = useQuery({
    queryKey: ["stats-top-artists"],
    queryFn: () => fetch(`${BRIDGE_URL}/api/stats/top-artists?limit=10&by=tracks`).then((r) => r.json()),
  });
  const { data: topTracks } = useQuery({
    queryKey: ["stats-top-tracks"],
    queryFn: () => fetch(`${BRIDGE_URL}/api/stats/top-tracks?limit=10`).then((r) => r.json()),
  });
  const { data: genres } = useQuery({
    queryKey: ["stats-genres"],
    queryFn: () => fetch(`${BRIDGE_URL}/api/stats/genres`).then((r) => r.json()),
  });
  const { data: bpm } = useQuery({
    queryKey: ["stats-bpm"],
    queryFn: () => fetch(`${BRIDGE_URL}/api/stats/bpm`).then((r) => r.json()),
  });
  const { data: keys } = useQuery({
    queryKey: ["stats-keys"],
    queryFn: () => fetch(`${BRIDGE_URL}/api/stats/keys`).then((r) => r.json()),
  });
  const { data: decades } = useQuery({
    queryKey: ["stats-decades"],
    queryFn: () => fetch(`${BRIDGE_URL}/api/stats/decades`).then((r) => r.json()),
  });
  const { data: stageStats } = useQuery({
    queryKey: ["stats-stages"],
    queryFn: () => fetch(`${BRIDGE_URL}/api/stats/stages`).then((r) => r.json()),
  });
  const { data: velocity } = useQuery({
    queryKey: ["stats-velocity"],
    queryFn: () => fetch(`${BRIDGE_URL}/api/stats/discovery-velocity`).then((r) => r.json()),
  });

  const maxBpm = bpm ? Math.max(...bpm.map((b: any) => b.count)) : 1;
  const maxDecade = decades ? Math.max(...decades.map((d: any) => d.count)) : 1;
  const maxStage = stageStats ? Math.max(...stageStats.map((s: any) => s.plays)) : 1;
  const maxVelocity = velocity ? Math.max(...velocity.map((v: any) => v.count)) : 1;
  const maxGenre = genres ? Math.max(...genres.map((g: any) => g.count)) : 1;

  return (
    <main className="p-6">
      {/* ═══ THE COLLECTION ═══ */}
      <SectionLabel label="// THE COLLECTION" />

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Top Artists */}
        <div className="clip-card p-5" style={{ background: "var(--color-bg-card)", border: "1px solid var(--border-subtle)" }}>
          <h3 className="font-[family-name:var(--font-terminal)] text-[10px] tracking-[0.1em] uppercase mb-4"
            style={{ color: "var(--color-text-muted)" }}>TOP ARTISTS</h3>
          {(topArtists ?? []).map((a: any, i: number) => (
            <div key={a.artist} className="flex items-center gap-3 py-1.5">
              <span className="font-[family-name:var(--font-display)] text-[14px] font-bold w-6 text-right"
                style={{ color: "var(--color-text-muted)" }}>{i + 1}</span>
              <span className="font-[family-name:var(--font-mono)] text-[12px] flex-1 truncate"
                style={{ color: "var(--color-text-primary)" }}>{a.artist}</span>
              <span className="font-[family-name:var(--font-terminal)] text-[10px]"
                style={{ color: "var(--color-accent)" }}>{a.count}</span>
            </div>
          ))}
        </div>

        {/* Top Tracks */}
        <div className="clip-card p-5" style={{ background: "var(--color-bg-card)", border: "1px solid var(--border-subtle)" }}>
          <h3 className="font-[family-name:var(--font-terminal)] text-[10px] tracking-[0.1em] uppercase mb-4"
            style={{ color: "var(--color-text-muted)" }}>TOP TRACKS</h3>
          {(topTracks ?? []).map((t: any, i: number) => (
            <div key={`${t.title}-${t.artist}`} className="flex items-center gap-3 py-1.5">
              <span className="font-[family-name:var(--font-display)] text-[14px] font-bold w-6 text-right"
                style={{ color: "var(--color-text-muted)" }}>{i + 1}</span>
              <div className="flex-1 min-w-0">
                <div className="font-[family-name:var(--font-mono)] text-[12px] truncate" style={{ color: "var(--color-text-primary)" }}>{t.title}</div>
                <div className="font-[family-name:var(--font-mono)] text-[11px] truncate" style={{ color: "var(--color-text-secondary)" }}>{t.artist}</div>
              </div>
              <span className="font-[family-name:var(--font-terminal)] text-[10px]" style={{ color: "var(--color-accent)" }}>{t.plays}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Genre Tag Cloud */}
      <div className="clip-card p-5 mt-4" style={{ background: "var(--color-bg-card)", border: "1px solid var(--border-subtle)" }}>
        <h3 className="font-[family-name:var(--font-terminal)] text-[10px] tracking-[0.1em] uppercase mb-4"
          style={{ color: "var(--color-text-muted)" }}>GENRES</h3>
        <div className="flex flex-wrap gap-2">
          {(genres ?? []).slice(0, 30).map((g: any) => {
            const size = 10 + (g.count / maxGenre) * 8;
            return (
              <span key={g.genre} className="font-[family-name:var(--font-terminal)] uppercase tracking-[0.05em] px-2 py-0.5"
                style={{ fontSize: `${size}px`, color: "var(--color-accent)", opacity: 0.4 + (g.count / maxGenre) * 0.6, border: "1px solid var(--border-subtle)" }}>
                {g.genre}
              </span>
            );
          })}
        </div>
      </div>

      {/* ═══ THE TASTE ═══ */}
      <SectionLabel label="// THE TASTE" />

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* BPM Distribution */}
        <div className="clip-card p-5" style={{ background: "var(--color-bg-card)", border: "1px solid var(--border-subtle)" }}>
          <h3 className="font-[family-name:var(--font-terminal)] text-[10px] tracking-[0.1em] uppercase mb-4"
            style={{ color: "var(--color-text-muted)" }}>BPM DISTRIBUTION</h3>
          <div className="flex items-end gap-px h-[120px]">
            {(bpm ?? []).map((b: any) => (
              <div key={b.bpm} className="flex-1 min-w-[2px]" title={`${b.bpm} BPM: ${b.count}`}
                style={{ height: `${(b.count / maxBpm) * 100}%`, background: "var(--color-accent)", opacity: 0.7 }} />
            ))}
          </div>
        </div>

        {/* Decades */}
        <div className="clip-card p-5" style={{ background: "var(--color-bg-card)", border: "1px solid var(--border-subtle)" }}>
          <h3 className="font-[family-name:var(--font-terminal)] text-[10px] tracking-[0.1em] uppercase mb-4"
            style={{ color: "var(--color-text-muted)" }}>DECADES</h3>
          {(decades ?? []).map((d: any) => (
            <div key={d.decade} className="flex items-center gap-3 py-1">
              <span className="font-[family-name:var(--font-terminal)] text-[11px] w-10" style={{ color: "var(--color-text-muted)" }}>{d.decade}s</span>
              <div className="flex-1 h-3" style={{ background: "var(--color-bg-elevated)" }}>
                <div className="h-full" style={{ width: `${(d.count / maxDecade) * 100}%`, background: "var(--color-accent)" }} />
              </div>
              <span className="font-[family-name:var(--font-terminal)] text-[10px] w-8 text-right" style={{ color: "var(--color-text-muted)" }}>{d.count}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Key Distribution */}
      <div className="clip-card p-5 mt-4" style={{ background: "var(--color-bg-card)", border: "1px solid var(--border-subtle)" }}>
        <h3 className="font-[family-name:var(--font-terminal)] text-[10px] tracking-[0.1em] uppercase mb-4"
          style={{ color: "var(--color-text-muted)" }}>CAMELOT KEY WHEEL</h3>
        <div className="flex flex-wrap gap-2">
          {(keys ?? []).map((k: any) => (
            <div key={k.key} className="text-center px-3 py-2"
              style={{ background: "var(--color-bg-elevated)", border: "1px solid var(--border-subtle)" }}>
              <div className="font-[family-name:var(--font-display)] text-[16px] font-bold" style={{ color: "var(--color-accent)" }}>{k.key}</div>
              <div className="font-[family-name:var(--font-terminal)] text-[9px]" style={{ color: "var(--color-text-muted)" }}>{k.count}</div>
            </div>
          ))}
        </div>
      </div>

      {/* ═══ THE BEHAVIOR ═══ */}
      <SectionLabel label="// THE BEHAVIOR" />

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Stage Distribution */}
        <div className="clip-card p-5" style={{ background: "var(--color-bg-card)", border: "1px solid var(--border-subtle)" }}>
          <h3 className="font-[family-name:var(--font-terminal)] text-[10px] tracking-[0.1em] uppercase mb-4"
            style={{ color: "var(--color-text-muted)" }}>STAGE DISTRIBUTION</h3>
          {(stageStats ?? []).map((s: any) => {
            const stg = STAGES.find((st) => st.id === s.stage_id);
            return (
              <div key={s.stage_id} className="flex items-center gap-3 py-1">
                <span className="font-[family-name:var(--font-terminal)] text-[10px] w-24 truncate"
                  style={{ color: stg?.color || "var(--color-text-muted)" }}>{stg?.name || s.stage_id}</span>
                <div className="flex-1 h-3" style={{ background: "var(--color-bg-elevated)" }}>
                  <div className="h-full" style={{ width: `${(s.plays / maxStage) * 100}%`, background: stg?.color || "var(--color-accent)" }} />
                </div>
                <span className="font-[family-name:var(--font-terminal)] text-[10px] w-10 text-right"
                  style={{ color: "var(--color-text-muted)" }}>{s.plays}</span>
              </div>
            );
          })}
        </div>

        {/* Discovery Velocity */}
        <div className="clip-card p-5" style={{ background: "var(--color-bg-card)", border: "1px solid var(--border-subtle)" }}>
          <h3 className="font-[family-name:var(--font-terminal)] text-[10px] tracking-[0.1em] uppercase mb-4"
            style={{ color: "var(--color-text-muted)" }}>DISCOVERY VELOCITY</h3>
          <div className="flex items-end gap-px h-[80px]">
            {(velocity ?? []).map((v: any, i: number) => (
              <div key={i} className="flex-1 min-w-[2px]" title={`${v.week}: ${v.count} tracks`}
                style={{ height: `${(v.count / maxVelocity) * 100}%`, background: "var(--color-accent)", opacity: 0.6 }} />
            ))}
          </div>
          <div className="font-[family-name:var(--font-terminal)] text-[9px] mt-2"
            style={{ color: "var(--color-text-ghost)" }}>
            TRACKS ADDED PER WEEK (52 WEEKS)
          </div>
        </div>
      </div>
    </main>
  );
}
