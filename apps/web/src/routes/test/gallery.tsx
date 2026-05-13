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
 * @internal Test fixture route. The QueryClient and Router context come from the app
 * shell (`main.tsx` → `QueryClientProvider` → router), so no extra providers are needed
 * for non-routePath fixtures.
 */
import { Component, type ErrorInfo, type ReactNode, useMemo } from 'react';
import {
  createFileRoute,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
  RouterProvider,
} from '@tanstack/react-router';
import { GALLERY_FIXTURES, type GalleryState } from './-gallery.fixtures';

export const Route = createFileRoute('/test/gallery')({
  component: ComponentGalleryPage,
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

  return (
    <div className="p-8" data-testid="component-gallery-page">
      <div className="mx-auto max-w-7xl">
        <h1 className="mb-2 text-2xl font-bold text-white">
          Component Visual Gallery (story 19-4)
        </h1>
        <p className="mb-8 text-sm text-[var(--text-secondary)]">
          Each section below is screenshotted per state by{' '}
          <code>tests/visual/components.visual.spec.ts</code>. <code>data-pen-node</code> links it
          to its <code>ux-design.pen</code> node (or <code>screen-section</code> /{' '}
          <code>utility</code>). See <code>_bmad-output/audit/visual-baseline-19-4.md</code>.
        </p>

        <div className="space-y-12">
          {GALLERY_FIXTURES.map((fx) => {
            // 19-4b Task 0 Fix C: `open` state is opt-in. Silently drop it if a
            // fixture forgot to set `openTrigger` (the spec would otherwise click
            // an undefined selector).
            const requestedStates = fx.statesOnly ?? ALL_STATES;
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
                          data-gallery-open-trigger={state === 'open' ? fx.openTrigger : undefined}
                          className="inline-block"
                          style={fx.width ? { width: fx.width } : undefined}
                        >
                          {innerContent}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </section>
            );
          })}
        </div>
      </div>
    </div>
  );
}

export default ComponentGalleryPage;
