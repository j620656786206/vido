---
name: ux-redesign-initiative
description: "Full-app UX redesign initiative — phased strangler migration; Phase 0 brief done 2026-06-12, Phase 1 (design language v2 + nav IA ADR) is next"
metadata: 
  node_type: memory
  type: project
  originSessionId: 3cdfaa85-5b71-4c78-8871-622075e2696b
---

Full-app UI/UX redesign initiative, kicked off 2026-06-12. **Strangler-fig phased migration, NOT big-bang.** Working folder: `_bmad-output/planning-artifacts/ux-redesign/` (its README.md is the phase plan + conventions: fresh session per phase, Fable 5, converse zh-TW, docs in English).

- **Phase 0 ✅ (2026-06-12):** `00-redesign-brief.md` — evidence-backed pain points (6 systemic root causes from pen-review + 10 retro failure modes + hotspot ranking), competitive scan (Plex/Jellyfin/Emby/arr/seerr), design principles N1–N6, open IA decisions D1–D4 (sidebar-vs-tabs, movies/TV first-class?, home-vs-discover-vs-library split, where tasks live). Deliberately undecided — Phase 1 owns the ADR.
- **Phase 1 (next):** architect + ux-designer → `01-design-language-v2.md` + `01-nav-ia-decision-adr.md`. Resolve D1–D4; brief §8 lists 4 product questions to put to Alexyu (anime tab?, Epic 10 hero attachment, status strip appetite, reserve IA slots for v5 ambitions?).
- **Phase 2:** pilot Browse (A) + Detail (B) behind a flag, go/no-go gate; 390/768/1440 browser-pixel verification is a gate criterion (mobile verify debt was perpetually deferred — brief P5).
- **Phase 3:** flow-by-flow cascade (C–J + Epic-12 detail screens).

Key evidence anchors: `pen-review-2026-06-12.md` (same folder), Epic 9 AI-subtitle frontend was never built (backend+design done), homepage identity flipped twice (Epic 4 task-dashboard → Epic 10 discovery), Plex 2025 top-nav backlash = cautionary evidence for D1. Phase 0 files are uncommitted as of 2026-06-12 (user decides when to ship). Related: [[pen-flow-layout-convention]].
