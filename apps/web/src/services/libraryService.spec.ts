import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { libraryService } from './libraryService';

const API_BASE = 'http://localhost:8080/api/v1';

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
});
