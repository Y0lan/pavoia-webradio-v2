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

        {/* Pavoia logo */}
        <img
          src="https://pavoia.com/wp-content/uploads/2022/10/PavoiaKV_Homebutton_300px.gif"
          alt="PAVOIA"
          className="w-[120px] md:w-[260px] h-auto mb-4"
        />

        {/* Title */}
        <h1 className="font-[family-name:var(--font-display)] text-[32px] md:text-[48px] font-extrabold tracking-[0.1em] mb-2"
          style={{ color: "var(--color-text-primary)" }}>
          Pavoia Webradio
        </h1>

        {/* Subtitle */}
        <div className="font-[family-name:var(--font-terminal)] text-[11px] tracking-[0.15em] uppercase mb-12"
          style={{ color: "var(--color-text-muted)" }}>
          9 stages · 24/7 · high quality audio
        </div>

        {/* Bio — real text from v1 */}
        <div className="max-w-2xl">
          <p className="font-[family-name:var(--font-editorial)] text-[17px] leading-[1.85]"
            style={{ color: "var(--color-text-secondary)" }}>
            This collection started years ago, built from artists heard at Pavoia and
            countless hours of digging for new sounds. Every time I fall for a track, I
            picture which stage it belongs to. What began as a personal obsession slowly
            grew into something worth sharing. Today, these nine streams are open to
            everyone, and the playlists keep growing with fresh discoveries.
          </p>
          <p className="font-[family-name:var(--font-editorial)] text-[17px] leading-[1.85] mt-6"
            style={{ color: "var(--color-text-secondary)" }}>
            All streams are in <strong>high quality audio</strong>, pure and unprocessed,
            just as the artists intended.
          </p>
        </div>

        {/* Links */}
        <div className="flex gap-3 mt-10 flex-wrap justify-center">
          <a href="https://instagram.com/wearepavoia" target="_blank" rel="noopener noreferrer"
            className="font-[family-name:var(--font-mono)] text-[12px] px-4 py-2 no-underline transition-all"
            style={{ color: "var(--color-accent)", border: "1px solid var(--border-default)", background: "var(--color-bg-card)" }}>
            Instagram <span style={{ color: "var(--color-text-muted)" }}>@wearepavoia</span>
          </a>
          <a href="https://soundcloud.com/pavoia" target="_blank" rel="noopener noreferrer"
            className="font-[family-name:var(--font-mono)] text-[12px] px-4 py-2 no-underline transition-all"
            style={{ color: "var(--color-accent)", border: "1px solid var(--border-default)", background: "var(--color-bg-card)" }}>
            SoundCloud <span style={{ color: "var(--color-text-muted)" }}>@pavoia</span>
          </a>
        </div>

        {/* Footer */}
        <div className="mt-16 font-[family-name:var(--font-terminal)] text-[11px]"
          style={{ color: "var(--color-text-ghost)" }}>
          made with ♥ by <a href="https://instagram.com/gaende_music" target="_blank" rel="noopener noreferrer"
            style={{ color: "var(--color-text-muted)" }}>gaende</a>
        </div>
      </div>
    </div>
  );
}
