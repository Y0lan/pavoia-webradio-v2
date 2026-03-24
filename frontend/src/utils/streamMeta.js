

export function getMetaFor(s) {
  const id = (s?.id || "").toLowerCase();
  const name = (s?.name || "").toLowerCase();
  const meta = { icon: "📻", title: s?.name || "", desc: "", button: "", mood: "", disabled: false };

  if (id.includes("gaende")) {
    return { icon: "💜", title: "gaende’s favorites", desc: "Personal selection of tracks that hit different. Every genre, every mood - find them across all stages!", button: "", mood: "", disabled: false };
  }
  if (id.includes("ambiance")) {
    return { icon: "🛋️", title: "Ambiance / Safe Space", desc: "The perfect comedown. Chill on the sofa with relaxing beats. Good vibes only in this safe space.", button: "", mood: "", disabled: false };
  }
  if (id.includes("bermuda") && (id.includes("day") || id.includes("oaza"))) {
    return { icon: "🌴", title: "Bermuda Before 18:00 / Oaza", desc: "Sunlit grooves • chat, move, breathe.", button: "", mood: "", disabled: false };
  }
  if (id.includes("bermuda") && id.includes("night")) {
    return { icon: "🌅", title: "Bermuda (18:00–00:00)", desc: "Progressive & Indie • sunset lift, growing tension.", button: "", mood: "", disabled: false };
  }
  if (id.includes("palac") && (id.includes("hypno") || id.includes("slow"))) {
    return { icon: "🌙", title: "Palac Feel", desc: "Melodic motion • hypnotic & tender, harsh and heartbroken. Dance your emotions away all night long.", button: "", mood: "", disabled: false };
  }
  if (id.includes("palac") && id.includes("dance")) {
    return { icon: "🏛️", title: "Palac Dance", desc: "High energy track on the darker side to make you dance all night long.", button: "", mood: "", disabled: false };
  }
  if (id.includes("fontanna") || id.includes("laputa")) {
    return { icon: "⛲", title: "Fontanna / Laputa", desc: "After-hour house, minimal, tech house — plus a few surprises.", button: "", mood: "", disabled: false };
  }
  if (id.includes("etage") || name.includes("etage 0")) {
    return { icon: "⛓️", title: "Etage 0", desc: "Fast-paced underground. Hard techno to euro dance, trance and groovy tracks. Too hard for upstairs!", button: "", mood: "", disabled: false };
  }
  if (id.includes("closing")) {
    return { icon: "🌟", title: "Closing", desc: "Closing is not hard — it just feels like the end, but the end is never the end. These are the tracks that could beautifully close PAVOIA. No categories.", button: "", mood: "", disabled: false };
  }
  if (id.includes("bus")) {
    return { icon: "🚌", title: "Bus", desc: "IRL only • you had to be there.", button: "", mood: "", disabled: true };
  }
  // Fallback – keep original name, no URL
  return meta;
}
