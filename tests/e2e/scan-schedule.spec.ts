/**
 * Scan schedule against the REAL backend (bugfix-scan-schedule-field-mismatch).
 *
 * The frontend sent and read `frequency`, the backend only knows `interval`, so
 * every save was a 400 and the select always read 僅手動 — from Story 7-3
 * (2026-03) until the TestSprite September run caught it. Both sides' unit
 * tests mocked their own idea of the field and stayed green; the e2e suite
 * stubbed /scanner/schedule too. This spec deliberately stubs NOTHING, so the
 * browser talks to the same handler users do (retro-dsr-AI1).
 *
 * @tags @e2e @scanner
 */

import { test, expect } from '../support/fixtures';

const API_URL = process.env.API_URL || 'http://localhost:8080/api/v1';

test.describe('Scan schedule (real backend) @e2e @scanner', () => {
  test.afterEach(async ({ request }) => {
    // Leave the shared backend on its default so other specs see manual scans.
    await request.put(`${API_URL}/scanner/schedule`, { data: { interval: 'manual' } });
  });

  test('[P0] choosing 每天 saves, and it is still 每天 after a reload', async ({ page }) => {
    await page.goto('/settings/scanner');

    const select = page.getByTestId('schedule-select');
    await expect(select).toBeVisible({ timeout: 15000 });

    const saved = page.waitForResponse(
      (r) => r.url().includes('/scanner/schedule') && r.request().method() === 'PUT'
    );
    await select.selectOption('daily');
    expect((await saved).status()).toBe(200);
    await expect(select).toHaveValue('daily');
    await expect(page.getByTestId('scanner-notification')).toHaveCount(0);

    await page.reload();
    await expect(page.getByTestId('schedule-select')).toHaveValue('daily', { timeout: 15000 });
  });
});
