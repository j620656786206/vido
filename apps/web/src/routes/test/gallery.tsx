// Implements: <route-only>
/**
 * Test Route: Component Gallery for Playwright visual baselines (Story 19-4 + 19-4b Task 0).
 *
 * Path: /test/gallery — DEV / test only (blocked in production builds, mirroring
 * `apps/web/src/routes/test/manual-search.tsx`).
 *
 * Renders every in-scope `apps/web/src/components/` component (see
 * `-gallery.fixtures.tsx`) inside a stable `<section data-gallery-id data-pen-node>`
 * wrapper with up to four `<div data-gallery-state="default|hover|focus|open">` blocks.
 * The Playwright `visual` project (`tests/visual/components.visual.spec.ts`) navigates
 * here and screenshots each state div → committed baselines under
 * `tests/visual/components.visual.spec.ts-snapshots/`.
 *
 * Each component is wrapped in a per-component ErrorBoundary so a broken fixture
 * renders a labelled placeholder (`data-gallery-error`) instead of crashing the page;
 * the visual spec skips snapshotting error placeholders.
 *
 * **19-4b Task 0 extensions** (Sally 2026-05-12 follow-ups):
 *   - **Fix A (focus :focus-visible):** each state div is preceded by a hidden
 *     `[data-gallery-sentinel="pre"]` focusable button. The spec focuses the sentinel
 *     then presses `Tab` so Chromium flags input modality as keyboard → `:focus-visible`
 *     rules paint on the subsequent in-state-div focus (programmatic `.focus()` did not).
 *   - **Fix B (router-dependent fixtures):** fixtures declaring `routePath` are wrapped
 *     in a nested memory `RouterProvider` whose history is pinned to that path. The
 *     fixture's `useRouterState()` resolves through the nearest provider → reports the
 *     stub path → router-state-driven UI (e.g. TabNavigation's active tab) paints.
 *   - **Fix C (interactive `open` state):** fixtures setting `openTrigger?: string` get
 *     an extra `<div data-gallery-state="open" data-gallery-open-trigger="…">` block;
 *     the spec clicks that selector before screenshotting (captures e.g. the open
 *     `SortDropdown 955EZ` panel of `library/SortSelector`).
 *
 * **19-4b Task 3 extensions** (Q-bucket / S-bucket infrastructure):
 *   - **`seedQueries`:** fixtures listing `{ queryKey, data }` pairs get their entries
 *     pre-loaded into the app-shell `queryClient` cache by `GalleryFixtureSeed` (below)
 *     BEFORE children mount, so child components calling `useQuery()` see the data on
 *     first render — no loading flash, no network attempt.
 *   - **`seedStore`:** fixtures providing a `() => void` lambda get it invoked
 *     synchronously (same `useState` init) so Zustand-backed components see seeded
 *     store state on first render. Currently no `components/` consumer reads a store
 *     directly (project-context.md Rule 5); the field stays for forward compatibility.
 *
 * **19-4b Task 4 extensions** (single-fixture-per-page isolation):
 *   - **`?fixture=<id>`:** renders only the matching fixture. The visual spec navigates
 *     here per fixture so `fixed inset-0` overlays (ui/Dialog, ui/SidePanel, 10 custom
 *     dialogs) can no longer intercept pointer events on OTHER fixtures globally. The
 *     spec failure that prompted this change: Radix `Dialog.Portal` overlay from
 *     `ui-dialog` (rendered with `open: true`) blocked the first fixture's hover.
 *   - **`?manifest=1`:** renders just the ID list (no components mounted). Visual spec
 *     hits this first to discover the worklist, then iterates `?fixture=<id>` per snapshot.
 *
 * @internal Test fixture route. The QueryClient and Router context come from the app
 * shell (`main.tsx` → `QueryClientProvider` → router), so no extra providers are needed
 * for non-routePath fixtures.
 */
import { Component, type ErrorInfo, type ReactNode, useMemo, useRef } from 'react';
import {
  createFileRoute,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
  RouterProvider,
} from '@tanstack/react-router';
import { useQueryClient } from '@tanstack/react-query';
import { GALLERY_FIXTURES, type GalleryFixture, type GalleryState } from './-gallery.fixtures';

type GallerySearchParams = {
  fixture?: string;
  manifest?: boolean;
};

// 19-4b Task 4: search params drive single-fixture-per-page isolation. Without
// this, fixtures rendering `fixed inset-0` overlays (ui/Dialog, ui/SidePanel, and
// 10 Task-2/3 custom dialogs) intercept pointer events globally and break every
// other fixture's hover/focus. The visual spec navigates to `?manifest=1` to
// discover ids, then `?fixture=<id>` per snapshot.
//
// bugfix-19-4b-1 (pre-existing failure fix, Epic 9c Retro AI-2): TanStack Router's
// default JSON-based search parser DOES parse numeric-looking strings — `?manifest=1`
// arrives as NUMBER 1, not STRING '1', so the strict `=== '1'` check introduced by
// 19-4b Task 6 CR L2 silently broke manifest mode (validateSearch returned undefined
// → router stripped the param → manifest never activated → `test:visual` timed out
// on `[data-testid="component-gallery-manifest"]`). Accept all three observable
// forms — string '1', number 1, boolean true — to be tolerant of TanStack Router
// parser variations across versions. CR L2's tightening to `=== '1'` only was an
// over-correction (the more permissive form is the correct contract for a boolean
// query-string flag).
export const Route = createFileRoute('/test/gallery')({
  component: ComponentGalleryPage,
  validateSearch: (search: Record<string, unknown>): GallerySearchParams => ({
    fixture: typeof search.fixture === 'string' ? search.fixture : undefined,
    manifest:
      search.manifest === '1' || search.manifest === 1 || search.manifest === true
        ? true
        : undefined,
  }),
});

// Mirror manual-search.tsx's environment guard, with a PROD short-circuit on top so a
// production build never enables the gallery — even when accessed via localhost (common
// for NAS deployments behind SSH tunnels / port-forwards). The hostname check below is
// the dev convenience the precedent route uses; it must not be reachable in a prod build.
const isTestEnvironment =
  !import.meta.env.PROD &&
  (import.meta.env.DEV ||
    import.meta.env.MODE === 'test' ||
    (typeof window !== 'undefined' && window.location.hostname === 'localhost'));

// Default state set when a fixture doesn't pin `statesOnly`. `open` is opt-in only —
// requires the fixture to also set `openTrigger` so the visual spec knows what to click.
const ALL_STATES: GalleryState[] = ['default', 'hover', 'focus'];

// 19-4b Task 0 Fix B: nested memory `RouterProvider` for fixtures whose components
// read router state. We register stub routes for every TabNavigation `matchPaths`
// entry (`/library`, `/downloads`, `/pending`, `/settings`) so `<Link>` resolution
// inside the wrapped component stays happy regardless of which path the fixture pins.
const STUB_TAB_PATHS = ['/library', '/downloads', '/pending', '/settings'] as const;
export type StubRoutePath = (typeof STUB_TAB_PATHS)[number];

/**
 * Wraps `children` inside a nested TanStack Router `RouterProvider` whose memory
 * history is pinned to `pathname`. `useRouterState()` inside `children` resolves
 * via the nearest provider → reports our stub path. Used by Fix B for the
 * `shell/TabNavigation` fixture and any future router-state-dependent fixture.
 *
 * `children` is captured into the `useMemo` closure at first render. Each fixture
 * mounts once for the visual snapshot and does not update its rendered props, so
 * a stale closure is benign here — re-creating the router on prop change would
 * thrash history subscriptions for no observable benefit. The exhaustive-deps
 * suppression below is deliberate.
 */
function StubbedRouter({ pathname, children }: { pathname: StubRoutePath; children: ReactNode }) {
  const router = useMemo(
    () => {
      const rootRoute = createRootRoute({ component: () => <Outlet /> });
      const stubChildren = STUB_TAB_PATHS.map((p) =>
        createRoute({
          getParentRoute: () => rootRoute,
          path: p,
          component: () => <>{children}</>,
        })
      );
      rootRoute.addChildren(stubChildren);
      return createRouter({
        routeTree: rootRoute,
        history: createMemoryHistory({ initialEntries: [pathname] }),
      });
    },
    // Intentional: `children` is captured at first render (see fn docstring).
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [pathname]
  );

  // The stub router's typed route tree is intentionally narrower than the main
  // app's `routeTree.gen.ts` — TS complains about `Router<…>` generic-parameter
  // mismatch; runtime is correct (RouterProvider provides its own router context,
  // and useRouterState inside the wrapped fixture reads from the nearest one).
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return <RouterProvider router={router as any} />;
}

/**
 * 19-4b Task 3 (+ Task 6 CR fix): Pre-seeds `queryClient` cache (and optionally a
 * Zustand store) for Q-bucket / S-bucket fixtures before their children mount.
 * The seeding fires synchronously during the wrapper's first render — inside the
 * render body but guarded by `useRef` so it runs exactly once per fixture mount.
 * Children mount AFTER this render returns, so `useQuery()` calls inside them see
 * the seeded data on their first read (no `isLoading` flash, no network attempt).
 *
 * Why a `useRef` guard and not `useState`-initializer-as-side-effect: under React 18
 * StrictMode (active in dev), `useState` initializers are double-invoked, which
 * would double-fire `seedStore` (caller-provided lambda — not guaranteed idempotent).
 * `useRef` is the canonical "exactly once per mount" instance flag. `setQueryData`
 * remains idempotent for the same key+data; the guard primarily protects `seedStore`.
 *
 * Idempotent across re-renders (the ref flag flips after first render); "last fixture
 * wins" across siblings sharing a queryKey (documented `seedStore` semantics).
 *
 * The wrapped `queryClient` is the app shell's instance (`main.tsx` →
 * `QueryClientProvider`) — `useQueryClient()` reads from the nearest provider in
 * context, and the gallery route is mounted under that provider.
 */
function GalleryFixtureSeed({ fx, children }: { fx: GalleryFixture; children: ReactNode }) {
  const queryClient = useQueryClient();
  // React 19's `react-hooks/refs` rule allows the `ref.current == null` lazy-init
  // pattern specifically — it's the canonical way to run a side effect "exactly
  // once per mount" before children commit. The side effects below are the seed
  // payload; the `null` sentinel flips to `true` so subsequent renders are no-ops.
  const seededRef = useRef<true | null>(null);
  if (seededRef.current === null) {
    seededRef.current = true;
    if (fx.seedQueries) {
      for (const { queryKey, data } of fx.seedQueries) {
        queryClient.setQueryData(queryKey, data);
      }
    }
    if (fx.seedStore) {
      fx.seedStore();
    }
  }
  return <>{children}</>;
}

class FixtureErrorBoundary extends Component<
  { id: string; children: ReactNode },
  { error: Error | null }
> {
  state: { error: Error | null } = { error: null };

  static getDerivedStateFromError(error: Error) {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    // Surfaced in the gallery card; also log so a `nx serve web` console shows which fixture broke.
    console.error(`[gallery] fixture "${this.props.id}" threw:`, error, info.componentStack);
  }

  render() {
    if (this.state.error) {
      return (
        <div
          data-gallery-error="true"
          className="rounded border border-[var(--error)] bg-red-950/40 px-3 py-2 text-xs text-[var(--error)]"
        >
          ⚠ fixture error: {this.state.error.message}
        </div>
      );
    }
    return this.props.children;
  }
}

function ComponentGalleryPage() {
  const { fixture, manifest } = Route.useSearch();

  if (!isTestEnvironment) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-center">
          <h1 className="mb-4 text-2xl font-bold text-white">Access Denied</h1>
          <p className="text-[var(--text-secondary)]">
            This page is only available in test environments.
          </p>
        </div>
      </div>
    );
  }

  // 19-4b Task 4: manifest mode emits only the ID list (no components mounted).
  // The visual spec hits this first to discover the fixture worklist, then
  // navigates per fixture to `?fixture=<id>` — fixtures never co-render, so
  // `fixed inset-0` overlays can no longer intercept pointer events globally.
  if (manifest) {
    return (
      <div className="p-8" data-testid="component-gallery-manifest">
        <h1 className="mb-2 text-2xl font-bold text-white">Component Gallery Manifest</h1>
        <p className="mb-4 text-sm text-[var(--text-secondary)]">
          {GALLERY_FIXTURES.length} fixtures. Visual spec consumes this list, then visits{' '}
          <code>/test/gallery?fixture=&lt;id&gt;</code> per fixture.
        </p>
        <ul>
          {GALLERY_FIXTURES.map((fx) => (
            <li key={fx.id} data-gallery-id={fx.id} className="font-mono text-xs">
              {fx.id}
            </li>
          ))}
        </ul>
      </div>
    );
  }

  // 19-4b Task 4: when `?fixture=<id>` is set, render only that fixture (isolated
  // per-page snapshot). When unset, render the full gallery for dev browsing —
  // overlay collisions are expected in browse mode but irrelevant to the spec.
  const fixturesToRender = fixture
    ? GALLERY_FIXTURES.filter((fx) => fx.id === fixture)
    : GALLERY_FIXTURES;

  return (
    <div className="p-8" data-testid="component-gallery-page">
      <div className="mx-auto max-w-7xl">
        <h1 className="mb-2 text-2xl font-bold text-white">
          Component Visual Gallery (story 19-4 / 19-4b)
        </h1>
        <p className="mb-8 text-sm text-[var(--text-secondary)]">
          Each section below is screenshotted per state by{' '}
          <code>tests/visual/components.visual.spec.ts</code>. <code>data-pen-node</code> links it
          to its <code>ux-design.pen</code> node (or <code>screen-section</code> /{' '}
          <code>utility</code>). See <code>_bmad-output/audit/visual-baseline-19-4.md</code>.
          {fixture && (
            <>
              {' '}
              · <strong>Single-fixture mode:</strong> <code>{fixture}</code>
            </>
          )}
        </p>

        <div className="space-y-12">
          {fixturesToRender.map((fx) => {
            // 19-4b Task 0 Fix C: `open` state is opt-in. Drop it if a fixture
            // forgot to set `openTrigger` (the spec would otherwise click an
            // undefined selector). Warn in dev so developers notice the mismatch
            // — 19-4b Task 6 CR fix: silent drop violated `statesOnly` contract.
            const requestedStates = fx.statesOnly ?? ALL_STATES;
            if (import.meta.env.DEV && requestedStates.includes('open') && !fx.openTrigger) {
              console.warn(
                `[gallery] fixture "${fx.id}" requested 'open' state in statesOnly ` +
                  `but no openTrigger selector — open state will be dropped. ` +
                  `Either remove 'open' from statesOnly or add an openTrigger.`
              );
            }
            const states = requestedStates.filter((s) => s !== 'open' || !!fx.openTrigger);
            const Comp = fx.component;
            const props = fx.props ?? {};
            return (
              <section
                key={fx.id}
                data-gallery-id={fx.id}
                data-pen-node={fx.penNode}
                className="border-t border-[var(--border-subtle)]/40 pt-4"
              >
                <h2 className="mb-3 text-sm font-medium text-[var(--text-muted)]">
                  {fx.label} <span className="opacity-60">· {fx.penNode}</span>
                </h2>
                {/* 19-4b Task 3: seed queryClient cache + Zustand store for Q/S-bucket
                    fixtures BEFORE the state divs render. No-op when neither
                    seedQueries nor seedStore is set. */}
                <GalleryFixtureSeed fx={fx}>
                  <div className="flex flex-wrap gap-8">
                    {states.map((state) => {
                      const renderedFixture = (
                        <FixtureErrorBoundary id={`${fx.id}:${state}`}>
                          <Comp {...props} />
                        </FixtureErrorBoundary>
                      );
                      // 19-4b Task 0 Fix B: if the fixture declares `routePath`,
                      // wrap the render in a nested memory `RouterProvider` so
                      // `useRouterState()` inside the component reports that path.
                      const innerContent = fx.routePath ? (
                        <StubbedRouter pathname={fx.routePath}>{renderedFixture}</StubbedRouter>
                      ) : (
                        renderedFixture
                      );
                      return (
                        <div key={state} className="space-y-1">
                          <div className="text-[10px] uppercase tracking-wider text-[var(--text-muted)]/70">
                            {state}
                          </div>
                          {/* 19-4b Task 0 Fix A — sentinel before each state div.
                            The visual spec focuses this then presses Tab to
                            enter the state div via keyboard, so Chromium flags
                            input modality as keyboard and `:focus-visible`
                            paints on the subsequent in-state-div focus. */}
                          <button
                            type="button"
                            data-gallery-sentinel="pre"
                            aria-hidden="true"
                            tabIndex={0}
                            className="sr-only"
                          />
                          <div
                            data-gallery-state={state}
                            data-gallery-open-trigger={
                              state === 'open' ? fx.openTrigger : undefined
                            }
                            className="inline-block"
                            style={fx.width ? { width: fx.width } : undefined}
                          >
                            {innerContent}
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </GalleryFixtureSeed>
              </section>
            );
          })}
        </div>
      </div>
    </div>
  );
}

export default ComponentGalleryPage;
