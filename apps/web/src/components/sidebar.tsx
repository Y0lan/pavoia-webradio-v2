"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const NAV = [
  { href: "/", label: "Stages", icon: "◈" },
  { href: "/dashboard", label: "Dashboard", icon: "▣" },
  { href: "/digging", label: "Digging", icon: "▦" },
  { href: "/history", label: "History", icon: "▤" },
  { href: "/artists", label: "Artists", icon: "◉" },
  { href: "/stats", label: "Stats", icon: "▥" },
  { href: "/about", label: "About", icon: "◇" },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside
      className="fixed left-0 top-0 bottom-[72px] w-[220px] z-50 hidden min-[900px]:flex flex-col"
      style={{
        background: "var(--color-bg-primary)",
        borderRight: "1px solid var(--border-subtle)",
      }}
    >
      {/* Logo */}
      <div className="p-5 pb-6">
        <Link
          href="/"
          className="font-[family-name:var(--font-display)] text-[14px] font-extrabold tracking-[0.3em] no-underline"
          style={{
            color: "var(--color-accent)",
            textShadow: "0 0 12px var(--color-accent-dim)",
          }}
        >
          GAENDE
        </Link>
        <div
          className="font-[family-name:var(--font-terminal)] text-[9px] tracking-[0.15em] uppercase mt-1"
          style={{ color: "var(--color-text-muted)" }}
        >
          ANTI-ALGORITHM RADIO
        </div>
      </div>

      {/* Nav items */}
      <nav className="flex flex-col gap-0.5 px-2">
        {NAV.map((item) => {
          const isActive =
            item.href === "/" ? pathname === "/" : pathname.startsWith(item.href);

          return (
            <Link
              key={item.href}
              href={item.href}
              className="relative flex items-center gap-3 py-[7px] px-[14px] no-underline transition-colors"
              style={{
                color: isActive ? "var(--color-accent)" : "var(--color-text-secondary)",
                background: isActive ? "var(--color-accent-glow)" : "transparent",
                fontFamily: "var(--font-mono)",
                fontSize: "12px",
              }}
            >
              {isActive && (
                <span
                  className="absolute left-0 top-1 bottom-1 w-[2px]"
                  style={{
                    background: "var(--color-accent)",
                    boxShadow: "0 0 6px var(--color-accent)",
                  }}
                />
              )}
              <span className="text-[11px] opacity-60">{item.icon}</span>
              {item.label}
            </Link>
          );
        })}
      </nav>

      {/* Footer */}
      <div className="mt-auto p-4">
        <div
          className="font-[family-name:var(--font-terminal)] text-[9px]"
          style={{ color: "var(--color-text-ghost)", animation: "flicker 4s infinite" }}
        >
          v1.0.0 // 9 STAGES // 24/7
        </div>
      </div>
    </aside>
  );
}
