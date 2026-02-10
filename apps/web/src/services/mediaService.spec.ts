import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mediaService } from './mediaService';

const API_BASE_URL = 'http://localhost:8080/api/v1';

describe('mediaService', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('getRecentMedia', () => {
    it('[P1] returns recent media data on success', async () => {
      // GIVEN: API returns successful response
      const mockMedia = [
        {
          id: 'movie-1',
          title: 'Test Movie',
          year: 2024,
          posterUrl: '/poster.jpg',
          mediaType: 'movie',
          justAdded: true,
          addedAt: '2026-02-10T10:00:00Z',
        },
      ];

      vi.mocked(fetch).mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: true, data: mockMedia }),
      } as Response);

      // WHEN: Calling getRecentMedia
      const result = await mediaService.getRecentMedia(10);

      // THEN: Returns the data array
      expect(result).toEqual(mockMedia);
      expect(fetch).toHaveBeenCalledWith(`${API_BASE_URL}/media/recent?limit=10`, {
        headers: { 'Content-Type': 'application/json' },
      });
    });

    it('[P1] throws error on non-ok HTTP response', async () => {
      // GIVEN: API returns 500
      vi.mocked(fetch).mockResolvedValue({
        ok: false,
        status: 500,
        json: () =>
          Promise.resolve({
            success: false,
            error: { code: 'DB_QUERY_FAILED', message: 'Database error' },
          }),
      } as Response);

      // WHEN/THEN: Should throw with error message from API
      await expect(mediaService.getRecentMedia()).rejects.toThrow('Database error');
    });

    it('[P1] throws error when API returns success: false', async () => {
      // GIVEN: API returns 200 but success: false
      vi.mocked(fetch).mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            success: false,
            error: { code: 'TMDB_TIMEOUT', message: 'TMDb timeout' },
          }),
      } as Response);

      // WHEN/THEN: Should throw with error message
      await expect(mediaService.getRecentMedia()).rejects.toThrow('TMDb timeout');
    });

    it('[P2] throws generic error when non-ok response has no JSON body', async () => {
      // GIVEN: API returns 500 with non-parseable body
      vi.mocked(fetch).mockResolvedValue({
        ok: false,
        status: 500,
        json: () => Promise.reject(new Error('Invalid JSON')),
      } as Response);

      // WHEN/THEN: Should throw generic error
      await expect(mediaService.getRecentMedia()).rejects.toThrow('API request failed: 500');
    });

    it('[P2] uses default limit of 10', async () => {
      // GIVEN: API returns success
      vi.mocked(fetch).mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: true, data: [] }),
      } as Response);

      // WHEN: Calling without limit parameter
      await mediaService.getRecentMedia();

      // THEN: Uses default limit=10
      expect(fetch).toHaveBeenCalledWith(`${API_BASE_URL}/media/recent?limit=10`, {
        headers: { 'Content-Type': 'application/json' },
      });
    });
  });
});
