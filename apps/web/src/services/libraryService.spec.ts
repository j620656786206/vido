import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { libraryService } from './libraryService';

const API_BASE = '/api/v1';

describe('libraryService', () => {
  const mockFetch = vi.fn();

  beforeEach(() => {
    vi.stubGlobal('fetch', mockFetch);
    mockFetch.mockReset();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  function mockSuccessResponse<T>(data: T) {
    return {
      ok: true,
      json: () => Promise.resolve({ success: true, data }),
    };
  }

  function mockErrorResponse(status: number, message = 'Error') {
    return {
      ok: false,
      status,
      json: () => Promise.resolve({ error: { message } }),
    };
  }

  describe('listLibrary', () => {
    it('calls GET /library with no params by default', async () => {
      mockFetch.mockResolvedValue(
        mockSuccessResponse({ items: [], page: 1, pageSize: 20, totalItems: 0, totalPages: 0 })
      );

      await libraryService.listLibrary();

      expect(mockFetch).toHaveBeenCalledWith(`${API_BASE}/library`, undefined);
    });

    it('includes page and page_size as query params', async () => {
      mockFetch.mockResolvedValue(
        mockSuccessResponse({ items: [], page: 2, pageSize: 10, totalItems: 0, totalPages: 0 })
      );

      await libraryService.listLibrary({ page: 2, pageSize: 10 });

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).toContain('page=2');
      expect(url).toContain('page_size=10');
    });

    it('includes type filter when not "all"', async () => {
      mockFetch.mockResolvedValue(
        mockSuccessResponse({ items: [], page: 1, pageSize: 20, totalItems: 0, totalPages: 0 })
      );

      await libraryService.listLibrary({ type: 'movie' });

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).toContain('type=movie');
    });

    it('omits type filter when "all"', async () => {
      mockFetch.mockResolvedValue(
        mockSuccessResponse({ items: [], page: 1, pageSize: 20, totalItems: 0, totalPages: 0 })
      );

      await libraryService.listLibrary({ type: 'all' });

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).not.toContain('type=');
    });

    it('includes sort params when provided', async () => {
      mockFetch.mockResolvedValue(
        mockSuccessResponse({ items: [], page: 1, pageSize: 20, totalItems: 0, totalPages: 0 })
      );

      await libraryService.listLibrary({ sortBy: 'title', sortOrder: 'asc' });

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).toContain('sort_by=title');
      expect(url).toContain('sort_order=asc');
    });

    it('throws on API error response', async () => {
      mockFetch.mockResolvedValue(mockErrorResponse(500, 'Internal error'));

      await expect(libraryService.listLibrary()).rejects.toThrow('Internal error');
    });

    it('throws on success=false response', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: false, error: { message: 'Invalid request' } }),
      });

      await expect(libraryService.listLibrary()).rejects.toThrow('Invalid request');
    });

    it('handles network error gracefully', async () => {
      mockFetch.mockRejectedValue(new Error('Network error'));

      await expect(libraryService.listLibrary()).rejects.toThrow('Network error');
    });

    it('handles malformed error JSON gracefully', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 500,
        json: () => Promise.reject(new Error('parse error')),
      });

      await expect(libraryService.listLibrary()).rejects.toThrow('API request failed: 500');
    });
  });

  describe('deleteMovie', () => {
    it('calls DELETE /library/movies/:id', async () => {
      mockFetch.mockResolvedValue({ ok: true });

      await libraryService.deleteMovie('movie-123');

      expect(mockFetch).toHaveBeenCalledWith(`${API_BASE}/library/movies/movie-123`, {
        method: 'DELETE',
      });
    });

    it('throws on error response', async () => {
      mockFetch.mockResolvedValue(mockErrorResponse(404, 'Movie not found'));

      await expect(libraryService.deleteMovie('bad-id')).rejects.toThrow('Movie not found');
    });

    it('throws fallback message when error JSON is unparseable', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 500,
        json: () => Promise.reject(new Error('parse error')),
      });

      await expect(libraryService.deleteMovie('id')).rejects.toThrow('Failed to delete movie');
    });
  });

  describe('deleteSeries', () => {
    it('calls DELETE /library/series/:id', async () => {
      mockFetch.mockResolvedValue({ ok: true });

      await libraryService.deleteSeries('series-456');

      expect(mockFetch).toHaveBeenCalledWith(`${API_BASE}/library/series/series-456`, {
        method: 'DELETE',
      });
    });

    it('throws on error response', async () => {
      mockFetch.mockResolvedValue(mockErrorResponse(404, 'Series not found'));

      await expect(libraryService.deleteSeries('bad-id')).rejects.toThrow('Series not found');
    });
  });

  describe('reparseMovie', () => {
    it('calls POST /library/movies/:id/reparse', async () => {
      mockFetch.mockResolvedValue(mockSuccessResponse({ id: 'movie-1', status: 'reparse_queued' }));

      const result = await libraryService.reparseMovie('movie-1');

      expect(mockFetch).toHaveBeenCalledWith(`${API_BASE}/library/movies/movie-1/reparse`, {
        method: 'POST',
      });
      expect(result).toEqual({ id: 'movie-1', status: 'reparse_queued' });
    });
  });

  describe('reparseSeries', () => {
    it('calls POST /library/series/:id/reparse', async () => {
      mockFetch.mockResolvedValue(
        mockSuccessResponse({ id: 'series-1', status: 'reparse_queued' })
      );

      const result = await libraryService.reparseSeries('series-1');

      expect(mockFetch).toHaveBeenCalledWith(`${API_BASE}/library/series/series-1/reparse`, {
        method: 'POST',
      });
      expect(result).toEqual({ id: 'series-1', status: 'reparse_queued' });
    });
  });

  describe('exportMovie', () => {
    it('calls POST /library/movies/:id/export', async () => {
      const exportData = { title: 'Test Movie', metadata: {} };
      mockFetch.mockResolvedValue(mockSuccessResponse(exportData));

      const result = await libraryService.exportMovie('movie-1');

      expect(mockFetch).toHaveBeenCalledWith(`${API_BASE}/library/movies/movie-1/export`, {
        method: 'POST',
      });
      expect(result).toEqual(exportData);
    });
  });

  describe('exportSeries', () => {
    it('calls POST /library/series/:id/export', async () => {
      const exportData = { title: 'Test Series', metadata: {} };
      mockFetch.mockResolvedValue(mockSuccessResponse(exportData));

      const result = await libraryService.exportSeries('series-1');

      expect(mockFetch).toHaveBeenCalledWith(`${API_BASE}/library/series/series-1/export`, {
        method: 'POST',
      });
      expect(result).toEqual(exportData);
    });
  });

  describe('listLibrary filter params', () => {
    it('[P0] includes genres as query param', async () => {
      mockFetch.mockResolvedValue(
        mockSuccessResponse({ items: [], page: 1, pageSize: 20, totalItems: 0, totalPages: 0 })
      );

      await libraryService.listLibrary({ genres: '科幻,動作' });

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).toContain('genres=%E7%A7%91%E5%B9%BB%2C%E5%8B%95%E4%BD%9C');
    });

    it('[P0] includes yearMin as year_min query param', async () => {
      mockFetch.mockResolvedValue(
        mockSuccessResponse({ items: [], page: 1, pageSize: 20, totalItems: 0, totalPages: 0 })
      );

      await libraryService.listLibrary({ yearMin: 2010 });

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).toContain('year_min=2010');
    });

    it('[P0] includes yearMax as year_max query param', async () => {
      mockFetch.mockResolvedValue(
        mockSuccessResponse({ items: [], page: 1, pageSize: 20, totalItems: 0, totalPages: 0 })
      );

      await libraryService.listLibrary({ yearMax: 2020 });

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).toContain('year_max=2020');
    });

    it('[P0] includes all filter params together', async () => {
      mockFetch.mockResolvedValue(
        mockSuccessResponse({ items: [], page: 1, pageSize: 20, totalItems: 0, totalPages: 0 })
      );

      await libraryService.listLibrary({
        type: 'movie',
        genres: '劇情',
        yearMin: 2000,
        yearMax: 2010,
        sortBy: 'title',
        sortOrder: 'asc',
        page: 2,
        pageSize: 10,
      });

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).toContain('type=movie');
      expect(url).toContain('genres=');
      expect(url).toContain('year_min=2000');
      expect(url).toContain('year_max=2010');
      expect(url).toContain('sort_by=title');
      expect(url).toContain('sort_order=asc');
      expect(url).toContain('page=2');
      expect(url).toContain('page_size=10');
    });

    it('[P1] omits yearMin when not provided', async () => {
      mockFetch.mockResolvedValue(
        mockSuccessResponse({ items: [], page: 1, pageSize: 20, totalItems: 0, totalPages: 0 })
      );

      await libraryService.listLibrary({ genres: '科幻' });

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).not.toContain('year_min');
      expect(url).not.toContain('year_max');
    });

    it('[P1] omits genres when not provided', async () => {
      mockFetch.mockResolvedValue(
        mockSuccessResponse({ items: [], page: 1, pageSize: 20, totalItems: 0, totalPages: 0 })
      );

      await libraryService.listLibrary({ yearMin: 2020 });

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).not.toContain('genres');
    });
  });

  describe('searchLibrary', () => {
    it('[P1] calls GET /library/search with query param', async () => {
      mockFetch.mockResolvedValue(
        mockSuccessResponse({ items: [], totalItems: 0, page: 1, pageSize: 20, totalPages: 0 })
      );

      await libraryService.searchLibrary('batman');

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).toContain('/library/search?');
      expect(url).toContain('q=batman');
    });

    it('[P1] includes type filter in search', async () => {
      mockFetch.mockResolvedValue(
        mockSuccessResponse({ items: [], totalItems: 0, page: 1, pageSize: 20, totalPages: 0 })
      );

      await libraryService.searchLibrary('test', { type: 'movie' });

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).toContain('q=test');
      expect(url).toContain('type=movie');
    });

    it('[P1] includes pagination params in search', async () => {
      mockFetch.mockResolvedValue(
        mockSuccessResponse({ items: [], totalItems: 0, page: 2, pageSize: 10, totalPages: 0 })
      );

      await libraryService.searchLibrary('query', { page: 2, pageSize: 10 });

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).toContain('page=2');
      expect(url).toContain('page_size=10');
    });

    it('[P1] omits type when "all"', async () => {
      mockFetch.mockResolvedValue(
        mockSuccessResponse({ items: [], totalItems: 0, page: 1, pageSize: 20, totalPages: 0 })
      );

      await libraryService.searchLibrary('test', { type: 'all' });

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).not.toContain('type=');
    });

    it('[P1] throws on API error', async () => {
      mockFetch.mockResolvedValue(mockErrorResponse(500, 'Search failed'));

      await expect(libraryService.searchLibrary('test')).rejects.toThrow('Search failed');
    });
  });

  describe('getGenres', () => {
    it('[P1] calls GET /library/genres', async () => {
      const genres = ['科幻', '動作', '劇情', '恐怖'];
      mockFetch.mockResolvedValue(mockSuccessResponse(genres));

      const result = await libraryService.getGenres();

      expect(mockFetch).toHaveBeenCalledWith(`${API_BASE}/library/genres`, undefined);
      expect(result).toEqual(genres);
    });

    it('[P1] throws on API error', async () => {
      mockFetch.mockResolvedValue(mockErrorResponse(500, 'Failed to fetch genres'));

      await expect(libraryService.getGenres()).rejects.toThrow('Failed to fetch genres');
    });
  });

  describe('getStats', () => {
    it('[P1] calls GET /library/stats', async () => {
      const stats = { yearMin: 1990, yearMax: 2024, movieCount: 100, tvCount: 50, totalCount: 150 };
      mockFetch.mockResolvedValue(mockSuccessResponse(stats));

      const result = await libraryService.getStats();

      expect(mockFetch).toHaveBeenCalledWith(`${API_BASE}/library/stats`, undefined);
      expect(result).toEqual(stats);
    });

    it('[P1] throws on API error', async () => {
      mockFetch.mockResolvedValue(mockErrorResponse(500, 'Failed to fetch stats'));

      await expect(libraryService.getStats()).rejects.toThrow('Failed to fetch stats');
    });
  });

  describe('batchDelete (Story 5-7)', () => {
    it('[P0] calls DELETE /library/batch with ids and type', async () => {
      const batchResult = { successCount: 3, failedCount: 0 };
      mockFetch.mockResolvedValue(mockSuccessResponse(batchResult));

      const result = await libraryService.batchDelete(['m1', 'm2', 'm3'], 'movie');

      expect(mockFetch).toHaveBeenCalledWith(`${API_BASE}/library/batch`, {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ids: ['m1', 'm2', 'm3'], type: 'movie' }),
      });
      expect(result).toEqual(batchResult);
    });

    it('[P0] sends series type correctly', async () => {
      mockFetch.mockResolvedValue(mockSuccessResponse({ successCount: 2, failedCount: 0 }));

      await libraryService.batchDelete(['s1', 's2'], 'series');

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(body.type).toBe('series');
      expect(body.ids).toEqual(['s1', 's2']);
    });

    it('[P1] returns partial failure result', async () => {
      const partialResult = {
        successCount: 2,
        failedCount: 1,
        errors: [{ id: 'm3', message: 'not found' }],
      };
      mockFetch.mockResolvedValue(mockSuccessResponse(partialResult));

      const result = await libraryService.batchDelete(['m1', 'm2', 'm3'], 'movie');

      expect(result.failedCount).toBe(1);
      expect(result.errors).toHaveLength(1);
      expect(result.errors![0].id).toBe('m3');
    });

    it('[P1] throws on API error response', async () => {
      mockFetch.mockResolvedValue(mockErrorResponse(500, 'Batch delete failed'));

      await expect(libraryService.batchDelete(['m1'], 'movie')).rejects.toThrow(
        'Batch delete failed'
      );
    });

    it('[P1] throws on success=false response', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({ success: false, error: { message: 'Invalid batch request' } }),
      });

      await expect(libraryService.batchDelete(['m1'], 'movie')).rejects.toThrow(
        'Invalid batch request'
      );
    });
  });

  describe('batchReparse (Story 5-7)', () => {
    it('[P0] calls POST /library/batch/reparse with ids and type', async () => {
      const batchResult = { successCount: 3, failedCount: 0 };
      mockFetch.mockResolvedValue(mockSuccessResponse(batchResult));

      const result = await libraryService.batchReparse(['m1', 'm2', 'm3'], 'movie');

      expect(mockFetch).toHaveBeenCalledWith(`${API_BASE}/library/batch/reparse`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ids: ['m1', 'm2', 'm3'], type: 'movie' }),
      });
      expect(result).toEqual(batchResult);
    });

    it('[P0] sends series type correctly', async () => {
      mockFetch.mockResolvedValue(mockSuccessResponse({ successCount: 1, failedCount: 0 }));

      await libraryService.batchReparse(['s1'], 'series');

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(body.type).toBe('series');
    });

    it('[P1] throws on API error response', async () => {
      mockFetch.mockResolvedValue(mockErrorResponse(500, 'Batch reparse failed'));

      await expect(libraryService.batchReparse(['m1'], 'movie')).rejects.toThrow(
        'Batch reparse failed'
      );
    });
  });

  describe('batchExport (Story 5-7)', () => {
    it('[P0] calls POST /library/batch/export with type as query param', async () => {
      const exportData = [{ title: 'Movie 1' }, { title: 'Movie 2' }];
      mockFetch.mockResolvedValue(mockSuccessResponse(exportData));

      const result = await libraryService.batchExport(['m1', 'm2'], 'movie');

      expect(mockFetch).toHaveBeenCalledWith(`${API_BASE}/library/batch/export?type=movie`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ids: ['m1', 'm2'], format: 'json' }),
      });
      expect(result).toEqual(exportData);
    });

    it('[P0] includes series type in query param', async () => {
      mockFetch.mockResolvedValue(mockSuccessResponse([{ title: 'Series 1' }]));

      await libraryService.batchExport(['s1'], 'series');

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).toContain('type=series');
    });

    it('[P1] sends format=json in body', async () => {
      mockFetch.mockResolvedValue(mockSuccessResponse([]));

      await libraryService.batchExport(['m1'], 'movie');

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(body.format).toBe('json');
    });

    it('[P1] throws on API error response', async () => {
      mockFetch.mockResolvedValue(mockErrorResponse(500, 'Batch export failed'));

      await expect(libraryService.batchExport(['m1'], 'movie')).rejects.toThrow(
        'Batch export failed'
      );
    });
  });

  describe('getRecentlyAdded', () => {
    it('[P2] calls GET /library/recent with default limit', async () => {
      mockFetch.mockResolvedValue(
        mockSuccessResponse({ items: [], page: 1, pageSize: 20, totalItems: 0, totalPages: 0 })
      );

      await libraryService.getRecentlyAdded();

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).toContain('/library/recent?limit=20');
    });

    it('[P2] calls GET /library/recent with custom limit', async () => {
      mockFetch.mockResolvedValue(
        mockSuccessResponse({ items: [], page: 1, pageSize: 10, totalItems: 0, totalPages: 0 })
      );

      await libraryService.getRecentlyAdded(10);

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).toContain('/library/recent?limit=10');
    });
  });
});
