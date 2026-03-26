"use client";

import { useQuery } from "@tanstack/react-query";
import { useParams } from "next/navigation";
import Link from "next/link";
import { BRIDGE_URL } from "@/lib/stages";

export default function ArtistDetailPage() {
  const { id } = useParams<{ id: string }>();

  const { data: artist } = useQuery({
    queryKey: ["artist", id],
    queryFn: () => fetch(`${BRIDGE_URL}/api/artists/${id}`).then((r) => r.json()),
  });

  const { data: tracksData } = useQuery({
    queryKey: ["artist-tracks", id],
    queryFn: () => fetch(`${BRIDGE_URL}/api/artists/${id}/tracks?per_page=50`).then((r) => r.json()),
  });

  const { data: similar } = useQuery({
    queryKey: ["artist-similar", id],
    queryFn: () => fetch(`${BRIDGE_URL}/api/artists/${id}/similar`).then((r) => r.json()),
  });

  const tracks = tracksData?.data ?? [];
  const similarArtists = similar ?? [];

  if (!artist) {
    return (
      <div className="flex items-center justify-center min-h-[50vh]">
        <span className="font-[family-name:var(--font-terminal)] text-[11px] uppercase tracking-[0.25em]"
          style={{ color: "var(--color-text-muted)" }}>// LOADING...</span>
      </div>
    );
  }

  return (
    <main className="p-6">
      {/* Header */}
      <div className="mb-8">
        <h1 className="font-[family-name:var(--font-display)] text-[28px] font-bold"
          style={{ color: "var(--color-text-primary)" }}>{artist.name}</h1>
        <div className="flex gap-2 mt-2 flex-wrap">
          {artist.country && (
            <span className="font-[family-name:var(--font-terminal)] text-[10px] tracking-[0.08em] uppercase px-2.5 py-1"
              style={{ color: "var(--color-data)", background: "var(--color-data-dim)", border: "1px solid rgba(255,170,0,0.15)" }}>
              {artist.country}
            </span>
          )}
          {artist.tags?.map((tag: string) => (
            <span key={tag} className="font-[family-name:var(--font-terminal)] text-[10px] tracking-[0.08em] px-2.5 py-1"
              style={{ color: "var(--color-accent)", background: "var(--color-accent-glow)", border: "1px solid var(--color-accent-dim)" }}>
              {tag}
            </span>
          ))}
        </div>
        <div className="flex gap-6 mt-3">
          <span className="font-[family-name:var(--font-terminal)] text-[10px]" style={{ color: "var(--color-text-muted)" }}>
            {artist.track_count} TRACKS
          </span>
          <span className="font-[family-name:var(--font-terminal)] text-[10px]" style={{ color: "var(--color-text-muted)" }}>
            {artist.play_count} PLAYS
          </span>
        </div>
      </div>

      {/* Bio */}
      {artist.bio && (
        <div className="mb-8">
          <p className="font-[family-name:var(--font-editorial)] text-[17px] leading-[1.85] max-w-2xl"
            style={{ color: "var(--color-text-secondary)" }}
            dangerouslySetInnerHTML={{ __html: artist.bio.replace(/<a /g, '<a style="color:var(--color-accent)" ') }} />
        </div>
      )}

      {/* Tracks */}
      <div className="flex items-center gap-2 mb-4">
        <span className="font-[family-name:var(--font-terminal)] text-[11px] tracking-[0.25em] uppercase"
          style={{ color: "var(--color-accent)" }}>// TRACKS</span>
        <div className="flex-1 h-px" style={{ background: "linear-gradient(90deg, var(--color-accent-dim), transparent)" }} />
      </div>

      <div className="overflow-x-auto mb-10">
        <table className="w-full" style={{ borderCollapse: "collapse" }}>
          <thead>
            <tr>
              {["TITLE", "ALBUM", "STAGE", "GENRE", "ADDED"].map((h) => (
                <th key={h} className="text-left font-[family-name:var(--font-terminal)] text-[10px] tracking-[0.1em] uppercase px-3 py-2"
                  style={{ color: "var(--color-text-muted)", borderBottom: "1px solid var(--border-default)" }}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {tracks.map((t: any) => (
              <tr key={t.id}>
                <td className="px-3 py-2 font-[family-name:var(--font-mono)] text-[12px]"
                  style={{ color: "var(--color-text-primary)", borderBottom: "1px solid var(--border-subtle)" }}>{t.title}</td>
                <td className="px-3 py-2 font-[family-name:var(--font-mono)] text-[12px]"
                  style={{ color: "var(--color-text-muted)", borderBottom: "1px solid var(--border-subtle)" }}>{t.album}</td>
                <td className="px-3 py-2 font-[family-name:var(--font-terminal)] text-[10px] uppercase"
                  style={{ color: "var(--color-text-muted)", borderBottom: "1px solid var(--border-subtle)" }}>{t.stage_id}</td>
                <td className="px-3 py-2 font-[family-name:var(--font-terminal)] text-[10px]"
                  style={{ color: "var(--color-text-muted)", borderBottom: "1px solid var(--border-subtle)" }}>{t.genre}</td>
                <td className="px-3 py-2 font-[family-name:var(--font-terminal)] text-[10px]"
                  style={{ color: "var(--color-text-muted)", borderBottom: "1px solid var(--border-subtle)" }}>
                  {new Date(t.added_at).toLocaleDateString()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Similar */}
      {similarArtists.length > 0 && (
        <>
          <div className="flex items-center gap-2 mb-4">
            <span className="font-[family-name:var(--font-terminal)] text-[11px] tracking-[0.25em] uppercase"
              style={{ color: "var(--color-accent)" }}>// SIMILAR ARTISTS</span>
            <div className="flex-1 h-px" style={{ background: "linear-gradient(90deg, var(--color-accent-dim), transparent)" }} />
          </div>
          <div className="flex gap-3 flex-wrap">
            {similarArtists.map((s: any) => (
              <Link key={s.id} href={`/artists/${s.id}`}
                className="clip-card px-4 py-2 no-underline transition-all hover:-translate-y-px"
                style={{ background: "var(--color-bg-card)", border: "1px solid var(--border-subtle)" }}>
                <span className="font-[family-name:var(--font-mono)] text-[12px]" style={{ color: "var(--color-text-primary)" }}>
                  {s.name}
                </span>
                <span className="font-[family-name:var(--font-terminal)] text-[9px] ml-2"
                  style={{ color: "var(--color-text-muted)" }}>
                  {Math.round(s.weight * 100)}%
                </span>
              </Link>
            ))}
          </div>
        </>
      )}
    </main>
  );
}
