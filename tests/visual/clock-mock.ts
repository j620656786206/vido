/**
 * Playwright clock-mock helper — story 19-9 AC #4 ([@contract-v1]).
 *
 * Pins the in-page wall clock to a fixed ISO timestamp so visual baselines of
 * components that read `Date.now()` / `new Date()` (ambient time) render
 * deterministically. Without this, `library-recently-added`'s `isWithin7Days`
 * predicate (or any other moving-window component) silently stales the moment
 * the real clock crosses a state boundary — the class of bug Rule 23 prevents.
 *
 * Wraps Playwright's GA `page.clock.install({ time, shouldAdvance: false })`
 * API (Playwright ≥1.45 — this repo pins 1.57.0 in `package.json`, so the GA
 * path is taken with no fallback). After install the in-page `Date.now()`,
 * `new Date()`, and `Date.parse(undefined)` all observe the pinned value;
 * `shouldAdvance: false` freezes time so the clock does NOT progress during
 * the screenshot.
 *
 * MUST be called BEFORE `page.goto(...)` for the fixture page — `page.clock`
 * installs init scripts that need to run before any time-dependent JS on the
 * target page. The visual spec (`tests/visual/components.visual.spec.ts`)
 * calls this in the per-fixture loop when the fixture row declares a
 * `clockTime` field.
 *
 * Pair with the Rule 23 component-side marker (one of):
 *   - `// Clock-mocked: gallery fixture {id} uses page.clock.setFixedTime`
 *   - `// Clock-injected: component accepts \`clock\` prop`
 *   - `// Time-bomb-exempt: <rationale>`
 *
 * @see project-context.md Rule 23
 * @see _bmad-output/audit/time-bomb-fixtures-2026-05.md
 */
import type { Page } from '@playwright/test';

/**
 * Install a frozen clock for the page at the given ISO timestamp.
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
 */
export async function withFixedClock(page: Page, isoTimestamp: string): Promise<void> {
  await page.clock.install({ time: new Date(isoTimestamp) });
}
