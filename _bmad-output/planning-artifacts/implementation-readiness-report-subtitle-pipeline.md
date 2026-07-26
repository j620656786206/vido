---
assessmentTarget: 'Subtitle Pipeline (M1)'
inputDocuments:
  - _bmad-output/planning-artifacts/prd.md
  - _bmad-output/planning-artifacts/subtitle-pipeline-architecture.md
  - ux-design.pen (flow F v2, generation-centric)
assessmentDate: '2026-07-26'
assessor: 'John (PM) — check-implementation-readiness, scoped to PRD↔Architecture'
scope: 'PRD↔Architecture readiness gate (pre-epics). Epic-coverage and epic-quality steps are N/A — no subtitle-pipeline epics/stories exist yet.'
verdict: 'READY FOR EPIC BREAKDOWN — NOT YET READY FOR PHASE 4 DEV'
---

# Implementation Readiness Report — Subtitle Pipeline (M1)

**Assessed:** `prd.md` (post FR6/FR28 edits) + `subtitle-pipeline-architecture.md`
**Date:** 2026-07-26 · **Assessor:** John (PM), adversarial review
**Standard:** `check-implementation-readiness` — scoped to the PRD↔Architecture gate

## Verdict

**READY FOR EPIC BREAKDOWN (`[ES]`) — NOT YET READY FOR PHASE 4 IMPLEMENTATION.**

The PRD and Architecture are coherent, mutually traceable, and — after today's FR6/FR28 edits — free of the contradictions the architecture review flagged. The single reason this is not "ready for implementation" is structural, not a defect: **no epics or stories exist for the subtitle pipeline yet**, so there is nothing to validate for epic coverage or story quality. The correct next step is `[ES] Create Epics and Stories`, then a full IR re-run.

## Scope note — why this is a partial IR

`check-implementation-readiness` validates PRD ↔ Architecture ↔ **Epics & Stories**. Its step-03 (epic coverage) and step-05 (epic quality) require epics/stories as input. The subtitle pipeline has none yet (`epics.md` / `epics/` are unrelated v4 epics). Those two steps are therefore **N/A at this time** and are the dominant "not ready for dev" finding. Everything assessable now — PRD completeness, PRD↔Architecture traceability, cross-document action status — was assessed in full.

## 1. Document inventory (scoped)

| Type | File | Status |
|---|---|---|
| PRD | `prd.md` | ✅ Used (edited today) |
| Architecture | `subtitle-pipeline-architecture.md` | ✅ Used (8-step BMAD output) |
| Epics/Stories | — | ❌ None for this feature |
| UX | `ux-design.pen` flow F v2 | ✅ Referenced (13 desktop + 4 mobile screens) |

**Duplicate-format note (false positives):** generic discovery flags `prd/`, `architecture/`, and `epics/` folders as duplicates of the whole `.md` files. They are **not** duplicates of the subtitle-pipeline docs — they are the older v4 project artifacts. No resolution needed; the subtitle-pipeline PRD and architecture are unambiguous single files.

## 2. PRD completeness & internal consistency

- ✅ 34 FRs across 9 capability areas, phase-tagged; 11 NFRs; 5 journeys; MVP/M1.5/P2/Tier-2 scoping with rationale.
- ✅ **FR28 fixed** — now carries `[P2]`, matching the Post-MVP prose and the glossary linkage. The legend-vs-prose contradiction is gone.
- ✅ **FR6 fixed** — content-based detection now scoped to the Traditional-vs-Simplified variant; source language identified from the track tag (per FR1). Removes the FR6-vs-P0 contradiction the architecture flagged.
- ⚠️ `prd-validation-report.md` gaps G2/G4 remain (NFRs lacking hard numbers — NFR-P1 threshold, translation-quality %). These are **deliberately deferred to the M1 pilot** and are not blocking; they must become measurable acceptance criteria at story time.

## 3. PRD ↔ Architecture traceability (M1)

Every M1 FR (~22 unmarked) maps to a named architecture component or decision:

| FR area | PRD (M1) | Architecture home | Status |
|---|---|---|---|
| A. Detection/extraction | FR1–5 | `ffprobe_service.go` (FR1) · `extractor.go` (FR2/3) · `sdh_filter.go` (FR4) · `subtitle_status` new value (FR5) | ✅ |
| B. Routing/conversion | FR6–9 | `router.go` + P0 tag rule · `detector.go` (CJK) · `converter.go` (s2twp) | ✅ (FR6 now aligned) |
| C. AI translation | FR10–13 | `pipeline.go` → `translation_service.go` → `ai/claude.go` (SDK) | ✅ FR12 endpoint (V2), FR13 scanner enqueue (V3) |
| D. Quality assurance | FR15–17 | `quality_gate.go` before OpenCC (P4) | ✅ |
| F. Keys | FR21, FR23 | env-var (M1) · capability gate in `pipeline.go` (V6) · NFR-S3 | ✅ |
| G. Metadata | FR26, FR27 | existing TMDb service · provenance table | ✅ |
| I. Delivery/status | FR32, FR33 | `placer.go` sole writer · SSE stage extension (D6) | ✅ |

**No orphaned M1 FR** (every requirement has an implementation home) and **no orphaned M1 component** (every new component traces to an FR). Deferred FRs (FR14, FR18–20, FR22, FR24, FR25, FR28, FR29–31, FR34) are correctly parked with existing groundwork (`asr.go`, `glossary_repository.go`, `engine.go`).

## 4. NFR coverage

✅ NFR-P1/P2 (I/O-bound, no local heavy compute in M1) · NFR-P3 (fixed concurrency 2 in M1, tiering→P2, V4) · NFR-S1 (encrypted secrets) · **NFR-S3 added** (key transit) · NFR-R1 (fail-soft) · NFR-R2 (single writer + P5 predicate) · NFR-R3 (resume via provenance, V5) · NFR-I1–I3 (Five Pillars, multi-arch+ffmpeg, sidecar naming).
⚠️ NFR-P1 and the trust-% target are still qualitative (G2/G4) — quantify at story/pilot time.

## 5. Cross-document actions (from the architecture review)

| Action | Status |
|---|---|
| FR6 PRD scope (V1a) | ✅ **Done today** |
| FR28 `[P2]` tag | ✅ **Done today** |
| `.pen` copy revisions (F2 "轉錄"→"抽取", F5 "前往設定" M1 behaviour, F5 FFmpeg framing) | ⚠️ **Open** — UX (Sally); affects M1.5 UI copy, not M1 backend. Non-blocking for backend story creation, but must close before the M1.5 UI stories. |

## 6. Blocking gap for Phase 4

**No epics/stories exist for the subtitle pipeline.** This is the one hard gate between "planned" and "buildable":
- Epic coverage (IR step-03) and story quality (IR step-05) cannot be assessed.
- The architecture already supplies the implementation sequence (7 ordered steps) — that is the natural skeleton for the epic breakdown, not a substitute for it.

## Recommendation

1. **Run `[ES]` (Create Epics and Stories)** for the subtitle pipeline, using the architecture's 7-step implementation sequence as the epic spine and the ~22 M1 FRs as the story source. Fold the G2/G4 measurable thresholds into story acceptance criteria.
2. **Close the `.pen` copy revisions** (UX) before the M1.5 UI stories — not required for the M1 backend stories.
3. **Re-run the full IR** once stories exist, so epic-coverage and story-quality (the currently-N/A steps) can be validated.

**Bottom line:** planning is coherent and traceable; the gate to dev is `[ES]`, not another planning fix.
