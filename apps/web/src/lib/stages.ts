export const STAGES = [
  {
    id: "gaende-favorites",
    name: "gaende's favorites",
    desc: "Personal selection of tracks that hit different. Every genre, every mood — find them across all stages!",
    icon: "💜",
    color: "#00ffc8",
  },
  {
    id: "ambiance-safe",
    name: "Ambiance / Safe Space",
    desc: "The perfect comedown. Chill on the sofa with relaxing beats. Good vibes only in this safe space.",
    icon: "🛋️",
    color: "#00ddff",
  },
  {
    id: "bermuda-day",
    name: "Bermuda Before 18:00 / Oaza",
    desc: "Sunlit grooves · chat, move, breathe.",
    icon: "🌴",
    color: "#ff4466",
  },
  {
    id: "bermuda-night",
    name: "Bermuda (18:00–00:00)",
    desc: "Progressive & Indie · sunset lift, growing tension.",
    icon: "🌅",
    color: "#ff44ff",
  },
  {
    id: "palac-slow-hypno",
    name: "Palac Feel",
    desc: "Melodic motion · hypnotic & tender, harsh and heartbroken. Dance your emotions away all night long.",
    icon: "🌙",
    color: "#44ddff",
  },
  {
    id: "palac-dance",
    name: "Palac Dance",
    desc: "High energy tracks on the darker side to make you dance all night long.",
    icon: "🏛️",
    color: "#ffaa00",
  },
  {
    id: "fontanna-laputa",
    name: "Fontanna / Laputa",
    desc: "After-hour house, minimal, tech house — plus a few surprises.",
    icon: "⛲",
    color: "#7b7bff",
  },
  {
    id: "etage-0",
    name: "Etage 0",
    desc: "Fast-paced underground. Hard techno to euro dance, trance and groovy tracks. Too hard for upstairs!",
    icon: "⛓️",
    color: "#ff0066",
  },
  {
    id: "closing",
    name: "Closing",
    desc: "Closing is not hard — it just feels like the end, but the end is never the end. No categories.",
    icon: "🌟",
    color: "#00ff88",
  },
] as const;

export type StageId = (typeof STAGES)[number]["id"];

export function getStage(id: string) {
  return STAGES.find((s) => s.id === id);
}

// When behind a reverse proxy (production), BRIDGE_URL is empty (same origin).
// For local dev, set NEXT_PUBLIC_BRIDGE_URL=http://localhost:3001
export const BRIDGE_URL = process.env.NEXT_PUBLIC_BRIDGE_URL || "";
