"use client";

/**
 * Camelot wheel — 12 positions × 2 rings (A = inner minor, B = outer major).
 *
 * The flat grid of key boxes was fine for data display but misses the one
 * thing a DJ actually uses the wheel for: seeing at a glance which zones of
 * the collection are deep vs thin. SVG radial layout matches DESIGN_SYSTEM
 * "custom SVG, segments glow on hover" and the canonical Mixed-In-Key
 * harmonic-mixing wheel that DJs already have in their head.
 *
 * No interactions yet beyond hover — clicking a segment to filter the
 * library by key is a TODO (needs a /stats route with a key filter).
 */

type KeyDatum = { key: string; count: number };

interface Props {
  data: KeyDatum[];
  size?: number;
}

// Canonical Camelot order around the wheel, 12 o'clock clockwise.
// Each slot carries the two ring entries (outer=B/major, inner=A/minor).
const WHEEL: { pos: number; outer: string; inner: string }[] = [
  { pos: 0,  outer: "12B", inner: "12A" },
  { pos: 1,  outer: "1B",  inner: "1A"  },
  { pos: 2,  outer: "2B",  inner: "2A"  },
  { pos: 3,  outer: "3B",  inner: "3A"  },
  { pos: 4,  outer: "4B",  inner: "4A"  },
  { pos: 5,  outer: "5B",  inner: "5A"  },
  { pos: 6,  outer: "6B",  inner: "6A"  },
  { pos: 7,  outer: "7B",  inner: "7A"  },
  { pos: 8,  outer: "8B",  inner: "8A"  },
  { pos: 9,  outer: "9B",  inner: "9A"  },
  { pos: 10, outer: "10B", inner: "10A" },
  { pos: 11, outer: "11B", inner: "11A" },
];

function toPolar(cx: number, cy: number, r: number, angleRad: number) {
  return { x: cx + r * Math.cos(angleRad), y: cy + r * Math.sin(angleRad) };
}

function arcSegment(
  cx: number,
  cy: number,
  rOuter: number,
  rInner: number,
  startAngle: number,
  endAngle: number,
) {
  // SVG wedge between two radii and two angles. Using a single path so the
  // stroke + fill apply uniformly.
  const p1 = toPolar(cx, cy, rOuter, startAngle);
  const p2 = toPolar(cx, cy, rOuter, endAngle);
  const p3 = toPolar(cx, cy, rInner, endAngle);
  const p4 = toPolar(cx, cy, rInner, startAngle);
  const largeArc = endAngle - startAngle > Math.PI ? 1 : 0;
  return [
    `M ${p1.x} ${p1.y}`,
    `A ${rOuter} ${rOuter} 0 ${largeArc} 1 ${p2.x} ${p2.y}`,
    `L ${p3.x} ${p3.y}`,
    `A ${rInner} ${rInner} 0 ${largeArc} 0 ${p4.x} ${p4.y}`,
    "Z",
  ].join(" ");
}

export function CamelotWheel({ data, size = 360 }: Props) {
  const byKey = new Map(data.map((d) => [d.key, d.count]));
  const max = Math.max(1, ...data.map((d) => d.count));

  const cx = size / 2;
  const cy = size / 2;
  // Outer ring (B / major) extends from rOuter2 → rOuter1.
  // Inner ring (A / minor) extends from rInner2 → rInner1.
  // Small gap between the two rings keeps the B/A boundary readable.
  const rOuter1 = size / 2 - 6;
  const rOuter2 = size * 0.34;
  const rInner1 = size * 0.33;
  const rInner2 = size * 0.20;

  const segAngle = (Math.PI * 2) / 12;
  // Shift so 12 o'clock is the 12B/12A slot. Default path math places 0rad
  // at 3 o'clock (east); subtract a quarter turn to rotate north.
  const startOffset = -Math.PI / 2 - segAngle / 2;

  return (
    <svg viewBox={`0 0 ${size} ${size}`} width={size} height={size} style={{ maxWidth: "100%", height: "auto" }}>
      {WHEEL.map(({ pos, outer, inner }) => {
        const startAngle = startOffset + pos * segAngle;
        const endAngle = startAngle + segAngle;
        const midAngle = (startAngle + endAngle) / 2;

        const outerCount = byKey.get(outer) ?? 0;
        const innerCount = byKey.get(inner) ?? 0;
        const outerFill = outerCount > 0
          ? `rgba(0, 255, 200, ${0.15 + 0.65 * (outerCount / max)})`
          : "var(--color-bg-elevated)";
        const innerFill = innerCount > 0
          ? `rgba(0, 255, 200, ${0.15 + 0.65 * (innerCount / max)})`
          : "var(--color-bg-elevated)";

        const outerLabel = toPolar(cx, cy, (rOuter1 + rOuter2) / 2, midAngle);
        const innerLabel = toPolar(cx, cy, (rInner1 + rInner2) / 2, midAngle);
        const outerCountPos = toPolar(cx, cy, (rOuter1 + rOuter2) / 2 + 12, midAngle);
        const innerCountPos = toPolar(cx, cy, (rInner1 + rInner2) / 2 - 10, midAngle);

        return (
          <g key={pos}>
            {/* Outer (B / major) wedge */}
            <path
              d={arcSegment(cx, cy, rOuter1, rOuter2, startAngle, endAngle)}
              fill={outerFill}
              stroke="var(--border-default)"
              strokeWidth={1}
            >
              <title>{`${outer}: ${outerCount} tracks`}</title>
            </path>
            {/* Inner (A / minor) wedge */}
            <path
              d={arcSegment(cx, cy, rInner1, rInner2, startAngle, endAngle)}
              fill={innerFill}
              stroke="var(--border-default)"
              strokeWidth={1}
            >
              <title>{`${inner}: ${innerCount} tracks`}</title>
            </path>

            {/* Labels in Syne display font, positioned in the middle of each
                wedge. Small shadow so they stay legible over a bright segment. */}
            <text
              x={outerLabel.x}
              y={outerLabel.y}
              textAnchor="middle"
              dominantBaseline="middle"
              fontFamily="var(--font-display)"
              fontSize={13}
              fontWeight={700}
              fill="var(--color-text-primary)"
              style={{ textShadow: "0 0 4px rgba(2,2,4,0.9)" }}
            >
              {outer}
            </text>
            {outerCount > 0 && (
              <text
                x={outerCountPos.x}
                y={outerCountPos.y}
                textAnchor="middle"
                dominantBaseline="middle"
                fontFamily="var(--font-terminal)"
                fontSize={9}
                fill="var(--color-accent)"
                opacity={0.8}
              >
                {outerCount}
              </text>
            )}
            <text
              x={innerLabel.x}
              y={innerLabel.y}
              textAnchor="middle"
              dominantBaseline="middle"
              fontFamily="var(--font-display)"
              fontSize={11}
              fontWeight={700}
              fill="var(--color-text-primary)"
              style={{ textShadow: "0 0 4px rgba(2,2,4,0.9)" }}
            >
              {inner}
            </text>
            {innerCount > 0 && (
              <text
                x={innerCountPos.x}
                y={innerCountPos.y}
                textAnchor="middle"
                dominantBaseline="middle"
                fontFamily="var(--font-terminal)"
                fontSize={8}
                fill="var(--color-accent)"
                opacity={0.8}
              >
                {innerCount}
              </text>
            )}
          </g>
        );
      })}

      {/* Center label — A / B guide so users who don't live in Camelot can
          still orient. */}
      <text
        x={cx}
        y={cy - 8}
        textAnchor="middle"
        dominantBaseline="middle"
        fontFamily="var(--font-terminal)"
        fontSize={9}
        letterSpacing="0.2em"
        fill="var(--color-text-muted)"
      >
        CAMELOT
      </text>
      <text
        x={cx}
        y={cy + 6}
        textAnchor="middle"
        dominantBaseline="middle"
        fontFamily="var(--font-terminal)"
        fontSize={8}
        letterSpacing="0.15em"
        fill="var(--color-text-ghost)"
      >
        A · MINOR / B · MAJOR
      </text>
    </svg>
  );
}
