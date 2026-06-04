/**
 * Saved Filter Presets API E2E Tests (Story 11-4)
 *
 * Exercises the LIVE backend stack — HTTP handler → service → repository →
 * SQLite (migration 023) — which the unit and UI suites cannot:
 *   - the handler unit test uses a MOCK service + a fresh gin router (not
 *     main.go's route registration);
 *   - the UI E2E (saved-filter-presets.spec.ts) MOCKS /filter-presets entirely.
 * This suite is the only coverage of the real wire contract, route wiring
 * (Rule 15), real persistence, and the filters-as-opaque-string round-trip.
 *
 * Non-destructive + self-cleaning: every preset created here is tracked and
 * DELETEd in afterEach, so a developer's real presets in the dev DB are never
 * wiped. The max-20 limit is intentionally NOT exercised here (it is covered at
 * the service + handler unit level, and creating 20 rows on a shared DB would be
 * destructive) — avoid-duplicate-coverage.
 *
 * Prerequisites: Go backend running on port 8080 (cd apps/api && go run ./cmd/api)
 *
 * @tags @api @discover @preset @story-11-4 @regression
 */

import { test, expect, type APIRequestContext } from '../support/fixtures';

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080/api/v1';

test.describe('Saved Filter Presets API @api @discover @preset', () => {
  // Track every preset created by a test so afterEach can clean only those rows.
  const createdIds: string[] = [];

  async function createPreset(
    request: APIRequestContext,
    name: string,
    filters: string
  ): Promise<{ status: number; body: { success: boolean; data?: { id: string } } }> {
    const response = await request.post(`${API_BASE_URL}/filter-presets`, {
      data: { name, filters },
    });
    const body = await response.json();
    if (body?.data?.id) createdIds.push(body.data.id);
    return { status: response.status(), body };
  }

  test.afterEach(async ({ request }) => {
    for (const id of createdIds) {
      await request.delete(`${API_BASE_URL}/filter-presets/${id}`);
    }
    createdIds.length = 0;
  });

  test('[P1] POST persists a preset and GET returns it with filters intact as a JSON string (AC #1, #2, #5)', async ({
    request,
  }) => {
    // GIVEN: a filters payload in URL-param shape (snake_case keys)
    const filters = '{"genre":"16","year_gte":2024,"region":"KR"}';

    // WHEN: creating the preset over the live API
    const { status, body } = await createPreset(request, 'API 測試動畫', filters);

    // THEN: it is created (201) and echoed back
    expect(status).toBe(201);
    expect(body.success).toBe(true);
    expect(body.data?.id).toBeTruthy();

    // AND: GET lists it, with filters preserved as a STRING (inner keys NOT mangled)
    const listResp = await request.get(`${API_BASE_URL}/filter-presets`);
    expect(listResp.status()).toBe(200);
    const list = await listResp.json();
    const found = list.data.presets.find((p: { id: string }) => p.id === body.data?.id);
    expect(found).toBeTruthy();
    expect(typeof found.filters).toBe('string');
    expect(JSON.parse(found.filters)).toMatchObject({ genre: '16', year_gte: 2024, region: 'KR' });
  });

  test('[P1] DELETE removes a persisted preset (AC #4)', async ({ request }) => {
    // GIVEN: a persisted preset
    const { body } = await createPreset(request, 'API 待刪除', '{"region":"JP"}');
    const id = body.data?.id as string;

    // WHEN: deleting it
    const delResp = await request.delete(`${API_BASE_URL}/filter-presets/${id}`);
    expect(delResp.status()).toBe(200);

    // THEN: it no longer appears in the list
    const listResp = await request.get(`${API_BASE_URL}/filter-presets`);
    const list = await listResp.json();
    expect(list.data.presets.find((p: { id: string }) => p.id === id)).toBeUndefined();
  });

  test('[P1] POST with an empty name is rejected with a validation error (AC #1)', async ({
    request,
  }) => {
    // WHEN: creating a preset with a blank name
    const { status, body } = await createPreset(request, '   ', '{"genre":"16"}');

    // THEN: 400 with the FILTER_PRESET validation code (no row created)
    expect(status).toBe(400);
    expect(body.success).toBe(false);
    expect((body as { error: { code: string } }).error.code).toBe(
      'FILTER_PRESET_VALIDATION_FAILED'
    );
  });

  test('[P1] POST with malformed filters JSON is rejected (AC #1)', async ({ request }) => {
    // WHEN: filters is not valid JSON
    const { status, body } = await createPreset(request, '壞資料', '{not json');

    // THEN: 400 validation error
    expect(status).toBe(400);
    expect((body as { error: { code: string } }).error.code).toBe(
      'FILTER_PRESET_VALIDATION_FAILED'
    );
  });

  test('[P2] DELETE of a non-existent preset returns 404', async ({ request }) => {
    // WHEN: deleting an id that does not exist
    const response = await request.delete(`${API_BASE_URL}/filter-presets/does-not-exist`);

    // THEN: 404 with the not-found code
    expect(response.status()).toBe(404);
    const body = await response.json();
    expect(body.error.code).toBe('FILTER_PRESET_NOT_FOUND');
  });
});
