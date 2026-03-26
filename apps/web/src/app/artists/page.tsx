"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useState } from "react";
import { BRIDGE_URL } from "@/lib/stages";

export default function ArtistsPage() {
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState("tracks");

  const params = new URLSearchParams({ page: String(page), per_page: "24", sort });
  if (search) params.set("search", search);

  const { data } = useQuery({
    queryKey: ["artists", page, search, sort],
    queryFn: () => fetch(`${BRIDGE_URL}/api/artists?${params}`).then((r) => r.json()),
  });

  const artists = data?.data ?? [];
  const total = data?.meta?.total ?? 0;
  const totalPages = Math.ceil(total / 24);

  return (
    <main className="p-6">
      <div className="flex items-center gap-2 mb-6">
        <span className="font-[family-name:var(--font-terminal)] text-[11px] tracking-[0.25em] uppercase"
          style={{ color: "var(--color-accent)" }}>// ARTISTS</span>
        <div className="flex-1 h-px" style={{ background: "linear-gradient(90deg, var(--color-accent-dim), transparent)" }} />
      </div>

      {/* Controls */}
      <div className="flex gap-3 mb-6 flex-wrap items-center">
        <input type="text" placeholder="Search artists..." value={search}
          onChange={(e) => { setSearch(e.target.value); setPage(1); }}
          className="font-[family-name:var(--font-mono)] text-[13px] px-3 py-2 w-64"
          style={{ background: "var(--color-bg-elevated)", border: "1px solid var(--border-subtle)", color: "var(--color-text-primary)" }} />
        <div className="flex gap-1">
          {(["tracks", "name"] as const).map((s) => (
            <button key={s} onClick={() => { setSort(s); setPage(1); }}
              className="font-[family-name:var(--font-terminal)] text-[10px] tracking-[0.08em] uppercase px-3 py-1.5"
              style={{
                color: sort === s ? "var(--color-accent)" : "var(--color-text-muted)",
                background: sort === s ? "var(--color-accent-glow)" : "transparent",
                border: `1px solid ${sort === s ? "var(--color-accent-dim)" : "var(--border-subtle)"}`,
              }}>{s}</button>
          ))}
        </div>
      </div>

      {/* Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 mb-6">
        {artists.map((a: any) => (
          <Link key={a.id} href={`/artists/${a.id}`}
            className="clip-card p-5 no-underline transition-all hover:-translate-y-px"
            style={{ background: "var(--color-bg-card)", border: "1px solid var(--border-subtle)" }}>
            <div className="flex items-start justify-between mb-2">
              <span className="font-[family-name:var(--font-mono)] text-[14px] font-medium"
                style={{ color: "var(--color-text-primary)" }}>{a.name}</span>
              <span className="font-[family-name:var(--font-display)] text-[24px] font-extrabold"
                style={{ color: "var(--color-text-primary)" }}>{a.track_count}</span>
            </div>
            {a.country && (
              <span className="font-[family-name:var(--font-terminal)] text-[10px] tracking-[0.08em] uppercase px-2 py-0.5 mr-1"
                style={{ color: "var(--color-data)", background: "var(--color-data-dim)", border: "1px solid rgba(255,170,0,0.15)" }}>
                {a.country}
              </span>
            )}
            {a.tags?.slice(0, 3).map((tag: string) => (
              <span key={tag} className="font-[family-name:var(--font-terminal)] text-[9px] tracking-[0.05em] px-1.5 py-0.5 mr-1"
                style={{ color: "var(--color-text-muted)", border: "1px solid var(--border-subtle)" }}>{tag}</span>
            ))}
          </Link>
        ))}
      </div>

      {artists.length === 0 && (
        <div className="text-center py-12 font-[family-name:var(--font-terminal)] text-[11px] uppercase tracking-[0.25em]"
          style={{ color: "var(--color-text-muted)" }}>// NO ARTISTS FOUND</div>
      )}

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-4">
          <button onClick={() => setPage(Math.max(1, page - 1))} disabled={page === 1}
            className="font-[family-name:var(--font-terminal)] text-[11px] px-3 py-1"
            style={{ color: page === 1 ? "var(--color-text-muted)" : "var(--color-accent)", border: "1px solid var(--border-subtle)" }}>PREV</button>
          <span className="font-[family-name:var(--font-terminal)] text-[11px]" style={{ color: "var(--color-text-muted)" }}>
            PAGE {page} / {totalPages}
          </span>
          <button onClick={() => setPage(Math.min(totalPages, page + 1))} disabled={page >= totalPages}
            className="font-[family-name:var(--font-terminal)] text-[11px] px-3 py-1"
            style={{ color: page >= totalPages ? "var(--color-text-muted)" : "var(--color-accent)", border: "1px solid var(--border-subtle)" }}>NEXT</button>
        </div>
      )}
    </main>
  );
}
