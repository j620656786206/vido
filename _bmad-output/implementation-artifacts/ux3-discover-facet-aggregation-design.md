# Story ux3-discover-facet-aggregation-design — revive per-chip FacetCountChip on Discover v2 rail (`.pen` I1-D-v2)

**Epic:** ux3-discover-v2 (UX Redesign Phase 3) · **Status:** ready-for-dev (design not started)
**Owner:** ux-designer (Sally, Pencil MCP) · **Type:** design · **Initiative:** Discover Contextual Facet Counts
**Critical path:** this is the LONG POLE of the facet-counts initiative — it **blocks** `ux3-discover-facet-aggregation-fe` **Task 5** (per-chip count UI). It has **no upstream prerequisite** (does not depend on the BE/FE being built) → **start now.**

## Story

As the design system,
I want the Discover v2 desktop rail's filter chips to **show a per-chip contextual result count** (the `FacetCountChip` revived), with its two new states — **computing / progressive fill-in** and **dead-end-dimmed-but-still-selectable `0`** — drawn into `.pen` **I1-D-v2** and the Component Library,
so that dev (`ux3-discover-facet-aggregation-fe` Task 5) builds the per-chip count UI against a settled spec instead of inventing it (memory: *design-must-conform-to-current-design-system*).

## Context — a revival, not a new draw

- **ux3-3-1** drew the Discover v2 persistent rail and **created the `FacetCountChip`** component (Component Library group **`filter-controls-v2`**) + the Design Language v2 §7 FilterRail catalog entry, with per-chip JetBrains Mono counts.
- **ux3-3-2 decision 2改** then **REMOVED the per-facet count nodes** from the **I1-D-v2** rail frame and shipped the **single live total `符合 N 部`** instead, because a true per-facet count needed a backend aggregation endpoint that did not exist yet (`total_results` only).
- That endpoint now exists as a ratified contract: **`ux3-discover-facet-aggregation-be` `[@contract-v1]`** returns `{ counts: { genre:{id:int}, region:{code:int}, rating:{value:int}, platform:{id:int} }, partial }`. So per-facet **contextual** counts are back on the table — this story re-adds them **with the two states the live behavior needs** (progressive fill + dead-end), which the original ux3-3-1 chip did not specify.

**The single-total footer STAYS** — it is the summary / fallback (FE AC6: facet-counts unavailable → fall back to `符合 N 部`). This story ADDS per-chip counts; it does not remove the total.

## Design scope — what to draw (`ux-design.pen`)

> All edits via **Pencil MCP** only (never `Read`/`Grep` the `.pen`). Token-only color, **JetBrains Mono** for the numeric counts, Noto Sans TC for labels, 44px touch floor, `prefers-reduced-motion` respected for any fill animation.

### 1. Component Library — extend `FacetCountChip` (`filter-controls-v2`) with its two NEW states

The chip's resting state (label + Mono count) already exists from ux3-3-1. Add/define:

- **State A — computing / progressive fill-in.** Before a count resolves (cold/uncached or mid-recompute), the chip shows a **subtle placeholder** for the number — a low-contrast shimmer or `text-disabled` dash, **NOT a spinner** (tech-spec Decision #2: no per-chip spinners; the rail must keep its instant-rail identity). Counts fill in subtly as they arrive. Define the placeholder treatment + the resolved→filled transition (respect reduced-motion: no shimmer, just a fade or instant swap).
- **State B — dead-end (`0`) dimmed-but-selectable.** A `count === 0` chip is **DIMMED** (reduced opacity / `text-disabled` label + count) to read as a dead-end, **but remains visually a live, clickable chip** — NOT the `disabled` treatment (tech-spec Decision #8 / FE AC2: the user may want to SWITCH to it, e.g. replace another filter). Make the affordance distinct from both the active chip and the normal resting chip (dimmed ≠ disabled ≠ active).
- Keep the existing **active** (selected, `accent-subtle`) and **resting** states; show the count in all of them. Document in the Component Library doc which state is which (Design System Reference §7 / Design Language v2 §7 FilterRail catalog — update the existing entries, don't fork a new component).

### 2. Frame — re-instance per-chip counts into **I1-D-v2** (`fxCVk`), desktop rail only

In the I1-D-v2 desktop rail's four filter sections (類型 / 地區 / 評分 / 平台), re-add a `FacetCountChip` per chip showing an example **contextual** Mono count, demonstrating all states in-context:

- Most chips: resting + a sample count (e.g. `動作 340`, `Netflix 540`).
- At least one chip per the rail in **State A (computing)** and at least one in **State B (dimmed `0`)** so the frame documents the live behavior the FE will build.
- The counts are **contextual** (count if that facet were added to the current selection) — annotate the frame so dev/reviewer reads them as contextual, not baseline (matches BE `[@contract-v1]` semantics + FE AC3).
- Keep the **single-total footer** (`符合 N 部` / `計算中…`) exactly as shipped — it is the fallback/summary, not removed.

### 3. Scope guard — DESKTOP RAIL ONLY

Do **NOT** add per-chip counts to the **mobile** frame **I2-M-v2** or the `FilterBottomSheet` design — mobile keeps its single draft total (tech-spec Out-of-Scope / FE AC7: facet-counts are desktop-rail only, to avoid per-facet fan-out on small screens / slow nets). Leave I2-M-v2 untouched.

## Key design decisions (resolved — inherited from the ratified tech-spec)

1. **Contextual counts, not baseline** (tech-spec Decision #1) — annotate the frame accordingly.
2. **No per-chip spinners; subtle progressive fill** (Decision #2) — drives State A's treatment. Protects the ux3-3-2 instant-rail identity (a toggle must not read as "everything is loading").
3. **Dead-end dimmed but selectable** (Decision #8) — drives State B; explicitly NOT `disabled`.
4. **Per-locale, approximate** (Decisions #3/#7, Q2=A) — counts are TMDb `total_results`, may drift slightly vs the grid; present as exact but the design need not promise precision (no design action, just don't add a "live exact" claim).
5. **Single-total footer is the fallback** (FE AC6) — keep it.

## Acceptance Criteria

1. **Given** the Component Library `filter-controls-v2` group, **when** `FacetCountChip` is updated, **then** it documents **four** states — resting (label + Mono count), active (selected + count), **computing/progressive-fill (State A, no spinner)**, and **dead-end-dimmed-`0` (State B, dimmed but NOT disabled)** — each visually distinct, token-only, count in JetBrains Mono.
2. **Given** I1-D-v2 (`fxCVk`), **when** redrawn, **then** the desktop rail's 類型/地區/評分/平台 chips each carry a `FacetCountChip` instance, with at least one chip shown in State A and at least one in State B, contextual-count annotated, and the single-total footer (`符合 N 部` / `計算中…`) retained unchanged.
3. **Given** the mobile frame I2-M-v2, **when** this story completes, **then** it is **untouched** — no per-chip counts on mobile (desktop-rail only).
4. **Given** the v2 Design Language, **when** the chip is drawn, **then** it conforms to DL-v2 tokens (no new palette), JetBrains Mono numerics, Noto Sans TC labels, 44px touch floor, and reduced-motion-safe fill — and the dimmed-`0` state is distinguishable from both active and disabled.
5. **Given** the design is complete, **when** handed off, **then** the FE story (`ux3-discover-facet-aggregation-fe` Task 5) has a settled spec for the chip's count placement (after the label), the dimmed-`0` styling, and the progressive-fill treatment — keyed by `genre.id` / `region.code` / `rating value` / `platform.id` to match `[@contract-v1]` inner keys.

## Screenshots workflow (MANDATORY — CLAUDE.md)

After the `.pen` edits:

1. **Cmd+S the `.pen` first** (memory: *pencil-mcp-edits-need-manual-save* — the export script renders live app state, which masks an unsaved file).
2. `python3 scripts/export-pen-screenshots.py` (Pencil.app running).
3. Re-stage **only** the screenshot whose design actually changed — **`i1-d.png`** (a full regen is non-deterministic; `git checkout` the re-render noise on the rest).
4. `git add` the `ux-design.pen` change **and** `_bmad-output/screenshots/flow-i-discover-v2/i1-d.png` together; commit `feat: update UX design — Discover facet-count chips (computing + dead-end states)`.
5. I2-M-v2 (`i2-m.png`) must NOT change (scope guard #3) — if it shows a diff, `git checkout` it.

## Dependencies & handoff

- **Consumes (contract semantics only):** `ux3-discover-facet-aggregation-be` `[@contract-v1]` — the count inner-keys (`genre.id` / `region.code` / `rating value` / `platform.id`) the chips display. No code dependency; design can proceed before the BE is built.
- **Blocks:** `ux3-discover-facet-aggregation-fe` **Task 5** (per-chip count UI). FE data-layer Tasks 1–4 are NOT blocked by this.
- **Pattern precedent:** design-first (`ux3-3-1-discover-design` → `ux3-3-2-discover-frontend`); this story is the facet-counts analogue of ux3-3-1.

## References

- [Source: `_bmad-output/implementation-artifacts/tech-spec-ux3-discover-facet-aggregation.md`] — Task 0 (design prerequisite), Technical Decisions #2/#8, Additional Context "Dependencies → Design (.pen) — PREREQUISITE", response semantics.
- [Source: `_bmad-output/implementation-artifacts/ux3-discover-facet-aggregation-fe.md`] — FE Task 5 + AC1/AC2 the chip must satisfy; `[@contract-v1]` inner-key alignment.
- [Source: `_bmad-output/implementation-artifacts/ux3-3-1-discover-design.md`] — the rail draw + the original `FacetCountChip` / `filter-controls-v2` creation + Design Language v2 §7 FilterRail catalog.
- [Source: `_bmad-output/screenshots/flow-i-discover-v2/i1-d.png`] — current rail (single-total, no per-chip counts) — the before state.
- [Source: `01-design-language-v2.md` §7 FilterRail catalog; Design System Reference §7] — the catalog entries to update.
- Memory: *design-must-conform-to-current-design-system*, *pencil-mcp-edits-need-manual-save*.

## Dev Agent Record (ux-designer)

### Status update

_(to be filled by ux-designer on completion — design state, MCP review result, `.pen` saved + i1-d.png staged, then sprint-status → done)_

### Discovery Triage

- **Did this design discover any work outside its current scope?** _(to be filled — e.g. if the FacetCountChip anatomy needs a new variant beyond the four states, or if I1-D-v2 layout can't fit a count without crowding, triage per Rule 24.)_
