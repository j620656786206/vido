# Vido UX Redesign Initiative — Working Folder

Artifacts for the **full-app UI/UX redesign** kicked off after Epic 42. Run as a
**phased, strangler-fig migration — NOT a big-bang rewrite**.

## Why
After 42 epics the UI accreted layer-by-layer; each epic was locally sensible but the
app lacks one coherent design language. A 40-screen `.pen` review (2026-06-12) surfaced
**six systemic root causes**: (1) CJK text set in DM Sans / Inter (no CJK glyphs →
uncontrolled fallback); (2) `--text-muted` low contrast on small text (fails WCAG AA);
(3) touch targets < 44px across mobile; (4) hardcoded semantic colors / token drift;
(5) navigation mixing *content* and *tasks* in one layer; (6) missing empty / loading /
error states + Epic-12 detail features with no design coverage.

## Phased plan
| Phase | Goal | Primary BMAD entry | Output |
|-------|------|--------------------|--------|
| **0** | Discovery & North Star — evidence + competitive + IA options. No screens, no code. | `analyst` (research) → product brief | `00-redesign-brief.md` |
| **1** | Design Language v2 + Nav-IA decision (the reusable foundation) | `architect` (create-architecture) + `ux-designer` (create-ux-design) | `01-design-language-v2.md`, `01-nav-ia-decision-adr.md` |
| **2** | Pilot 2 flows end-to-end (Browse A + Detail B), behind a flag, validate. **go/no-go gate.** | `ux-designer` → `sm` → `dev` → `tea` | `02-pilot-validation.md` |
| **3** | Cascade flow-by-flow (C–J + Epic-12 detail), strangler migration | `sprint-planning` + `create-epics-and-stories` → per-flow vertical slices | per-flow epics |

## Artifact index
- `pen-review-2026-06-12.md` — the 40-screen `.pen` review (4 parallel agents: A+C, B,
  D–G, H/I/J+Design System) + cross-cutting Tier 1–4 triage. **Primary evidence input for
  Phase 0.** Recorded in zh-TW (as conducted, preserves the Chinese UI strings & node IDs).
- `00-redesign-brief.md` — **Phase 0 output (done 2026-06-12)**: evidence-backed pain-point
  inventory (6 systemic root causes + 10 recurring failure modes + hotspot ranking),
  2025–26 competitive scan, design principles N1–N6, open IA decisions D1–D4 for Phase 1.
- `01-design-language-v2.md`, `01-nav-ia-decision-adr.md` — **Phase 1 output (done 2026-06-13, PR #64)**
- `02-pilot-validation.md` — **Phase 2 output (done 2026-06-14) — ✅ GO to Phase 3 CONFIRMED**
  after Alexyu's 390/768/1440 browser-pixel pass. Pilot = PRs #67/#68/#69; the
  test-quality fallout (Epic 20: bugfix-20-1 #72, story-20-2 #75, story-20-3 #74)
  is closed. **Next: open a fresh session (Fable 5) for Phase 3.**
- See also `../../design-context-pack.md` — project primer (real design tokens from
  `apps/web/src/styles.css`, the `.pen` A–J canvas convention).

## Conventions (every phase session)
- Open a fresh Claude Code session per phase; switch model to **Fable 5**.
- Converse in **zh-TW**; write docs in **English** (per `_bmad/bmm/config.yaml`
  `document_output_language`). `project-context.md` is the bible.
- Any `ux-design.pen` change → run `python3 scripts/export-pen-screenshots.py` and commit
  only genuinely-changed PNGs.
- PR/merge with gh account `j620656786206` (switch in the same shell command; the active
  account silently reverts — verify with `gh api user --jq .login`).
