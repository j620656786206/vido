# Story 19.4b: Visual-Baseline Bulk Fill (remaining ~99 components)

Status: ready-for-dev

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

- [ ] **0a. Fix A — Keyboard-driven `focus` state (Sally follow-up #1).** Programmatic `locator.focus()` does not trigger Chromium's `:focus-visible` rules; this story replaces it with a sentinel-then-Tab pattern so the input modality is set to keyboard.
  - [ ] In `apps/web/src/routes/test/gallery.tsx`: render a hidden, focusable sentinel button immediately before each `<div data-gallery-state>`. Use `<button type="button" data-gallery-sentinel="pre" aria-hidden="true" tabIndex={0} className="sr-only" />` (sr-only keeps it out of the visible gallery without removing it from the tab order).
  - [ ] In `tests/visual/components.visual.spec.ts`, for `state === 'focus'`: locate the preceding sentinel via `stateDiv.locator('xpath=preceding-sibling::*[@data-gallery-sentinel="pre"][1]')`, call `await sentinel.focus()` then `await page.keyboard.press('Tab')`. Keep the existing fallback for state-divs with no focusable descendant (scroll into view, capture default-equivalent baseline). Update the spec header doc-comment to reference 19-4b Task 0 Fix A.
  - [ ] Regenerate the 10 existing 3-state-baseline focus PNGs via `pnpm run test:visual:update`. Many components have identical `:focus` and `:focus-visible` styles, so several focus PNGs may not actually change — that is expected. Note in Completion Notes which baselines changed pixel-wise and which were re-blessed unchanged.

- [ ] **0b. Fix B — TabNavigation active-tab state (Sally follow-up #2).** `TabNavigation` reads `useRouterState().location.pathname` and matches against `TABS.matchPaths = ['/library', '/downloads', '/pending', '/settings']`; the gallery route `/test/gallery` matches none → no active tab. Pick **one** of the two implementations below and stick to it (record the choice + rationale in Completion Notes):
  - **Option B1 — nested memory `RouterProvider` (preferred).** In the gallery route, for the `shell-tab-navigation` fixture (and any future router-dependent fixture), wrap the rendered component in a sub-router with `createMemoryHistory({ initialEntries: ['/library'] })`. The `useRouterState` hook reads from the nearest provider, so the inner snapshot reports `/library` → `TabActive (TboA7)` paints. `<Link to>` inside `TabNavigation` may be typed against the main app `routeTree.gen.ts` and complain about the stub router's untyped tree — suppress with a targeted `@ts-expect-error` + a comment pointing back to this AC. Add to fixture: `routePath?: '/library' | '/downloads' | '/pending' | '/settings'` and let `shell-tab-navigation` declare `routePath: '/library'`.
  - **Option B2 — sibling route file matching a TAB path.** Add `apps/web/src/routes/test/-tabnav-{library,downloads,pending,settings}.tsx` (the `-` prefix keeps each out of the route tree — they're imported by the gallery only). Each just renders `<TabNavigation />`. The visual spec, when it sees `routePath` on a fixture, navigates to that path and screenshots the locator from there instead of from `/test/gallery`. Rejected if the new route files would themselves need data mocking; B1 is the cleaner of the two.
  - [ ] Update `-gallery.fixtures.tsx`'s `shell-tab-navigation` fixture: remove the FIXME comment block (it points at this task), add `routePath: '/library'` (or whichever path the chosen option needs).
  - [ ] Regenerate `shell-tab-navigation/{default,hover,focus}-visual-darwin.png`. Sally re-reviews these three at the gallery sign-off.

- [ ] **0c. Fix C — Interactive `open` state via `openTrigger` (Sally follow-up #3).** Add an `open` state for fixtures that have a click-to-open sub-UI (dropdown / menu / modal — e.g. `SortSelector`'s `SortDropdown 955EZ`, `PosterCardMenu`, any future modal/dialog).
  - [ ] Extend `GalleryFixture` in `-gallery.fixtures.tsx`: add `'open'` to the `GalleryState` union; add `openTrigger?: string` (CSS selector relative to the rendered component, searched inside the state div).
  - [ ] In `gallery.tsx`: when iterating `requestedStates`, filter out `'open'` if `!fx.openTrigger` (silent drop — fixtures without a trigger don't get an `open` baseline). Render the `open` state div with `data-gallery-open-trigger={fx.openTrigger}` so the spec can read it.
  - [ ] In `components.visual.spec.ts`, add a new branch: `else if (state === 'open') { const trigger = await stateDiv.getAttribute('data-gallery-open-trigger'); if (trigger) await stateDiv.locator(trigger).first().click(); }`. The existing `expect(stateDiv).toHaveScreenshot(...)` then captures the post-click state.
  - [ ] Add `library-sort-selector`'s `open` state to the 19-4 fixture (the reference case): set `statesOnly: ['default', 'hover', 'focus', 'open']` + `openTrigger: '[data-testid="sort-selector-button"]'`. This produces a new baseline `library-sort-selector/open-visual-darwin.png` that captures the open dropdown panel `955EZ`.
  - [ ] Inventory other 19-4 reference fixtures that have an obvious `openTrigger` and opt them in — leave the rest for Task 2/3 to declare per-fixture.
  - [ ] Update `tests/visual/README.md`: document the `open` state + `openTrigger` field in the "Adding a component" section.

- [ ] **0d. Burn-in + commit.** `pnpm run test:visual` ×3 → 0 flake. Lint + Prettier clean on touched files. Update the existing audit doc's "Delivered baselines" table — keep header `(25 unique components / 26 fixture entries / 46 PNGs)` and note in Completion Notes that 19-4b's Task 0 has added `library-sort-selector/open` (now 27 fixture entries / 47 PNGs) and re-blessed the 10 focus baselines. Commit: `feat(19-4b): Task 0 harness-quality fixes — Sally follow-ups 1/2/3`.

### Task 1: Inventory remaining components & bucket data-driven vs. presentational (AC: #2, #4)

- [ ] Generate the full in-scope list. Quick recipe (record exact commands run in Debug Log References):
  ```bash
  # All .tsx components, minus tests/index barrels
  find apps/web/src/components -name '*.tsx' ! -name '*.spec.tsx' ! -name '*.test.tsx' \
    ! -name 'index.ts' | sort > /tmp/all-components.txt
  # Already-covered fixture ids → component-paths (reverse the kebab → path mapping)
  grep -oE "'[a-z][a-z0-9-]+'" apps/web/src/routes/test/-gallery.fixtures.tsx \
    | head -n 200 > /tmp/covered-ids.txt
  ```
- [ ] For each not-yet-covered component, classify into one of FOUR buckets — record the bucket in a working notes section under "Debug Log References":
  - **P (Presentational)** — pure props-in, no `useQuery` / `useMutation` / store reads / router reads. → goes in Task 2.
  - **Q (Query-driven)** — needs React-Query data. → goes in Task 3 (seed via `seedQueries`).
  - **S (Store-driven)** — reads from a Zustand store (selection, library-filters, etc.). → goes in Task 3 (seed via `seedStore`).
  - **L (Layout shell / no isolated render)** — `AppShell`, `DashboardLayout`, `SettingsLayout`, `SetupWizard`, etc. → recorded as deliberate skip per AC #4 unless a trivial isolated fixture exists.
- [ ] Confirm the four type/util-only files stay skipped (`parse/types.ts`, `degradation/types.ts`, `downloads/formatters.ts`, `parse/useParseProgress.ts`) — these were skipped in 19-4 and remain skipped here.
- [ ] Cross-check the inventory against `_bmad-output/audit/drift-19-3-2026-05.md` Category-A/B/C tables — any newly-classified Category-A (real `.pen` Reusable-Component mapping) gets that node id in its fixture's `penNode`; everything else uses `'screen-section'` (the 19-3 Phase-2 placeholder) or `'utility'`.

### Task 2: Add fixtures — Presentational bucket first (AC: #2)

- [ ] For each P-bucket component, add a fixture entry. Reuse the component's own `*.spec.tsx` mock-data shapes for props (do not re-invent — Rule per `project-context.md`). The `penNode` value MUST come from the component file's `// Implements:` header (Rule 21-enforced by `local/implements-pen-node-id` ESLint rule).
- [ ] Group fixtures in the file by `components/` subfolder (the existing convention — `ui/` → `media/` → `degradation/` → `library/` → …). Keep one-fixture-per-line readable formatting (Prettier handles wrapping).
- [ ] For badges / skeletons / static labels: `statesOnly: ['default']` (no meaningful hover/focus). For interactive elements (buttons, links, inputs): keep the default three states.
- [ ] For inline / auto-width components that would collapse to 0-width in isolation (badges, chips): set a sensible `width` (typically 200–640 px).
- [ ] Spot-check renders in `pnpm nx serve web` → `/test/gallery` as you add batches of ~10. Use the per-fixture `FixtureErrorBoundary` to identify broken props quickly (`[data-gallery-error]` placeholders render with the error message; the spec already skips them).

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

### Debug Log References

### Completion Notes List

### File List

## Change Log

| Date | Change |
| ---- | ------ |
| 2026-05-13 | SM Bob /create-story (YOLO) COMPLETE. backlog → ready-for-dev. Story file: 19-4b-visual-baseline-bulk-fill.md (6 ACs; Task 0 with 3 fix sub-tasks + 6 numbered tasks covering inventory / presentational fill / data-driven fill / regen+UX-review+burn-in / Linux-baseline strategy / regression+close; ALL frontend / 0 backend → single story per cross-stack split check). Scope: (1) Task 0 lands the 3 Sally follow-ups from the 19-4 review — Fix A keyboard-Tab focus via sentinel, Fix B TabNavigation active-tab via nested memory `RouterProvider` (preferred) or sibling route, Fix C interactive `open` state via `openTrigger?` fixture field; regenerates the affected 19-4 focus baselines + adds `library-sort-selector/open`. (2) Tasks 1-3 inventory ~99 remaining `apps/web/src/components/**/*.tsx`, bucket into Presentational / Query-driven / Store-driven / Layout-shell, add fixture entries (reusing each component's `*.spec.tsx` mock shapes; `penNode` from the `// Implements:` header per `drift-19-3-2026-05.md`); extends the gallery infrastructure with `seedQueries?: Array<{ queryKey: readonly unknown[]; data: unknown }>` + `seedStore?: () => void` + `routePath?: string` additive fields on `GalleryFixture`. (3) Task 4 regenerates the full baseline set, Sally /ux-designer reviews `/test/gallery` (AC #3 close gate — mirrors 19-4's), burn-in `test:visual` ×5 = 0 flake. (4) Task 5 decides the Linux-baseline strategy 19-5 needs — `scripts/visual-baseline.sh` Docker helper OR documented CI-regen-on-first-run; updates `tests/visual/README.md` accordingly. (5) Task 6 updates `visual-baseline-19-4.md` "Delivered" table to the full set + closes the "Pending" section + full regression. 📎 Contract Stamps: NONE this story (the harness contracts are 19-4's `[@contract-v1]` AC #1–#5, consumed unchanged; `openTrigger?`/`seedQueries?`/`seedStore?`/`routePath?` are additive fixture-interface fields, not stamped contracts). 🔗 AC Drift: N/A (additive — no AC observable behaviour change on prior stories). 🔒 Rule 7 Wire Format: N/A (pure FE, no Go error codes). 🎨 UX: reads `ux-design.pen` mapping via the 19-3 audit doc only — no `.pen` modification, screenshot workflow not triggered. Depends on 19-4 (done — consumes harness; no Rule 20 ack needed, harness `[@contract-v1]` is consumed unchanged). Out of scope: CI workflow (19-5), component-vs-`.pen` diff sweep + `bugfix-N` filing (19-8), upgrading `<screen-section …>` placeholders (19-8), TestSprite (19-6/19-7), any `apps/web/src/components/` source edits. → DEV /dev-story next (use a different LLM than this SM session per workflow tip; run /code-review after with a third — and TEA *test-automate stays mostly N/A here since the visual spec IS the test). |
| 2026-05-12 | Created by the Story 19-4 Party Mode ruling (2026-05-12) — split out the bulk fixture/baseline fill (~99 components) so 19-4 could land the harness + ~25 reference components atomically (19-5 depends on the harness). backlog. |
