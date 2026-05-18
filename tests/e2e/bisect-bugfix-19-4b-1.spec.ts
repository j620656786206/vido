/**
 * BISECT probe for bugfix-19-4b-1 — "Maximum update depth exceeded" warnings in
 * `/test/gallery` multi-fixture browse. Two-phase probe in a single test:
 *
 *   Phase A (multi-fixture browse mode): loads `/test/gallery` with NO `?fixture=`
 *     filter so all 123 fixtures co-render. Reproduces Sally's original observation
 *     (5-7 warnings in `nx serve web`). Also doubles as the fixture-id discovery
 *     step (scraping `section[data-gallery-id]` from the rendered DOM) — the
 *     `?manifest=1` endpoint is unusable as of the 19-4b Task 6 CR L2 fix (it
 *     tightened `validateSearch` to `=== '1'` but TanStack Router auto-parses
 *     `?manifest=1` to NUMBER 1, so manifest mode never activates — separate bug,
 *     out of scope here, see Completion Notes).
 *
 *   Phase B (per-fixture isolation): walks every scraped id, navigates per id to
 *     `/test/gallery?fixture=<id>`, counts per-id warnings + extracts component-
 *     stack frames under `apps/web/src/components/`.
 *
 * Writes structured JSON to
 * `_bmad-output/implementation-artifacts/bisect-bugfix-19-4b-1-${MODE}.json`
 * (default MODE=dev).
 *
 * Run against `nx serve web` (port 4200, React 18 StrictMode active):
 *
 *   BISECT_MODE=dev BASE_URL=http://localhost:4200 \
 *     pnpm exec playwright test tests/e2e/bisect-bugfix-19-4b-1.spec.ts --project=chromium
 *
 * NOTE: a preview-mode (port 4201, prod build) probe is NOT supported because
 * `apps/web/src/routes/test/gallery.tsx:90-97` gates the route on
 * `!import.meta.env.PROD` — the gallery returns "Access Denied" in any prod
 * build, so preview cannot render fixtures. Bucket D was therefore ruled out
 * STRUCTURALLY (the loop is callback-prop-identity drift, StrictMode-
 * independent), not empirically. See bisect-bugfix-19-4b-1.md § "Bucket D
 * ruled out (preview probe limitation)".
 *
 * Bucket verdict matrix (dev-only probe):
 *   - Phase A dev > 0  AND Phase B has offender(s) → A (single) or B (cluster)
 *   - Phase A dev > 0  AND Phase B per-id sum ≈ 0 → C (harness-only)
 *   - Phase A dev == 0                            → resolved, no offender
 *
 * Skipped in non-chromium projects so a `pnpm test:e2e` run picks the spec up exactly
 * once (the per-fixture walk takes ~5 min; running it × 5 browsers would be wasteful
 * and the warning is a React internal — browser-agnostic). Bugfix-10-3 spike precedent:
 * `_bmad-output/implementation-artifacts/spike-bugfix-10-3-findings.md` § "Methodology".
 *
 * @tags @bisect @story-19-4b-1
 */
import { test, expect, type ConsoleMessage, type Page } from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';

const MODE = process.env.BISECT_MODE ?? 'dev';
const OUT_DIR = path.resolve(__dirname, '../../_bmad-output/implementation-artifacts');
const OUT_PATH = path.resolve(OUT_DIR, `bisect-bugfix-19-4b-1-${MODE}.json`);

// In Vite dev, sources are served at `/src/components/...` (no `apps/web/` prefix).
// In preview (prod build), sources are bundled — stack frames point to `assets/index-*.js`
// which is opaque. Match the dev form first; the preview probe is mostly for Phase A
// dev-vs-prod presence comparison anyway.
const COMPONENT_FRAME_REGEX = /\/src\/components\/[^\s)'"]+(?::\d+:\d+)?/g;

type IdResult = {
  id: string;
  warnCount: number;
  componentFrames: string[];
  navigationError?: string;
};

type MultiPhaseResult = {
  warnCount: number;
  componentFrames: string[];
  navigationError?: string;
};

function attachWarningCollector(page: Page) {
  // React emits "Maximum update depth exceeded" as a console.error with the message
  // string only — no React component stack is appended (vs. other React warnings).
  // To capture origin frames we rely on the `__warnStacks` window array seeded by
  // the test's `addInitScript` (see test body): the wrapper records `new Error().stack`
  // at every console.error matching the pattern. The collector reads that array at
  // detach time and extracts frames matching COMPONENT_FRAME_REGEX.
  const state = { warnCount: 0, frames: new Set<string>() };
  const onConsole = (msg: ConsoleMessage) => {
    const text = msg.text();
    if (!/Maximum update depth exceeded/.test(text)) return;
    state.warnCount++;
  };
  const onPageError = (err: Error) => {
    const text = `${err.message}\n${err.stack ?? ''}`;
    if (!/Maximum update depth exceeded/.test(text)) return;
    state.warnCount++;
  };
  page.on('console', onConsole);
  page.on('pageerror', onPageError);
  return {
    state,
    detach: async () => {
      page.off('console', onConsole);
      page.off('pageerror', onPageError);
      // Pull origin frames from the in-page wrapper.
      try {
        const stacks: string[] = await page.evaluate(() => {
          // @ts-expect-error window.__warnStacks is seeded by addInitScript.
          const arr = (window.__warnStacks as string[] | undefined) ?? [];
          // @ts-expect-error reset for the next phase.
          window.__warnStacks = [];
          return arr;
        });
        for (const stack of stacks) {
          const matches = stack.match(COMPONENT_FRAME_REGEX);
          if (matches) for (const m of matches) state.frames.add(m);
        }
      } catch {
        // Page may already be closed if test was interrupted; tolerate it.
      }
    },
  };
}

test.describe('@bisect-19-4b-1 max-update-depth probe', () => {
  test.beforeEach(async ({ page, context, browserName }) => {
    test.skip(
      browserName !== 'chromium',
      'bisect probe is browser-agnostic (React internal warning) — run once in chromium only'
    );
    // /test/gallery is gated behind `!import.meta.env.PROD` (gallery.tsx:105) so a
    // production-build CI run returns "Access Denied". This bisect probe is for
    // local development reproduction only — it relies on Vite dev mode where
    // `import.meta.env.PROD === false`. CI E2E shard runs `nx build --configuration=production`
    // + `npx serve dist`, which permanently triggers the prod gate and makes Phase A's
    // `ids.length > 0` assertion impossible. Skip in CI (no value lost — the actual
    // regression gate is the `useParseProgress.spec.ts` unit test added by 19-4b-1 CR M1).
    test.skip(
      process.env.CI === 'true',
      'bisect probe is local-only — prod-build CI cannot reach /test/gallery (Access Denied gate)'
    );
    // Seed an in-page wrapper around console.error that captures `new Error().stack`
    // every time a Maximum-update-depth warning fires. React 18 emits the message
    // string only (no component stack arg), so we synthesise a JS stack on the call
    // site — Vite dev's module URLs (`/src/components/...`) survive into the stack
    // and identify the offending hook/component. Drained per phase by the collector.
    await context.addInitScript(() => {
      const orig = console.error.bind(console);
      // @ts-expect-error window-scoped buffer is read by the test's collector.
      window.__warnStacks = [];
      console.error = function (...args: unknown[]) {
        const msg = args[0];
        if (typeof msg === 'string' && msg.includes('Maximum update depth exceeded')) {
          const stack = new Error().stack ?? '';
          // @ts-expect-error see above
          if (window.__warnStacks.length < 50) window.__warnStacks.push(stack);
        }
        return orig(...args);
      };
    });
    // Playwright route handlers run in REVERSE order of registration (LIFO):
    // the most-recently added matching handler wins. So the catch-all MUST be
    // registered FIRST, and specific stubs registered AFTER it, for the
    // specific stubs to win their patterns.
    //
    // Catch-all for any /api/v1/* — return empty 200 so dev (with vite proxy
    // forwarding to API:8080) and preview (no proxy) see equivalent network
    // behavior even when no specific stub matches.
    await page.route('**/api/v1/**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
    );
    // Setup-status stub so `__root.tsx` doesn't redirect to /setup before reaching the gallery.
    await page.route('**/api/v1/setup/status', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ needsSetup: false }),
      })
    );
    // Reproduce Sally's observed conditions: the 3 pre-existing 500s on
    // library/genres + explore-blocks/blk-1/content. The seedQueries Q-bucket
    // infrastructure pre-loads most fixture data, so most fixtures never hit the
    // network — these 3 endpoints are the ones whose consumers (ExploreBlock for
    // content, library filter UI for genres) read keys that fixtures didn't seed.
    // These MUST be registered AFTER the catch-all above so LIFO matching picks
    // them first for their patterns.
    await page.route('**/api/v1/library/genres', (route) =>
      route.fulfill({ status: 500, contentType: 'application/json', body: '{"error":"stub"}' })
    );
    await page.route('**/api/v1/explore-blocks/*/content', (route) =>
      route.fulfill({ status: 500, contentType: 'application/json', body: '{"error":"stub"}' })
    );
  });

  test('Phase A + B — multi-fixture browse warnings + per-fixture isolation walk', async ({
    page,
  }) => {
    // Multi-fixture render (~3 s settle for all 123) + 123 per-fixture nav × ~1.5 s
    // settle each ⇒ ~3-5 min wall clock. 10 min budget covers worst case (slow Vite
    // recompiles on dev mode after many sequential navigations) without parking CI
    // for a runaway 20 minutes — escalate manually if the budget is hit, since a
    // hang likely indicates a probe regression worth surfacing fast.
    test.setTimeout(10 * 60 * 1000);

    // ---------- Phase A: multi-fixture browse mode (also discovers fixture ids) ----------
    const phaseA = attachWarningCollector(page);
    let phaseANavError: string | undefined;
    let ids: string[] = [];
    try {
      await page.goto('/test/gallery', { timeout: 60_000, waitUntil: 'domcontentloaded' });
      await page.waitForSelector('[data-testid="component-gallery-page"]', {
        state: 'visible',
        timeout: 30_000,
      });
      // Multi-fixture mode needs a longer settle — all 123 fixtures must complete
      // their initial render + StrictMode double-mount cycle. Sally observed
      // 5-7 warnings during this window; budget for all to fire.
      await page.waitForTimeout(3000);

      // Scrape fixture ids from the rendered DOM (manifest mode unreachable — see
      // file-header note on the Task 6 CR L2 regression).
      ids = await page
        .locator('section[data-gallery-id]')
        .evaluateAll((els) =>
          els
            .map((el) => el.getAttribute('data-gallery-id'))
            .filter((id): id is string => typeof id === 'string' && id.length > 0)
        );
    } catch (err) {
      phaseANavError = err instanceof Error ? err.message : String(err);
    }
    await phaseA.detach();

    const multi: MultiPhaseResult = {
      warnCount: phaseA.state.warnCount,
      componentFrames: [...phaseA.state.frames].slice(0, 10),
      ...(phaseANavError ? { navigationError: phaseANavError } : {}),
    };

    // Persist Phase A immediately so a Phase B failure still leaves a useful artifact.
    fs.mkdirSync(OUT_DIR, { recursive: true });
    const partialPayload = {
      mode: MODE,
      baseURL: process.env.BASE_URL ?? 'http://localhost:4200',
      totalIds: ids.length,
      multi,
      // Phase B placeholder — overwritten on success below.
      perFixtureSum: null,
      offenderCount: null,
      results: [],
    };
    fs.writeFileSync(OUT_PATH, JSON.stringify(partialPayload, null, 2) + '\n');

    expect(ids.length, 'gallery rendered at least one fixture section').toBeGreaterThan(0);

    // ---------- Phase B: per-fixture walk ----------
    const results: IdResult[] = [];

    for (const id of ids) {
      const collector = attachWarningCollector(page);
      let navigationError: string | undefined;

      try {
        await page.goto(`/test/gallery?fixture=${encodeURIComponent(id)}`, {
          timeout: 60_000,
          waitUntil: 'domcontentloaded',
        });
        await page.waitForSelector('[data-testid="component-gallery-page"]', {
          state: 'visible',
          timeout: 30_000,
        });
        // Per-fixture settle — covers StrictMode double-mount + initial effect cycle.
        await page.waitForTimeout(800);
      } catch (err) {
        navigationError = err instanceof Error ? err.message : String(err);
      }

      await collector.detach();

      results.push({
        id,
        warnCount: collector.state.warnCount,
        componentFrames: [...collector.state.frames].slice(0, 5),
        ...(navigationError ? { navigationError } : {}),
      });
    }

    // ---------- Write final JSON ----------
    const perFixtureSum = results.reduce((acc, r) => acc + r.warnCount, 0);
    const offenderCount = results.filter((r) => r.warnCount > 0).length;
    fs.writeFileSync(
      OUT_PATH,
      JSON.stringify(
        {
          mode: MODE,
          baseURL: process.env.BASE_URL ?? 'http://localhost:4200',
          totalIds: results.length,
          multi,
          perFixtureSum,
          offenderCount,
          results,
        },
        null,
        2
      ) + '\n'
    );

    const summary = `[bisect-${MODE}] multi=${multi.warnCount} · per-fixture sum=${perFixtureSum} · offenders=${offenderCount}/${results.length} · ${OUT_PATH}`;
    test.info().annotations.push({ type: 'bisect-summary', description: summary });
    console.log(summary);

    // CI regression gate (AC #2): post-fix the loop must stay closed forever.
    // Spec doubles as the regression assertion — if a future change re-introduces
    // the callback-prop-identity churn, these expects fail loudly instead of
    // silently leaving a stale JSON on disk.
    expect(
      multi.warnCount,
      'Phase A multi-fixture browse must emit 0 "Maximum update depth exceeded" warnings'
    ).toBe(0);
    const offendersList = results
      .filter((r) => r.warnCount > 0)
      .map((r) => `${r.id}=${r.warnCount}`)
      .join(', ');
    expect(
      offenderCount,
      `Phase B per-fixture isolation must report 0 offenders; got: ${offendersList || 'none'}`
    ).toBe(0);
  });
});
