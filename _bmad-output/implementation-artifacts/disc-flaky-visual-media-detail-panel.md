# Story disc-flaky-visual-media-detail-panel: Deterministic Image-Load Fallback for MediaDetailPanel (fix flaky visual baseline)

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

> **Origin:** Discovered by `disc-nav-entry-discover-route` (Rule 24 ③ — backlog-with-carry-forward-link).
> While re-blessing the `shell-tab-navigation` baseline, `pnpm run test:visual:update` ALSO re-emitted
> `components/media-media-detail-panel/{default,focus}-visual-darwin.png` despite **zero code change** to
> `MediaDetailPanel`. The re-emit was reverted (not blessed) and filed here. This is **flaky**, not real
> drift: the baseline is non-deterministic and will noise-fail a future rebless. Non-blocking, frontend-only.

## Story

As a Vido maintainer,
I want `MediaDetailPanel`'s poster and backdrop images to render a **deterministic** fallback when the image fails to load,
so that its visual-regression baseline is stable (no flaky re-emits) and a broken/missing image shows a designed placeholder instead of the browser's native broken-image glyph or raw `alt` text.

## Root Cause (confirmed this session)

- `apps/web/src/components/media/MediaDetailPanel.tsx` renders the **backdrop** (`MediaDetailPanel.tsx:81-86`) and **poster** (`MediaDetailPanel.tsx:98-106`) as **raw `<img loading="lazy">` with NO `onError` handler**.
- `getImageUrl()` (`apps/web/src/lib/image.ts`) composes TMDb URLs (`https://image.tmdb.org/t/p/...`).
- The visual harness `abortTmdbImages()` (`tests/visual/components.visual.spec.ts:68-76`) aborts every `image.tmdb.org` request so renders are deterministic — but it relies on each component painting a **deterministic error fallback**. `MediaDetailPanel` has none, so the aborted `<img>` settles to the browser's native broken-image state (the poster carries `alt={title}` → may paint the alt text "銀翼殺手 2049"), and the screenshot **races that settle** → non-deterministic pixels → flaky baseline.
- **NOT Rule 23** (time drift): the fixture feeds a fixed `createdAt: '2024-03-20T10:30:00Z'`, and `MediaDetailPanel.tsx:253` `new Date(createdAt).toLocaleDateString('zh-TW')` parses a fixed string (argument present → wall-clock-independent). This is the **bugfix-10-4 async-image class**.
- **Precedent:** `PosterCard.tsx:160-183` already solves this exact class — `onError={() => setImageError(true)}` swaps the `<img>` for a deterministic CSS+text fallback (🎬 on `bg-[var(--bg-tertiary)]`). `MediaDetailPanel` must adopt the same pattern, styled per the new design spec.

## Design Spec (authored this session — Pencil)

The fallback visual is specced on the canvas (read-only via Pencil MCP):

- **`B9-D` (node `Tn4Gz`)** — desktop image-load fallback spec. Screenshot: `_bmad-output/screenshots/flow-b-detail-interaction/b9-d.png`.
- **`B9-M` (node `jH6rM`)** — mobile (bottom-sheet) variant. Screenshot: `_bmad-output/screenshots/flow-b-detail-interaction/b9-m.png`.

Fallback tokens (deterministic — pure CSS + text, **no second network request**):

- **Backdrop fail →** 135° linear gradient `#4338CA → #6D28D9 → #7C3AED` (reuses Screen `B7` / pending-fallback token).
- **Poster / image fail →** initial-letter treatment: a circle `fill #FFFFFF18`, the title's **first character** centered, `#FFFFFFCC`, Noto Sans TC 36 / 700. Scaled/placed into the poster slot.
- **case A vs case B:** "no TMDb metadata at all" (parsing / failed) is **case A** → handled by `FallbackPending`/`FallbackFailed` (Screens `B6`/`B7`), NOT this story. This story is **case B** only: metadata exists, only the image file failed.

> **Design-reconciliation note (for the Sally gate):** the component has BOTH a backdrop `<img>` AND a poster thumbnail `<img>`; the B9 spec illustrates the gradient backdrop + initial-letter circle. Dev applies: backdrop-fail → gradient; poster-fail → initial-letter circle/box sized to the `h-48 w-32` poster slot. Sally confirms the exact poster-slot treatment during the gallery review (AC #5).

## Acceptance Criteria

1. Given the backdrop image fails to load (`onError`), when `MediaDetailPanel` renders, then the broken `<img>` is **not** shown and a deterministic 135° gradient backdrop (`#4338CA → #6D28D9 → #7C3AED`) is shown in its place — no network request, no `alt` text, no native broken-image glyph.
2. Given the poster image fails to load (`onError`), when `MediaDetailPanel` renders, then the broken `<img>` is **not** shown and a deterministic initial-letter fallback (per B9 tokens) fills the `h-48 w-32` poster slot — no network request, no native broken-image glyph.
3. Given an image loads successfully, when `MediaDetailPanel` renders, then the real image is shown exactly as today (no visual change to the happy path; existing `loading="lazy"` retained).
4. Given the `media-media-detail-panel` visual fixture under `abortTmdbImages()`, when `pnpm run test:visual` runs **3–5 times consecutively** (burn-in), then `default` / `hover` / `focus` produce **zero pixel diff** against the newly-blessed baselines every run (flake eliminated).
5. Given the new fallback renders, when Sally reviews the `B9-D`/`B9-M` gallery renders, then the poster-slot + backdrop fallback treatment is **APPROVED** before any baseline is blessed (UX gate, per three-gate workflow).
6. Given the baseline is re-blessed, when CI runs, then both `-darwin` (local) and `-linux` (CI incremental-bootstrap) `media-media-detail-panel` baselines are committed and the `Visual Regression` check is green on main.

## Tasks / Subtasks

- [ ] **Task 1: Add deterministic `onError` fallback to MediaDetailPanel images** (AC: #1, #2, #3)
  - [ ] 1.1 In `apps/web/src/components/media/MediaDetailPanel.tsx`, add `backdropError` + `posterError` state (mirror `PosterCard.tsx:55-56` `imageError` pattern). Hooks before any early return (rules-of-hooks; note the `isLoading || !details` early return at L54).
  - [ ] 1.2 Backdrop (`MediaDetailPanel.tsx:81-86`): render the `<img onError={() => setBackdropError(true)}>` only while `backdropUrl && !backdropError`; otherwise render a `<div>` with the 135° gradient (`#4338CA → #6D28D9 → #7C3AED`) at the same `h-48 w-full` box. Keep the existing bottom gradient overlay.
  - [ ] 1.3 Poster (`MediaDetailPanel.tsx:98-106`): render the `<img onError={() => setPosterError(true)}>` only while `posterUrl && !posterError`; otherwise render the deterministic initial-letter fallback (circle `#FFFFFF18` + first char of `title`, `#FFFFFFCC`, Noto Sans TC 700) sized to `h-48 w-32 rounded-lg`. `data-testid="detail-poster-fallback"`.
  - [ ] 1.4 Keep `loading="lazy"` on the happy-path `<img>`. Do NOT add a second network request for the fallback (pure CSS/text only — that is what makes it deterministic).
  - [ ] 1.5 Update the Rule 21 header comment (`MediaDetailPanel.tsx:1`) to also reference the fallback spec: `// Design ref: ux-design.pen B3-D Detail Panel (RgSxQ) + B9-D image-load fallback (Tn4Gz)`. Verify `local/implements-pen-node-id` ESLint rule still passes.

- [ ] **Task 2: Unit tests** (AC: #1, #2, #3)
  - [ ] 2.1 In `apps/web/src/components/media/MediaDetailPanel.spec.tsx`, add: firing `onError` on the backdrop hides the `<img>` and shows the gradient fallback; firing `onError` on the poster shows `detail-poster-fallback` with the title initial. Use specific matchers (`toBeInTheDocument`, `toHaveTextContent`) per Rule 16.
  - [ ] 2.2 Assert the happy path unchanged: with images present and no error, `detail-poster` `<img>` is in the document and no fallback element is.

- [ ] **Task 3: Visual fixture + baseline rebless** (AC: #4, #6)
  - [ ] 3.1 Confirm the `media-media-detail-panel` fixture (`apps/web/src/routes/test/-gallery.fixtures.tsx:2375`) still feeds TMDb `posterPath`/`backdropPath`; under `abortTmdbImages()` they will now deterministically hit the new `onError` fallback. No fixture change expected (verify; if the fixture needs a tweak, keep it minimal).
  - [ ] 3.2 **Burn-in (Murat):** run `pnpm run test:visual` 3–5× — must be zero-diff each run BEFORE blessing. If any diff, the fallback still has a non-deterministic element — fix before proceeding.
  - [ ] 3.3 `pnpm run test:visual:update` to regenerate `components/media-media-detail-panel/{default,hover,focus}-visual-darwin.png`. Confirm ONLY this component's darwin baselines changed (revert any unrelated re-render noise — full regen is non-deterministic; see project-context Rule 22 tooling note + `.claude/memory` screenshot caveat).
  - [ ] 3.4 Commit the darwin baselines as a **separate** `test(visual): rebaseline media-media-detail-panel …` commit (NOT mixed with the component logic), and only AFTER Sally's gallery sign-off (Task 4). `tests/visual/README.md` commit discipline.

- [ ] **Task 4: UX (Sally) gallery review** (AC: #5)
  - [ ] 4.1 Sally reviews the regenerated `B9-D`/`B9-M` + `media-media-detail-panel` darwin renders against the B9 spec tokens. APPROVE before any rebless commit. Remove `requires-manual-review` on the PR once approved.

- [ ] **Task 5: `-linux` rebless via CI incremental bootstrap** (AC: #6)
  - [ ] 5.1 The PR `Visual Regression / PR` job is verify-only and cannot regenerate `-linux` from a darwin machine. Delete the 3 stale `media-media-detail-panel/*-linux.png` (stale→missing) so main's `update-missing` incremental bootstrap regenerates them post-merge and opens a `chore(visual): bootstrap … -linux baselines (incremental)` PR with `requires-manual-review`. **Exact mechanism + precedent: `disc-nav-entry-discover-route.md` Task 4.3 / 4.3a–c.**
  - [ ] 5.2 Sally re-approves the CI-generated `-linux` PNGs; owner admin-merges; confirm main `Visual Regression` goes steady-state green.

## Dev Notes

### What this story is (and is NOT)

- **IS:** a deterministic `onError` fallback on two `<img>` in one component + tests + a one-component baseline rebless. **Frontend-only. Zero backend tasks** (Cross-Stack Split check: PASS — single story).
- **IS NOT:** a change to `FallbackPending`/`FallbackFailed` (case A — no metadata), to `PosterCard`, to the visual harness `abortTmdbImages` strategy (rejected this session — global stub would rebless every image baseline), or to `getImageUrl`.

### Architecture Compliance

- **Reuse / pattern:** copy `PosterCard.tsx:55-56,160-183`'s `imageError` state + conditional-render approach. Do NOT invent a new mechanism (Rule 21 origin = bugfix-10-4 was an independently-invented divergent component).
- **Determinism is the contract:** the fallback MUST be pure CSS + text. Any image fill, `srcSet`, or network fetch in the fallback re-introduces the race. This is the whole point of the story.
- **Rule 23 (time):** `new Date(createdAt)` has an argument (parses a fixed prop) → wall-clock-independent → the `local/time-dependent-fixture-stability` ESLint rule should not flag it (it targets `Date.now()` / no-arg `new Date()`). See "Time-dependent visual coverage" below.
- **Shared component:** `MediaDetailPanel` renders on both desktop and the mobile bottom-sheet, so one component fix covers `B9-D` + `B9-M`.

### Project Structure Notes

- Touch: `apps/web/src/components/media/MediaDetailPanel.tsx` (logic) + `apps/web/src/components/media/MediaDetailPanel.spec.tsx` (tests) + `tests/visual/components.visual.spec.ts-snapshots/components/media-media-detail-panel/*` (rebless). Possibly `-gallery.fixtures.tsx` (only if Task 3.1 finds a needed tweak).
- `.pen` is **read-only** for this story (B9 spec already authored + screenshots committed). No `set_variables`/`batch_design` → CLAUDE.md screenshot-export workflow does NOT re-trigger here.

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()`?**
  - **NO new wall-clock read.** `MediaDetailPanel.tsx:253` already contains `new Date(createdAt)` but it parses a **fixed prop** (argument present), not the wall clock → `N/A — wall-clock-independent` (Time-bomb-exempt). This story adds only `onError` handlers + CSS/text fallbacks; no clock dependency. Rule 23 not applicable.
- Reference: `project-context.md` Rule 23; the fixture's `createdAt` is a fixed ISO string.

### References

- [Source: apps/web/src/components/media/MediaDetailPanel.tsx#L81-L106] — backdrop + poster `<img>` to add `onError` to
- [Source: apps/web/src/components/media/PosterCard.tsx#L55-L56,L160-L183] — `imageError` onError fallback precedent to mirror
- [Source: apps/web/src/lib/image.ts] — `getImageUrl` → TMDb URLs
- [Source: tests/visual/components.visual.spec.ts#L68-L76] — `abortTmdbImages` (why the fallback must be deterministic)
- [Source: apps/web/src/routes/test/-gallery.fixtures.tsx#L2375] — `media-media-detail-panel` fixture
- [Source: ux-design.pen — B9-D (Tn4Gz) / B9-M (jH6rM)] — image-load fallback spec
- [Source: _bmad-output/screenshots/flow-b-detail-interaction/b9-d.png, b9-m.png] — spec renders
- [Source: _bmad-output/implementation-artifacts/disc-nav-entry-discover-route.md#Task 4.3] — `-linux` incremental-bootstrap rebless mechanism + precedent
- [Source: project-context.md — Rule 16, Rule 21, Rule 22, Rule 23] — assertion quality / pen-node header / visual baselines / time-fixtures
- [Source: _bmad-output/implementation-artifacts/sprint-status.yaml — disc-flaky-visual-media-detail-panel] — origin tracking entry

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Discovery Triage

<!-- Rule 24 (project-context.md). Forward-only. -->

- **Did this story discover any work outside its current scope?**
  - **YES — all already triaged & resolved this session (during story authoring):**
    - **① expand-scope-in-place — `scripts/export-pen-screenshots.py` broken (Pencil 1.1.61 removed `--http`/`--http-port`):** rewritten to stdio JSON-RPC. Tracked + committed `5bace9e` (also `9ffdb85`). Resolved.
    - **① expand-scope-in-place — screenshot export non-deterministic (full regen re-renders every PNG):** documented in `.claude/memory/project_pen_flow_layout_convention.md` + MEMORY.md; mitigation = only commit genuinely-changed screens. Resolved (memory committed `b6ea1e7`).
    - **③ backlog/separate — canvas IA + design-guidelines cleanup (caption alignment, orphan nodes, imported-kit removal):** handled by Alexyu via Pencil in-app agent (A–J merged-block rollout + component-library cleanup), screenshot pipeline re-synced in `9ffdb85`. No open `sprint-status` debt from this story.
- Reference: `project-context.md` Rule 24.

### File List

### Change Log

| Date       | Change                                                                 |
| ---------- | --------------------------------------------------------------------- |
| 2026-06-05 | SM Bob `/create-story` (YOLO): authored ready-for-dev. Root cause = raw `<img>` no `onError` → non-deterministic aborted-image paint (bugfix-10-4 class, NOT Rule 23). Fix = deterministic `onError` fallback per B9-D/B9-M spec (gradient backdrop + initial-letter circle). Frontend-only, single story. backlog → ready-for-dev. |
