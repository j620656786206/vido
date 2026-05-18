# Visual-regression baselines (`visual` Playwright project — story 19-4)

Per-component pixel baselines for `apps/web/src/components/`. Part of **epic-19 (Design-Implementation
Drift Audit)**: catch the `HoverPreviewCard.tsx` ↔ `Component/PosterCardHover` class of drift
(bugfix-10-4 root cause) mechanically. 19-5 wires this into a PR-scoped CI job; 19-8 uses the
`data-pen-node` mapping to do the full component-vs-`.pen` sweep; Rule 22 retros use `pnpm run test:visual`
as the diff tool.

## How it works

- **`apps/web/src/routes/test/gallery.tsx`** — a DEV/test-only TanStack Router route (`/test/gallery`,
  mirrors `routes/test/manual-search.tsx`; inert in production builds) that renders every in-scope
  component inside `<section data-gallery-id="…" data-pen-node="…">` → up to three
  `<div data-gallery-state="default|hover|focus">` blocks. Each component is wrapped in a
  per-component `ErrorBoundary` so a broken fixture renders a labelled `[data-gallery-error]`
  placeholder instead of crashing the page (the spec skips snapshotting error placeholders).
- **`apps/web/src/routes/test/-gallery.fixtures.tsx`** — one entry per component
  (`{ id, label, component, props?, penNode, statesOnly?, width? }`). The `-` prefix keeps it out
  of the route tree (TanStack Router convention). QueryClient + Router context come from the app
  shell (`main.tsx`), so no extra providers are needed.
- **`tests/visual/components.visual.spec.ts`** — runs only under the `visual` project; derives the
  worklist from the live DOM (adding a component = adding a fixture entry, nothing here changes),
  applies each state (`hover()` / focus the first focusable descendant), and asserts
  `toHaveScreenshot(['components', <id>, '<state>.png'])`. Tagged `@visual @story-19-4`.
- **Determinism** (`playwright.config.ts` → `visual` project + `expect.toHaveScreenshot`):
  single browser (Chromium), `viewport` 1280×800, `colorScheme: 'dark'` (Vido's only theme),
  `reducedMotion: 'reduce'` + `animations: 'disabled'` (CSS transitions finish instantly before
  capture), `caret: 'hide'`, `maxDiffPixelRatio: 0.001` (≈0.1 % — loose end of the 0.05–0.1 % band;
  tighten once stable).

## `data-gallery-id` ↔ `.pen` node convention

`data-gallery-id` = kebab of the import path (`media/PosterCard` → `media-poster-card`).
`data-pen-node` is one of:

- a real `ux-design.pen` Reusable-Component node id (`RusTY`, `otvKh`, …) for Category-A files
  (the `// Implements: Component/X (id)` header — see `_bmad-output/audit/drift-19-3-2026-05.md`);
- the literal `screen-section` for files carrying `// Implements: <screen-section — pending epic-19-8 mapping>`;
- `utility` for in-scope Category-B files (`// Implements: <utility — no .pen counterpart>`).

epic-19-8 keys its component-vs-`.pen` sweep off `data-pen-node`. The full in-scope component table
(delivered vs. pending) lives in `_bmad-output/audit/visual-baseline-19-4.md`.

## Running

```bash
pnpm run test:visual          # verify against committed baselines (CI uses this — 19-5)
pnpm run test:visual:update   # (re)generate baselines  ← see discipline below
```

The `visual` project is **not** part of `pnpm run test:e2e` (the feature-E2E scripts list their
projects explicitly) — so the feature-E2E test count is unaffected.

## Baseline-update discipline (read before running `:update`)

1. Regenerate baselines **only** via `pnpm run test:visual:update` — never hand-edit PNGs.
2. Regenerate **only** after a _deliberate, reviewed_ design change. A diff that surprises you is a
   bug to investigate, not a baseline to bless (the bugfix-10-4 lesson: don't let test artifacts
   drift away from reality).
3. A baseline regeneration is **its own commit** — never mixed with logic changes. Commit message:
   `test(visual): rebaseline {component(s)} — {what changed & why}`.

   **Exception — architectural harness changes.** When a spec / gallery-route change _forces_ a
   baseline regeneration (the new and old baselines can only be interpreted under one architecture
   each), the spec/route change + the regenerated baselines MUST land atomically in one commit.
   Reverting only one side would leave the harness broken. Precedents: 19-4b Task 0 (Sally
   follow-ups A/B/C — spec sentinel/Tab + 4 re-blessed focus baselines + 1 new `open` baseline)
   and 19-4b Task 4 (single-fixture-per-page Plan-D — spec + gallery search params + 215 new
   baselines + 12 viewport-mode captures for `fixed inset-0` overlays). Document the exception in
   the commit body and the story's Completion Notes — do NOT generalise this to pure-baseline
   rebless work, which still follows the one-commit-per-rebless rule above.

4. **UX (Sally) reviews the rendered gallery** (`/test/gallery`) before a baseline set is committed —
   "first-Sally-approved web rendering, NOT `.pen` export". Record the review in the relevant story's
   Completion Notes.
5. **Platform suffix — Linux baselines are bootstrapped by 19-5's CI on first run** (decided
   19-4b Task 5, 2026-05-14). Baselines carry a `-{platform}` suffix (`-darwin` / `-linux`); the
   committed set is currently `darwin` (dev-machine). The `-linux` set will be generated by 19-5's
   CI workflow on its first execution and committed back via a one-off PR. After that PR merges,
   CI runs in verify-only mode (`pnpm run test:visual`) on every PR thereafter.

   **CI first-run decision tree** (for the 19-5 owner — implement in `.github/workflows/visual-regression.yml`):

   ```text
   if no `-linux.png` baseline exists for any fixture:
     run `pnpm run test:visual:update`
     open a one-off PR committing the `-linux` set
     PR MUST be tagged `requires-manual-review` (no auto-merge)
       — Sally reviews the Linux variants vs the darwin set already in repo
       — content drift unexpected (Sally already approved darwin content);
         expect only rendering drift (font hinting, emoji glyphs, sub-pixel)
     PR body MUST append a line to `_bmad-output/audit/visual-baseline-19-4.md`:
       `Linux baselines bootstrapped {YYYY-MM-DD} via CI image {full-digest-sha256:…}`
   else:
     run `pnpm run test:visual` (verify-only — fail → block PR)
   ```

   **Image pinning — pin the digest, not just the tag.** The Playwright Docker tag
   (`mcr.microsoft.com/playwright:vX.Y.Z-jammy`) is mutable: Ubuntu security updates to
   `fontconfig` / `freetype` will silently shift Linux glyph rendering and trigger mass-rebless.
   The CI workflow MUST resolve and pin a `sha256:…` digest at workflow-config time, and any
   image bump MUST be paired with a deliberate baseline-regeneration PR (same tag handling as
   any other reviewed-design-change rebless — `:update` + Sally gate + own commit).

   **Rejected alternative (Option A) — local `scripts/visual-baseline.sh` Docker helper.** A
   thin wrapper that `docker run`s `mcr.microsoft.com/playwright:vX.Y.Z-jammy` with the repo
   mounted and runs `:update` inside the container, producing `-linux` PNGs on the dev machine.
   Rejected because: (i) creates a second authoritative baseline-generation source (dev local +
   CI Linux) which violates "one authoritative environment" — risk of "I produced locally vs
   CI produced" drift; (ii) the `scripts/`-pinned image version bit-rots vs whatever 19-5's CI
   actually uses (digest drift); (iii) it crosses 19-4b's bounded context into 19-5's CI tooling
   scope (Party Mode 2026-05-12 ruling deliberately separated harness/coverage from CI
   plumbing). Re-considerable if 19-5 surfaces a concrete need for local Linux preview that
   the CI-regen path cannot serve. See `_bmad-output/audit/visual-baseline-19-4.md` for the
   full decision record.

## Adding a component

1. Add a fixture entry to `apps/web/src/routes/test/-gallery.fixtures.tsx` (`penNode` from the
   component's `// Implements:` header). Use `statesOnly: ['default']` for badges/skeletons/static
   components; set `width` if the component is inline/auto-width and would otherwise collapse.
2. `pnpm run test:visual:update` → new baseline PNGs appear under
   `tests/visual/components.visual.spec.ts-snapshots/components/{id}/`.
3. Eyeball `/test/gallery` (`pnpm nx serve web`), get UX sign-off, commit baselines + fixture together.

## `GalleryFixture` optional fields (19-4b harness extensions)

Beyond the core `{ id, label, component, props?, penNode, statesOnly?, width? }` shape, the fixture
interface supports four opt-in extensions added by 19-4b. The authoritative source for prop types
and JSDoc is the `GalleryFixture` interface in `-gallery.fixtures.tsx` (search for
`export interface GalleryFixture`); this section is a quick discovery overview.

- **`openTrigger?: string`** (Task 0c). CSS selector for an `interactive` open state. When set, the
  gallery emits an additional `<div data-gallery-state="open" data-gallery-open-trigger="…">`
  block and the visual spec clicks the selector inside the state div before screenshotting, then
  waits for `:is([role="listbox"], [role="menu"], [role="dialog"])` to be visible (1 s timeout +
  `.catch` fallback for popups that don't carry one of those roles). Add `'open'` to `statesOnly`
  to opt the fixture in. Reference fixture: `library-sort-selector` (4-state: default/hover/focus/
  open; `openTrigger: '[data-testid="sort-selector-button"]'`).

- **`routePath?: '/library' | '/downloads' | '/pending' | '/settings'`** (Task 0b). Renders the
  fixture inside a nested memory `RouterProvider` (`createMemoryHistory({ initialEntries: [routePath] })`)
  so `useRouterState()` consumers (e.g. `shell/TabNavigation` highlighting the active tab) paint
  the right state. The four allowed pathnames match `TabNavigation`'s `TABS.matchPaths`. Reference
  fixture: `shell-tab-navigation` (`routePath: '/library'` → `媒體庫` tab styled active).

- **`seedQueries?: ReadonlyArray<{ queryKey: readonly unknown[]; data: unknown }>`** (Task 3 Step A).
  Pre-populates the React-Query cache before the component mounts, so `useQuery()`-driven
  fixtures see seeded data on first read (no `isLoading` flash, no network attempt). Seeds fire
  synchronously inside a `useState(() => { ... })` initializer in `<GalleryFixtureSeed>`, which
  runs once during the first render BEFORE children commit. Reference fixtures: any in the Q-bucket
  (see `_bmad-output/audit/visual-baseline-19-4.md` Story-19-4b sub-tables — entries flagged
  "Q-bucket — seeded …").

- **`seedStore?: () => void`** (Task 3 Step A). Forward-compatibility hook for Zustand store-driven
  fixtures (called inside the same `useState` init as `seedQueries`). NOT used by any fixture today
  — `apps/web/src/components/` has no S-bucket consumers (stores live at the route level per
  project-context Rule 5). Stays in the interface so a future route-level fixture can opt in.

### Single-fixture-per-page architecture (Task 4 Plan-D)

For fixtures with `position: fixed` overlays (Radix `Dialog.Portal`, custom `fixed inset-0`
wrappers, side panels), multi-fixture rendering caused pointer-event collisions and made any
neighbour hover/focus impossible. The gallery route therefore accepts two opt-in search params:

- `?manifest=1` — renders a flat discovery list `<ul><li data-gallery-id>…</li></ul>` (no
  components mounted, no seeds, no state-divs). The visual spec uses this to enumerate fixture
  IDs once per run before navigating to each fixture individually.
- `?fixture=<id>` — filters `GALLERY_FIXTURES` to a single entry. Each fixture renders on its own
  page → zero cross-fixture interference.
- Neither param set → full gallery (dev-browse mode).

When a fixture's state-div has zero bbox (Radix portals escape to `document.body`; `position:fixed`
children leave the inline flow), the spec falls back to a viewport screenshot
(`expect(page).toHaveScreenshot(...)`) so the overlay paint is still captured. 12 fixtures hit
this fallback path today (the 12 `fixed inset-0` overlays — see audit doc Plan-D note).
