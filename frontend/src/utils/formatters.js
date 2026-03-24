

export function fmtTime(s) {
  if (!Number.isFinite(s)) return "—:—";
  s = Math.max(0, Math.floor(s));
  const m = Math.floor(s / 60);
  const ss = String(s % 60).padStart(2, "0");
  return `${m}:${ss}`;
}
