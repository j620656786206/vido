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
4. **UX (Sally) reviews the rendered gallery** (`/test/gallery`) before a baseline set is committed —
   "first-Sally-approved web rendering, NOT `.pen` export". Record the review in the relevant story's
   Completion Notes.
5. Platform suffix: baselines carry a `-{platform}` suffix (`-darwin` / `-linux`). The committed set
   is currently `darwin` (dev-machine). CI (Linux, 19-5) will either regenerate the `-linux` set via
   `:update` in a one-off commit, or 19-4b/19-5 adds `scripts/visual-baseline.sh` to generate the
   Linux set inside the CI Docker image. See `_bmad-output/audit/visual-baseline-19-4.md`.

## Adding a component

1. Add a fixture entry to `apps/web/src/routes/test/-gallery.fixtures.tsx` (`penNode` from the
   component's `// Implements:` header). Use `statesOnly: ['default']` for badges/skeletons/static
   components; set `width` if the component is inline/auto-width and would otherwise collapse.
2. `pnpm run test:visual:update` → new baseline PNGs appear under
   `tests/visual/components.visual.spec.ts-snapshots/components/{id}/`.
3. Eyeball `/test/gallery` (`pnpm nx serve web`), get UX sign-off, commit baselines + fixture together.
