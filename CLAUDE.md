# CLAUDE.md

## Rules

- **Never mention Claude, AI, or co-authored-by in git commits, PR descriptions, or changelogs.** No `Co-Authored-By` trailers. No "Generated with Claude Code" footers. Clean commit messages only.
- Challenge spec decisions if a better approach exists. The spec is directional, not sacred.

## Project

GAENDE Radio — 9-stage 24/7 webradio with cyber-brutalist terminal aesthetic.

- **Repo:** https://github.com/Y0lan/pavoia-webradio-v2
- **Seedbox:** yolan@orange.whatbox.ca (SSH key: `~/.ssh/id_ed25519_whatbox`)
- **Design system:** `docs/DESIGN_SYSTEM.md`
- **Architecture:** `docs/DESIGN.md`

## Tech Stack

- **Frontend:** Next.js 15 (App Router) + React 19 + Tailwind 4 + Framer Motion 11
- **Bridge API:** Go 1.22+ (bare binary on host, not containerized)
- **Database:** PostgreSQL 16 (Podman container)
- **Cache:** Redis 7 (Podman container)
- **Search:** Meilisearch (Podman container)
- **State:** Zustand 5
- **Data fetching:** TanStack Query v5
- **Fonts:** Syne, JetBrains Mono, Space Mono, Instrument Serif
- **Audio:** Raw Web Audio API (dual-slot crossfade)
- **Containers:** Podman (rootless, no Docker)
