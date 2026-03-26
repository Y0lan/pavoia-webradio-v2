"use client";

import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { BRIDGE_URL, STAGES } from "@/lib/stages";

export default function HistoryPage() {
  const [page, setPage] = useState(1);
  const [stage, setStage] = useState("");
  const [search, setSearch] = useState("");

  const params = new URLSearchParams({ page: String(page), per_page: "20" });
  if (stage) params.set("stage", stage);
  if (search) params.set("search", search);

  const { data } = useQuery({
    queryKey: ["history", page, stage, search],
    queryFn: () => fetch(`${BRIDGE_URL}/api/history?${params}`).then((r) => r.json()),
  });

  const entries = data?.data ?? [];
  const total = data?.meta?.total ?? 0;
  const totalPages = Math.ceil(total / 20);

  return (
    <main className="p-6">
      <div className="flex items-center gap-2 mb-6">
        <span className="font-[family-name:var(--font-terminal)] text-[11px] tracking-[0.25em] uppercase"
          style={{ color: "var(--color-accent)" }}>// HISTORY</span>
        <div className="flex-1 h-px" style={{ background: "linear-gradient(90deg, var(--color-accent-dim), transparent)" }} />
      </div>

      {/* Filters */}
      <div className="flex gap-3 mb-6 flex-wrap">
        <select value={stage} onChange={(e) => { setStage(e.target.value); setPage(1); }}
          className="font-[family-name:var(--font-mono)] text-[13px] px-3 py-2"
          style={{ background: "var(--color-bg-elevated)", border: "1px solid var(--border-subtle)", color: "var(--color-text-primary)" }}>
          <option value="">All stages</option>
          {STAGES.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
        </select>
        <input type="text" placeholder="Search artist or title..." value={search}
          onChange={(e) => { setSearch(e.target.value); setPage(1); }}
          className="font-[family-name:var(--font-mono)] text-[13px] px-3 py-2 w-64"
          style={{ background: "var(--color-bg-elevated)", border: "1px solid var(--border-subtle)", color: "var(--color-text-primary)" }} />
      </div>

      {/* Table */}
      <div className="overflow-x-auto">
        <table className="w-full" style={{ borderCollapse: "collapse" }}>
          <thead>
            <tr>
              {["TIME", "ARTIST", "TITLE", "ALBUM", "STAGE"].map((h) => (
                <th key={h} className="text-left font-[family-name:var(--font-terminal)] text-[10px] tracking-[0.1em] uppercase px-3 py-2"
                  style={{ color: "var(--color-text-muted)", borderBottom: "1px solid var(--border-default)" }}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {entries.length === 0 ? (
              <tr><td colSpan={5} className="text-center py-12 font-[family-name:var(--font-terminal)] text-[11px] uppercase tracking-[0.25em]"
                style={{ color: "var(--color-text-muted)" }}>// NO PLAYS YET</td></tr>
            ) : entries.map((e: any) => (
              <tr key={e.id} className="transition-colors" style={{ cursor: "default" }}
                onMouseEnter={(ev) => (ev.currentTarget.style.background = "var(--color-bg-card)")}
                onMouseLeave={(ev) => (ev.currentTarget.style.background = "transparent")}>
                <td className="px-3 py-2.5 font-[family-name:var(--font-terminal)] text-[11px]"
                  style={{ color: "var(--color-text-muted)", borderBottom: "1px solid var(--border-subtle)" }}>
                  {new Date(e.played_at).toLocaleString()}
                </td>
                <td className="px-3 py-2.5 font-[family-name:var(--font-mono)] text-[12px]"
                  style={{ color: "var(--color-text-secondary)", borderBottom: "1px solid var(--border-subtle)" }}>{e.artist}</td>
                <td className="px-3 py-2.5 font-[family-name:var(--font-mono)] text-[12px]"
                  style={{ color: "var(--color-text-primary)", borderBottom: "1px solid var(--border-subtle)" }}>{e.title}</td>
                <td className="px-3 py-2.5 font-[family-name:var(--font-mono)] text-[12px]"
                  style={{ color: "var(--color-text-muted)", borderBottom: "1px solid var(--border-subtle)" }}>{e.album}</td>
                <td className="px-3 py-2.5 font-[family-name:var(--font-terminal)] text-[10px] uppercase"
                  style={{ color: "var(--color-text-muted)", borderBottom: "1px solid var(--border-subtle)" }}>{e.stage_id}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-4 mt-6">
          <button onClick={() => setPage(Math.max(1, page - 1))} disabled={page === 1}
            className="font-[family-name:var(--font-terminal)] text-[11px] px-3 py-1"
            style={{ color: page === 1 ? "var(--color-text-muted)" : "var(--color-accent)", border: "1px solid var(--border-subtle)" }}>
            PREV
          </button>
          <span className="font-[family-name:var(--font-terminal)] text-[11px]"
            style={{ color: "var(--color-text-muted)" }}>
            PAGE {page} / {totalPages}
          </span>
          <button onClick={() => setPage(Math.min(totalPages, page + 1))} disabled={page >= totalPages}
            className="font-[family-name:var(--font-terminal)] text-[11px] px-3 py-1"
            style={{ color: page >= totalPages ? "var(--color-text-muted)" : "var(--color-accent)", border: "1px solid var(--border-subtle)" }}>
            NEXT
          </button>
        </div>
      )}
    </main>
  );
}
