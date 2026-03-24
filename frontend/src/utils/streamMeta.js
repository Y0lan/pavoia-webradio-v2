

export function getMetaFor(s) {
  const id = (s?.id || "").toLowerCase();
  const name = (s?.name || "").toLowerCase();
  const fallback = { icon: "📻", title: s?.name || "", desc: "", disabled: false, gradientFrom: "#0b0d10", gradientVia: "#111520", gradientTo: "#0b0d10", accentColor: "#64748b" };

  if (id.includes("gaende")) {
    return { icon: "💜", title: "gaende's favorites", desc: "Personal selection of tracks that hit different. Every genre, every mood - find them across all stages!", disabled: false, gradientFrom: "#2d1b4e", gradientVia: "#1a0f30", gradientTo: "#0d0618", accentColor: "#a78bfa" };
  }
  if (id.includes("ambiance")) {
    return { icon: "🛋️", title: "Ambiance / Safe Space", desc: "The perfect comedown. Chill on the sofa with relaxing beats. Good vibes only in this safe space.", disabled: false, gradientFrom: "#3d1f0a", gradientVia: "#2a1508", gradientTo: "#1a0c04", accentColor: "#f59e0b" };
  }
  if (id.includes("bermuda") && (id.includes("day") || id.includes("oaza"))) {
    return { icon: "🌴", title: "Bermuda Before 18:00 / Oaza", desc: "Sunlit grooves • chat, move, breathe.", disabled: false, gradientFrom: "#0a3d3d", gradientVia: "#062a2a", gradientTo: "#041a1a", accentColor: "#2dd4bf" };
  }
  if (id.includes("bermuda") && id.includes("night")) {
    return { icon: "🌅", title: "Bermuda (18:00–00:00)", desc: "Progressive & Indie • sunset lift, growing tension.", disabled: false, gradientFrom: "#1a1040", gradientVia: "#2d1520", gradientTo: "#0d0618", accentColor: "#f97316" };
  }
  if (id.includes("palac") && (id.includes("hypno") || id.includes("slow"))) {
    return { icon: "🌙", title: "Palac Feel", desc: "Melodic motion • hypnotic & tender, harsh and heartbroken. Dance your emotions away all night long.", disabled: false, gradientFrom: "#141230", gradientVia: "#0d0b20", gradientTo: "#080714", accentColor: "#94a3b8" };
  }
  if (id.includes("palac") && id.includes("dance")) {
    return { icon: "🏛️", title: "Palac Dance", desc: "High energy track on the darker side to make you dance all night long.", disabled: false, gradientFrom: "#1a0a3d", gradientVia: "#0d1040", gradientTo: "#0a0620", accentColor: "#c084fc" };
  }
  if (id.includes("fontanna") || id.includes("laputa")) {
    return { icon: "⛲", title: "Fontanna / Laputa", desc: "After-hour house, minimal, tech house — plus a few surprises.", disabled: false, gradientFrom: "#0a2d1a", gradientVia: "#061a10", gradientTo: "#040e08", accentColor: "#34d399" };
  }
  if (id.includes("etage") || name.includes("etage 0")) {
    return { icon: "⛓️", title: "Etage 0", desc: "Fast-paced underground. Hard techno to euro dance, trance and groovy tracks. Too hard for upstairs!", disabled: false, gradientFrom: "#1a1a1a", gradientVia: "#121212", gradientTo: "#0a0a0a", accentColor: "#71717a" };
  }
  if (id.includes("closing")) {
    return { icon: "🌟", title: "Closing", desc: "Closing is not hard — it just feels like the end, but the end is never the end. These are the tracks that could beautifully close PAVOIA. No categories.", disabled: false, gradientFrom: "#3d350a", gradientVia: "#2a2408", gradientTo: "#1a1604", accentColor: "#fbbf24" };
  }
  if (id.includes("bus")) {
    return { icon: "🚌", title: "Bus", desc: "Some things must be experienced in person.", disabled: true, gradientFrom: "#3d2a0a", gradientVia: "#2a1a08", gradientTo: "#1a1004", accentColor: "#fb923c" };
  }
  return fallback;
}
