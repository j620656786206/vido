---
status: 'approved'
workflow: 'correct-course (BMAD 4-implementation)'
author: 'Alexyu + Sally/PM session, 2026-08-26'
change_trigger: 'Home v3 identity rework — impeccable critique R1–R3 + shape session'
inputDocuments:
  - _bmad-output/planning-artifacts/home-v3-identity-brief.md
  - _bmad-output/planning-artifacts/ux-redesign/03-phase3-destination-epic-map.md
  - _bmad-output/implementation-artifacts/sprint-status.yaml
  - _bmad-output/screenshots/flow-h-homepage-v3/ (design, merged PR #323)
---

# Sprint Change Proposal — Home v2 → Home v3 identity rework

## 1. Issue Summary

Three measured `/impeccable critique homepage` rounds (2026-08-25/26, score 20 → 25 → 27/40)
established that the remaining score ceiling is **identity**, not polish: the homepage's largest
surface (TMDb Hero + Explore) sells content the user does NOT own, while the product's moat
(unattended zh-TW subtitle closure) got an 11px badge. A dedicated shape session with Alexyu
(2026-08-26, five rulings over two rounds) produced `home-v3-identity-brief.md`; the visual
design landed as PR #323 (flow-h-homepage-v3: H1-D-v3 / H7-D-v3 / H2-M-v3 / H8-SPEC-v3,
Sally MCP-reviewed twice). This proposal folds the change into the BMAD planning artifacts and
routes implementation.

## 2. Impact Analysis

- **Epic impact**: `ux3-home-v2` (done) is superseded at the page-composition level. Its D3
  law (own-above-external) SURVIVES and is strengthened; its deliverables (HomeBrowseV2 shell,
  RecentlyAddedRowV2, shell-v2 gating, CW-slot decision) are reused, not reverted. New epic
  **`ux3-home-v3`** under the same Phase-3 destination #1 (首頁 Home).
- **Story impact**: new stories (see §4); no in-flight story is disrupted (subtitle-pipeline M3
  wave is orthogonal).
- **Artifact conflicts**:
  - `ux-redesign/03-phase3-destination-epic-map.md` — destination #1 row needs a v3 addendum.
  - `flow-h-homepage-v2` designs renamed 〔已淘汰→v3〕 in PR #323; deleted only after v3 ships.
  - DESIGN.md untouched (v3 uses existing tokens; 動骨架不動世界).
- **Technical impact**: ONE net-new backend surface — a cheap aggregate for the readout band
  (extend `GET /api/v1/activity` or new `GET /api/v1/home-summary`; architect decides in
  tech-spec). Explore owned-filter is frontend-only (hoisted ownership already exists). TMDb
  hero retirement deletes/repurposes `HeroBanner` usage; e2e specs in `tests/e2e/` MUST be
  swept (the #318 lesson).

## 3. Recommended Approach

**Direct Adjustment** (no rollback, no MVP change): add epic `ux3-home-v3` with a BE story
gating two FE stories. Design is already merged, so implementation can start at the tech-spec.
Risk: low — page is shell-gated, degradation modes are designed (H7), and the readout band
fails soft per the brief's honesty rules. Effort: 1 BE story + 2 FE stories.

## 4. Detailed Change Proposals

1. **Epic map addendum** (`03-phase3-destination-epic-map.md` §1 row 1): note that Home v2
   composition is superseded by Home v3 (readout band + own-library static hero + TMDb tail
   filtered/degraded), brief + design references, D3 unchanged.
2. **sprint-status.yaml**: new epic `ux3-home-v3` with stories:
   - `ux3-1-5-home-v3-design`: **done** (retroactive — shipped via impeccable shape + Inline
     Agent + Sally MCP review, PR #323).
   - `ux3-1-6-home-v3-summary-api` (BE): aggregate endpoint for the four readout cells
     (coverage 42/55, processed-today, needs-attention failures + spend, in-flight count).
     Gated on architect tech-spec `tech-spec-ux3-home-summary.md` (open decisions brief §7:
     endpoint shape, time anchor, spend source — builders may NOT invent these).
   - `ux3-1-7-home-v3-readout-band-fe` (FE): the band itself — 4 cells, doors, amber exception
     cell, 0/0 first-run state, 一切正常 state, amount display per H8-SPEC-v3, mobile 2×2.
   - `ux3-1-8-home-v3-hero-and-tail-fe` (FE): own-library static hero (manual dots, no
     autoplay, subtitle badge, not rendered without backdrops), TMDb hero retirement, explore
     rows filter owned (frontend), TMDb block absent-when-degraded + amber footer door
     (H7-D-v3), e2e sweep.
3. **Story sequencing**: 1-6 → 1-7 (needs API) ; 1-8 parallelizable with 1-7 after 1-6 lands
   (hero needs backdrop data already available via library API — confirm in tech-spec).

## 5. Implementation Handoff

Scope: **Moderate** (backlog reorganization; one net-new API). Route:
- **Architect** → `tech-spec-ux3-home-summary.md` (endpoint choice, exact JSON field names,
  time-anchor + spend-source decisions, backed by code facts).
- **SM (Bob)** → `create-story` for ux3-1-6 first, then 1-7/1-8.
- **Dev (Amelia)** → `dev-story` per story; three-gate verification (Dev → Bob → Sally → User)
  unchanged.

Success criteria: homepage renders the four-cell readout band from live data with honest
degraded states; own-library hero static with manual switching; TMDb tail filtered and
absent-when-degraded; all four designed states (H1/H7 + mobile) match screenshots; e2e green.
