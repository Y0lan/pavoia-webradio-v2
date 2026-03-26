export const STAGES = [
  { id: "gaende-favorites", name: "Main Stage", genre: "Progressive Melodic Techno", color: "#00ffc8" },
  { id: "etage-0", name: "Techno Bunker", genre: "Techno", color: "#ff0066" },
  { id: "ambiance-safe", name: "Ambient Horizon", genre: "Ambient", color: "#00ddff" },
  { id: "palac-dance", name: "Indie Floor", genre: "Indie Dance", color: "#ffaa00" },
  { id: "fontanna-laputa", name: "Deep Current", genre: "Deep House", color: "#7b7bff" },
  { id: "palac-slow-hypno", name: "Chill Terrace", genre: "Chillout", color: "#44ddff" },
  { id: "bermuda-night", name: "Bass Cave", genre: "DnB", color: "#ff44ff" },
  { id: "bermuda-day", name: "World Frequencies", genre: "Afro House", color: "#ff4466" },
  { id: "closing", name: "Live Sets", genre: "Live", color: "#00ff88" },
] as const;

export type StageId = (typeof STAGES)[number]["id"];

export function getStage(id: string) {
  return STAGES.find((s) => s.id === id);
}

export const BRIDGE_URL = process.env.NEXT_PUBLIC_BRIDGE_URL || "http://localhost:3001";
