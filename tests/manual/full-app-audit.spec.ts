/**
 * Full App Audit — Comprehensive browser + API exploration test
 *
 * Target: http://192.168.50.52:8088 (Vido on NAS)
 *
 * This script navigates every page, clicks every interactive element,
 * and calls every API endpoint to produce a complete bug report.
 *
 * Run: npx playwright test tests/manual/full-app-audit.spec.ts --project=chromium --headed
 * Or headless: npx playwright test tests/manual/full-app-audit.spec.ts --project=chromium
 */

import { test, type Page } from '@playwright/test';
import fs from 'fs';
import path from 'path';

const BASE_URL = 'http://192.168.50.52:8088';
const API_URL = `${BASE_URL}/api/v1`;
const REPORT_DIR = path.resolve(__dirname, '../../test-results/app-audit');
const SCREENSHOTS_DIR = path.join(REPORT_DIR, 'screenshots');

interface BugReport {
  id: number;
  severity: 'critical' | 'major' | 'minor' | 'info';
  category: string;
  page: string;
  description: string;
  expected: string;
  actual: string;
  screenshot?: string;
}

const bugs: BugReport[] = [];
let bugId = 0;

function reportBug(
  severity: BugReport['severity'],
  category: string,
  page: string,
  description: string,
  expected: string,
  actual: string,
  screenshot?: string
) {
  bugs.push({
    id: ++bugId,
    severity,
    category,
    page,
    description,
    expected,
    actual,
    screenshot,
  });
}

async function screenshot(page: Page, name: string): Promise<string> {
  const filename = `${name.replace(/[^a-z0-9-]/gi, '_')}.png`;
  const filepath = path.join(SCREENSHOTS_DIR, filename);
  await page.screenshot({ path: filepath, fullPage: true });
  return filename;
}

// ============================================================================
// Setup
// ============================================================================
test.beforeAll(async () => {
  fs.mkdirSync(SCREENSHOTS_DIR, { recursive: true });
});

test.afterAll(async () => {
  // Generate markdown report
  const reportPath = path.join(REPORT_DIR, 'bug-report.md');
  let md = `# Vido App Audit Report\n\n`;
  md += `**Date:** ${new Date().toISOString()}\n`;
  md += `**Target:** ${BASE_URL}\n`;
  md += `**Total Issues Found:** ${bugs.length}\n\n`;

  const critical = bugs.filter((b) => b.severity === 'critical');
  const major = bugs.filter((b) => b.severity === 'major');
  const minor = bugs.filter((b) => b.severity === 'minor');
  const info = bugs.filter((b) => b.severity === 'info');

  md += `| Severity | Count |\n|---|---|\n`;
  md += `| Critical | ${critical.length} |\n`;
  md += `| Major | ${major.length} |\n`;
  md += `| Minor | ${minor.length} |\n`;
  md += `| Info | ${info.length} |\n\n`;

  for (const bug of bugs) {
    md += `## #${bug.id} [${bug.severity.toUpperCase()}] ${bug.description}\n\n`;
    md += `- **Category:** ${bug.category}\n`;
    md += `- **Page:** ${bug.page}\n`;
    md += `- **Expected:** ${bug.expected}\n`;
    md += `- **Actual:** ${bug.actual}\n`;
    if (bug.screenshot) {
      md += `- **Screenshot:** [${bug.screenshot}](screenshots/${bug.screenshot})\n`;
    }
    md += `\n---\n\n`;
  }

  fs.writeFileSync(reportPath, md, 'utf-8');
  console.log(`\n📋 Bug report saved to: ${reportPath}`);
  console.log(`📸 Screenshots saved to: ${SCREENSHOTS_DIR}`);
  console.log(
    `🐛 Total issues: ${bugs.length} (${critical.length} critical, ${major.length} major, ${minor.length} minor, ${info.length} info)\n`
  );
});

// ============================================================================
// PART 1: API Endpoint Audit
// ============================================================================
test.describe('API Endpoint Audit', () => {
  test('Health check', async ({ request }) => {
    const res = await request.get(`${BASE_URL}/health`);
    const data = await res.json();
    const _img = undefined;

    if (res.status() !== 200) {
      reportBug(
        'critical',
        'API',
        '/health',
        'Health endpoint not returning 200',
        '200 OK',
        `${res.status()}`
      );
    }
    if (!data.database) {
      reportBug(
        'major',
        'API',
        '/health',
        'Health endpoint missing database info',
        'database field present',
        'missing'
      );
    }
  });

  test('Library list', async ({ request }) => {
    const res = await request.get(
      `${API_URL}/library?page=1&page_size=5&sort_by=created_at&sort_order=desc&type=all`
    );
    const json = await res.json();

    if (!json.success) {
      reportBug(
        'critical',
        'API',
        '/library',
        'Library list API failed',
        'success: true',
        `success: ${json.success}`
      );
      return;
    }

    const data = json.data;

    // Check pagination fields (snake_case from API)
    if (data.total_items === undefined && data.totalItems === undefined) {
      reportBug(
        'major',
        'API',
        '/library',
        'Library response missing total_items',
        'total_items field',
        'missing'
      );
    }

    // Check items structure
    if (data.items?.length > 0) {
      const item = data.items[0];
      const movie = item.movie || item.series;

      if (!movie) {
        reportBug(
          'major',
          'API',
          '/library',
          'Library item missing movie/series object',
          'movie or series field',
          'missing'
        );
      } else {
        // Check critical fields
        if (!movie.id)
          reportBug('major', 'API', '/library', 'Movie missing id field', 'UUID id', 'missing');
        if (movie.poster_path === undefined && movie.posterPath === undefined) {
          // poster_path can be null, but should exist as a field
        }
        if (movie.parse_status === '' || movie.parse_status === null) {
          reportBug(
            'info',
            'Data',
            '/library',
            `Movie "${movie.title}" has empty parse_status`,
            'pending or success',
            `"${movie.parse_status}"`
          );
        }
        if (movie.tmdb_id === null && movie.parse_status !== 'failed') {
          reportBug(
            'info',
            'Data',
            '/library',
            `Movie "${movie.title}" has no TMDB metadata`,
            'tmdb_id populated',
            'null'
          );
        }
      }
    }
  });

  test('Library stats', async ({ request }) => {
    const res = await request.get(`${API_URL}/library/stats`);
    const json = await res.json();

    if (!json.success) {
      reportBug(
        'major',
        'API',
        '/library/stats',
        'Stats API failed',
        'success: true',
        `${json.success}`
      );
    }
  });

  test('Library genres', async ({ request }) => {
    const res = await request.get(`${API_URL}/library/genres`);
    const json = await res.json();

    if (!json.success) {
      reportBug(
        'major',
        'API',
        '/library/genres',
        'Genres API failed',
        'success: true',
        `${json.success}`
      );
    }
  });

  test('Library search', async ({ request }) => {
    const res = await request.get(`${API_URL}/library/search?q=test&page=1&page_size=5`);
    const json = await res.json();

    if (!json.success) {
      reportBug(
        'major',
        'API',
        '/library/search',
        'Search API failed',
        'success: true',
        `${json.success}`
      );
    }
  });

  test('Scanner status', async ({ request }) => {
    const res = await request.get(`${API_URL}/scanner/status`);
    const json = await res.json();

    if (!json.success) {
      reportBug(
        'major',
        'API',
        '/scanner/status',
        'Scanner status API failed',
        'success: true',
        `${json.success}`
      );
    }
  });

  test('Scanner schedule', async ({ request }) => {
    const res = await request.get(`${API_URL}/scanner/schedule`);
    const json = await res.json();

    if (!json.success) {
      reportBug(
        'major',
        'API',
        '/scanner/schedule',
        'Scanner schedule API failed',
        'success: true',
        `${json.success}`
      );
    }
  });

  test('Settings', async ({ request }) => {
    const res = await request.get(`${API_URL}/settings`);
    const json = await res.json();

    if (!json.success) {
      reportBug(
        'major',
        'API',
        '/settings',
        'Settings API failed',
        'success: true',
        `${json.success}`
      );
    }
  });

  test('Service health', async ({ request }) => {
    const res = await request.get(`${API_URL}/health/services`);
    const json = await res.json();

    if (!json.success) {
      reportBug(
        'major',
        'API',
        '/health/services',
        'Service health API failed',
        'success: true',
        `${json.success}`
      );
    } else {
      // Check for expected services
      const services = json.data?.services || json.data;
      if (Array.isArray(services)) {
        const serviceNames = services.map((s: any) => s.name || s.service);
        if (!serviceNames.some((n: string) => n?.toLowerCase().includes('tmdb'))) {
          reportBug(
            'minor',
            'API',
            '/health/services',
            'TMDB service not in health check',
            'tmdb service listed',
            'missing'
          );
        }
      }
    }
  });

  test('Enrichment status', async ({ request }) => {
    const res = await request.get(`${API_URL}/scanner/enrich/status`);
    if (res.status() === 404) {
      reportBug(
        'info',
        'API',
        '/scanner/enrich/status',
        'Enrichment endpoint not deployed yet (expected if NAS not rebuilt)',
        '200',
        '404 — needs Docker rebuild'
      );
    }
  });

  test('SSE events endpoint', async ({ request }) => {
    // Just check it's accessible (don't hold connection)
    const _res = await request.get(`${API_URL}/events`, { timeout: 3000 }).catch(() => null);
    // SSE may timeout which is fine — it's a streaming endpoint
  });

  test('TMDB search proxy', async ({ request }) => {
    const res = await request.get(`${API_URL}/tmdb/search/movie?query=Inception&language=zh-TW`);
    const json = await res.json();

    if (!json.success) {
      reportBug(
        'major',
        'API',
        '/tmdb/search',
        'TMDB search proxy failed',
        'success: true',
        `${json.success}, error: ${json.error?.message}`
      );
    } else if (!json.data?.results?.length && !json.data?.items?.length) {
      reportBug(
        'major',
        'API',
        '/tmdb/search',
        'TMDB search returned no results for "Inception"',
        'at least 1 result',
        '0 results'
      );
    }
  });

  test('Non-existent endpoint returns proper error', async ({ request }) => {
    const res = await request.get(`${API_URL}/nonexistent`);
    const _json = await res.json();

    if (res.status() !== 404) {
      reportBug(
        'minor',
        'API',
        '/nonexistent',
        'Non-existent endpoint should return 404',
        '404',
        `${res.status()}`
      );
    }
  });
});

// ============================================================================
// PART 2: UI Navigation & Interaction Audit
// ============================================================================
test.describe('UI Navigation Audit', () => {
  test('Homepage / Library page loads', async ({ page }) => {
    await page.goto(BASE_URL, { waitUntil: 'networkidle', timeout: 30000 });
    const img = await screenshot(page, '01-homepage');

    // Check page title or header
    await page.title();
    const headerText = await page
      .locator('text=vido')
      .first()
      .isVisible()
      .catch(() => false);

    if (!headerText) {
      reportBug(
        'major',
        'UI',
        '/',
        'App header/logo not visible on homepage',
        'vido logo visible',
        'not found',
        img
      );
    }

    // Check if navigation tabs exist
    const navLinks = page.locator('nav a, header a');
    const navCount = await navLinks.count();
    if (navCount < 2) {
      reportBug(
        'minor',
        'UI',
        '/',
        'Navigation has fewer than 2 links',
        '>=2 navigation items',
        `${navCount}`,
        img
      );
    }
  });

  test('Library page — media grid', async ({ page }) => {
    await page.goto(
      `${BASE_URL}/library?sortBy=created_at&sortOrder=desc&page=1&pageSize=20&type=all`,
      {
        waitUntil: 'networkidle',
        timeout: 30000,
      }
    );
    const img = await screenshot(page, '02-library-grid');

    // Check for media cards
    const cards = page.locator('[data-testid="poster-card"]');
    const cardCount = await cards.count();

    if (cardCount === 0) {
      reportBug(
        'major',
        'UI',
        '/library',
        'No media cards rendered in library grid',
        'media cards visible',
        'zero cards',
        img
      );
    }

    // Check if poster images loaded
    if (cardCount > 0) {
      const fallbacks = page.locator('[data-testid="poster-fallback"]');
      const fallbackCount = await fallbacks.count();

      if (fallbackCount === cardCount) {
        reportBug(
          'major',
          'UI',
          '/library',
          'All poster cards show fallback (no images)',
          'poster images loaded',
          `all ${cardCount} cards show fallback`,
          img
        );
      }

      // Check for skeleton state stuck
      const skeletons = page.locator('[data-testid="poster-skeleton"]');
      const skeletonCount = await skeletons.count();
      if (skeletonCount > 0) {
        reportBug(
          'minor',
          'UI',
          '/library',
          `${skeletonCount} poster cards stuck in loading skeleton`,
          'images loaded or fallback shown',
          `${skeletonCount} skeletons visible`,
          img
        );
      }
    }
  });

  test('Library page — sort and filter controls', async ({ page }) => {
    await page.goto(
      `${BASE_URL}/library?sortBy=created_at&sortOrder=desc&page=1&pageSize=20&type=all`,
      {
        waitUntil: 'networkidle',
        timeout: 30000,
      }
    );

    // Check sort controls exist
    const sortButton = page
      .locator('text=新增日期')
      .or(page.locator('text=排序'))
      .or(page.locator('[data-testid*="sort"]'));
    const sortVisible = await sortButton
      .first()
      .isVisible()
      .catch(() => false);
    const img = await screenshot(page, '03-library-controls');

    if (!sortVisible) {
      reportBug(
        'minor',
        'UI',
        '/library',
        'Sort controls not visible',
        'sort dropdown/button visible',
        'not found',
        img
      );
    }

    // Check filter controls
    const filterButton = page.locator('text=篩選').or(page.locator('[data-testid*="filter"]'));
    const filterVisible = await filterButton
      .first()
      .isVisible()
      .catch(() => false);

    if (!filterVisible) {
      reportBug(
        'minor',
        'UI',
        '/library',
        'Filter controls not visible',
        'filter button visible',
        'not found',
        img
      );
    }
  });

  test('Library page — click first media card', async ({ page }) => {
    await page.goto(
      `${BASE_URL}/library?sortBy=created_at&sortOrder=desc&page=1&pageSize=20&type=all`,
      {
        waitUntil: 'networkidle',
        timeout: 30000,
      }
    );

    const cards = page.locator('[data-testid="poster-card"]');
    const cardCount = await cards.count();

    if (cardCount === 0) {
      reportBug(
        'critical',
        'UI',
        '/library',
        'Cannot test card click — no cards rendered',
        'cards to click',
        'zero cards'
      );
      return;
    }

    // Click first card and check navigation
    const firstCard = cards.first();
    const href = await firstCard.getAttribute('href');
    await firstCard.click();
    await page.waitForTimeout(2000);

    const currentUrl = page.url();
    const img = await screenshot(page, '04-card-click-result');

    // Check for 404
    const has404 = await page
      .locator('text=404')
      .isVisible()
      .catch(() => false);
    const hasNotFound = await page
      .locator('text=找不到')
      .isVisible()
      .catch(() => false);

    if (has404 || hasNotFound) {
      reportBug(
        'critical',
        'UI',
        currentUrl,
        `Clicking media card leads to 404 (href: ${href})`,
        'media detail page',
        '404 page',
        img
      );
    }

    // Check if URL has /media/movie/0 or /media/tv/0
    if (currentUrl.includes('/0') || currentUrl.endsWith('/0')) {
      reportBug(
        'critical',
        'UI',
        currentUrl,
        'Media card links to ID=0 (tmdbId is null)',
        'valid media ID in URL',
        'ID is 0',
        img
      );
    }
  });

  test('Search page — navigation', async ({ page }) => {
    await page.goto(BASE_URL, { waitUntil: 'networkidle', timeout: 30000 });

    // Try to find and click search/搜尋 nav link
    const searchLink = page.locator('a[href*="search"]').or(page.locator('text=搜尋'));
    const searchVisible = await searchLink
      .first()
      .isVisible()
      .catch(() => false);

    if (!searchVisible) {
      // Try the global search bar
      const searchBar = page
        .locator('input[placeholder*="搜尋"]')
        .or(page.locator('input[type="search"]'));
      const barVisible = await searchBar
        .first()
        .isVisible()
        .catch(() => false);

      if (!barVisible) {
        const img = await screenshot(page, '05-search-not-found');
        reportBug(
          'major',
          'UI',
          '/',
          'No search functionality accessible from homepage',
          'search bar or link',
          'none found',
          img
        );
        return;
      }
    }

    const _img = await screenshot(page, '05-search-page');
  });

  test('Search — perform a search', async ({ page }) => {
    await page.goto(BASE_URL, { waitUntil: 'networkidle', timeout: 30000 });

    // Find search input
    const searchInput = page
      .locator('input[placeholder*="搜尋"]')
      .or(page.locator('input[placeholder*="search"]'))
      .or(page.locator('input[type="search"]'));
    const inputVisible = await searchInput
      .first()
      .isVisible()
      .catch(() => false);

    if (!inputVisible) {
      reportBug('major', 'UI', '/', 'Search input not found', 'search input visible', 'not found');
      return;
    }

    await searchInput.first().fill('Inception');
    await page.waitForTimeout(2000); // debounce
    const img = await screenshot(page, '06-search-results');

    // Check for results or no-results message
    const hasResults = await page.locator('[data-testid="poster-card"]').count();
    const hasNoResults = await page
      .locator('text=沒有找到')
      .or(page.locator('text=no results'))
      .first()
      .isVisible()
      .catch(() => false);

    if (hasResults === 0 && !hasNoResults) {
      reportBug(
        'minor',
        'UI',
        '/search',
        'Search for "Inception" shows neither results nor no-results message',
        'results or empty state',
        'unclear state',
        img
      );
    }
  });

  test('Settings page', async ({ page }) => {
    await page.goto(BASE_URL, { waitUntil: 'networkidle', timeout: 30000 });

    // Navigate to settings
    const settingsLink = page.locator('a[href*="setting"]').or(page.locator('text=設定'));
    const visible = await settingsLink
      .first()
      .isVisible()
      .catch(() => false);

    if (!visible) {
      // Try settings icon
      const settingsIcon = page
        .locator('[aria-label*="設定"]')
        .or(page.locator('a[href*="setting"]'));
      const iconVisible = await settingsIcon
        .first()
        .isVisible()
        .catch(() => false);
      if (iconVisible) {
        await settingsIcon.first().click();
      } else {
        await page.goto(`${BASE_URL}/settings`, { waitUntil: 'networkidle', timeout: 30000 });
      }
    } else {
      await settingsLink.first().click();
    }

    await page.waitForTimeout(1000);
    const img = await screenshot(page, '07-settings-page');
    page.url();

    // Check settings page loaded
    const has404 = await page
      .locator('text=404')
      .isVisible()
      .catch(() => false);
    if (has404) {
      reportBug('major', 'UI', '/settings', 'Settings page shows 404', 'settings form', '404', img);
    }
  });

  test('Downloads page', async ({ page }) => {
    await page
      .goto(`${BASE_URL}/downloads`, { waitUntil: 'networkidle', timeout: 30000 })
      .catch(async () => {
        // Try alternate URL
        await page.goto(BASE_URL, { waitUntil: 'networkidle', timeout: 30000 });
      });

    // Navigate to downloads
    const downloadsLink = page.locator('a[href*="download"]').or(page.locator('text=下載中'));
    const visible = await downloadsLink
      .first()
      .isVisible()
      .catch(() => false);

    if (visible) {
      await downloadsLink.first().click();
      await page.waitForTimeout(1000);
    }

    const img = await screenshot(page, '08-downloads-page');
    const has404 = await page
      .locator('text=404')
      .isVisible()
      .catch(() => false);

    if (has404) {
      reportBug(
        'major',
        'UI',
        '/downloads',
        'Downloads page shows 404',
        'downloads list',
        '404',
        img
      );
    }
  });

  test('Parse queue page', async ({ page }) => {
    await page.goto(BASE_URL, { waitUntil: 'networkidle', timeout: 30000 });

    const parseLink = page.locator('a[href*="parse"]').or(page.locator('text=待解析'));
    const visible = await parseLink
      .first()
      .isVisible()
      .catch(() => false);

    if (visible) {
      await parseLink.first().click();
      await page.waitForTimeout(1000);
      const img = await screenshot(page, '09-parse-queue');

      const has404 = await page
        .locator('text=404')
        .isVisible()
        .catch(() => false);
      if (has404) {
        reportBug(
          'major',
          'UI',
          '/parse-queue',
          'Parse queue page shows 404',
          'parse queue list',
          '404',
          img
        );
      }
    }
  });

  test('Responsive — mobile viewport', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 812 }); // iPhone X
    await page.goto(
      `${BASE_URL}/library?sortBy=created_at&sortOrder=desc&page=1&pageSize=20&type=all`,
      {
        waitUntil: 'networkidle',
        timeout: 30000,
      }
    );
    const img = await screenshot(page, '10-mobile-library');

    // Check cards render in mobile
    const cards = page.locator('[data-testid="poster-card"]');
    const cardCount = await cards.count();

    if (cardCount === 0) {
      reportBug(
        'major',
        'UI',
        '/library (mobile)',
        'No media cards on mobile viewport',
        'responsive grid',
        'empty',
        img
      );
    }

    // Check for horizontal overflow
    const bodyWidth = await page.evaluate(() => document.body.scrollWidth);
    const viewportWidth = 375;
    if (bodyWidth > viewportWidth + 10) {
      reportBug(
        'minor',
        'UI',
        '/library (mobile)',
        `Horizontal overflow detected (body: ${bodyWidth}px, viewport: ${viewportWidth}px)`,
        'no horizontal scroll',
        `${bodyWidth - viewportWidth}px overflow`,
        img
      );
    }
  });

  test('Console errors check', async ({ page }) => {
    const errors: string[] = [];
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });

    await page.goto(
      `${BASE_URL}/library?sortBy=created_at&sortOrder=desc&page=1&pageSize=20&type=all`,
      {
        waitUntil: 'networkidle',
        timeout: 30000,
      }
    );
    await page.waitForTimeout(3000);

    if (errors.length > 0) {
      const uniqueErrors = [...new Set(errors)];
      for (const err of uniqueErrors.slice(0, 10)) {
        reportBug(
          'minor',
          'Console',
          '/library',
          `Console error: ${err.substring(0, 200)}`,
          'no console errors',
          err.substring(0, 200)
        );
      }
    }
  });

  test('Library pagination', async ({ page }) => {
    await page.goto(
      `${BASE_URL}/library?sortBy=created_at&sortOrder=desc&page=1&pageSize=20&type=all`,
      {
        waitUntil: 'networkidle',
        timeout: 30000,
      }
    );

    // Scroll to bottom to check for pagination
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
    await page.waitForTimeout(1000);
    const img = await screenshot(page, '11-pagination');

    // Check for pagination controls or infinite scroll trigger
    const pagination = page
      .locator('[data-testid*="pagination"]')
      .or(page.locator('text=下一頁'))
      .or(page.locator('button:has-text(">")'));
    const hasPagination = await pagination
      .first()
      .isVisible()
      .catch(() => false);

    if (!hasPagination) {
      reportBug(
        'info',
        'UI',
        '/library',
        'No visible pagination controls (may use infinite scroll)',
        'pagination or load-more',
        'none visible',
        img
      );
    }
  });

  test('Library type filter — movies only', async ({ page }) => {
    await page.goto(
      `${BASE_URL}/library?sortBy=created_at&sortOrder=desc&page=1&pageSize=20&type=movie`,
      {
        waitUntil: 'networkidle',
        timeout: 30000,
      }
    );
    const _img = await screenshot(page, '12-library-movies-filter');

    const cards = page.locator('[data-testid="poster-card"]');
    const _cardCount = await cards.count();
    // Just screenshot — visual check
  });

  test('Library type filter — TV only', async ({ page }) => {
    await page.goto(
      `${BASE_URL}/library?sortBy=created_at&sortOrder=desc&page=1&pageSize=20&type=tv`,
      {
        waitUntil: 'networkidle',
        timeout: 30000,
      }
    );
    const _img = await screenshot(page, '13-library-tv-filter');
  });
});
