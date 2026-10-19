/**
 * setup-wizard-byok AC #3 — a newcomer with only a Gemini key gets through the
 * wizard, and the key leaves the browser under the wire name the Go binding
 * reads (`gemini_api_key`). The server half — that the stored key reaches
 * translation — is proven against the real secrets service in
 * `setup_gemini_integration_test.go`; this spec covers the browser half.
 *
 * The API is stubbed: the dev server's real instance has finished setup long
 * ago, and the wizard only runs while /setup/status says needsSetup.
 */
import { test, expect, type Route } from '../support/fixtures';

const ROUTE_API = '**/api/v1';

const jsonOk = <T>(body: T) => ({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify({ success: true, data: body }),
});

test.describe('Setup wizard — Gemini-only key (setup-wizard-byok)', () => {
  test('says who pays, takes a Gemini key alone, and sends it as gemini_api_key', async ({
    page,
  }) => {
    let setupDone = false;
    const validated: Record<string, unknown>[] = [];
    let completeBody: Record<string, unknown> | null = null;

    await page.route(`${ROUTE_API}/setup/status`, (route: Route) =>
      route.fulfill(jsonOk({ needs_setup: !setupDone }))
    );
    await page.route(`${ROUTE_API}/setup/validate-step`, async (route: Route) => {
      validated.push(route.request().postDataJSON());
      await route.fulfill(jsonOk({ valid: true }));
    });
    await page.route(`${ROUTE_API}/setup/complete`, async (route: Route) => {
      completeBody = route.request().postDataJSON();
      setupDone = true;
      await route.fulfill(jsonOk({ message: 'Setup completed successfully' }));
    });
    await page.route(`${ROUTE_API}/settings/keys`, (route: Route) =>
      route.fulfill(
        jsonOk({
          writable: true,
          keys: [
            { name: 'claude', configured: false, source: 'none' },
            { name: 'gemini', configured: false, source: 'none' },
            { name: 'tmdb', configured: true, source: 'bundled' },
          ],
        })
      )
    );

    await page.goto('/setup');

    await page.getByTestId('next-button').click(); // welcome → qBittorrent
    await page.getByTestId('skip-button').click(); // → media library
    await page.getByTestId('library-path-0').fill('/media');
    await page.getByTestId('next-button').click(); // → API keys

    const step = page.getByTestId('api-keys-step');
    await expect(step).toContainText('費用直接記在你的 Claude 或 Gemini 帳號');
    await expect(page.getByTestId('gemini-key-link')).toHaveAttribute(
      'href',
      'https://aistudio.google.com/apikey'
    );
    await expect(page.getByTestId('skip-warning')).toBeVisible();

    await page.getByTestId('gemini-key-input').fill('AIza-e2e');
    await expect(page.getByTestId('skip-warning')).toBeHidden();
    await page.getByTestId('next-button').click(); // → usage report

    await expect(page.getByTestId('usage-report-step')).toBeVisible();
    await page.getByTestId('next-button').click(); // → complete

    const summary = page.getByTestId('complete-step');
    await expect(summary.getByText('Gemini 金鑰')).toBeVisible();
    await page.getByTestId('finish-button').click();

    await expect.poll(() => completeBody).not.toBeNull();
    expect(completeBody).toMatchObject({ gemini_api_key: 'AIza-e2e' });
    expect(completeBody).not.toHaveProperty('claude_api_key');
    expect(validated).toContainEqual({
      step: 'api-keys',
      data: { tmdb_api_key: '', claude_api_key: '', gemini_api_key: 'AIza-e2e' },
    });
  });
});
