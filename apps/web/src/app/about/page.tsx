export default function AboutPage() {
  return (
    <div className="relative min-h-screen overflow-hidden">
      {/* Grid background */}
      <div className="absolute inset-0 opacity-30" style={{
        backgroundImage: "linear-gradient(var(--border-subtle) 1px, transparent 1px), linear-gradient(90deg, var(--border-subtle) 1px, transparent 1px)",
        backgroundSize: "60px 60px",
        maskImage: "radial-gradient(ellipse at center, rgba(0,0,0,0.4) 0%, transparent 70%)",
        WebkitMaskImage: "radial-gradient(ellipse at center, rgba(0,0,0,0.4) 0%, transparent 70%)",
      }} />

      <div className="relative z-10 flex flex-col items-center pt-20 px-6">
        {/* Section label */}
        <div className="flex items-center gap-2 mb-12 w-full max-w-2xl">
          <span className="font-[family-name:var(--font-terminal)] text-[11px] tracking-[0.25em] uppercase"
            style={{ color: "var(--color-accent)" }}>// ABOUT</span>
          <div className="flex-1 h-px" style={{ background: "linear-gradient(90deg, var(--color-accent-dim), transparent)" }} />
        </div>

        {/* Title */}
        <h1 className="font-[family-name:var(--font-display)] text-[64px] font-extrabold tracking-[0.4em] mb-4"
          style={{ color: "var(--color-text-primary)" }}>
          GAENDE
        </h1>

        {/* Anti-Algorithm badge */}
        <div className="clip-skew px-4 py-1.5 mb-12" style={{
          background: "rgba(255, 34, 68, 0.15)",
          border: "1px solid rgba(255, 34, 68, 0.3)",
        }}>
          <span className="font-[family-name:var(--font-terminal)] text-[11px] font-bold tracking-[0.1em] uppercase"
            style={{ color: "var(--color-error)" }}>
            Anti-Algorithm Radio
          </span>
        </div>

        {/* Bio */}
        <div className="max-w-2xl">
          <p className="font-[family-name:var(--font-editorial)] text-[17px] leading-[1.85]"
            style={{ color: "var(--color-text-secondary)" }}>
            GAENDE is a DJ, producer, and software engineer running a 9-stage 24/7 webradio
            from a seedbox. The curation process IS the content. Every track, every stat,
            every digging session — radically transparent. No algorithms, no curtain.
          </p>
          <p className="font-[family-name:var(--font-editorial)] text-[17px] leading-[1.85] mt-6"
            style={{ color: "var(--color-text-secondary)" }}>
            Nine themed stages run around the clock — from progressive melodic techno to
            ambient, DnB to Afro house. Each playlist is hand-curated from a personal Plex
            library of thousands of tracks, synced and enriched through Last.fm and
            MusicBrainz. What you see here is the full depth of that collection.
          </p>
        </div>

        {/* Cities */}
        <div className="mt-20 font-[family-name:var(--font-terminal)] text-[11px] tracking-[0.15em]"
          style={{ color: "var(--color-text-muted)" }}>
          PARIS ◆ BERLIN ◆ TBILISI
        </div>
      </div>
    </div>
  );
}
