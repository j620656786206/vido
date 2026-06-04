import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { filterPresetService } from './filterPresetService';

const API_BASE_URL = '/api/v1';

/**
 * Service-boundary contract tests (Story 11-4 TA gap-fill, P2).
 *
 * The critical design decision for 11-4 is that `filters` is an OPAQUE JSON
 * STRING, not a nested object — because the API boundary case-transforms object
 * keys (Rule 18) and a nested object's snake_case URL-param keys (year_gte)
 * would be mangled to camelCase on the round-trip. These tests lock that
 * contract at the layer where the mangling would happen: they assert the
 * outbound request body keeps `filters` as an untouched string, and the inbound
 * response preserves the inner snake_case keys.
 */
describe('filterPresetService', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('getAll', () => {
    it('[P2] returns the presets envelope on success', async () => {
      const presets = [
        {
          id: 'p1',
          name: '韓劇',
          filters: '{"region":"KR","year_gte":2024}',
          sort_order: 0,
          created_at: '2026-06-04T00:00:00Z',
        },
      ];
      vi.mocked(fetch).mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: true, data: { presets } }),
      } as Response);

      const result = await filterPresetService.getAll();

      expect(fetch).toHaveBeenCalledWith(`${API_BASE_URL}/filter-presets`, {
        headers: { 'Content-Type': 'application/json' },
      });
      // snakeToCamel maps sort_order→sortOrder / created_at→createdAt...
      expect(result.presets[0]).toMatchObject({ id: 'p1', sortOrder: 0 });
      // ...but the `filters` STRING value is left untouched (year_gte NOT → yearGte).
      expect(typeof result.presets[0].filters).toBe('string');
      expect(JSON.parse(result.presets[0].filters)).toMatchObject({ year_gte: 2024 });
    });
  });

  describe('create', () => {
    it('[P2] POSTs the name + filters string WITHOUT mangling the inner keys', async () => {
      vi.mocked(fetch).mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            success: true,
            data: {
              id: 'new',
              name: '高評分動畫',
              filters: '{"genre":"16","rating_gte":8}',
              sort_order: 1,
              created_at: '2026-06-04T00:00:00Z',
            },
          }),
      } as Response);

      const filtersJson = '{"genre":"16","rating_gte":8}';
      const result = await filterPresetService.create({ name: '高評分動畫', filters: filtersJson });

      // Outbound: the body's `filters` is the exact JSON string we passed in.
      const [, init] = vi.mocked(fetch).mock.calls[0];
      expect(init?.method).toBe('POST');
      const sentBody = JSON.parse(init?.body as string);
      expect(sentBody).toEqual({ name: '高評分動畫', filters: filtersJson });
      // The inner snake_case key survived camelToSnake (string value untouched).
      expect(JSON.parse(sentBody.filters)).toMatchObject({ rating_gte: 8 });

      // Inbound: response parsed, filters string preserved.
      expect(result).toMatchObject({ id: 'new', sortOrder: 1 });
      expect(JSON.parse(result.filters)).toMatchObject({ genre: '16', rating_gte: 8 });
    });

    it('[P2] throws with the backend error message on a non-ok response (e.g. limit reached)', async () => {
      vi.mocked(fetch).mockResolvedValue({
        ok: false,
        status: 409,
        json: () =>
          Promise.resolve({
            success: false,
            error: {
              code: 'FILTER_PRESET_LIMIT_REACHED',
              message: 'filter preset limit reached: max 20 presets',
            },
          }),
      } as Response);

      await expect(filterPresetService.create({ name: '太多', filters: '{}' })).rejects.toThrow(
        'filter preset limit reached: max 20 presets'
      );
    });
  });

  describe('remove', () => {
    it('[P2] DELETEs the preset by id', async () => {
      vi.mocked(fetch).mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: true, data: { deleted: true } }),
      } as Response);

      await filterPresetService.remove('p1');

      expect(fetch).toHaveBeenCalledWith(`${API_BASE_URL}/filter-presets/p1`, {
        headers: { 'Content-Type': 'application/json' },
        method: 'DELETE',
      });
    });
  });
});
