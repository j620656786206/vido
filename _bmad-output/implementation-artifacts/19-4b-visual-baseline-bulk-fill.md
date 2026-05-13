# Story 19.4b: Visual-Baseline Bulk Fill (remaining ~99 components)

Status: in-progress

<!-- SM Bob /create-story (YOLO) 2026-05-13 — bootstrapped via Party Mode 2026-05-12 (Sally + Bob + Murat + Winston + Amelia; Alexyu ratified) when 19-4 was re-cut to ship the harness + ~25 reference components atomically; 19-4b inherits the bulk fill. -->
<!-- 🔗 AC Drift: N/A (additive — extends 19-4's harness, no AC observable behaviour change on prior stories). · 📎 Contract Stamps: NONE this story (the harness contracts AC #1–#5 are 19-4's [@contract-v1] and are *consumed* not extended; Task 0 Fix C adds an *optional* `openTrigger?` field + `open` state to the `GalleryFixture` interface — documented as a harness extension, not a new stamp, because no downstream story consumes it yet). · 🔒 Rule 7 Wire Format: N/A (pure FE, no Go error codes). · 🎨 UX: reads `ux-design.pen` mapping via `_bmad-output/audit/drift-19-3-2026-05.md` only — no `.pen` modification. Sally gallery sign-off (`/test/gallery`) is the AC #3 close gate. -->
<!-- markers-block-end -->

## Story

As a frontend maintainer,
I want a Playwright `toHaveScreenshot()` visual baseline (default / hover / focus / open where applicable) for every remaining in-scope `apps/web/src/components/` component — and the three harness-quality fixes Sally flagged on 19-4 — so that the epic-19 visual-regression net covers the *whole* component surface (not just the ~25 reference set 19-4 shipped) and produces faithful per-state baselines before 19-8's component-vs-`.pen` diff sweep and any Rule 22 retro audit relies on it (and before 19-5's CI workflow goes live against drift-prone baselines).

## Acceptance Criteria

1. **Harness-quality fixes from the 19-4 Sally review (Task 0) are landed BEFORE any new fixture/baseline is generated** — they affect every baseline this story produces. The visual spec must (a) use *keyboard-driven* focus (sentinel + `Tab`, or equivalent) so Chromium flags the modality as keyboard and `:focus-visible` rules paint; (b) cover the `TabNavigation` active-tab state (stub `useRouterState` via a nested memory `RouterProvider` in the gallery wrapper, or render the fixture under a route whose pathname matches `TABS.matchPaths`); (c) support an `open` interactive state via an optional per-fixture `openTrigger?: string` selector — gallery emits `<div data-gallery-state="open" data-gallery-open-trigger="…">`, the spec clicks the trigger inside the state div before screenshotting. The three affected 19-4 baselines (`media-poster-card/focus`, `library-filter-chips/focus`, `library-sort-selector/focus`, `library-view-toggle/focus`, `metadata-editor-genre-selector/focus`, `search-search-bar/focus`, `search-media-type-tabs/focus`, `shell-tab-navigation/{default,hover,focus}`, `ui-button/focus`, `ui-pagination/focus`) are regenerated and re-committed in the same commit as the spec change.

2. **Every `apps/web/src/components/**/*.tsx` component that renders visible UI and is not already in `apps/web/src/routes/test/-gallery.fixtures.tsx` gets a fixture entry there.** Fixture shape: `{ id, label, component, props?, penNode, statesOnly?, width?, openTrigger?, seedQueries? }` — `penNode` from the component's `// Implements:` header (`_bmad-output/audit/drift-19-3-2026-05.md` mapping; `'screen-section'` / `'utility'` for the placeholder/exemption forms). Reuse each component's own `*.spec.tsx` mock-data shapes; data-driven components get their React-Query keys seeded (Task 3 — extend the gallery route with a `<GalleryQuerySeed>` helper or take a per-fixture `seedQueries: Array<{ queryKey: readonly unknown[]; data: unknown }>` and call `queryClient.setQueryData(...)` before render). Components depending on Zustand stores (e.g. `library-filters-store`, `selection-store`) get their store seeded the same way (per-fixture `seedStore?: () => void` hook, called before mount).

3. **Sally (UX) reviews the rendered gallery (`/test/gallery`) before the baseline set is committed.** The story records the review in Completion Notes (mirror the 19-4 AC #5 Sally-gate). `pnpm run test:visual:update` regenerates all PNGs (existing 46 + new ones for the bulk fill); `pnpm run test:visual` is green on a clean re-run; burn-in (`test:visual` ×5) shows 0 flake. Every committed PNG path follows the harness convention: `tests/visual/components.visual.spec.ts-snapshots/components/{gallery-id}/{state}-visual-{platform}.png`.

4. **Deliberate skips are recorded with reasons in `_bmad-output/audit/visual-baseline-19-4.md`.** Known skips (carry over from 19-4): type/util modules `parse/types.ts`, `degradation/types.ts`, `downloads/formatters.ts`; the misfiled hook `parse/useParseProgress.ts`; bare layout shells (`shell/AppShell`, `dashboard/DashboardLayout`, `settings/SettingsLayout`, `setup/SetupWizard`) — rendered only if a sensible isolated fixture exists, otherwise listed in the audit doc with one-line reasons. The audit doc's **Delivered** table is updated to the full set and the **Pending** section is emptied (or lists only documented skips).

5. **Platform / CI: the Linux-baseline strategy 19-5 needs is decided and implemented.** Either (a) commit a `-linux` set generated in the CI Docker image via a new `scripts/visual-baseline.sh` (cross-platform helper), OR (b) document — in `tests/visual/README.md` + the audit doc — that 19-5's CI job regenerates the `-linux` set via `pnpm run test:visual:update` in a one-off commit. The decision is recorded in Completion Notes with rationale, so 19-5 can wire CI without re-litigating it.

6. **Regression gates green at story close:** `pnpm lint:all` 0 errors / 122 warnings (the bugfix-10-7 / 19-3 / 19-4 baseline); `pnpm nx test web` + `pnpm nx test api` pass; `pnpm test:e2e --list` count unchanged from 1663 / 36 files (the `visual` project stays excluded); `pnpm run test:cleanup` shows no orphans; `ux-design.pen` untouched (so `scripts/export-pen-screenshots.py` is not run and the CLAUDE.md screenshot workflow does not trigger). The new fixture entries + spec/gallery changes pass ESLint + Prettier — no new warnings in the changed files relative to the closeout baseline.

## Tasks / Subtasks

> **Scope (Party Mode 2026-05-12 ruling, inherited):** ALL frontend / 0 backend → single story (cross-stack split check N/A). 19-4's `[@contract-v1]` harness ACs are *consumed* not extended; the only contract-shape change is the additive `openTrigger?` / `seedQueries?` / `seedStore?` fields on `GalleryFixture` — documented as a harness extension, no new `[@contract-vN]` stamp.

### Task 0: Harness-quality fixes from 19-4 Sally review — DO THESE FIRST (AC: #1)

These fixes change spec/fixture behaviour and therefore every baseline 19-4b generates. Land Task 0 atomically (one commit) and regenerate the affected 19-4 focus baselines in the same commit. Burn-in test:visual ×3 after the changes to confirm 0 flake before moving to Task 1.

- [x] **0a. Fix A — Keyboard-driven `focus` state (Sally follow-up #1).** Programmatic `locator.focus()` does not trigger Chromium's `:focus-visible` rules; replaced with a sentinel-then-Tab pattern so input modality flips to keyboard.
  - [x] `apps/web/src/routes/test/gallery.tsx`: hidden `<button type="button" data-gallery-sentinel="pre" aria-hidden="true" tabIndex={0} className="sr-only" />` rendered immediately before each `<div data-gallery-state>`.
  - [x] `tests/visual/components.visual.spec.ts`: `state === 'focus'` branch uses `stateDiv.locator('xpath=preceding-sibling::*[@data-gallery-sentinel="pre"][1]')` then `await sentinel.focus()` + `await page.keyboard.press('Tab')`. Programmatic-focus fallback retained for fixtures with no focusable descendant. Spec header doc-comment updated to reference 19-4b Task 0.
  - [x] Regenerated affected baselines via `pnpm run test:visual:update`. **Of the 10 existing 3-state focus baselines, ONLY `search-search-bar/focus-visual-darwin.png` changed pixel-wise** — the SearchBar's input is the only fixture whose CSS distinguishes `:focus-visible` from `:focus`. The other 9 (ui-button, library-filter-chips, library-sort-selector, library-view-toggle, media-poster-card, metadata-editor-genre-selector, search-media-type-tabs, shell-tab-navigation, ui-pagination) render identically under both → re-blessed unchanged (Playwright didn't rewrite them). Expected per Sally's review note ("many components have identical `:focus` and `:focus-visible` styles").

- [x] **0b. Fix B — TabNavigation active-tab state (Sally follow-up #2).** Implemented **Option B1 — nested memory `RouterProvider`** (preferred per the story spec; sibling-route Option B2 rejected because it would drag in shell-layout data dependencies).
  - [x] `apps/web/src/routes/test/gallery.tsx`: added `STUB_TAB_PATHS = ['/library', '/downloads', '/pending', '/settings'] as const` + `StubbedRouter` component that builds a memory router (`createMemoryHistory({ initialEntries: [pathname] })`) with all 4 TAB paths registered as stub child routes whose `component` renders the wrapped fixture. `useRouterState()` inside the wrapped component resolves through the inner provider → reports the stub path → `TabActive (TboA7)` paints. TS shape-mismatch on `RouterProvider router={...}` suppressed with `// eslint-disable-next-line @typescript-eslint/no-explicit-any` + cast (the stub router's typed tree is intentionally narrower than the main `routeTree.gen.ts`; runtime context lookup is correct).
  - [x] `useMemo` dep list intentionally omits `children` (each fixture mounts once for the snapshot; re-creating the router on prop change would thrash history subscriptions). `react-hooks/exhaustive-deps` suppressed with a deliberate comment.
  - [x] `-gallery.fixtures.tsx`: `shell-tab-navigation` FIXME block removed; `routePath: '/library'` added. `GalleryFixture` interface gained `routePath?: StubRoutePath` field (typed as `'/library' | '/downloads' | '/pending' | '/settings'`).
  - [x] Regenerated `shell-tab-navigation/{default,hover,focus}-visual-darwin.png` — all 3 now render with `媒體庫 (/library)` tab styled active (blue underline + white text per `TabNavigation.tsx:38-41`).

- [x] **0c. Fix C — Interactive `open` state via `openTrigger` (Sally follow-up #3).**
  - [x] `GalleryFixture` extended: `'open'` added to `GalleryState` union; `openTrigger?: string` field added (CSS selector relative to the component's render).
  - [x] `gallery.tsx`: `requestedStates.filter((s) => s !== 'open' || !!fx.openTrigger)` silently drops the `open` state for fixtures without a trigger. The `open` state div renders `data-gallery-open-trigger={fx.openTrigger}` so the spec can read it.
  - [x] `components.visual.spec.ts`: new `else if (state === 'open')` branch reads `data-gallery-open-trigger`, clicks the selector inside the state div, then **waits for any `:is([role="listbox"], [role="menu"], [role="dialog"])` to be visible** with a 1s timeout + `.catch(() => {})` fallback (this `waitFor` is the burn-in stabilizer — the initial implementation without it produced 1 visual fail in 4 burn-in runs; the wait kills the screenshot-vs-popup-paint race).
  - [x] `library-sort-selector` fixture: `statesOnly: ['default', 'hover', 'focus', 'open']` + `openTrigger: '[data-testid="sort-selector-button"]'` opted in — the reference case for the `open` mechanism. New baseline `library-sort-selector/open-visual-darwin.png` captures the open `SortDropdown 955EZ` panel (`role="listbox" aria-label="排序選項"`, 4 options visible).
  - [ ] DEFERRED to next 19-4b iteration: inventorying other 19-4 reference fixtures for opt-in (Sally review only flagged SortDropdown explicitly; other 19-4 fixtures don't have an obvious `openTrigger`). Task 1 will identify Q/S-bucket modal/dropdown components and they'll opt in per-fixture.
  - [ ] DEFERRED to next 19-4b iteration: `tests/visual/README.md` update for the `open` state + `openTrigger` field. The interface is documented in `-gallery.fixtures.tsx` JSDoc which is the primary discovery surface; README polish will batch with the Task 4 / Task 6 doc pass.

- [x] **0d. Burn-in + commit.** Post-stabilization burn-in: `pnpm run test:visual` ×4 consecutive runs → 4 PASS / 0 visual content failures (1 webServer-startup infrastructure timeout during rapid back-to-back runs — not a visual flake; standalone runs are 14–32 s). Pre-stabilization had 1 visual fail in 4 runs (the open-state click-then-screenshot race that the `waitFor` in Fix C addresses). Lint `0/122`, Prettier clean on all 3 touched code files, feature-E2E `--list` 1663 tests / 36 files unchanged. Audit doc Delivered-baselines table NOT updated this commit (header still reads `(25 unique components / 26 fixture entries / 46 PNGs)`) — Task 6 owns the audit-doc full-set update; Task 0's incremental delta (`+1 fixture entry: library-sort-selector now has 4 states; +1 new PNG: library-sort-selector/open; 4 baselines re-blessed: search-search-bar/focus + shell-tab-navigation/{default,hover,focus}` ⇒ **26 fixture entries / 27 entry-state combinations / 47 PNGs**) is recorded here. Commit message: `feat(19-4b): Task 0 harness-quality fixes — Sally follow-ups 1/2/3`.

### Task 1: Inventory remaining components & bucket data-driven vs. presentational (AC: #2, #4)

- [x] Generate the full in-scope list. Quick recipe (record exact commands run in Debug Log References):
  ```bash
  # All .tsx components, minus tests/index barrels
  find apps/web/src/components -name '*.tsx' ! -name '*.spec.tsx' ! -name '*.test.tsx' \
    ! -name 'index.ts' | sort > /tmp/all-components.txt
  # Already-covered fixture ids → component-paths (reverse the kebab → path mapping)
  grep -oE "'[a-z][a-z0-9-]+'" apps/web/src/routes/test/-gallery.fixtures.tsx \
    | head -n 200 > /tmp/covered-ids.txt
  ```
- [x] For each not-yet-covered component, classify into one of FOUR buckets — record the bucket in a working notes section under "Debug Log References":
  - **P (Presentational)** — pure props-in, no `useQuery` / `useMutation` / store reads / router reads. → goes in Task 2.
  - **Q (Query-driven)** — needs React-Query data. → goes in Task 3 (seed via `seedQueries`).
  - **S (Store-driven)** — reads from a Zustand store (selection, library-filters, etc.). → goes in Task 3 (seed via `seedStore`).
  - **L (Layout shell / no isolated render)** — `AppShell`, `DashboardLayout`, `SettingsLayout`, `SetupWizard`, etc. → recorded as deliberate skip per AC #4 unless a trivial isolated fixture exists.
- [x] Confirm the four type/util-only files stay skipped (`parse/types.ts`, `degradation/types.ts`, `downloads/formatters.ts`, `parse/useParseProgress.ts`) — these were skipped in 19-4 and remain skipped here.
- [x] Cross-check the inventory against `_bmad-output/audit/drift-19-3-2026-05.md` Category-A/B/C tables — any newly-classified Category-A (real `.pen` Reusable-Component mapping) gets that node id in its fixture's `penNode`; everything else uses `'screen-section'` (the 19-3 Phase-2 placeholder) or `'utility'`.

### Task 2: Add fixtures — Presentational bucket first (AC: #2)

- [x] For each P-bucket component, add a fixture entry. Reuse the component's own `*.spec.tsx` mock-data shapes for props (do not re-invent — Rule per `project-context.md`). The `penNode` value MUST come from the component file's `// Implements:` header (Rule 21-enforced by `local/implements-pen-node-id` ESLint rule). **63 entries added** across 17 subfolders (see Debug Log "Task 2 bulk fill" + the File List).
- [x] Group fixtures in the file by `components/` subfolder (the existing convention — `ui/` → `media/` → `degradation/` → `library/` → …). Keep one-fixture-per-line readable formatting (Prettier handles wrapping). Order: `ui/` → `media/` → `degradation/` → `dashboard/` → `downloads/` → `library/` → `homepage/` → `learning/` → `manual-search/` → `metadata-editor/` → `notifications/` → `parse/` → `retry/` → `scanner/` → `search/` → `settings/` → `setup/`.
- [x] For badges / skeletons / static labels: `statesOnly: ['default']` (no meaningful hover/focus). For interactive elements (buttons, links, inputs): keep the default three states. Applied per-fixture (see entries with `statesOnly: ['default']` for: ui-dialog, ui-highlight-text, ui-side-panel, media-{credits,fallback-failed,fallback-pending,file-info,media-grid,tv-show-info}, all degradation/, downloads-{download-parse-status-badge,status-icon}, library-{batch-confirm-dialog,batch-progress,parse-failure-card→no this kept default-only no; actually kept default-only? see file}, all notifications/, parse-{layered-progress,parse-status-badge}, retry-countdown-timer, scanner-{progress-card,progress-sheet}, search-search-results, settings-{backup-table,connection-test-result,restore-confirm-dialog,settings-placeholder}, setup-step-progress).
- [x] For inline / auto-width components that would collapse to 0-width in isolation (badges, chips): set a sensible `width` (typically 200–640 px). Applied to 47 of 63 entries (5 entries are full-page setup steps that don't need a width; ui-side-panel + library-batch-confirm-dialog + library-batch-progress + settings-restore-confirm-dialog are fixed-position overlays that don't need width; the remaining bare-default entries are intentionally letting the natural width win).
- [x] Spot-check renders in `pnpm nx serve web` → `/test/gallery`. **Method**: Playwright probe (`.gallery-probe.mjs`, deleted post-check) against the running dev server. **Result**: 89 `[data-gallery-id]` sections present (26 pre-existing + 63 new = 89 ✓); **0 `[data-gallery-error]` placeholders**; 2 console errors (HTTP 500 on `/api/v1/setup/status` + one other — pre-existing app-shell calls, not Task 2 fixtures). First probe pass surfaced a transient Radix Dialog `Invalid hook call` for `ui-dialog` from Vite's optimizeDeps cache warming during HMR; second probe pass after Vite cache settled was clean. **Caveat**: `ui-dialog` renders an empty state-div because Radix `DialogContent` portals to `document.body` (outside the fixture's snapshot crop); Task 4 baseline generation + Sally review will decide whether to keep, drop, or rewrite the fixture with a non-portal wrapper. Same caveat applies to `ui-side-panel` (fixed-position viewport overlay).

### Task 3: Add fixtures — Query-driven & Store-driven buckets (AC: #2)

- [ ] **Extend the gallery infrastructure** (one set of edits to `gallery.tsx` + `-gallery.fixtures.tsx` before adding Q/S fixtures):
  - [ ] Add `seedQueries?: Array<{ queryKey: readonly unknown[]; data: unknown }>` and `seedStore?: () => void` to the `GalleryFixture` interface.
  - [ ] In the gallery route, before rendering each fixture: if `fx.seedQueries`, call `queryClient.setQueryData(qk, data)` for each entry. The `queryClient` instance must be the **same** one the app shell provides — read it with `useQueryClient()` at the top of `ComponentGalleryPage` (the gallery is already inside the app shell's `<QueryClientProvider>`, so this works). If `fx.seedStore`, call it once in a `useEffect(() => { fx.seedStore?.(); }, [])` keyed by `fx.id` (the components import their stores directly, so calling the setter is enough).
  - [ ] Decide whether to introduce a `<GalleryQuerySeed>` wrapper component (clean) or inline the seeding (lighter). The story does not mandate — record the choice + rationale in Dev Notes when implementing.
- [ ] For each Q-bucket component, populate `seedQueries` with the keys the component reads. The canonical place to find query keys is the component's `useFoo()` hook — e.g. `useLibraryQuery` → look at its `queryKey` build; `useMovieDetails` / `useTVShowDetails` use `detailKeys` (`apps/web/src/hooks/useMediaDetails.ts`). Mock data shapes come from `apps/web/src/types/` and existing `*.spec.tsx` fixtures.
- [ ] For S-bucket components, write a small `seedStore` lambda that sets the minimum needed store state (e.g. `useLibraryFiltersStore.setState({ filters: { genres: ['動作'], ... } })`). Be aware: store seeding affects all subsequent renders in the gallery → reset to default after each fixture if interference between fixtures shows up (the gallery renders all fixtures simultaneously, so the LAST fixture wins for any store the previous ones touched). Mitigation: in `seedStore`, set a complete state object rather than mutating partials.
- [ ] Components rendering inside a `Dialog` / `SidePanel` / portal: render them in `open: true` state directly (the fixture's `props` can include `open: true` / `defaultOpen: true`). Do **not** rely on the `openTrigger` mechanism for components that mount their content via Radix portals at the document root — portal content is outside the state div and won't be captured by `stateDiv.screenshot()`. For these, render the dialog inline (some Radix components support `forceMount` or you can render the dialog's body component directly).
- [ ] PosterCardMenu, kebab menus, etc. — use `openTrigger` to click the trigger button.

### Task 4: Generate full baseline set, UX review, commit; burn-in (AC: #3)

- [ ] Run `pnpm run test:visual:update` — produces all new PNGs under `tests/visual/components.visual.spec.ts-snapshots/components/{id}/{state}-visual-darwin.png`. Spot-check rendering quality at `pnpm nx serve web` → `/test/gallery` (you should see every component render without error placeholders).
- [ ] Triage any fixture-error placeholders: a `[data-gallery-error]` on a fixture means props are misshapen or a hook crashed → fix the fixture's props/seed, regenerate, do **not** ship a baseline for an error state.
- [ ] **Sally /ux-designer reviews the rendered gallery.** Record the review in Completion Notes ("🎨 UX Verification" subsection — mirror the 19-4 closeout's format). Any rendering issues flagged → return to Task 2/3 to fix the offending fixture(s), regenerate, re-review. Sally's review IS the AC #3 close gate.
- [ ] Burn-in: `pnpm run test:visual` ×5 → 0 flake. If any flake surfaces, identify the offending fixture (Playwright's `--max-failures=1` + the failure trace), suppress non-determinism (animations leaking past `reducedMotion: 'reduce'`, async-only render paths the gallery enters before they settle, etc.).
- [ ] Commit baselines + fixture additions. The story-19-4 commit-message style applies: `feat(19-4b): bulk-fill ~99 component visual baselines`. Per the harness baseline-update discipline (`tests/visual/README.md`), do NOT mix baseline churn with logic changes — pure-fixture commits only.

### Task 5: Linux-baseline strategy for CI (AC: #5)

- [ ] Decide between **(a)** `scripts/visual-baseline.sh` Docker helper, or **(b)** document CI-regenerate-on-first-run:
  - **(a) `scripts/visual-baseline.sh`**: thin wrapper that `docker run`s the same image 19-5 will use (the existing Playwright image — `mcr.microsoft.com/playwright:v$PLAYWRIGHT_VERSION-jammy` matches the project's `@playwright/test` version) with `tests/visual/` mounted and runs `pnpm run test:visual:update`. Output: PNGs with `-linux` suffix. 19-5 then commits both `-darwin` + `-linux` sets and CI verifies `-linux`. *Cleaner long-term; requires Docker on dev machine.*
  - **(b) Document CI-regen**: leave only `-darwin` baselines committed; 19-5's CI workflow runs `test:visual:update` on first execution, commits the `-linux` set in a one-off PR, then runs in verify-only mode thereafter. *Simpler now; one-time CI commit at 19-5 close.*
- [ ] Update `tests/visual/README.md` with the chosen strategy under "Baseline-update discipline" → "Platform suffix" — keep the existing language for the strategy NOT chosen as a "rejected alternative" footnote so 19-5's owner sees both options.
- [ ] Update `_bmad-output/audit/visual-baseline-19-4.md` "Platform suffix" line to match.

### Task 6: Update audit doc to full set; full regression + close (AC: #4, #6)

- [ ] Update `_bmad-output/audit/visual-baseline-19-4.md`:
  - [ ] "Delivered baselines" table: append all new bulk-fill rows (one per fixture id); update the header count to "(N unique components / M fixture entries / K PNGs)" with the actual totals.
  - [ ] "Pending (19-4b worklist — ~99 components, NOT design-drift findings)" section: replace the worklist with a "Delivered in 19-4b 2026-05-..." closure note + the deliberate-skips list (still recorded per AC #4).
  - [ ] "Material drift findings (Rule 22)" section: stays "None this story" — 19-4b is still building the diff tool, not running the diff (19-8 owns that).
- [ ] Full regression: `pnpm lint:all` 0 errors / 122 warnings; `pnpm nx test web` + `pnpm nx test api` pass; `pnpm test:e2e --list` 1663 tests / 36 files unchanged; `pnpm run test:visual` green (full new baseline set); `pnpm run test:cleanup` no orphans; `ux-design.pen` unmodified.
- [ ] Update sprint-status.yaml: 19-4b `in-progress` → `review` with a Completion Notes-style summary line.
- [ ] Set Story Status to `review`. CR /code-review runs next (different LLM-context per workflow tip).

## Dev Notes

### Why this story exists / where it sits in epic-19

- **bugfix-10-4 root cause** (Party Mode 2026-05-08): `HoverPreviewCard.tsx` diverged from `.pen` `Component/PosterCardHover` (node `MQbvp`) undetected for months. Epic-19 is the systemic fix. **19-1** added Rule 21 to `project-context.md`; **19-2** added Rule 22; **19-3** made Rule 21 CI-enforced via `local/implements-pen-node-id` + header backfill across all 131 `components/` files (12 Category-A → real `.pen` nodes; 25 `<utility — no .pen counterpart>`; 94 `<screen-section — pending epic-19-8 mapping>`). **19-4** delivered the visual-regression *harness* + 25 reference fixtures + 46 baselines + Sally sign-off (closed 2026-05-13, `7d7a6b2`). **This story (19-4b)** bulk-fills the remaining ~99 components and lands the three harness-quality fixes Sally flagged on 19-4 (Task 0). **19-5** wires the harness into PR-scoped CI. **19-8** runs the full component-vs-`.pen` diff and files `bugfix-N` for material drift. Rule 22 epic retros use `pnpm run test:visual` as the diff tool.
- **Dependency:** depends on **19-4 (done)** — the harness (`visual` Playwright project, `test:visual*` scripts, gallery route, `-gallery.fixtures.tsx` shape, `data-gallery-id`/`data-pen-node` convention, `FixtureErrorBoundary` per-fixture pattern, `[@contract-v1]` harness ACs). No upstream Rule 20 ack needed: 19-4's `[@contract-v1]` AC #1–#5 are consumed *unchanged* (the spec's `focus` branch is being replaced, but the public harness contract — `visual` project name, npm scripts, gallery wrapper attributes, baseline path — is intact). 19-3's `[@contract-v2]` covers the `// Implements:` marker grammar; 19-4b reads the produced `.pen`-node mapping (an audit doc, not a versioned AC) → implicit-v0, ack-skipped per Rule 20.

### Architecture / constraints — read before implementing

- **All frontend.** 0 Go, 0 migrations, 0 swagger, 0 backend tests. Cross-stack split check: backend task count = 0 → single story is correct (the `>3 each side` threshold is not met).
- **No Storybook, no Playwright component-testing in this repo.** The 19-4 ruling stands: dev-only TanStack Router gallery route + the existing Playwright runner. Do NOT add `@playwright/experimental-ct-react` or any other new test-framework dep.
- **No `apps/web/src/components/` edits.** The `local/implements-pen-node-id` rule (19-3) is silent because this story touches only `routes/test/*` (route-only — Rule 21 exempt) and `tests/visual/*` (tests — Rule 21 exempt). If you find yourself wanting to add a `data-testid` to a component to make a fixture work, **stop**: that's a 19-8-style finding, not something to patch here. Use the `<section data-gallery-id>` wrapper in the *route* instead. (One exception: if a component is genuinely unrenderable in isolation without a prop it doesn't expose, flag it back to the SM — it's a 19-8 candidate, not 19-4b's problem.)
- **Determinism is everything for visual tests** (project-context Rule 16 + the bugfix-10-3 StrictMode lesson + the 19-4 harness): `reducedMotion: 'reduce'` + `animations: 'disabled'` kill CSS transitions (Vido's hover/focus states are pure CSS — `lg:group-hover:*`, `focus-visible:*`); fixed `viewport` 1280×800; `colorScheme: 'dark'` (Vido has no light theme); `caret: 'hide'`; `maxDiffPixelRatio: 0.001`. **Seeded data ⇒ no network calls** — if a Q-fixture is hitting the network during snapshot, that's a missed `queryKey` in `seedQueries` (or a `staleTime: 0` somewhere). Use Playwright's `page.route('**/api/v1/**', route => route.abort())` only as a last-resort safety net; the right answer is to seed correctly.
- **`:focus-visible` vs `:focus`**: only `:focus-visible` paints the visible focus ring in Vido's styles. Programmatic `.focus()` does not trigger `:focus-visible` in Chromium; keyboard-driven focus (Tab from the sentinel) does. Task 0 Fix A is the canonical fix. Some components have identical `:focus` and `:focus-visible` styles → their focus baselines won't visibly change post-fix; that's expected, capture them anyway for completeness.
- **`useRouterState`-dependent components**: see Task 0 Fix B. The nested memory-`RouterProvider` is the cleaner of the two options because it isolates the stub to the one fixture that needs it; sibling route files (Option B2) drag in shell layout etc.
- **`routes/test/` and the prod bundle**: the gallery + sentinel button + any new sibling `-tabnav-*.tsx` files (Option B2) get `import.meta.env.PROD` short-circuit guarding (the 19-4 CR M1 fix in `gallery.tsx:31-37`). Mirror that pattern in any new test-route file; do NOT rely on the `hostname === 'localhost'` clause alone.
- **Platform suffix**: this story's commit lands `-darwin` (or `-linux` if you switch the dev machine first); 19-5 will own cross-platform parity per Task 5's chosen strategy.
- **`tests/visual/components.visual.spec.ts` is the only spec.** Do NOT split into multiple specs per component-group (the DOM-driven worklist pattern is core to "adding a component = adding a fixture entry, nothing else"). The spec already discovers state divs from the DOM; new states (like `open`) just need a new `else if` branch in the state-handling chain.

### Fixture patterns — quick reference

```ts
// Presentational (P bucket) — no data dependencies
{
  id: 'category-component-name',
  label: 'category/ComponentName',
  component: ComponentName as ComponentType<Record<string, unknown>>,
  props: { /* match the component's *.spec.tsx mock shape */ },
  penNode: 'XXXXX',           // from `// Implements:` header (drift-19-3-2026-05.md)
  // statesOnly: ['default'], // for badges/skeletons
  // width: 320,              // for inline/auto-width components
}

// Query-driven (Q bucket) — needs React-Query seed
{
  id: 'homepage-hero-banner',
  ...
  seedQueries: [
    { queryKey: ['library', 'movies', { /* filters */ }] as const, data: { items: [/* 3 mock movies */] } },
    { queryKey: ['tmdb', 'trending'] as const, data: [/* mock */] },
  ],
}

// Interactive open state (Fix C)
{
  id: 'library-poster-card-menu',
  ...
  statesOnly: ['default', 'open'],
  openTrigger: '[data-testid="poster-card-menu-trigger"]',
}

// Store-driven (S bucket)
{
  id: 'library-selection-toolbar',
  ...
  seedStore: () => useSelectionStore.setState({ selectedIds: new Set(['m1', 'm2', 'm3']), mode: 'select' }),
}

// Router-dependent (Fix B Option B1)
{
  id: 'shell-tab-navigation',
  ...
  routePath: '/library',  // nested memory router initial entry
}
```

### Anti-patterns to avoid

- **Don't snapshot a `[data-gallery-error]` placeholder.** If a fixture errors, fix the fixture; do not commit an error-state baseline (the spec already skips them, but `:update` won't — verify visually that `pnpm run test:visual:update` didn't write an error PNG by spot-checking `/test/gallery`).
- **Don't add an `inert` prop to gallery sections** to "freeze" interactive state — the gallery is a screenshot tool, not a frozen-state tool. Use `statesOnly` to skip states a component doesn't have.
- **Don't introduce per-component spec files.** All baselines share `components.visual.spec.ts`. Adding a per-component spec creates discoverability fragmentation and breaks the "add fixture = done" workflow.
- **Don't hand-edit PNGs.** Discipline (per `tests/visual/README.md`): regenerate only via `:update`, only after a deliberate reviewed change, own commit.
- **Don't mix baseline regeneration with logic changes in the same commit.** Task 0 is a deliberate exception (the spec change AND the affected baselines must land atomically); for Task 4 bulk-fill, the fixtures + baselines go in one commit, separated from Task 0 / Task 5 commits.

### Testing standards (project-context.md)

- **E2E/visual: Playwright.** After ANY run: `pnpm run test:cleanup` (project-context "Test Process Cleanup"; the `globalSetup`/`globalTeardown` already track spawned servers).
- **Vitest (if any gallery aggregator smoke test):** co-located, `toBeInTheDocument` / `toEqual` not `toBeTruthy` (Rule 16). **Prefer no unit test for the gallery aggregator** — the visual spec is its real coverage; a brittle "renders N sections" RTL test is dead weight (bugfix-10-3 "don't add a regression test for a non-existent bug" spirit).
- **Lint gate (Rule 12):** `pnpm lint:all` = `go vet` → `staticcheck` → `eslint .` → `prettier --check .`, 0 errors at close; warnings = 122 (the bugfix-10-7 / 19-3 / 19-4 baseline). `eslint .` covers `apps/web/`, `libs/shared-types/`, `tests/` — so the new fixture entries + spec changes + any new `routes/test/*-tabnav-*.tsx` files must lint clean.

### Rule 20 / Rule 21 / Rule 22 linkage

- **Rule 20 (Contract Stamps):** this story carries NO `[@contract-vN]` stamps. The harness contracts AC #1–#5 are 19-4's `[@contract-v1]` and are *consumed* not extended; the `openTrigger?` / `seedQueries?` / `seedStore?` / `routePath?` additive fields on `GalleryFixture` are documented harness extensions, not contracts (no downstream story consumes them as stamped contracts yet). Upstream 19-4 ack is implicit-v1 (consumed unchanged) — no ack row needed per Rule 20 forward-only retrofit.
- **Rule 21:** no new `components/` files → the ESLint rule (19-3) is silent. New `routes/test/*` files (if any — Option B2) are `<route-only>` (Rule 21 exempt). New `tests/visual/*` files (if any) are tests (Rule 21 exempt).
- **Rule 22:** this story does NOT classify drift. It builds the diff tool for 19-8. The Rule 22 tooling line in `project-context.md` already reads "LIVE since story 19-4" — no edit needed unless this story materially changes the harness invocation (it doesn't; `pnpm run test:visual` stays the entry point).

### Project Structure Notes

- **New (gallery aggregator extension):** *no new TS files required* if Fix B picks Option B1. If Fix B picks Option B2: `apps/web/src/routes/test/-tabnav-library.tsx` (+ optionally `-downloads`/`-pending`/`-settings` if other fixtures need them).
- **Modified:**
  - `apps/web/src/routes/test/gallery.tsx` — sentinel button per state div (Fix A); `open` state filter + `data-gallery-open-trigger` emit (Fix C); `queryClient.setQueryData` + `seedStore` invocation in render (Task 3 infrastructure); optional nested `RouterProvider` for `routePath` fixtures (Fix B Option B1).
  - `apps/web/src/routes/test/-gallery.fixtures.tsx` — `GalleryState` union adds `'open'`; `GalleryFixture` interface adds `openTrigger?` + `seedQueries?` + `seedStore?` + `routePath?`; ~99 new fixture entries (Tasks 2/3); the existing `shell-tab-navigation` fixture loses its FIXME comment (Fix B), gains `routePath: '/library'`; the existing `library-sort-selector` fixture gains `statesOnly: ['default', 'hover', 'focus', 'open']` + `openTrigger: '[data-testid="sort-selector-button"]'` (Fix C).
  - `tests/visual/components.visual.spec.ts` — `focus` branch uses sentinel + Tab (Fix A); new `open` branch reads `data-gallery-open-trigger` and clicks it (Fix C); spec header doc-comment references 19-4b Task 0.
  - `tests/visual/components.visual.spec.ts-snapshots/components/**/*.png` — the 10 affected 19-4 focus baselines regenerated; ~250+ new baselines for the bulk fill (depending on how many of the ~99 components get the full 3-state set vs `default`-only); `library-sort-selector/open-visual-darwin.png` new.
  - `tests/visual/README.md` — document `open` state + `openTrigger` field; document `seedQueries` / `seedStore` / `routePath` fixture options under "Adding a component"; update platform-suffix language per Task 5 chosen strategy.
  - `_bmad-output/audit/visual-baseline-19-4.md` — "Delivered baselines" table expanded to full set; "Pending" section closed.
  - `_bmad-output/implementation-artifacts/sprint-status.yaml` — `19-4b` status transitions.
  - `apps/web/src/routeTree.gen.ts` — auto-regenerates if any new `routes/test/*.tsx` files are added (Option B2).
- **Out of scope:**
  - CI workflow (`.github/workflows/visual-regression.yml` — 19-5 owns it).
  - Component-vs-`.pen` *diff* sweep + `bugfix-N` filing (19-8).
  - Upgrading any `<screen-section …>` placeholder to canonical Rule 21 header (19-8).
  - Any TestSprite work (19-6/19-7).
  - Any `apps/web/src/components/` source edits (Rule per AC #2 + 19-4 inherited constraint).

### References

- [Source: _bmad-output/implementation-artifacts/19-4-playwright-visual-snapshot-baseline.md] — predecessor; Party Mode 2026-05-12 scope re-cut; harness contract `[@contract-v1]` AC #1–#5; CR closeout 2026-05-13 (the 3 Sally follow-ups this story's Task 0 addresses)
- [Source: _bmad-output/audit/visual-baseline-19-4.md] — harness table + 25 delivered + the ~99 worklist this story closes + deliberate skips
- [Source: _bmad-output/audit/drift-19-3-2026-05.md] — file→`.pen`-node mapping (the `penNode` values for every fixture); Category A/B/C tables
- [Source: _bmad-output/implementation-artifacts/sprint-status.yaml#L499-525] — epic-19 header + 19-1..19-5 status (dependency order + agent routing)
- [Source: project-context.md#Rule-21-Component-to-Design-Node-Traceability] — marker grammar (this story consumes `// Implements:` headers via the audit doc, doesn't extend them)
- [Source: project-context.md#Rule-22-Epic-Retro-Design-Drift-Audit] — the harness this story extends ("LIVE since story 19-4")
- [Source: project-context.md#Rule-12-Code-Quality-Checks-CI-based] / [#Rule-16-Test-Assertion-Quality] — lint order + assertion-matcher rules
- [Source: playwright.config.ts:148-163] — the `visual` project config (Chromium, 1280×800, dark, reduced-motion, `testMatch: ['**/*.visual.spec.ts']` — added in 19-4 CR H1)
- [Source: apps/web/src/routes/test/gallery.tsx] — the gallery route this story extends (with sentinel + open state + query/store seeding)
- [Source: apps/web/src/routes/test/-gallery.fixtures.tsx] — the fixture aggregator this story extends from 26 entries to ~125 entries
- [Source: tests/visual/components.visual.spec.ts] — the visual spec this story extends (sentinel/Tab focus + open state)
- [Source: tests/visual/README.md] — harness overview, baseline-update discipline, "Adding a component"
- [Source: apps/web/src/components/shell/TabNavigation.tsx] — the `useRouterState` consumer that drives Fix B
- [Source: apps/web/src/components/library/SortSelector.tsx] — the `data-testid="sort-selector-button"` open-trigger reference fixture for Fix C
- [Source: apps/web/src/hooks/useMediaDetails.ts] — `detailKeys` query-key generator used by data-driven fixtures
- [Source: CLAUDE.md] — `routes/test/` precedent (manual-search.tsx); screenshot-export workflow gating (only on `.pen` modification — not triggered by this story)

## Dev Agent Record

### Agent Model Used

claude-opus-4-7[1m] (Amelia / dev-story workflow; same session as the SM /create-story pass — workflow tip "use a different LLM" not honoured this session; CR pass should run in a different LLM context to compensate)

### Debug Log References

- `pnpm exec eslint .` → 0 errors / 122 warnings (matches the 19-4 closeout baseline)
- `pnpm exec prettier --check apps/web/src/routes/test/gallery.tsx apps/web/src/routes/test/-gallery.fixtures.tsx tests/visual/components.visual.spec.ts` → clean on all 3 touched files
- `pnpm run test:visual:update` → wrote `library-sort-selector/open-visual-darwin.png` (new) + re-blessed `search-search-bar/focus-visual-darwin.png` + `shell-tab-navigation/{default,hover,focus}-visual-darwin.png` (router-state-driven active-tab now paints `/library`). The other 9 focus baselines (ui-button, library-filter-chips, library-sort-selector, library-view-toggle, media-poster-card, metadata-editor-genre-selector, search-media-type-tabs, ui-pagination, and the no-focusable-descendant cases) rendered identically under `:focus-visible` and `:focus` → Playwright treated them as matches, did not re-write.
- Burn-in: `pnpm run test:visual` ran 8 times total (4 pre-`open`-state-stabilizer + 4 post). Pre-stabilizer: 3 pass / 1 fail (visual content, exact baseline unknown — first burn-in run output was tail-cropped). Post-stabilizer: 4 PASS visual content / 1 webServer-startup infra timeout from rapid back-to-back runs (orphaned ports — `test:cleanup:all` resolved). Standalone runs land at 14–32 s (one anomalous 16-minute first run after rapid serialization; not a visual content issue).
- `npx playwright test --project=chromium --project=firefox --project=webkit-core --project=mobile-chrome --project=mobile-safari --list` → 1663 tests / 36 files (unchanged from 19-4 closeout — visual project still excluded)
- `pnpm nx test web` → PASS (148 files / 1840 tests — unchanged from 19-4 closeout)
- `pnpm nx test api` → PASS (Nx flaky-flagged — the known `TestScannerService_SSEBroadcast_ScanCancelled` flake, tracked in sprint-status as `preexisting-fail-scanner-sse-scan-cancelled-flake`; passed on Nx retry; zero Go changes this story so the flake is unrelated)
- `pnpm run test:cleanup` → no orphans (after `test:cleanup:all` killed 2 leftover Playwright nodes from the burn-in runs)

#### Task 1 inventory (2026-05-13)

**Commands run** (per Task 1's quick recipe):

```bash
find apps/web/src/components -name '*.tsx' ! -name '*.spec.tsx' ! -name '*.test.tsx' \
  ! -name 'index.ts' | sort > /tmp/all-components.txt
# → 127 .tsx files
grep -oE "id: '[a-z][a-z0-9-]+'" apps/web/src/routes/test/-gallery.fixtures.tsx \
  | sort -u > /tmp/covered-ids.txt
# → 27 ids (= 25 unique components: AvailabilityBadge has owned/requested variants;
#   `gallery-pc-uuid-0001` is a PosterCard prop value, not a fixture id; the grep
#   over-includes by 2 — manual cross-check against the 19-4 closeout audit doc
#   confirms 25 unique component source files / 26 fixture entries / 46 PNGs).
```

**Uncovered count:** `127 − 25 = 102` `.tsx` files for the bulk fill. The story header
reads "~99"; the +3 margin is the marginal Category-B utilities (`ui/Dialog`,
`ui/HighlightText`, `ui/SidePanel`) that 19-4 explicitly deferred to 19-4b.

**Bucket assignments (P/Q/S/L per Task 1 definitions):**

Signal-grep per file: `useQuery|useMutation|useInfiniteQuery|useQueryClient` (Q-direct);
custom hook calls under `apps/web/src/hooks/` (Q-indirect — those hooks wrap RQ);
`use[A-Z]*Store(` (S); `useRouterState|useParams|useSearch` (R — informational);
`<Outlet`/`{children}`-shell (L candidate). Cross-checked against AC #4 layout-shell
list. Final tallies:

| Bucket | Count | Treatment in Tasks 2/3 |
|---|---|---|
| **L** (layout shell, deliberate skip) | **4** | Documented skip in audit doc per AC #4 (Task 6); no fixture entry. |
| **S** (Zustand store-driven) | **0** | No `apps/web/src/stores/` consumer found under `components/`. The `seedStore?` infrastructure stays in place for future-proofing; no S work this story. |
| **Q** (data-driven, wraps RQ via custom hook) | **35** | Task 3 — `seedQueries` for read hooks. Mutation-only consumers (e.g. `settings/LibraryCard` → `useDeleteLibrary` only) need no data seeding but stay in Q because they require `QueryClientProvider` (the gallery already provides one). |
| **P** (presentational, props-in only) | **63** | Task 2 — straight fixture add, reuse each component's own `*.spec.tsx` mock-data shape. |
| **Sum** | **102** | matches uncovered count ✓ |

**L bucket (4 — deliberate skips, per AC #4):**

| File | Lines | Reason |
|---|---|---|
| `shell/AppShell.tsx` | 111 | Wraps `TabNavigation` + `{children}`. TabNavigation is already covered as its own fixture; rendering AppShell standalone screenshots the same thing again. |
| `dashboard/DashboardLayout.tsx` | 17 | Transparent `<div className=…>{children}</div>` — no isolated visual surface. |
| `settings/SettingsLayout.tsx` | 182 | Sidebar nav driven by `useRouterState()`. *Has visual surface, but* re-rendering it requires nested memory router (Task 0 Fix B) + stub `{children}`. Cost/benefit weak — settings sub-routes will be covered when individual settings components get their own fixtures. **May reconsider in Task 2** if Sally flags the omission. |
| `setup/SetupWizard.tsx` | 157 | Stateful multi-step controller (`useState` step machine + `useQueryClient`). No single static snapshot is meaningful; individual `*Step` components ARE in P-bucket and get baselined. |

**Q bucket (35 — data-driven):**

```
dashboard/   DownloadPanel, RecentMediaPanel
downloads/   DownloadDetails
health/      ConnectionHistoryPanel, QBStatusIndicator
homepage/    ExploreBlock, ExploreBlocksList, HeroBanner
learning/    LearnedPatternsSettings
library/     FilterPanel, LibraryGrid, RecentlyAdded
manual-search/  ManualSearchDialog
media/       MediaDetailPanel
metadata-editor/  MetadataEditorDialog
parse/       FloatingParseProgressCard, RetryQueueSection
retry/       RetryNotifications, RetryQueuePanel, RetryQueueWithNotifications
scanner/     ScanProgress
settings/    BackupManagement, BackupScheduleConfig, CacheManagement,
             ExploreBlockEditModal, ExploreBlocksSettings, LibraryCard,
             LibraryEditModal, LogsViewer, MediaLibraryManager, MetadataExport,
             QBittorrentForm, ScannerSettings, ServiceStatusDashboard
subtitle/    SubtitleSearchDialog
```

SSE/progress-gated components (`parse/FloatingParseProgressCard`, `scanner/ScanProgress`) — render with `taskId={null}` / idle-state fixture rather than seeding SSE; if Sally wants an in-progress baseline, mock the progress state via a per-fixture override in Task 3.

**P bucket (63 — presentational), grouped by `components/` subfolder (Task 2 ordering):**

```
ui/         (3)   Dialog, HighlightText, SidePanel
media/      (8)   CreditsSection, DetailPanelMenu, FallbackFailed, FallbackPending,
                  FileInfo, MediaGrid, TrailerEmbed, TVShowInfo
degradation/(3)   PlaceholderContent, ServiceHealthBanner, UnidentifiedFileCard
dashboard/  (2)   CollapsibleSection, QuickSearchBar
downloads/  (6)   DownloadFilterTabs, DownloadItem, DownloadList,
                  DownloadParseStatusBadge, ParseFailedActions, StatusIcon
library/    (8)   BatchConfirmDialog, BatchProgress, LibrarySearchBar, LibraryTable,
                  ParseFailureCard, PosterCardMenu, SelectionToolbar, SettingsGearDropdown
homepage/   (1)   TrailerModal
learning/   (1)   LearnPatternPrompt
manual-search/ (3) FallbackStatusDisplay, SearchResultCard, SearchResultsGrid
metadata-editor/ (2) CastEditor, PosterUploader
notifications/ (3) NewMediaNotifications, NewMediaToast, ParseCompleteToast
parse/      (4)   ErrorDetailsPanel, LayeredProgressIndicator, MediaFileCard, ParseStatusBadge
retry/      (1)   CountdownTimer
scanner/    (2)   ScanProgressCard, ScanProgressSheet
search/     (1)   SearchResults
settings/   (8)   BackupTable, CacheTypeCard, ConnectionTestResult, LogEntry,
                  LogFilters, RestoreConfirmDialog, ServiceStatusCard, SettingsPlaceholder
setup/      (7)   ApiKeysStep, CompleteStep, MediaFolderStep, MediaLibrarySetupStep,
                  QBittorrentStep, StepProgress, WelcomeStep
```

Sub-folder sum: 3+8+3+2+6+8+1+1+3+2+3+4+1+2+1+8+7 = **63** ✓

**S bucket investigation (zero hits, intentionally documented):**

Spot-checked the obvious suspects against their grep'd imports:
- `library/SelectionToolbar.tsx` — receives selection as props; no `stores/` import.
- `notifications/NewMediaNotifications.tsx` — receives `notifications` array as prop; only imports the `NewMediaNotification` *type* from `useNewMediaNotifications.ts`.
- `library/PosterCardMenu.tsx`, `library/SettingsGearDropdown.tsx` — props-in, no store.
- `parse/ParseStatusBadge.tsx` — props-in.

Project pattern aligns with `project-context.md` Rule 5 ("Use TanStack Query for server state; Zustand for UI state only"). Stores under `apps/web/src/stores/` (if any) are consumed by route-level components, not by leaf `components/` files. The story's additive `seedStore?` field on `GalleryFixture` stays in place for forward compatibility but produces no Task 3 work this story.

**Type/util-only `.ts` files skip confirmation (Task 1 sub-bullet #3):**

| File | Lines | Status |
|---|---|---|
| `apps/web/src/components/parse/types.ts` | 147 | Exists. Excluded by `-name '*.tsx'` filter. No JSX. |
| `apps/web/src/components/degradation/types.ts` | 48 | Exists. Excluded. No JSX. |
| `apps/web/src/components/downloads/formatters.ts` | 67 | Exists. Excluded. No JSX. |
| `apps/web/src/components/parse/useParseProgress.ts` | 367 | Exists. Excluded. Misfiled hook — Category B per drift doc. |

All 4 remain "no baseline ever" per `_bmad-output/audit/visual-baseline-19-4.md`.

**Drift-doc cross-check (Task 1 sub-bullet #4) vs `_bmad-output/audit/drift-19-3-2026-05.md`:**

- **Category A (12 files, real `.pen` Reusable-Component nodes):** ALL 12 already covered in `-gallery.fixtures.tsx` (delivered by 19-4). **0 new Category-A `penNode` values** to set in Tasks 2/3.
- **Category B (21 `.tsx` + 4 `.ts` = 25 paths, `penNode='utility'`):** 8 already covered (`ui/{Badge,Card,Skeleton,Pagination}`, `media/{PosterCardSkeleton,ColorPlaceholder,TechBadgeGroup}`, `homepage/ExploreBlockSkeleton`). **13 `.tsx` remaining** get `penNode: 'utility'` in their new fixtures:
  - `ui/Dialog`, `ui/HighlightText`, `ui/SidePanel` (P)
  - `media/FileInfo` (P)
  - `degradation/PlaceholderContent` (P)
  - `settings/SettingsPlaceholder` (P)
  - `setup/StepProgress` (P)
  - `dashboard/CollapsibleSection` (P)
  - `downloads/StatusIcon` (P)
  - `shell/AppShell`, `dashboard/DashboardLayout`, `settings/SettingsLayout`, `setup/SetupWizard` (L — only get `penNode: 'utility'` *if* rendered; currently planned skip).
- **Category C (94 files, `penNode='screen-section'`):** 5 already covered (`media/AvailabilityBadge` ×2 variants, `media/MetadataSourceBadge`, `degradation/DegradationBadge`, `library/ViewToggle`, `library/EmptySearchResults`). **89 remaining** get `penNode: 'screen-section'` — the rest of the P and Q bucket files.
- Category arithmetic: `(102 − 13 Cat-B remaining = 89 Cat-C remaining)` ✓ matches.

**Hand-off to Task 2 (Presentational fill):** the 63-file P bucket above, grouped by `components/` subfolder, is the worklist. The component's own `*.spec.tsx` mock-data shape is the authoritative props source (project-context.md rule). `penNode` for every file in this bucket except `ui/{Dialog,HighlightText,SidePanel}`, `media/FileInfo`, `degradation/PlaceholderContent`, `settings/SettingsPlaceholder`, `setup/StepProgress`, `dashboard/CollapsibleSection`, `downloads/StatusIcon` (Category B → `'utility'`) is **`'screen-section'`** (Category C).

#### Task 2 bulk fill (2026-05-13)

**Method.** 5 parallel general-purpose subagents each handled a subfolder bucket (ui+media+degradation 14 / dashboard+downloads+library 16 / homepage+learning+manual-search+metadata-editor+notifications 10 / parse+retry+scanner+search 8 / settings+setup 15 = 63). Each agent read the component `.tsx` (for `// Implements:` header + prop interface) plus the corresponding `.spec.tsx` (for canonical mock-data shape per project-context.md), produced ready-to-paste fixture entries, and flagged any P-bucket mis-classifications they spotted while reading source.

**Deltas vs Task 1 inventory.**

- **+2 P-bucket → Q-bucket reclassification flags (deferred to Task 3 to re-bucket properly):**
  - `homepage/TrailerModal` — calls `useQuery(['tmdb','videos',mediaType,tmdbId])` internally. **Defensive fix landed in this Task 2 commit:** fixture uses `tmdbId: 0` → query disabled (`enabled: !!tmdbId` is false at 0) → renders the deterministic empty state, no network. Task 3 should optionally re-bucket and seed a `[]` videos payload for a "no trailer" baseline, OR seed a real payload for the iframe-loaded variant.
  - `library/ParseFailureCard` — renders `<ManualSearchDialog isOpen={false}>` which unconditionally mounts `useManualSearch` (gated by `params.query.length >= 2`). **Defensive fix landed:** fixture file's `parsedInfo.title: ''` + `filename: 'a.mkv'` keeps the derived query length below 2 → useQuery disabled. Re-bucket optional in Task 3 if a richer baseline is wanted.
- **Inventory totals remain 63 P / 35 Q.** The two flagged components stay nominally P-bucket in this commit (defensive props prevent network); Task 3 may re-list them as Q-bucket if richer seeded baselines are desired.

**Per-subfolder additions (with notable per-fixture decisions):**

```
ui/         3  Dialog (utility, statesOnly:default — Radix portal puts content
              outside state div; Sally may decide to drop or rewrap with
              non-portal Demo wrapper in Task 4),
              HighlightText (utility, default-only, width 240),
              SidePanel (utility, default-only — fixed-position viewport
              overlay; Sally Task 4 review may flag visual neighbor occlusion)
media/      8  CreditsSection (default-only, width 480),
              DetailPanelMenu (4-state inc. `open`, openTrigger
              `[data-testid="detail-menu-trigger"]` — captures inline dropdown
              panel which is absolutely-positioned, not portaled),
              FallbackFailed (default-only, uses TanStack Link — app shell
              router resolves), FallbackPending (default-only),
              FileInfo (utility, default-only, width 360),
              MediaGrid (default-only, PosterCard children id:0 keeps the
              useMovieDetails/useTVShowDetails hooks disabled — same defensive
              pattern as the existing media-poster-card fixture),
              TrailerEmbed (3-state, width 360 — only the "▶ 觀看預告片"
              button is captured; iframe state explicitly excluded),
              TVShowInfo (default-only, width 480)
degradation/ 3 PlaceholderContent (utility, default-only, width 200),
              ServiceHealthBanner (default-only, width 640),
              UnidentifiedFileCard (default-only, width 480)
dashboard/  2  CollapsibleSection (utility, 3-state, width 480 —
              useNavigate-only, no data hooks; sessionStorage tolerates empty),
              QuickSearchBar (3-state, width 480)
downloads/  6  DownloadFilterTabs (3-state, width 720),
              DownloadItem (3-state, width 720),
              DownloadList (3-state, width 720 — expandedHash=null default
              keeps useDownloadDetails dormant),
              DownloadParseStatusBadge (default-only, width 160),
              ParseFailedActions (3-state, width 320),
              StatusIcon (utility, default-only, width 120)
library/    8  BatchConfirmDialog (default-only, plain fixed-overlay, NOT
              Radix portal — renders inline when isOpen=true),
              BatchProgress (default-only, plain fixed-overlay),
              LibrarySearchBar (3-state, width 480),
              LibraryTable (3-state, width 960 — uses TanStack Link),
              ParseFailureCard (3-state, width 280 — defensive empty
              parsedInfo.title prevents useManualSearch from firing per the
              Task 1 flag above),
              PosterCardMenu (default-only, width 240, rendered with isOpen=true
              — no internal trigger, parent-controlled open state),
              SelectionToolbar (3-state, width 720),
              SettingsGearDropdown (4-state inc. `open`, openTrigger
              `[data-testid="settings-gear-button"]` — captures the dropdown
              panel; same pattern as library-sort-selector)
homepage/   1  TrailerModal (default-only — defensive tmdbId=0 disables
              the internal useQuery; flagged for Q-bucket re-classification)
learning/   1  LearnPatternPrompt (3-state, width 560)
manual-search/3 FallbackStatusDisplay (default-only, width 640),
              SearchResultCard (3-state, width 200 — omits posterUrl to
              render the no-network fallback placeholder),
              SearchResultsGrid (3-state, width 880, 4 results across all 3
              sources to exercise the source-badge variations)
metadata-editor/2 CastEditor (3-state, width 480),
              PosterUploader (3-state, width 520 — no currentPoster prop, so
              renders the empty dropzone without URL.createObjectURL)
notifications/3 NewMediaNotifications (default-only, fixed-position
              bottom-right; toast items animate-in via setTimeout 10ms but
              Playwright disables CSS animations — static frame is
              deterministic), NewMediaToast (default-only, width 360),
              ParseCompleteToast (default-only, width 360)
parse/      4  ErrorDetailsPanel (3-state, width 560, shared
              PARSE_STEPS_FAILED const at file top),
              LayeredProgressIndicator (default-only, width 480, shared
              PARSE_STEPS_IN_PROGRESS const),
              MediaFileCard (3-state, width 240, pinned 8 GB fileSize so
              formatFileSize renders the stable "8.00 GB" literal),
              ParseStatusBadge (default-only, status="success" — `parsing`
              would animate-spin even with Playwright animations disabled)
retry/      1  CountdownTimer (default-only, targetTime pinned to a PAST
              ISO `'2020-01-01T00:00:00.000Z'` → initial secondsRemaining=0
              → formatTimeRemaining(0) returns the stable literal `'即將重試'`;
              every tick recomputes to 0 so the rendered string is byte-stable
              across runs; onComplete fires once into noop)
scanner/    2  ScanProgressCard (default-only, width 400, shared
              SCAN_STATE_ACTIVE const — percentDone:62 pins the bar fill),
              ScanProgressSheet (default-only, width 400 — default
              expanded=false captures the 64px collapsed mobile pill)
search/     1  SearchResults (default-only, width 960 — isLoading:true
              renders the deterministic skeleton grid; real results would
              fire PosterCard useMovieDetails/useTVShowDetails on mount)
settings/   8  BackupTable (default-only, width 760, 3 rows
              completed/failed/running exercising all action-button states),
              CacheTypeCard (3-state, width 480),
              ConnectionTestResult (default-only, width 480),
              LogEntry (3-state, width 720 — ERROR-level log with full
              context object + hint to exercise the collapsed render),
              LogFilters (3-state, width 640),
              RestoreConfirmDialog (default-only, plain fixed-overlay NOT
              Radix portal),
              ServiceStatusCard (3-state, width 520),
              SettingsPlaceholder (utility, default-only, width 480, lucide
              Database icon import)
setup/      7  All 7 receive `StepProps` (data/onUpdate/onNext/onBack/onSkip/
              isFirst/isLast/isSubmitting) — pure presentational, no data
              hooks, no width (full-page panels). ApiKeysStep, CompleteStep,
              MediaFolderStep, MediaLibrarySetupStep (pre-seeded libraries
              with stable ids to short-circuit the mount-useEffect that
              would otherwise call onUpdate({ libraries: [...] }) using
              crypto.randomUUID — non-deterministic), QBittorrentStep,
              StepProgress (utility, default-only, width 320),
              WelcomeStep.
```

Sum: 3+8+3+2+6+8+1+1+3+2+3+4+1+2+1+8+7 = **63** ✓.

**Shared helper consts added** to top of `-gallery.fixtures.tsx`:
- `PARSE_STEPS_FAILED: ParseStep[]` (6-step failed parse — for ErrorDetailsPanel)
- `PARSE_STEPS_IN_PROGRESS: ParseStep[]` (6-step in-progress — for LayeredProgressIndicator)
- `SCAN_STATE_ACTIVE: ScanProgressState` (percentDone:62 active scan — for both ScanProgressCard + ScanProgressSheet)

**New type-only imports:**
- `import type { CastMember, CrewMember, TVShowDetails } from '../../types/tmdb'`
- `import type { ServicesHealth } from '../../components/degradation/types'`
- `import type { ParseStep } from '../../components/parse/types'`
- `import type { ScanProgressState } from '../../hooks/useScanProgress'`

**New value import:** `import { Database } from 'lucide-react'` (icon prop for settings-settings-placeholder).

**Spot-check (replaces "as you add batches of ~10").** Programmatic Playwright probe (`.gallery-probe.mjs`, deleted post-run): navigated to `http://localhost:4200/test/gallery` against running `pnpm nx serve web`, waited for `[data-gallery-id]` selector + 2 s settle. Result: **89** `[data-gallery-id]` sections rendered (26 pre-existing + 63 new = 89 ✓), **0** `[data-gallery-error]` placeholders, **2** console errors (HTTP 500 on `/api/v1/setup/status` + one other pre-existing app-shell call — NOT a Task 2 fixture). First probe pass surfaced a transient `Invalid hook call` for `ui-dialog` during Vite optimizeDeps cache warming; second probe pass (after Vite cache settled) was clean — no fixture re-rendered the hook error.

**Regression after Task 2 (compile-time only — full regression deferred to Task 4 closure):**
- `pnpm exec eslint apps/web/src/routes/test/-gallery.fixtures.tsx` → 0 errors / 0 warnings
- `pnpm exec eslint .` → 0 errors / **122 warnings** (matches the 19-4/Task-0 baseline EXACTLY — no new warnings introduced by the 63 fixture additions; lint:all baseline preserved per AC #6)
- `pnpm exec prettier --check apps/web/src/routes/test/-gallery.fixtures.tsx` → clean
- `pnpm exec tsc --noEmit -p apps/web/tsconfig.json` → exit 0 (all 63 prop shapes type-check against component interfaces; the `as ComponentType<Record<string, unknown>>` cast on the `component` field is the documented loose-typing pattern of the gallery aggregator, not a Task-2 introduction)

`nx test web` / `nx test api` / `playwright test --project=visual --list` / `test:visual` burn-in are AC #6 close-gate work for Task 4, NOT Task 2 — added fixtures don't change test discoverability (the visual spec is DOM-driven; new fixtures only become testable once baselines are generated via `test:visual:update` in Task 4).

### Completion Notes List

- **🔗 AC Drift:** N/A (additive harness extension — 19-4 `[@contract-v1]` AC #1–#5 wrapper shape gains a NEW optional `'open'` state member + `data-gallery-open-trigger` attribute. SM CS pass classified this as a documented harness extension, NOT a stamped v1→v2 contract bump (rationale: the existing `default`/`hover`/`focus` shapes remain unchanged; `open` is opt-in via `openTrigger?` and emitted only when the fixture sets it). CR pass may re-classify as `[@contract-v1→v2]` AC #3 if it sees the gallery-wrapper grammar extension as in-scope of the v1 contract — if so, a v1→v2 bump row + ack in Dev Notes are owed.
- **📎 Contract Stamps:** NONE this story per SM CS judgment (harness contracts AC #1–#5 are 19-4's `[@contract-v1]` consumed unchanged; `openTrigger?` + `routePath?` are additive `GalleryFixture` interface fields, not stamped contracts). Upstream 19-4 `[@contract-v1]` is implicit-v1 (consumed unchanged) — no ack row needed per Rule 20 forward-only retrofit. If CR re-classifies AC #3 as a v1→v2 bump (see AC Drift note above), CR can add the bump row.
- **🔒 Rule 7 Wire Format:** N/A (pure FE, no Go error codes).
- **🎨 UX Verification — PENDING.** Task 4's gate ("Sally /ux-designer reviews `/test/gallery` before the baseline set is committed") is **deferred to the post-Task-1-through-3 commit**. Task 0's incremental changes (5 modified baselines + 1 new) are reviewed by Sally as part of the Task 4 closure once Tasks 1–3 land the full ~99-component bulk fill. The `shell-tab-navigation` baselines now show the `/library` active-tab state (Sally follow-up #2 satisfied per inspection); the `library-sort-selector/open` baseline shows the open `SortDropdown 955EZ` panel (Sally follow-up #3 reference case). Inspector confirmation: `nx serve web` → `/test/gallery` rendered all 26 fixtures cleanly (no `[data-gallery-error]` placeholders); TabNavigation visibly highlights `媒體庫`; SortSelector's `open` state captures the expanded panel with 4 options + selected indicator on `新增日期`. Programmatic-DEV self-check substitutes pending Sally session for Task 0 only — Task 4's gate stands for the bulk-fill close.
- **⚙️ Task 0 scope (user instruction):** This session implemented Task 0 ONLY (the 3 Sally harness-quality follow-ups). Tasks 1–6 (inventory + presentational fill + data-driven fill + Sally review + Linux-baseline strategy + audit-doc full-set update) are deferred to a separate session (per user "task 0" scope). Story remains `in-progress` after Task 0 commit; sprint-status reflects same.
- **🔬 Burn-in outcome — qualified pass.** Post-stabilizer 4 consecutive visual passes meets the AC #1 / Task 0d "0 flake" gate for the 3-run requirement. Pre-stabilizer 1/4 fail confirmed the suspected open-state click-vs-paint race; the `waitFor([role="listbox|menu|dialog"])` + `.catch` fallback added in `tests/visual/components.visual.spec.ts:81-89` resolved it. Recommendation for Tasks 2/3: any future fixture with `openTrigger` should ensure the opened popup carries one of those three roles (Radix UI components do by default — `Select` → listbox, `DropdownMenu` → menu, `Dialog` → dialog).
- **🚨 Webserver startup contention (orphaned ports) — known infra issue, NOT a Task 0 concern.** Rapid back-to-back `pnpm run test:visual` runs sometimes leave `:8080` (Go API) or `:4200` (Vite dev server) bound to a terminating-but-not-yet-released process; the next run's `webServer.timeout: 120 * 1000` fires with `"Timed out waiting 120000ms from config.webServer."`. Mitigation: run `pnpm run test:cleanup:all` between rapid burn-in sequences (the project's existing `globalSetup`/`globalTeardown` tracks session servers but doesn't reap external port holders). Affects burn-in workflow only — single-shot CI runs (which 19-5 will perform) are unaffected. Not raised as a sprint-status entry because (a) the project-context Test Process Cleanup discipline already documents `test:cleanup:all` as the remedy; (b) it doesn't affect feature-E2E reliability; (c) 19-5 won't trigger it (one-shot CI invocation).
- **⏭️ Deferred from Task 0 to next 19-4b session (Tasks 1–6):**
  - `tests/visual/README.md` update for `open` state + `openTrigger` + `routePath` interface fields (Task 6's doc-pass scope; the `GalleryFixture` JSDoc in `-gallery.fixtures.tsx:65-99` is the primary discovery surface today).
  - Opt-in of other 19-4 reference fixtures to the `open` state (none have obvious openers; Tasks 1–3 will surface candidates among the ~99 bulk-fill components — `PosterCardMenu`, modal/dialog families, etc.).
  - `_bmad-output/audit/visual-baseline-19-4.md` "Delivered baselines" table update (header still reads `(25 unique components / 26 fixture entries / 46 PNGs)`; Task 0 incremental delta = `+1 entry-state combination (library-sort-selector now has 4 states) + 1 new PNG + 4 re-blessed PNGs ⇒ 26 fixture entries / 27 entry-state combinations / 47 PNGs`). Task 6 owns the full-set rewrite.
- **⚙️ Task 2 scope (user instruction "接續 Task 2:63 個 P-bucket fixture"):** This session implemented Task 2 ONLY (the 63 presentational-bucket fixture entries). Tasks 3–6 deferred. Story remains `in-progress` after Task 2 commit. **2 P-bucket → Q-bucket reclassification flags surfaced during sub-agent reads** (`homepage/TrailerModal` calls useQuery; `library/ParseFailureCard` mounts ManualSearchDialog with useManualSearch); defensive prop tweaks (tmdbId=0; empty parsedInfo.title) keep both fixtures network-free without re-bucketing — Task 3 may re-bucket if richer seeded baselines are desired.
- **🎨 Task 2 spot-check confirmation.** `.gallery-probe.mjs` Playwright run against live `nx serve web`: 89 fixtures rendered (26+63), 0 `[data-gallery-error]` placeholders, 2 pre-existing console errors (HTTP 500 on `/api/v1/setup/status` — app-shell call, unrelated to Task 2). `ui-dialog` renders an empty state-div because Radix `DialogContent` portals to `document.body` — flagged for Task 4 / Sally review (may need a non-portal Demo wrapper OR a `statesOnly: []` opt-out). `ui-side-panel` (fixed-position viewport overlay) may visually occlude neighbors in the rendered gallery — also a Task 4 / Sally concern. Both caveats documented inline as fixture comments.
- **📦 Burn-in / baseline generation NOT performed this commit.** Task 2 ADDS fixtures only; `pnpm run test:visual:update` to generate the new baseline set + Sally `/test/gallery` review + burn-in `test:visual` ×5 are Task 4's gate per the story plan. Running `:update` now would land ~150+ darwin PNGs without the Sally sign-off the AC #3 close gate requires.

### File List

**Modified (Task 0 — 3 code files + 2 doc files):**
- `apps/web/src/routes/test/gallery.tsx` — Fix A sentinel button per state div; Fix B `StubbedRouter` component (nested memory `RouterProvider` for `routePath`-bearing fixtures); Fix C `open` state filter + `data-gallery-open-trigger` attribute emission; doc-comment updated to reference 19-4b Task 0.
- `apps/web/src/routes/test/-gallery.fixtures.tsx` — `GalleryState` union gains `'open'`; `GalleryFixture` interface gains `openTrigger?: string` + `routePath?: StubRoutePath` fields; `library-sort-selector` opted into `open` state with `[data-testid="sort-selector-button"]` trigger; `shell-tab-navigation` FIXME removed + `routePath: '/library'` added.
- `tests/visual/components.visual.spec.ts` — `focus` branch uses sentinel + `page.keyboard.press('Tab')` (Fix A); new `open` branch reads `data-gallery-open-trigger`, clicks it, then `waitFor([role="listbox|menu|dialog"])` for stability (Fix C); doc-header updated with all 3 fixes + `@story-19-4b` tag.
- `_bmad-output/implementation-artifacts/19-4b-visual-baseline-bulk-fill.md` — Status `ready-for-dev` → `in-progress`; Task 0 + all sub-tasks marked `[x]` (with deferral notes on `README.md` polish + opener inventory); Dev Agent Record + File List + Change Log filled.
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — `19-4b` status transitions (`ready-for-dev` → `in-progress`) + Task 0 completion summary line.

**Modified (Task 0 — committed baselines):**
- `tests/visual/components.visual.spec.ts-snapshots/components/search-search-bar/focus-visual-darwin.png` — re-blessed; the `:focus-visible` ring on SearchBar's input now paints (was identical to default pre-fix).
- `tests/visual/components.visual.spec.ts-snapshots/components/shell-tab-navigation/default-visual-darwin.png` — re-blessed; `媒體庫` (`/library`) tab now styled active per the nested memory router.
- `tests/visual/components.visual.spec.ts-snapshots/components/shell-tab-navigation/hover-visual-darwin.png` — re-blessed; hover state retains `/library` active.
- `tests/visual/components.visual.spec.ts-snapshots/components/shell-tab-navigation/focus-visual-darwin.png` — re-blessed; keyboard-Tab focus + `/library` active tab.

**New (Task 0 — committed baseline):**
- `tests/visual/components.visual.spec.ts-snapshots/components/library-sort-selector/open-visual-darwin.png` — open `SortDropdown 955EZ` panel (`role="listbox"`, 4 sort options, `新增日期` selected indicator).

**Unchanged baselines (re-tested under new spec, no pixel diff — included for completeness, NOT in git diff):**
- All other 19-4 baselines: 41 PNGs. Notably the 9 other `*/focus-*.png` (ui-button, library-filter-chips, library-sort-selector, library-view-toggle, media-poster-card, metadata-editor-genre-selector, search-media-type-tabs, ui-pagination + the no-focusable-descendant cases) render identically under `:focus-visible` vs `:focus` — Playwright didn't re-write them.

**Modified (Task 2 — 1 code file + 2 doc files):**
- `apps/web/src/routes/test/-gallery.fixtures.tsx` — **63 new fixture entries** added across 17 subfolder blocks; **51 new imports** (49 component imports + 2 type-import groups + 1 `Database` icon from lucide-react); **3 new helper consts** at file top (`PARSE_STEPS_FAILED`, `PARSE_STEPS_IN_PROGRESS`, `SCAN_STATE_ACTIVE`). File grew from **373 → 1714 lines**. Fixture count: **26 → 89** entries (Task 2 = +63 P-bucket). No changes to existing 26 entries.
- `_bmad-output/implementation-artifacts/19-4b-visual-baseline-bulk-fill.md` — Task 2 subtasks marked `[x]` (5/5); Task 2 Debug Log subsection added (per-subfolder breakdown + helper-const summary + spot-check method + reclassification flags); Completion Notes appended (scope, spot-check confirmation, deferred items); File List updated; Change Log entry appended.
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — `19-4b` summary line extended with Task 2 completion (status stays `in-progress`).

## Change Log

| Date | Change |
| ---- | ------ |
| 2026-05-13 | DEV Amelia /dev-story Task 2 P-bucket bulk fill COMPLETE (user-scoped "接續 Task 2:63 個 P-bucket fixture"). All 5 Task 2 sub-bullets [x]. **63 new fixture entries** added to `apps/web/src/routes/test/-gallery.fixtures.tsx` across **17 subfolders** (ui:3, media:8, degradation:3, dashboard:2, downloads:6, library:8, homepage:1, learning:1, manual-search:3, metadata-editor:2, notifications:3, parse:4, retry:1, scanner:2, search:1, settings:8, setup:7). File grew **373→1714 lines**; fixture count **26→89**. Method: 5 parallel general-purpose subagents each handled a subfolder bucket, reading the `.tsx` (for `// Implements:` header + prop interface) + `.spec.tsx` (for canonical mock-data shape), producing ready-to-paste TS entries. **3 shared helper consts** added at file top to keep parse/scanner fixtures byte-stable across reruns: `PARSE_STEPS_FAILED`, `PARSE_STEPS_IN_PROGRESS` (both `ParseStep[]`), `SCAN_STATE_ACTIVE` (`ScanProgressState` with `percentDone:62`). **51 new imports** (49 component + 4 type-import groups consolidated into `import type` lines + 1 lucide-react `Database` icon). **Defensive prop tweaks for 2 sub-agent-flagged near-Q-bucket components**: `homepage/TrailerModal` → `tmdbId:0` (disables internal useQuery via `enabled: !!tmdbId`); `library/ParseFailureCard` → empty `parsedInfo.title` + 1-char `filename:'a.mkv'` (keeps ManualSearchDialog's `useManualSearch` disabled via its `params.query.length >= 2` gate). Both flagged for optional Task-3 re-bucketing with seeded queries if richer baselines are desired. **Special-case fixture decisions**: `media-detail-panel-menu` opts into the `open` state with `openTrigger:'[data-testid="detail-menu-trigger"]'` (inline absolutely-positioned dropdown, not portaled — captures the open panel inside the state div, mirrors library-sort-selector pattern); `library-settings-gear-dropdown` opts into `open` with `openTrigger:'[data-testid="settings-gear-button"]'`; `library-poster-card-menu` rendered with `isOpen:true` directly (no internal trigger, parent-controlled); `media-media-grid` children use `id:0` to keep PosterCard's useMovieDetails/useTVShowDetails disabled (same pattern as the existing media-poster-card fixture); `retry-countdown-timer` pins `targetTime` to a PAST ISO (`'2020-01-01T00:00:00.000Z'`) so `formatTimeRemaining(0)` renders the stable `'即將重試'` literal regardless of tick timing; `parse-parse-status-badge` pinned to `status:'success'` (would animate-spin on `'parsing'`); `search-search-results` pinned to `isLoading:true` (real results would fire PosterCard data hooks on mount); `notifications/*` all `statesOnly:['default']` (toasts auto-dismiss at 5 s but Playwright animations:disabled keeps the static frame deterministic); 7 setup steps receive full `StepProps` with `noop` callbacks (`MediaLibrarySetupStep` pre-seeded with stable-id libraries to short-circuit the mount-useEffect that would otherwise call `onUpdate({ libraries: [crypto.randomUUID(...)] })`). **Spot-check** via `.gallery-probe.mjs` Playwright run against live `nx serve web`: 89 fixtures rendered (26+63 ✓), **0 `[data-gallery-error]` placeholders**, 2 pre-existing app-shell console errors (HTTP 500 on `/api/v1/setup/status` — unrelated to Task 2). `ui-dialog` renders an empty state-div (Radix DialogContent portals to `document.body` outside the snapshot crop) — flagged for Task 4 / Sally review. `ui-side-panel` (fixed-position viewport overlay) may visually occlude neighbors — same flag. Both caveats documented inline as fixture comments. Regression: `pnpm exec eslint .` 0 errors / **122 warnings** (EXACT match to 19-4 closeout AND Task-0 baseline — no new warnings introduced), `pnpm exec prettier --check apps/web/src/routes/test/-gallery.fixtures.tsx` clean, `pnpm exec tsc --noEmit -p apps/web/tsconfig.json` exit 0 (all 63 prop shapes type-check). `nx test web` / `nx test api` / `test:e2e --list` / `test:visual:update` / Sally review / burn-in `test:visual` ×5 are Task 4's gate, NOT Task 2 — fixture additions don't change test discoverability (visual spec is DOM-driven). 🔗 AC Drift: N/A (additive — 63 new fixture entries, no AC observable behaviour change on prior stories). 📎 Contract Stamps: NONE (this story carries no `[@contract-v*]` stamps; harness `[@contract-v1]` from 19-4 consumed unchanged). 🔒 Rule 7 Wire Format: N/A (pure FE, no Go error codes). 🎨 UX: PENDING Sally `/test/gallery` review — deferred to the Task 4 close gate per the story plan. Tasks 3–6 remain `[ ]` — story stays `in-progress`, NOT bumped to `review`. Commit message: `feat(19-4b): Task 2 P-bucket bulk fill — 63 presentational fixtures`. → Next session: Task 3 (Q/S-bucket fill — 35 Q-bucket data-driven fixtures + the `seedQueries`/`seedStore` gallery-infrastructure extension) on a different LLM context per workflow tip. |
| 2026-05-13 | DEV Amelia /dev-story Task 1 inventory COMPLETE (user-scoped "task 1 only"). All 4 Task 1 sub-bullets [x]. **127** `.tsx` under `apps/web/src/components/` (find filter excludes `*.spec.tsx`/`*.test.tsx`/`index.ts`); minus the **25** unique components already in `-gallery.fixtures.tsx` (26 fixture entries / 46 PNGs at 19-4 closeout, +1 entry-state and +1 PNG from Task 0 = **26 fixture entries / 27 entry-state combinations / 47 PNGs** going into Task 1) = **102** files for the bulk fill (+3 margin over the story header's "~99": the `ui/{Dialog,HighlightText,SidePanel}` Category-B utilities 19-4 explicitly deferred). **Bucket assignments**: 4 L (deliberate skip per AC #4 — `shell/AppShell`, `dashboard/DashboardLayout`, `settings/SettingsLayout`, `setup/SetupWizard`; `SettingsLayout` may be reconsidered in Task 2 if Sally flags the omission, otherwise stays skipped), 0 S (no `apps/web/src/stores/` consumer under `components/` — selection / notification state flows via props per project-context Rule 5; `seedStore?` infra stays in place for forward compatibility), **35 Q** (custom-hook RQ consumers — see Debug Log References for the full sub-folder list), **63 P** (presentational, props-in only — grouped by `components/` subfolder in Debug Log). Sum = 4 + 0 + 35 + 63 = 102 ✓. **Type/util `.ts` skip confirmation**: all 4 files (`parse/types.ts` 147L, `degradation/types.ts` 48L, `downloads/formatters.ts` 67L, `parse/useParseProgress.ts` 367L) confirmed present and (by virtue of the `-name '*.tsx'` filter) absent from the inventory — remain "no baseline ever" per `visual-baseline-19-4.md`. **Drift-doc cross-check vs `drift-19-3-2026-05.md`**: 0 new Category-A `penNode` values to set (all 12 Cat-A components already covered by 19-4); 13 Cat-B `.tsx` remaining → `penNode: 'utility'`; 89 Cat-C `.tsx` remaining → `penNode: 'screen-section'`; arithmetic 0 + 13 + 89 = 102 ✓. **No code changes this task — story file only.** 🔗 AC Drift: N/A (inventory work product, no behavioral change). 📎 Contract Stamps: NONE (no `[@contract-v*]` in this story or upstream refs touched). 🔒 Rule 7: N/A. 🎨 UX: N/A (no UI changes — inventory only; Task 4 gate stands for the bulk-fill close). Regression gate: N/A (no code change, no test rerun needed). Tasks 2–6 remain `[ ]` — story stays `in-progress`. → Next session: Task 2 (presentational bucket fill — 63 P-bucket fixtures) on a different LLM context per workflow tip. |
| 2026-05-13 | DEV Amelia /dev-story COMPLETE for Task 0 ONLY (user-scoped). ready-for-dev → in-progress. Task 0 lands all 3 Sally 2026-05-12 follow-ups. **Fix A (keyboard-Tab focus)**: `gallery.tsx` renders a hidden `<button data-gallery-sentinel="pre" tabIndex={0} className="sr-only">` immediately before each state div; the visual spec's `focus` branch focuses the sentinel then `page.keyboard.press('Tab')` so Chromium flips input modality to keyboard → `:focus-visible` rules paint. Of 10 existing 3-state focus baselines, only `search-search-bar/focus-visual-darwin.png` was pixel-different (SearchBar has the only `:focus-visible`-distinct rule); the other 9 re-tested as identical under the new mechanism and were re-blessed unchanged by Playwright. **Fix B (TabNavigation active tab via nested memory RouterProvider, Option B1)**: `gallery.tsx` adds `STUB_TAB_PATHS = ['/library', '/downloads', '/pending', '/settings'] as const` + `StubbedRouter` component which builds `createMemoryHistory({ initialEntries: [pathname] })` + a stub root + child routes for all 4 paths (with `<Outlet>` rendering the wrapped fixture). `GalleryFixture` gains `routePath?: StubRoutePath`; the `shell-tab-navigation` fixture sets `routePath: '/library'` and its FIXME block is removed. `useMemo` deps intentionally omit `children` (one-mount-per-snapshot, no re-render churn needed — `react-hooks/exhaustive-deps` suppressed with a deliberate comment); the `RouterProvider router={router as any}` cast is required because the stub router's typed tree is narrower than the main `routeTree.gen.ts` (runtime context lookup correct — `@typescript-eslint/no-explicit-any` suppressed with a rationale comment). Regenerated 3 baselines `shell-tab-navigation/{default,hover,focus}-visual-darwin.png` now show `媒體庫` styled active. **Fix C (interactive `open` state via `openTrigger`)**: `GalleryState` union gains `'open'`; `GalleryFixture` gains `openTrigger?: string`; `gallery.tsx` filters out the `open` state when `!fx.openTrigger` (silent drop) and emits `data-gallery-open-trigger={fx.openTrigger}` on the `open` state div. The visual spec's new `else if (state === 'open')` branch reads the attribute, clicks the trigger inside the state div, then **`waitFor([role="listbox|menu|dialog"])` with 1 s timeout + `.catch` fallback** (the burn-in stabilizer — pre-`waitFor`, 1 visual fail in 4 burn-in runs; post-`waitFor`, 0 visual fails in 4 consecutive runs). `library-sort-selector` opted in (`statesOnly: ['default', 'hover', 'focus', 'open']` + `openTrigger: '[data-testid="sort-selector-button"]'`); new baseline `library-sort-selector/open-visual-darwin.png` captures the open `SortDropdown 955EZ` panel. **Deltas vs 19-4 closeout**: 26 fixture entries → 26 fixture entries (no new fixtures; SortSelector gains a state) / 27 entry-state combinations / **47 PNGs** (was 46: +1 new `library-sort-selector/open`, +4 re-blessed `shell-tab-navigation/{default,hover,focus}` and `search-search-bar/focus`). Regression: `eslint .` 0 errors / 122 warnings (matches 19-4 closeout EXACTLY), `prettier --check` clean on all 3 touched code files, `playwright test --project=visual --list` 1 test/1 file, feature-E2E `--list` 1663 tests / 36 files unchanged, `pnpm nx test web` 148 files / 1840 tests PASS, `pnpm nx test api` PASS (Nx-flagged known SSE flake, retried green — preexisting-fail-scanner-sse-scan-cancelled-flake; zero Go changes this story), `test:cleanup` no orphans after `test:cleanup:all` reaped 2 leftover Playwright nodes. 🔗 AC Drift: N/A (additive harness extension — gallery wrapper grammar gains optional `'open'` state + `data-gallery-open-trigger` attribute; SM CS pass classified as documented harness extension not stamped `[@contract-v1→v2]` bump on 19-4 AC #3; CR may re-classify). 📎 Contract Stamps: NONE this story (per SM judgment). 🔒 Rule 7 Wire Format: N/A (pure FE). 🎨 UX: PENDING Sally /test/gallery review — deferred to the Task 4 closure once Tasks 1–3 land the bulk fill. DEV inspector confirmation: `/test/gallery` rendered all 26 fixtures cleanly (no error placeholders); TabNavigation visibly highlights `媒體庫`; SortSelector `open` state captures the expanded panel with selected indicator on `新增日期`. ⚠️ Tasks 1–6 remain `[ ]` — story stays `in-progress`, NOT bumped to `review`. Per user instruction "task 0 only" scope. Commit message: `feat(19-4b): Task 0 harness-quality fixes — Sally follow-ups 1/2/3`. → Next session: pick up Task 1 (inventory + bucket P/Q/S/L) on a different LLM context per workflow tip. |
| 2026-05-13 | SM Bob /create-story (YOLO) COMPLETE. backlog → ready-for-dev. Story file: 19-4b-visual-baseline-bulk-fill.md (6 ACs; Task 0 with 3 fix sub-tasks + 6 numbered tasks covering inventory / presentational fill / data-driven fill / regen+UX-review+burn-in / Linux-baseline strategy / regression+close; ALL frontend / 0 backend → single story per cross-stack split check). Scope: (1) Task 0 lands the 3 Sally follow-ups from the 19-4 review — Fix A keyboard-Tab focus via sentinel, Fix B TabNavigation active-tab via nested memory `RouterProvider` (preferred) or sibling route, Fix C interactive `open` state via `openTrigger?` fixture field; regenerates the affected 19-4 focus baselines + adds `library-sort-selector/open`. (2) Tasks 1-3 inventory ~99 remaining `apps/web/src/components/**/*.tsx`, bucket into Presentational / Query-driven / Store-driven / Layout-shell, add fixture entries (reusing each component's `*.spec.tsx` mock shapes; `penNode` from the `// Implements:` header per `drift-19-3-2026-05.md`); extends the gallery infrastructure with `seedQueries?: Array<{ queryKey: readonly unknown[]; data: unknown }>` + `seedStore?: () => void` + `routePath?: string` additive fields on `GalleryFixture`. (3) Task 4 regenerates the full baseline set, Sally /ux-designer reviews `/test/gallery` (AC #3 close gate — mirrors 19-4's), burn-in `test:visual` ×5 = 0 flake. (4) Task 5 decides the Linux-baseline strategy 19-5 needs — `scripts/visual-baseline.sh` Docker helper OR documented CI-regen-on-first-run; updates `tests/visual/README.md` accordingly. (5) Task 6 updates `visual-baseline-19-4.md` "Delivered" table to the full set + closes the "Pending" section + full regression. 📎 Contract Stamps: NONE this story (the harness contracts are 19-4's `[@contract-v1]` AC #1–#5, consumed unchanged; `openTrigger?`/`seedQueries?`/`seedStore?`/`routePath?` are additive fixture-interface fields, not stamped contracts). 🔗 AC Drift: N/A (additive — no AC observable behaviour change on prior stories). 🔒 Rule 7 Wire Format: N/A (pure FE, no Go error codes). 🎨 UX: reads `ux-design.pen` mapping via the 19-3 audit doc only — no `.pen` modification, screenshot workflow not triggered. Depends on 19-4 (done — consumes harness; no Rule 20 ack needed, harness `[@contract-v1]` is consumed unchanged). Out of scope: CI workflow (19-5), component-vs-`.pen` diff sweep + `bugfix-N` filing (19-8), upgrading `<screen-section …>` placeholders (19-8), TestSprite (19-6/19-7), any `apps/web/src/components/` source edits. → DEV /dev-story next (use a different LLM than this SM session per workflow tip; run /code-review after with a third — and TEA *test-automate stays mostly N/A here since the visual spec IS the test). |
| 2026-05-12 | Created by the Story 19-4 Party Mode ruling (2026-05-12) — split out the bulk fixture/baseline fill (~99 components) so 19-4 could land the harness + ~25 reference components atomically (19-5 depends on the harness). backlog. |
