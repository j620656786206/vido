/**
 * Playwright clock-mock helper — story 19-9 AC #4 ([@contract-v1]).
 *
 * Pins the in-page wall clock to a fixed ISO timestamp so visual baselines of
 * components that read `Date.now()` / `new Date()` (ambient time) render
 * deterministically. Without this, `library-recently-added`'s `isWithin7Days`
 * predicate (or any other moving-window component) silently stales the moment
 * the real clock crosses a state boundary — the class of bug Rule 23 prevents.
 *
 * Two-call pattern (CR /code-review 2026-05-28 M3 fix): Playwright's
 * `page.clock.install({ time, shouldAdvance: false })` (GA in Playwright ≥1.45;
 * this repo pins 1.58.0 in `package.json`) can be called only ONCE per `Page`
 * — subsequent calls error with "clock is already installed". When `withFixedClock`
 * is invoked for a SECOND clock-mocked fixture in the same test run (the visual
 * spec iterates `library-recently-added/recent` THEN `/stale` in one loop), we
 * MUST re-pin via `page.clock.setFixedTime(time)` — install-once, set-as-many-
 * times-as-needed is the Playwright-canonical pattern. We track per-`Page`
 * install state via a module-level `WeakSet<Page>` so the same Page instance
 * across loop iterations dispatches to `setFixedTime`, and a fresh Page
 * (different test, different worker) starts clean with `install`.
 *
 * After install the in-page `Date.now()`, `new Date()`, and `Date.parse(undefined)`
 * all observe the pinned value; `shouldAdvance: false` freezes time so the clock
 * does NOT progress during the screenshot.
 *
 * MUST be called BEFORE `page.goto(...)` for the FIRST fixture — `page.clock`
 * installs init scripts that need to run before any time-dependent JS on the
 * target page. For subsequent fixtures `setFixedTime` updates the already-frozen
 * clock and takes effect on the next navigation. The visual spec
 * (`tests/visual/components.visual.spec.ts`) calls this in the per-fixture loop
 * when the fixture row declares a `clockTime` field.
 *
 * Pair with the Rule 23 component-side marker (one of):
 *   - `// Clock-mocked: gallery fixture {id} uses page.clock.setFixedTime`
 *     (canonical — the marker phrase intentionally references `setFixedTime`
 *     because that's the steady-state mechanism for re-pinning across fixtures;
 *     `install` is the one-time setup the helper does on first call.)
 *   - `// Clock-injected: component accepts \`clock\` prop`
 *   - `// Time-bomb-exempt: <rationale>`
 *
 * @see project-context.md Rule 23
 * @see _bmad-output/audit/time-bomb-fixtures-2026-05.md
 */
import type { Page } from '@playwright/test';

// Per-Page install state. `WeakSet` lets the GC reap Page references when tests
// finish — no leak across worker reuse. Module-scope is safe because Playwright
// workers run in separate Node processes; each worker has its own copy.
const installedPages = new WeakSet<Page>();

/**
 * Install a frozen clock for the page at the given ISO timestamp.
 *
 * Idempotent across multiple calls on the same `Page`: first call installs the
 * clock, subsequent calls re-pin the fixed time via `setFixedTime`. Safe to
 * invoke in a per-fixture loop where some fixtures declare `clockTime` and
 * others don't — the helper is only called when needed, and a different `Page`
 * instance (fresh test) starts a new install cycle automatically.
 *
 * @param page         Playwright `Page` instance.
 * @param isoTimestamp ISO 8601 timestamp (e.g. `'2026-05-15T00:00:00Z'`). The
 *                     in-page `Date.now()` will return this value across every
 *                     call until the page navigates away or the test ends.
 *
 * @example
 *   // In a Playwright spec, before goto:
 *   await withFixedClock(page, '2026-05-15T00:00:00Z');
 *   await page.goto('/test/gallery?fixture=library-recently-added/recent');
 *   // ... `Date.now()` inside the page now returns 2026-05-15T00:00:00Z
 *
 *   // Same test, second fixture — `setFixedTime` re-pins (no re-install error):
 *   await withFixedClock(page, '2026-05-30T00:00:00Z');
 *   await page.goto('/test/gallery?fixture=library-recently-added/stale');
 */
export async function withFixedClock(page: Page, isoTimestamp: string): Promise<void> {
  const time = new Date(isoTimestamp);
  if (installedPages.has(page)) {
    await page.clock.setFixedTime(time);
  } else {
    await page.clock.install({ time });
    installedPages.add(page);
  }
}
