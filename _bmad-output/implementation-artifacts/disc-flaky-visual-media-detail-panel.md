# Story disc-flaky-visual-media-detail-panel: Deterministic Image-Load Fallback for MediaDetailPanel (fix flaky visual baseline)

Status: in-progress — Dev implementation complete (Tasks 1, 2, 3.1–3.3); AC #1–#4 verified. HALTED at the Sally UX gallery gate (Task 4 / AC #5); baseline commit (3.4) + `-linux` CI rebless (Task 5 / AC #6) sequenced after sign-off.

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

- [x] **Task 1: Add deterministic `onError` fallback to MediaDetailPanel images** (AC: #1, #2, #3)
  - [x] 1.1 In `apps/web/src/components/media/MediaDetailPanel.tsx`, add `backdropError` + `posterError` state (mirror `PosterCard.tsx:55-56` `imageError` pattern). Hooks before any early return (rules-of-hooks; note the `isLoading || !details` early return at L54). — added both `useState(false)` immediately after `subtitleDialogOpen`, before the L54 early return.
  - [x] 1.2 Backdrop: render the `<img onError={() => setBackdropError(true)}>` only while `backdropUrl && !backdropError`; otherwise render a `<div>` with the 135° gradient (`#4338CA → #6D28D9 → #7C3AED`) at the same `h-48 w-full` box. Keep the existing bottom gradient overlay. — gradient encoded as module-level `IMAGE_FALLBACK_GRADIENT` inline-style (matches `ColorPlaceholder.tsx`); backdrop `<img>` got `data-testid="detail-backdrop"` for test targeting; fallback `data-testid="detail-backdrop-fallback"`.
  - [x] 1.3 Poster: render the `<img onError={() => setPosterError(true)}>` only while `posterUrl && !posterError`; otherwise render the deterministic initial-letter fallback (circle `#FFFFFF18` + first char of `title`, `#FFFFFFCC`, 700) sized to `h-48 w-32 rounded-lg`. `data-testid="detail-poster-fallback"`. — outer `posterUrl &&` preserved so case-A (null path) still renders nothing; circle is `w-20 h-20`, `text-4xl` (36px) `font-bold`, painted over the same gradient slot bg.
  - [x] 1.4 Keep `loading="lazy"` on the happy-path `<img>`. Do NOT add a second network request for the fallback (pure CSS/text only — that is what makes it deterministic). — both fallbacks are pure inline-style + text; happy-path imgs retain `loading="lazy"`.
  - [x] 1.5 Update the Rule 21 header comment (`MediaDetailPanel.tsx:1`) to also reference the fallback spec. Verify `local/implements-pen-node-id` ESLint rule still passes. — ⚠️ DEVIATION: kept the literal `Screen ` token required by `DESIGN_REF_RE` (`implements-pen-node-id.js:60`) → header set to `// Design ref: ux-design.pen Screen B3-D Detail Panel (RgSxQ) + B9-D image-load fallback (Tn4Gz)`. The story's verbatim text (`ux-design.pen B3-D …`, no `Screen`) would FAIL the same ESLint rule this subtask mandates passing. ESLint exit 0 verified.

- [x] **Task 2: Unit tests** (AC: #1, #2, #3)
  - [x] 2.1 In `MediaDetailPanel.spec.tsx`, added: firing `onError` on the backdrop hides the `<img>` and shows the gradient fallback; firing `onError` on the poster shows `detail-poster-fallback` with the title initial (`測` of `測試電影`). Specific matchers (`toBeInTheDocument`, `queryBy…not.toBeInTheDocument`, `toHaveTextContent`) per Rule 16.
  - [x] 2.2 Assert the happy path unchanged: with images present and no error, `detail-poster` + `detail-backdrop` `<img>` are in the document and no fallback element is. Plus a case-A guard (null posterPath → neither poster img nor fallback). 49/49 spec tests pass.

- [ ] **Task 3: Visual fixture + baseline rebless** (AC: #4, #6) — _3.1–3.3 done; 3.4 BLOCKED on Sally (Task 4)_
  - [x] 3.1 Confirmed the `media-media-detail-panel` fixture (`-gallery.fixtures.tsx:2375`) still feeds TMDb `posterPath`/`backdropPath` (`/gajva2…jpg` + `/ilRyazd…jpg`); under `abortTmdbImages()` they now deterministically hit the new `onError` fallback. **No fixture change needed.**
  - [x] 3.2 **Burn-in:** ran `pnpm run test:visual` **3×** consecutively — **zero-diff each run** (43.7s / 42.3s / 42.2s, all `1 passed`). Flake eliminated → AC #4 satisfied.
  - [x] 3.3 `pnpm run test:visual:update` regenerated `components/media-media-detail-panel/{default,hover,focus}-visual-darwin.png`. **Only these 3 baselines changed** (`git status` = 3 files, ZERO unrelated re-render noise) → no revert needed.
  - [ ] 3.4 Commit the darwin baselines as a **separate** `test(visual): rebaseline media-media-detail-panel …` commit — **BLOCKED: pending Sally's gallery sign-off (Task 4) per the story's own ordering. Baselines are regenerated + burn-in-proven in the working tree, uncommitted.**

- [ ] **Task 4: UX (Sally) gallery review** (AC: #5) — _BLOCKED: requires the Sally (UX) persona / human gate_
  - [ ] 4.1 Sally reviews the regenerated `B9-D`/`B9-M` + `media-media-detail-panel` darwin renders against the B9 spec tokens. APPROVE before any rebless commit. **Dev self-check (Step 9) PASS — see UX Verification table; poster-slot uses the same 135° gradient bg + translucent circle, the one reconciliation decision flagged for Sally's confirmation.**

- [ ] **Task 5: `-linux` rebless via CI incremental bootstrap** (AC: #6) — _BLOCKED: post-merge CI + owner admin-merge_
  - [ ] 5.1 Delete the 3 stale `media-media-detail-panel/*-linux.png` so main's `update-missing` incremental bootstrap regenerates them post-merge. **Mechanism + precedent: `disc-nav-entry-discover-route.md` Task 4.3 / 4.3a–c.** (Not done — sequenced after Sally sign-off + PR.)
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

Amelia (Developer Agent) — `claude-opus-4-8[1m]` via BMM `/dev-story` workflow, 2026-06-05.

### Debug Log References

- Targeted spec: `vitest run MediaDetailPanel.spec.tsx` → **49/49 pass** (45 existing + 4 new).
- Full web gate: `pnpm nx test web` → **1965/1965** (one transient `InstantSearchBar` debounce flake on the first run; passed 2/2 in isolation AND on a clean full re-run — filed `preexisting-fail-instant-search-debounce-flake: backlog`).
- Full api gate: `pnpm nx test api` → **PASS**.
- ESLint (`MediaDetailPanel.{tsx,spec.tsx}`) → exit 0 (Rule 21 header accepted). Prettier `--check` → clean.
- Visual: `test:visual:update` → only the 3 `media-media-detail-panel` darwin baselines changed (zero unrelated noise). Burn-in `test:visual` ×3 → zero-diff each (43.7s / 42.3s / 42.2s). No orphaned processes (ports 4200/8080 clean, cleanup script clean).

### Completion Notes List

- ✅ **AC #1 (backdrop fallback):** on `onError`, the backdrop `<img>` is replaced by a deterministic 135° gradient `<div>` (`#4338CA → #6D28D9 → #7C3AED`, `data-testid="detail-backdrop-fallback"`) — no network, no `alt`, no broken-image glyph.
- ✅ **AC #2 (poster fallback):** on `onError`, the poster `<img>` is replaced by an initial-letter fallback (`data-testid="detail-poster-fallback"`) — a `w-20 h-20` circle (`#FFFFFF18`) with the title's first char (`#FFFFFFCC`, `text-4xl`/36px, `font-bold`) over the same gradient slot, sized `h-48 w-32 rounded-lg`. Case-A (null `posterPath`) still renders nothing (outer `posterUrl &&` preserved).
- ✅ **AC #3 (happy path):** real images render exactly as before; `loading="lazy"` retained; the only added attr is `onError` + a `data-testid="detail-backdrop"` for test targeting.
- ✅ **AC #4 (determinism):** burn-in ×3 zero-diff. The fallback is pure inline-style + text → no aborted-`<img>`-settle race (the bugfix-10-4 class).
- ⏸️ **AC #5 (Sally gate):** Dev self-check PASS (table below); awaiting Sally's gallery sign-off (Task 4) — the only design judgment deferred is the poster-slot background (I used the same 135° gradient + translucent circle to match the B9-D illustration; per the story's design-reconciliation note Sally confirms the exact treatment).
- ⏸️ **AC #6 (`-linux` CI):** not started — sequenced after Sally sign-off → baseline commit (3.4) → PR → main `update-missing` incremental bootstrap (Task 5).
- ⚠️ **Header deviation (Task 1.5):** kept the literal `Screen ` token the ESLint `DESIGN_REF_RE` requires (`implements-pen-node-id.js:60`); the story's verbatim header text omitted it and would have failed the same rule the subtask mandates passing. Final header: `// Design ref: ux-design.pen Screen B3-D Detail Panel (RgSxQ) + B9-D image-load fallback (Tn4Gz)`.
- 🔗 **AC Drift: NONE** (checked: `onError|imageError|posterError|backdropError|detail-poster-fallback` across `_bmad-output/implementation-artifacts/*.md` — hits are prior per-component fallbacks (PosterCard 2-3/bugfix-10-4/10-7, HeroBanner 10-2), all REUSE of the same `imageError` pattern; none alter `MediaDetailPanel`'s prior contract. Happy-path AC #3 explicitly preserves the existing render — this is additive error-path behavior on a component that previously had none).
- 📎 **Contract Stamps: NONE** (no `[@contract-v*]` stamps in this story; upstream `disc-nav-entry-discover-route` + 19-4/19-5 baseline convention carry no stamps either — pre-Rule-20 / implicit v0. Normal for a baseline-rebless story that defines no wire contract).
- 🔒 **Rule 7 Wire Format: N/A** (pure frontend; no Go error codes).
- 🕑 **Rule 23: N/A** (no new wall-clock read; `MediaDetailPanel.tsx:253` `new Date(createdAt)` parses a fixed prop → wall-clock-independent; baseline date renders fixed `2024/3/20`).

#### 🎨 UX Verification (Step 9 — Dev self-check vs `flow-b-detail-interaction/b9-d.png`)

| Area | Design Spec (B9-D) | Implementation (regenerated darwin baseline) | Match? | Fix Needed |
|------|--------------------|----------------------------------------------|--------|-----------|
| Backdrop fail | 135° gradient `#4338CA→#6D28D9→#7C3AED` | gradient `<div>` via `IMAGE_FALLBACK_GRADIENT` inline style | ✅ | — |
| Poster fail — circle | translucent circle `#FFFFFF18` | `w-20 h-20 rounded-full`, `backgroundColor:#FFFFFF18` | ✅ | — |
| Poster fail — initial | first char, `#FFFFFFCC`, 36/700 | `銀` (of `銀翼殺手 2049`), `color:#FFFFFFCC`, `text-4xl font-bold` | ✅ | — |
| Poster-slot bg | (B9 illustrates circle on gradient) | same 135° gradient slot bg | ⚠️ flag for Sally | confirm at AC #5 |
| Happy path | unchanged | title/rating/genres/overview/cast/buttons/file-info all intact | ✅ | — |

→ `🎨 UX Verification: PASS (Dev self-check)` — one reconciliation decision (poster-slot gradient bg) explicitly flagged for the Sally gate (AC #5).

### Discovery Triage

<!-- Rule 24 (project-context.md). Forward-only. -->

- **Did this story discover any work outside its current scope?**
  - **YES — all already triaged & resolved this session (during story authoring):**
    - **① expand-scope-in-place — `scripts/export-pen-screenshots.py` broken (Pencil 1.1.61 removed `--http`/`--http-port`):** rewritten to stdio JSON-RPC. Tracked + committed `5bace9e` (also `9ffdb85`). Resolved.
    - **① expand-scope-in-place — screenshot export non-deterministic (full regen re-renders every PNG):** documented in `.claude/memory/project_pen_flow_layout_convention.md` + MEMORY.md; mitigation = only commit genuinely-changed screens. Resolved (memory committed `b6ea1e7`).
    - **③ backlog/separate — canvas IA + design-guidelines cleanup (caption alignment, orphan nodes, imported-kit removal):** handled by Alexyu via Pencil in-app agent (A–J merged-block rollout + component-library cleanup), screenshot pipeline re-synced in `9ffdb85`. No open `sprint-status` debt from this story.
- Reference: `project-context.md` Rule 24.

### File List

- `apps/web/src/components/media/MediaDetailPanel.tsx` — modified (Rule 21 header → B3-D + B9-D; `IMAGE_FALLBACK_GRADIENT` const; `backdropError`/`posterError` state; backdrop + poster `onError` → deterministic gradient / initial-letter fallbacks; `data-testid` `detail-backdrop` / `detail-backdrop-fallback` / `detail-poster-fallback`).
- `apps/web/src/components/media/MediaDetailPanel.spec.tsx` — modified (new `describe('Image-load fallback …')` block: 4 tests — backdrop fallback, poster fallback, happy path, case-A null path).
- `tests/visual/components.visual.spec.ts-snapshots/components/media-media-detail-panel/default-visual-darwin.png` — regenerated (burn-in-proven; **uncommitted, pending Sally AC #5**).
- `tests/visual/components.visual.spec.ts-snapshots/components/media-media-detail-panel/hover-visual-darwin.png` — regenerated (**uncommitted, pending Sally**).
- `tests/visual/components.visual.spec.ts-snapshots/components/media-media-detail-panel/focus-visual-darwin.png` — regenerated (**uncommitted, pending Sally**).
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — modified (story → `in-progress`; filed `preexisting-fail-instant-search-debounce-flake: backlog`).

### Change Log

| Date       | Change                                                                 |
| ---------- | --------------------------------------------------------------------- |
| 2026-06-05 | DEV Amelia `/dev-story`: implemented deterministic `onError` fallbacks on `MediaDetailPanel` backdrop (135° gradient) + poster (initial-letter circle) per B9-D spec; +4 unit tests (49/49 pass); regenerated 3 `media-media-detail-panel` darwin baselines; burn-in ×3 zero-diff (AC #1–#4 ✅). Full gates green (web 1965/1965, api PASS, ESLint+Prettier clean). Filed `preexisting-fail-instant-search-debounce-flake` backlog. **HALTED at Sally UX gate (Task 4 / AC #5)**; baseline commit (3.4) + `-linux` CI rebless (Task 5 / AC #6) pending. Header kept ESLint-required `Screen ` token (deviation noted). `ready-for-dev → in-progress`. |
| 2026-06-05 | SM Bob `/create-story` (YOLO): authored ready-for-dev. Root cause = raw `<img>` no `onError` → non-deterministic aborted-image paint (bugfix-10-4 class, NOT Rule 23). Fix = deterministic `onError` fallback per B9-D/B9-M spec (gradient backdrop + initial-letter circle). Frontend-only, single story. backlog → ready-for-dev. |
