// Implements: <route-only>
/**
 * Test Route: Component Gallery for Playwright visual baselines (Story 19-4).
 *
 * Path: /test/gallery — DEV / test only (blocked in production builds, mirroring
 * `apps/web/src/routes/test/manual-search.tsx`).
 *
 * Renders every in-scope `apps/web/src/components/` component (see
 * `gallery.fixtures.tsx`) inside a stable `<section data-gallery-id data-pen-node>`
 * wrapper with up to three `<div data-gallery-state="default|hover|focus">` blocks.
 * The Playwright `visual` project (`tests/visual/components.visual.spec.ts`) navigates
 * here and screenshots each state div → committed baselines under
 * `tests/visual/components.visual.spec.ts-snapshots/`.
 *
 * Each component is wrapped in a per-component ErrorBoundary so a broken fixture
 * renders a labelled placeholder (`data-gallery-error`) instead of crashing the page;
 * the visual spec skips snapshotting error placeholders.
 *
 * @internal Test fixture route. The QueryClient and Router context come from the app
 * shell (`main.tsx` → `QueryClientProvider` → router), so no extra providers are needed.
 */
import { Component, type ErrorInfo, type ReactNode } from 'react';
import { createFileRoute } from '@tanstack/react-router';
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

const ALL_STATES: GalleryState[] = ['default', 'hover', 'focus'];

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
            const states = fx.statesOnly ?? ALL_STATES;
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
                  {states.map((state) => (
                    <div key={state} className="space-y-1">
                      <div className="text-[10px] uppercase tracking-wider text-[var(--text-muted)]/70">
                        {state}
                      </div>
                      <div
                        data-gallery-state={state}
                        className="inline-block"
                        style={fx.width ? { width: fx.width } : undefined}
                      >
                        <FixtureErrorBoundary id={`${fx.id}:${state}`}>
                          <Comp {...props} />
                        </FixtureErrorBoundary>
                      </div>
                    </div>
                  ))}
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
