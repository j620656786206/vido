# Story 19.4: Playwright Per-Component Visual-Snapshot Baselines (Design-Drift Audit — Phase 2 Tooling)

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a frontend maintainer,
I want a Playwright visual-regression harness that renders every in-scope `apps/web/src/components/` component in its default / hover / focus states and pins a committed `toHaveScreenshot()` baseline per state,
so that design-implementation drift (the `HoverPreviewCard.tsx` ↔ `Component/PosterCardHover` divergence — bugfix-10-4 root cause) is caught automatically on every PR (19-5 wires this into CI) and the epic-19-8 sweep + every future Rule 22 retro audit has a real pixel-diff tool instead of a manual eyeball pass.

## Acceptance Criteria

1. [@contract-v1] **A dedicated Playwright "visual" project exists.** `playwright.config.ts` gains a project named `visual` with `testMatch: ['**/*.visual.spec.ts']` (these files live under `tests/visual/`, NOT `tests/e2e/`, so the existing `chromium`/`firefox`/`mobile-*`/`webkit-core` projects do **not** pick them up and the feature-level E2E count is unchanged). The `visual` project pins deterministic rendering: `use: { ...devices['Desktop Chrome'], viewport: { width: 1280, height: 800 }, colorScheme: 'dark', reducedMotion: 'reduce' }` (dark — Vido's only theme; `reducedMotion` kills CSS transitions so hover/focus snapshots are stable). Snapshot tolerance: `expect: { toHaveScreenshot: { maxDiffPixelRatio: 0.001, animations: 'disabled', caret: 'hide' } }` (≈0.1 % — the upper end of the sprint-status 0.05–0.1 % band; tighten to 0.0005 in a follow-up if the harness proves stable). The project is **excluded from the default `playwright test` run that CI's feature-E2E job uses** (CI calls the feature projects explicitly; 19-5 adds the separate visual job) — document how (e.g. the feature-E2E npm script / CI invocation lists projects explicitly, or `visual` is opt-in via `--project=visual`). Do not change any existing project's `testMatch`/`testDir`.

2. [@contract-v1] **Root npm scripts.** `package.json` gains `"test:visual": "playwright test --project=visual"` and `"test:visual:update": "playwright test --project=visual --update-snapshots"`. (Mirrors the existing `test:e2e*` script family. 19-5's CI job invokes `test:visual`; the `:update` script is the only sanctioned way to (re)generate baselines — see AC #6.)

3. [@contract-v1] **A dev-only component gallery route renders every in-scope component in isolation.** A new TanStack Router route — `apps/web/src/routes/test/gallery.tsx` (sibling of the existing `apps/web/src/routes/test/manual-search.tsx`; same `routes/test/` precedent — a dev aid that ships harmlessly, lazy-loaded) — renders each in-scope component inside a stable wrapper:
   ```tsx
   <section data-gallery-id="{kebab-id}" data-pen-node="{nodeId | 'screen-section' | 'utility'}">
     <div data-gallery-state="default"> <Component {...defaultFixture} /> </div>
     <div data-gallery-state="hover">   <Component {...defaultFixture} /> </div>   {/* Playwright hovers this one */}
     <div data-gallery-state="focus">   <Component {...defaultFixture} /> </div>   {/* Playwright focuses the first focusable descendant */}
   </section>
   ```
   - `{kebab-id}` is derived from the component's import path (e.g. `media/PosterCard.tsx` → `media-poster-card`).
   - `{nodeId}` is taken from the `// Implements:` header of the source file (the `_bmad-output/audit/drift-19-3-2026-05.md` mapping) — `RusTY` etc. for Category-A, the literal string `screen-section` for Category-C (`<screen-section …>`), `utility` for the few Category-B files that still render visible UI and are in scope. This attribute is what 19-8 keys off to line a baseline up against its `.pen` node.
   - Representative props live in **one fixtures module** — `apps/web/src/routes/test/gallery.fixtures.ts` (or a `gallery/` subfolder) — one entry per component, typed against the component's own `Props` type. Fixtures use the same mock-data shapes the component's `*.spec.tsx` already uses where possible (reuse, do not reinvent — Rule per project-context). Components needing providers (TanStack Query, Router context, a store) get wrapped by a shared `<GalleryProviders>` shell in the route.
   - Components rendered: every `apps/web/src/components/**/*.tsx` that renders visible UI — i.e. **exclude** the Category-B pure-utility/layout/type-module/hook files enumerated in `drift-19-3-2026-05.md` *that have no meaningful standalone rendering* (`parse/types.ts`, `degradation/types.ts`, `downloads/formatters.ts`, `parse/useParseProgress.ts`, the bare layout shells `AppShell`/`DashboardLayout`/`SettingsLayout`/`SetupWizard` — these get rendered only if a sensible isolated fixture exists; skips are listed in `_bmad-output/audit/visual-baseline-19-4.md` with a one-line reason). Target ≈ 80–100 components × 3 states (matches the sprint-status estimate). Skeletons / placeholders (`PosterCardSkeleton`, `ExploreBlockSkeleton`, `Skeleton`, `ColorPlaceholder`, `PlaceholderContent`, `SettingsPlaceholder`) ARE in scope (default state only — they have no hover/focus; the spec emits only the `default` snapshot for components flagged `statesOnly: ['default']` in the fixtures module).

4. [@contract-v1] **The visual spec drives the gallery.** `tests/visual/components.visual.spec.ts` navigates to `/test/gallery`, then for each `[data-gallery-id]` section and each `[data-gallery-state]` it owns: scrolls it into view, (for `hover`) `locator.hover()`, (for `focus`) focuses the first focusable descendant, waits for `reducedMotion` to have settled (no animation — a short `expect(locator).toBeVisible()` is enough), and asserts `await expect(stateLocator).toHaveScreenshot(['components', '{gallery-id}', '{state}.png'])`. One assertion per component-state. The spec derives the worklist **from the DOM the gallery renders** (so adding a component = adding a fixture entry, nothing else) — it does not hard-code the component list. Spec is tagged `@visual @story-19-4`.

5. [@contract-v1] **Committed baselines.** `pnpm run test:visual:update` is run, producing PNG baselines under `tests/visual/components.visual.spec.ts-snapshots/components/{gallery-id}/{state}.png` (Playwright's default snapshot path; one platform — `darwin`/`linux` suffix is whatever the dev machine produces, and 19-5's CI job will regenerate the Linux set or the harness pins `--ignore-snapshots`-style platform handling — note the chosen approach). **Every baseline PNG is committed.** A human (Sally / UX) reviews the generated gallery rendering *before* the baselines are committed — the story's Completion Notes must record that this review happened and link the gallery URL / screenshot evidence; an un-reviewed baseline set is not a valid story close (the sprint-status note: "Baseline source: first-Sally-approved web rendering, NOT .pen export").

6. **Baseline-update discipline is documented.** A short `tests/visual/README.md` states: baselines are regenerated *only* via `pnpm run test:visual:update`, *only* after a deliberate, reviewed design change, and the regeneration must be its own commit (no mixing baseline churn with logic changes) — mirrors the bugfix-10-4 lesson about test claims drifting from reality. The README also documents the `data-gallery-id` ↔ `.pen`-node mapping convention so 19-8 and Rule 22 audits can use it.

7. **`_bmad-output/audit/visual-baseline-19-4.md`** is created: lists every component in scope with its `data-gallery-id`, its `.pen` node (or `screen-section` / `utility`), the states captured, and any deliberate skips with reasons. This is the durable handoff doc for 19-5 (CI) and 19-8 (sweep) — analogous to `drift-19-3-2026-05.md`. It supersedes/extends the Category-A/B/C tables in `drift-19-3-2026-05.md` with the actual rendered-baseline status. _(Also: fix the now-stale "Kept in `scripts/` as the auditable record" sentence in `drift-19-3-2026-05.md` — `scripts/backfill-rule21-headers.mjs` was removed in the 19-3 CR follow-up, commit `cf10b20`; the mapping lives in that audit doc itself.)_

8. **No production behaviour change, no `components/` source change.** The gallery route is the only new app-source file; it is lazy-loaded and either guarded so it renders nothing in a production build or left as a harmless `routes/test/` dev aid (state which, following the `manual-search.tsx` precedent). Zero edits to files under `apps/web/src/components/` ⇒ the `local/implements-pen-node-id` ESLint rule (19-3) stays trivially green. `pnpm lint:all` is **0 errors** at close (the new `.visual.spec.ts`, the gallery route, the fixtures module, and `gallery.fixtures.ts` must pass ESLint + Prettier; route files are `<route-only>` / out-of-scope for Rule 21, spec/fixtures are tests). Warning count matches the prior baseline (bugfix-10-7 / 19-3: 122).

9. **Regression + framework hygiene.** `pnpm run test:visual` is green (all baselines match on a clean re-run). `pnpm nx test web` and `pnpm nx test api` still pass (the gallery route adds a `routes/test/gallery.spec.tsx`-style smoke test only if trivial — otherwise the visual spec is its only coverage; do not add a brittle RTL test for a gallery aggregator). `pnpm test:e2e` (feature E2E) test count is **unchanged** (the `visual` project is separate). `pnpm run test:cleanup` shows no orphaned processes. `ux-design.pen` is **not** modified (this story reads the 19-3 audit-doc mapping; it does not touch `.pen`), so `scripts/export-pen-screenshots.py` is not run and the CLAUDE.md screenshot workflow does not trigger.

10. **`project-context.md` Rule 22 tooling line is updated.** The line `Tooling: Playwright toHaveScreenshot() (story 19-4) automates diff calculation.` → present-tense, naming the harness: the `visual` Playwright project + `pnpm run test:visual` + the gallery route + where baselines live + where the audit doc is. The `Last Updated` header gets a story-19-4 entry. No other Rule 21/22 prose changes.

## Tasks / Subtasks

- [ ] Task 1: Playwright `visual` project + npm scripts (AC: #1, #2)
  - [ ] Add the `visual` project to `playwright.config.ts` (`testDir`/`testMatch` → `tests/visual/**/*.visual.spec.ts`; `viewport` 1280×800; `colorScheme: 'dark'`; `reducedMotion: 'reduce'`; `expect.toHaveScreenshot` `{ maxDiffPixelRatio: 0.001, animations: 'disabled', caret: 'hide' }`). Do NOT alter existing projects.
  - [ ] Confirm the feature-E2E invocation (root `test:e2e` script / CI job) does not pick up the `visual` project — if `playwright test` with no `--project` would now include it, make the feature run explicit (`--project=chromium --project=firefox …`) or move `visual` behind an opt-in. Document the decision in `tests/visual/README.md`.
  - [ ] Add `package.json` scripts `test:visual` + `test:visual:update`.
- [ ] Task 2: Gallery providers shell + fixtures module (AC: #3)
  - [ ] `apps/web/src/routes/test/gallery.fixtures.ts` — one typed entry per in-scope component: `{ id, component, defaultProps, penNode, statesOnly? }`. Reuse mock shapes from each component's existing `*.spec.tsx` where one exists.
  - [ ] `<GalleryProviders>` shell wrapping QueryClientProvider (test client, no retries), any required store providers, and a Router stub if a component calls `useNavigate`/`useRouter` (mock per the project-context "TanStack Router useNavigate in tests: mock it directly" note).
  - [ ] Resolve every `penNode` from the source file's `// Implements:` header (Category A → real id; `<screen-section …>` → `'screen-section'`; `<utility …>` → `'utility'`). Cross-check against `drift-19-3-2026-05.md`.
- [ ] Task 3: Gallery route (AC: #3, #8)
  - [ ] `apps/web/src/routes/test/gallery.tsx` — iterates the fixtures module, renders each as the `<section data-gallery-id data-pen-node>` → 1–3 `<div data-gallery-state>` blocks (skip `hover`/`focus` blocks when `statesOnly` is set). Lazy-load the route; guard so it's inert (or 404s) under `import.meta.env.PROD` OR document why the `routes/test/` precedent makes that unnecessary.
  - [ ] Visual sanity: `pnpm nx serve web`, open `http://localhost:4200/test/gallery`, eyeball that every component renders without crashing (broken fixtures show here).
- [ ] Task 4: Visual spec (AC: #4)
  - [ ] `tests/visual/components.visual.spec.ts` — navigate `/test/gallery`, enumerate `[data-gallery-id]`, for each owned `[data-gallery-state]`: scroll into view, hover/focus as appropriate, `await expect(stateLocator).toHaveScreenshot(['components', id, `${state}.png`])`. Worklist derived from the live DOM, not hard-coded. Tag `@visual @story-19-4`.
- [ ] Task 5: Generate, review & commit baselines (AC: #5)
  - [ ] `pnpm run test:visual:update` → PNG baselines under `tests/visual/components.visual.spec.ts-snapshots/`.
  - [ ] **Sally / UX review** of the rendered gallery BEFORE committing — record in Completion Notes (gallery URL + that the review happened + any fixture fixes that resulted). Note the platform-suffix situation (`-darwin`/`-linux`) and the chosen handling (regenerate-on-CI vs. commit-dev-platform vs. Docker-generated Linux baselines — pick one; 19-5 will rely on it).
  - [ ] `git add` every baseline PNG. Re-run `pnpm run test:visual` → all green.
- [ ] Task 6: Docs & audit (AC: #6, #7, #10)
  - [ ] `tests/visual/README.md` — update discipline + `data-gallery-id` ↔ `.pen`-node convention.
  - [ ] `_bmad-output/audit/visual-baseline-19-4.md` — full in-scope component table (id, pen node, states, skips+reasons).
  - [ ] Fix the stale `scripts/backfill-rule21-headers.mjs` sentence in `drift-19-3-2026-05.md` (script removed in commit `cf10b20`).
  - [ ] `project-context.md` Rule 22 tooling line → present tense naming the harness; `Last Updated` header += story-19-4 entry.
- [ ] Task 7: Full regression + close (AC: #8, #9)
  - [ ] `pnpm lint:all` → 0 errors / 122 warnings; `prettier --check .` clean.
  - [ ] `pnpm run test:visual` green; `pnpm nx test web` + `pnpm nx test api` pass; `pnpm test:e2e` count unchanged.
  - [ ] `pnpm run test:cleanup` → no orphans. `ux-design.pen` untouched.

## Dev Notes

### Why this story exists / where it sits in epic-19

- bugfix-10-4 root cause (Party Mode 2026-05-08): `HoverPreviewCard.tsx` diverged from `.pen` `Component/PosterCardHover` (node `MQbvp`) undetected for months. Epic-19 is the systemic fix. **19-1** added Rule 21 to `project-context.md`; **19-3** made it CI-enforced (the `local/implements-pen-node-id` ESLint rule + header backfill across all 131 `components/` files — 12 mapped to `.pen` Reusable Components, 25 `<utility …>`, 94 `<screen-section — pending epic-19-8 mapping>`). **This story (19-4)** is the *visual* half: render those components and pin a pixel baseline so future drift is mechanically detectable. **19-5** wires `test:visual` into a PR-scoped GitHub Actions job. **19-8** uses this harness (the `data-pen-node` attributes + the audit doc) to do the full component-vs-`.pen` sweep and file `bugfix-N` stories for material drift. **Rule 22** retros use `pnpm run test:visual` as the diff tool.
- Dependency: **depends on 19-3 (done)** — specifically on `_bmad-output/audit/drift-19-3-2026-05.md`, which is the file→`.pen`-node mapping this story embeds as `data-pen-node`. (No Rule 20 ack needed: 19-3's `[@contract-v2]` stamps cover the `// Implements:` *marker grammar* — a code-comment contract this story does not extend; this story consumes the audit *doc* (not a versioned AC) and the ESLint rule's *existence* (not its grammar). Implicit-v0 / ack-skipped per Rule 20.)

### Architecture / constraints — read before implementing

- **All frontend.** 0 Go, 0 migrations, 0 swagger, 0 backend tests. Cross-stack split check: backend task count = 0 → single story is correct (the `>3 each side` threshold is not met). This story is *large* (≈80–100 components × ≤3 states ≈ 240 baselines + a fixture per component) — batch Task 2/3 by `components/` subfolder; nothing here needs to be done in one sitting, but it is one story (the work is homogeneous and the harness must land atomically before 19-5/19-8).
- **No Storybook, no Playwright component-testing in this repo.** Decision (this story): use a **dev-only TanStack Router gallery route + the existing Playwright runner**, not `@playwright/experimental-ct-react`. Rationale: (a) the codebase already has `apps/web/src/routes/test/manual-search.tsx` as a shipped dev aid — direct precedent; (b) rendering inside the real app gives real Tailwind/theme/provider context (the whole point — we want to catch drift in *how the component actually renders in the app*, not in an artificial mount); (c) adding `@playwright/experimental-ct-react` means a new dep + a second Vite config + a second test-runner config with zero existing precedent — rejected as disproportionate. The gallery route is the one new app-source file.
- **Determinism is everything for visual tests** (project-context Rule 16 + the bugfix-10-3 StrictMode lesson): `reducedMotion: 'reduce'` + `animations: 'disabled'` kill CSS transitions (Vido's hover/focus states are pure CSS — `lg:group-hover:*`, `focus-visible:*`); fixed `viewport`; `colorScheme: 'dark'` (Vido has no light theme — but pin it so a future light theme doesn't silently rebaseline everything); `caret: 'hide'`; mock all data via fixtures (no network — `webServer` still starts the Go backend for the dev server, but the gallery components get props directly and shouldn't fetch — if one does, mock it in `<GalleryProviders>`). `maxDiffPixelRatio: 0.001` ≈ the loose end of the sprint band; font-rendering differs across OSes (the bugfix-10-7 / 10-4 spikes noted Chromium GPU quirks) — hence the platform-suffix discussion in AC #5 / Task 5.
- **Hover/focus** — Vido components express these via Tailwind `group-hover:` (parent-driven) and `focus-visible:`. For the `hover` state block, `locator.hover()` over the section is usually enough (the `group` is the component root); for `focus`, focus the first `button`/`a`/`[tabindex]` descendant (`section.locator(':is(button,a,input,select,textarea,[tabindex]):not([tabindex="-1"])').first().focus()`). Components with no interactive descendant → `statesOnly: ['default']` (skeletons, badges, layout shells). PosterCard's hover is the canonical hard case (the bugfix-10-4 viewport-flip + bugfix-10-7 info-density work) — its fixture should exercise the library-admin `metadataSource` badge path that bugfix-10-4 H2 silently broke.
- **`routes/test/` and the prod bundle** — `manual-search.tsx` already ships in prod, so a `gallery.tsx` sibling is consistent. Prefer `createLazyFileRoute` so it's a separate chunk. A cheap belt-and-braces guard: have the route component early-return a `<NotFound />`/redirect when `import.meta.env.PROD` — document whichever you pick.
- **Snapshot file location** — Playwright's default is `<specfile>-snapshots/`. Keep that (`tests/visual/components.visual.spec.ts-snapshots/`). Do not invent a custom `snapshotPathTemplate` unless the default produces collisions — if you do, document it in `tests/visual/README.md` and tell 19-5.
- **CI is out of scope** — 19-5 owns `.github/workflows/visual-regression.yml`. This story must leave `pnpm run test:visual` green *locally* and the baselines committed; it must NOT add a CI workflow file. (If the dev-machine vs. CI-Linux font-rendering gap looks like it'll bite 19-5, the cleanest mitigation — generate baselines inside the same Docker image CI uses — can be set up here as a `scripts/visual-baseline.sh` helper; optional, note it.)

### Project Structure Notes

- New: `apps/web/src/routes/test/gallery.tsx`, `apps/web/src/routes/test/gallery.fixtures.ts` (+ optional `gallery/` subfolder if the fixtures file gets unwieldy — split by `components/` subdir), `tests/visual/components.visual.spec.ts`, `tests/visual/README.md`, `tests/visual/components.visual.spec.ts-snapshots/**` (the committed PNGs), `_bmad-output/audit/visual-baseline-19-4.md`.
- Modified: `playwright.config.ts` (one new project), `package.json` (two scripts), `project-context.md` (Rule 22 tooling line + `Last Updated`), `_bmad-output/audit/drift-19-3-2026-05.md` (one stale-sentence fix), `_bmad-output/implementation-artifacts/sprint-status.yaml` (19-4 status).
- **Zero** edits under `apps/web/src/components/` — if you find yourself wanting to add a `data-testid` to a component to make the gallery work, stop: use the `<section data-gallery-id>` wrapper in the *route* instead. (One exception worth flagging back to the SM if it comes up: if a component is genuinely unrenderable in isolation without a prop it doesn't expose, that's a 19-8-style finding, not something to patch here.)
- Out of scope: CI workflow (19-5), the comprehensive component-vs-`.pen` *diff* sweep + `bugfix-N` filing (19-8), any TestSprite work (19-6/19-7), upgrading any `<screen-section …>` placeholder to a canonical Rule 21 header (19-8), Rule 22 retro execution.

### Testing standards (project-context.md)

- E2E/visual: Playwright. After ANY run: `pnpm run test:cleanup` (project-context "Test Process Cleanup"; the `globalSetup`/`globalTeardown` already track spawned servers).
- Don't test CSS hover transitions with `toBeVisible()` in the *narrow* sense the MEMORY note warns about — but here we explicitly disable animations, so `toHaveScreenshot` on the post-transition steady state is the correct assertion (the warning is about racing an in-flight `opacity` transition; `reducedMotion: 'reduce'` removes the race).
- Vitest (if any gallery smoke test): co-located, `toBeInTheDocument`/`toEqual` not `toBeTruthy` (Rule 16). Prefer *no* unit test for the gallery aggregator — the visual spec is its real coverage; a brittle "renders 80 sections" RTL test is dead weight (bugfix-10-3 "don't add a regression test for a non-existent bug" spirit).
- Lint gate (Rule 12): `pnpm lint:all` = `go vet` → `staticcheck` → `eslint .` → `prettier --check .`, 0 errors at close; warnings = 122 (the bugfix-10-7 / 19-3 baseline). `eslint .` covers `apps/web/`, `libs/shared-types/`, `tests/` — so the new `tests/visual/*.ts` must lint clean.

### Rule 21 / Rule 22 linkage

- This story produces `_bmad-output/audit/visual-baseline-19-4.md` (the rendered-baseline status table) and the live `data-pen-node` attributes in the gallery — both are what epic-19-8 builds on. It updates the Rule 22 tooling line to point at the now-real harness. It does NOT itself classify drift (that's 19-8 + per-epic Rule 22 retros).
- Rule 21: no new `components/` files → ESLint rule (19-3) is satisfied trivially. The gallery *route* is `<route-only>` per Rule 21's exemptions; spec/fixtures are tests (exempt). Phase-1 SM-template `// Implements:` Dev-Notes gate: N/A (no designed-component file added).

### References

- [Source: _bmad-output/implementation-artifacts/sprint-status.yaml:523] — the 19-4 charter line (per-component default+hover+focus, ~80 components × 3 states, baseline = first-Sally-approved web rendering not `.pen` export, pixel diff 0.05–0.1 %, depends on 19-3)
- [Source: _bmad-output/implementation-artifacts/sprint-status.yaml:494-518] — epic-19 header (origin, dependency order, agent routing)
- [Source: _bmad-output/audit/drift-19-3-2026-05.md] — the file→`.pen`-node mapping this story embeds as `data-pen-node` (Category A/B/C tables; "durable record for epic-19-4" per the doc itself)
- [Source: _bmad-output/implementation-artifacts/19-3-eslint-pen-node-id-rule.md] — predecessor story; the `// Implements:` headers this story reads; the 19-3 CR follow-up (commit `cf10b20`) that removed `scripts/backfill-rule21-headers.mjs`
- [Source: project-context.md#Rule-22-Epic-Retro-Design-Drift-Audit] — the tooling line this story makes real (`Playwright toHaveScreenshot() (story 19-4)`); the 0.5 %/5 % classification bands
- [Source: project-context.md#Rule-21-Component-to-Design-Node-Traceability] — marker grammar, exemptions (`<route-only>`, tests exempt)
- [Source: project-context.md#Rule-12-Code-Quality-Checks-CI-based] / [#Rule-16-Test-Assertion-Quality] — `pnpm lint:all` order; assertion-matcher rules
- [Source: playwright.config.ts] — current project list (`chromium`/`firefox`/`webkit-core`/`mobile-chrome`/`mobile-safari`), `expect.timeout` 10s, `webServer` (Go backend on :8080 + Vite on :4200), `globalSetup`/`globalTeardown`
- [Source: apps/web/src/routes/test/manual-search.tsx] — precedent for a shipped `routes/test/` dev-aid route
- [Source: tests/e2e/poster-card-hover.spec.ts:32] — the existing "`Visual regression baseline → story 19-4 (Playwright toHaveScreenshot)`" comment this story fulfils; the PosterCard hover behaviour to baseline
- [Source: package.json#scripts] — the `test:e2e*` script family `test:visual*` mirrors
- [Source: CLAUDE.md] — `routes/test/` precedent; screenshot-export workflow only triggers on `.pen` *modification* (not the case here)

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List

## Change Log

| Date | Change |
| ---- | ------ |
| 2026-05-12 | [@contract-v0→v1] AC #1–#5 stamped on creation — what's defined: the `visual` Playwright project (name `visual`, `testMatch` `tests/visual/**/*.visual.spec.ts`, deterministic `viewport`/`colorScheme`/`reducedMotion`, `maxDiffPixelRatio` 0.001), the `test:visual` / `test:visual:update` npm scripts, the dev-only `apps/web/src/routes/test/gallery.tsx` route + its `<section data-gallery-id data-pen-node>` wrapper contract + the `gallery.fixtures.ts` shape, the `tests/visual/components.visual.spec.ts` DOM-derived worklist, and the committed-baseline location (`tests/visual/components.visual.spec.ts-snapshots/components/{id}/{state}.png`). What breaks downstream: 19-5's CI job depends on the `test:visual` script + the `visual` project name + the baseline path; 19-8 depends on the `data-pen-node` attribute convention + `_bmad-output/audit/visual-baseline-19-4.md` + the gallery route existing; Rule 22 retros depend on `pnpm run test:visual` being the diff tool. Upstream 19-3 carries `[@contract-v2]` on its `// Implements:` marker grammar — not consumed here (this story adds no `components/` headers); the consumed artifact is the `drift-19-3-2026-05.md` mapping doc (not a versioned AC) → no ack, implicit-v0 per Rule 20. |
| 2026-05-12 | SM Bob /create-story (YOLO) — story drafted ready-for-dev. ALL frontend / 0 backend → single story (cross-stack split check N/A). 10 ACs (#1–#5 stamped `[@contract-v1]`), 7 tasks. Key decision recorded in Dev Notes: dev-only TanStack Router gallery route + existing Playwright runner (NOT `@playwright/experimental-ct-react`) — `routes/test/manual-search.tsx` precedent + real-app render context. Depends on 19-3 (done). 🔒 Rule 7 Wire Format: N/A (pure FE). 🎨 UX: reads `ux-design.pen` mapping via the 19-3 audit doc only — no `.pen` modification, screenshot workflow not triggered; Sally/UX review of the rendered gallery is a story-close gate (AC #5). |
