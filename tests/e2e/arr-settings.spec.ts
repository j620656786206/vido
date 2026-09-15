/**
 * Sonarr / Radarr connection cards on 連線設定 (story 13-6).
 *
 * Browser-level proof that the cards talk to the 13-4a endpoints with the right wire shape
 * (snake_case, write-only key) — the component spec mocks the hooks, so only this test sees
 * the real service + query wiring. The API is stubbed; no *arr server is needed.
 */
import { test, expect } from '../support/fixtures';

const ROUTE_API = '**/api/v1';

const NEVER_SET_UP = {
  url: '',
  enabled: false,
  quality_profile_id: 0,
  root_folder_path: '',
  has_api_key: false,
  health: { status: 'unconfigured', last_checked_at: null, message: 'plugin not configured' },
};

const ok = (data: unknown) => ({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify({ success: true, data }),
});

test.describe('Sonarr / Radarr settings @settings @ui @13-6', () => {
  test('[P1] first setup: test + save send snake_case, then the saved card unlocks its lists', async ({
    page,
  }) => {
    // Stateful stub: GET returns whatever the last PUT stored, like the real server.
    let sonarr: Record<string, unknown> = { ...NEVER_SET_UP };
    const testBodies: unknown[] = [];
    const saveBodies: Record<string, unknown>[] = [];
    let listRequests = 0;

    await page.route(`${ROUTE_API}/settings/radarr`, (route) => route.fulfill(ok(NEVER_SET_UP)));
    await page.route(`${ROUTE_API}/settings/sonarr`, (route) => {
      if (route.request().method() === 'PUT') {
        const body = route.request().postDataJSON() as Record<string, unknown>;
        saveBodies.push(body);
        sonarr = {
          url: body.url,
          enabled: body.enabled,
          quality_profile_id: body.quality_profile_id,
          root_folder_path: body.root_folder_path,
          has_api_key: true,
          health: { status: 'healthy', last_checked_at: '2026-09-15T00:00:00Z', message: '' },
        };
        return route.fulfill(ok({ message: 'Configuration saved' }));
      }
      return route.fulfill(ok(sonarr));
    });
    await page.route(`${ROUTE_API}/settings/sonarr/test`, (route) => {
      testBodies.push(route.request().postDataJSON());
      return route.fulfill(ok({ message: 'Connection successful' }));
    });
    await page.route(`${ROUTE_API}/settings/sonarr/quality-profiles`, (route) => {
      listRequests++;
      return route.fulfill(ok({ quality_profiles: [{ id: 4, name: 'HD-1080p' }] }));
    });
    await page.route(`${ROUTE_API}/settings/sonarr/root-folders`, (route) => {
      listRequests++;
      return route.fulfill(ok({ root_folders: [{ id: 1, path: '/data/media/tv' }] }));
    });

    await page.goto('/settings/connection');
    const card = page.getByTestId('arr-card-sonarr');
    await expect(card).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('arr-card-radarr')).toBeVisible();
    await expect(card.getByTestId('arr-health-sonarr')).toHaveText('未設定');
    await expect(card.getByRole('switch', { name: '啟用' })).toHaveAttribute(
      'aria-checked',
      'true'
    );
    expect(listRequests, 'lists must not be fetched before a saved, enabled connection').toBe(0);

    await card.getByLabel('網址').fill('http://192.168.1.100:8989');
    await card.getByLabel('API 金鑰').fill('secret-key');
    await card.getByRole('button', { name: '測試連線' }).click();
    await expect(card.getByTestId('arr-test-result-sonarr')).toContainText('連得到 Sonarr');
    expect(testBodies).toEqual([{ url: 'http://192.168.1.100:8989', api_key: 'secret-key' }]);

    await card.getByRole('button', { name: '儲存設定' }).click();
    await expect(card.getByText('設定已儲存')).toBeVisible();
    expect(saveBodies).toEqual([
      {
        url: 'http://192.168.1.100:8989',
        api_key: 'secret-key',
        enabled: true,
        quality_profile_id: 0,
        root_folder_path: '',
      },
    ]);

    // After the save: the card reads back the stored config, the key box is empty again, the
    // lists unlock, and the card says a profile + folder are still needed.
    await expect(card.getByTestId('arr-health-sonarr')).toHaveText('已連線');
    await expect(card.getByLabel('API 金鑰')).toHaveValue('');
    await expect(card.getByLabel('品質設定檔')).toBeEnabled();
    await expect(card.getByTestId('arr-setup-note')).toContainText('請求會停在等待中');
  });

  test('[P1] a refused save (409 DVR_TEST_FAILED around DVR_AUTH_FAILED) names both facts', async ({
    page,
  }) => {
    await page.route(`${ROUTE_API}/settings/sonarr`, (route) => route.fulfill(ok(NEVER_SET_UP)));
    await page.route(`${ROUTE_API}/settings/radarr/**`, (route) =>
      route.fulfill({
        status: 400,
        contentType: 'application/json',
        body: JSON.stringify({
          success: false,
          error: { code: 'DVR_CONNECTION_FAILED', message: '無法連線到 Radarr' },
        }),
      })
    );
    await page.route(`${ROUTE_API}/settings/radarr`, (route) => {
      if (route.request().method() === 'PUT') {
        return route.fulfill({
          status: 409,
          contentType: 'application/json',
          body: JSON.stringify({
            success: false,
            error: {
              code: 'DVR_TEST_FAILED',
              message: '儲存 Radarr 設定失敗',
              suggestion: 'radarr rejected the API key',
              cause_code: 'DVR_AUTH_FAILED',
            },
          }),
        });
      }
      return route.fulfill(
        ok({
          ...NEVER_SET_UP,
          url: 'http://192.168.1.100:7878',
          has_api_key: true,
          enabled: true,
          health: { status: 'unhealthy', last_checked_at: null, message: 'timeout' },
        })
      );
    });

    await page.goto('/settings/connection');
    const card = page.getByTestId('arr-card-radarr');
    await expect(card).toBeVisible({ timeout: 15000 });
    await expect(card.getByTestId('arr-health-radarr')).toHaveText('連不上');
    await expect(card.getByLabel('品質設定檔')).toHaveText(/讀不到 Radarr 的清單/);

    await card.getByRole('button', { name: '儲存設定' }).click();
    const alert = card.getByRole('alert');
    await expect(alert).toContainText('設定沒有儲存');
    await expect(alert).toContainText('API 金鑰不對');
  });
});
