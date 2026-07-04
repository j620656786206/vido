# Story chore-pen-design-system-token-debt — `.pen` design-system debt sync (R5 `text-muted` + library registration)

**Epic:** — (standalone design-system chore; spawned by 13-0 review) · **Status:** review (executed 2026-07-04 — PR pending merge)
**Owner:** ux-designer (Pencil MCP) · **Type:** design-only chore · **Source:** 13-0 requests-design review 2026-07-04 (PR #118)

## Story

As the design system,
I want the `.pen` file's `text-muted` variable synced to the ratified R5 value and every shipped v2 component registered in the Component Library frame,
so that the `.pen` ↔ `styles.css` single-source-of-truth contract (DL-v2 §2.1 "no forking") actually holds, and future design stories can find every reusable v2 unit in one place.

## Context — two verified debts, both `.pen`-only

1. **`text-muted` value lag (R5).** DL-v2 §2.2 (Alexyu decision 2026-06-13, "both" remediation): `--text-muted` becomes **`#A0AABE`** — "the single value change in v2" (old `#808080` fails AA: 3.97 / 3.33 / 2.83 on the three dark surfaces). **Verified 2026-07-04:**
   - `apps/web/src/styles.css:31` = `#a0aabe` ✅ (code adopted the correction long ago)
   - `.pen` variable `text-muted` = `#808080` ❌ (never updated — `get_variables` confirmed)
   - `.pen` DL-v2 reference frame even *documents* the correction in its §2 panel text — the variable just never followed.
   - `text-disabled` `#6E7891` matches §2.2 in both surfaces ✅ (only `text-muted` lags).

   **Consequence today:** every mockup renders muted text ~2 shades darker than the shipped product, and any AA judgment made from the canvas for `$text-muted` strings is judged against a failing value. Design-only fix: no app code, no test, **no `-linux` visual-baseline impact**.

2. **`Component/DownloadCard-v2` unregistered.** The component exists (node `Mz428`, created by ux3-4-1) but has no cell in the Component Library frame (`sJzat`) — the `content-cards-v2` row (`ISilG`) holds only `PosterCard-v2` and `ActivityRow-v2` (+ `RequestRow-v2` since 13-0). Registration convention (per 13-0 / memory `feedback_design_system_conformance_pen`): **cell = vertical frame, gap 8 → [component `ref` instance (width may be narrowed, e.g. 720→480) + Noto Sans TC 12 `$text-muted` caption below]**.

## Scope

- **In:** `ux-design.pen` (variable + library frame), full screenshot regen, this story's tracking docs.
- **Out:** any `apps/web` change (styles.css already correct); `text-muted` **usage-rule** sweep (§2.2's "≥14px metadata only" rule — separate audit if ever needed); the `.pen` frames that hardcode muted-ish literals (e.g. legacy v1 flows) — only the **variable** is in scope.

## ⚠️ Blast radius — the screenshot rule inverts for this story

Changing the variable is a **global, genuine visual change**: every frame using `$text-muted` re-renders slightly brighter. The usual CLAUDE.md rule ("stage only genuinely-changed PNGs, checkout the noise") **inverts** — after the regen, expect most PNGs to be legitimately changed and byte-diff indistinguishable from noise anyway. Procedure:

1. Full regen via `scripts/export-pen-screenshots.py`.
2. **Sample-verify** the brightness change on 2–3 muted-text-heavy screens (e.g. a sidebar group label, poster metadata, facet counts) before staging.
3. Stage **all** regenerated PNGs + `ux-design.pen` in one commit (this is the documented exception; cite this story in the commit body).
4. Known flake: `flow-a-browse/a1-d.png` failed to export on 2026-07-04's run (1/128) — retry; if it persists, investigate the node/script before shipping.

## Acceptance Criteria

1. **Given** DL-v2 §2.2, **then** `get_variables` shows `text-muted = #A0AABE`; the change is made with a **merge** `SetVariables` (no other variable or theme touched, `replace` NOT used).
2. **Given** the registration convention, **then** a `DownloadCard-v2` cell (ref `Mz428` + caption) exists in the Component Library `content-cards-v2` row, alongside the existing cells, with no layout problems (`snapshot_layout` clean).
3. **Given** the component inventory, **then** an audit confirms every `Component/*-v2` reusable has a library cell (RequestRow-v2 ✅ 13-0; list any other stragglers found and register or file them).
4. **Given** the blast-radius procedure, **then** the commit contains `ux-design.pen` + **all** regenerated screenshots, with the sample-verification noted in the story/commit; the a1-d.png export failure is resolved or explained.
5. **Given** scope, **then** zero changes under `apps/*` in the diff.

## Tasks / Subtasks (designer)

- [x] (AC #1) Pencil MCP: `get_variables` snapshot → `SetVariables({"text-muted": {type:"color", value:"#A0AABE"}})` (merge) → re-read to confirm; spot-check 2–3 muted-heavy frames visually.
- [x] (AC #2) `batch_design`: insert `DownloadCard-v2 cell` into `ISilG` (ref `Mz428`, narrowed width if needed, Noto 12 `$text-muted` caption); `snapshot_layout` problemsOnly clean; screenshot the row.
- [x] (AC #3) Audit: diff `get_editor_state` reusable-component list vs library cells; register or file any other unregistered `*-v2` component.
- [x] (AC #4) User Cmd+S → full regen → sample-verify → stage ALL changed PNGs + `.pen` → commit `chore(design-system): sync .pen text-muted to R5 #A0AABE + register DownloadCard-v2` → PR.
- [x] (AC #4) Retry/resolve `flow-a-browse/a1-d.png` export failure.
- [ ] Close-out: mark sprint-status entry done after merge.

## Dev Notes

- **Authority chain:** DL-v2 §2.2 R5 decision (2026-06-13) → `styles.css:31` (already adopted) → `.pen` variable (this story). §2.1: "styles.css stays the single token file… never forks a second palette" — the `.pen` mirror lagging IS a fork until this ships.
- **Registration convention:** 13-0 changelog + memory `feedback_design_system_conformance_pen` (cell anatomy, `content-cards-v2` row `ISilG`, library frame `sJzat`).
- **Pencil mechanics:** `SetVariables` values must be `{type, value}` objects; variable names without `$` prefix in definitions. Edits need **manual Cmd+S** before screenshot regen (memory: _pencil-mcp-edits-need-manual-save_).
- **Do NOT** "fix" per-frame literals (e.g. `#0F1420AA` scrims, gradient stops, canvas-annotation greys `#888888`/`#666666`/`#222222`) — out of scope; only the variable.

## Change Log

| Date       | Change                                                                                                        |
| ---------- | ------------------------------------------------------------------------------------------------------------- |
| 2026-07-04 | Story created (Sally, from the 13-0 review's deferred findings; both debts verified against `.pen` + `styles.css`). Status → ready-for-dev. |
| 2026-07-04 | Executed (Sally, same day). AC1: `text-muted` → `#A0AABE` via merge SetVariables; re-read confirmed all 35 other variables untouched. AC2: `DownloadCard-v2 cell` (ref `Mz428` @480) registered in `ISilG`; row renders clean. AC3 audit: ALL `Component/*-v2` now have library cells; only non-cell reusable is legacy `s11main` (a screen marked reusable, not a `Component/*` — out of scope, noted). AC4: full regen **128/128** (a1-d.png succeeded on retry — 2026-07-04 failure was transient); brightness sample-verified old-vs-new on `i1-d.png` (rail facet counts + card meta visibly brighter, AA-passing); ALL PNGs staged per the documented inversion. Pre-existing finding (not fixed, v1 scope): `EmptyLibrary-NoQBT` internal icon 54px in 40px box → partially clipped (`snapshot_layout`); untouched to avoid v1 re-render churn. Status → review. |
