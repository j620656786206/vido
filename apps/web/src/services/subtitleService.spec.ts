import { describe, it, expect, vi, beforeEach } from 'vitest';
import { subtitleService } from './subtitleService';

// Mock global fetch
const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

function mockSuccessResponse<T>(data: T) {
  return {
    ok: true,
    json: () => Promise.resolve({ success: true, data }),
  };
}

function mockErrorResponse(status: number, message: string) {
  return {
    ok: false,
    status,
    json: () => Promise.resolve({ error: { message } }),
  };
}

describe('subtitleService', () => {
  beforeEach(() => {
    mockFetch.mockReset();
  });

  describe('searchSubtitles', () => {
    it('sends POST request with correct params', async () => {
      const mockResults = [{ id: '1', source: 'assrt', score: 0.85 }];
      mockFetch.mockResolvedValueOnce(mockSuccessResponse(mockResults));

      const result = await subtitleService.searchSubtitles({
        mediaId: 'movie-1',
        mediaType: 'movie',
        query: 'Test',
      });

      expect(mockFetch).toHaveBeenCalledOnce();
      const [url, options] = mockFetch.mock.calls[0];
      expect(url).toContain('/subtitles/search');
      expect(options.method).toBe('POST');
      expect(JSON.parse(options.body)).toEqual({
        media_id: 'movie-1',
        media_type: 'movie',
        query: 'Test',
      });
      expect(result).toEqual(mockResults);
    });

    it('throws on API error', async () => {
      mockFetch.mockResolvedValueOnce(mockErrorResponse(400, 'Invalid request'));
      await expect(
        subtitleService.searchSubtitles({ mediaId: '1', mediaType: 'movie' })
      ).rejects.toThrow('Invalid request');
    });
  });

  describe('downloadSubtitle', () => {
    it('sends POST request with convertToTraditional', async () => {
      const mockResult = { subtitlePath: '/path/sub.srt', language: 'zh-Hant', score: 0.9 };
      mockFetch.mockResolvedValueOnce(mockSuccessResponse(mockResult));

      const result = await subtitleService.downloadSubtitle({
        mediaId: 'movie-1',
        mediaType: 'movie',
        mediaFilePath: '/media/movie.mkv',
        subtitleId: 'sub-1',
        provider: 'assrt',
        convertToTraditional: false,
      });

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(body.convert_to_traditional).toBe(false);
      expect(result).toEqual(mockResult);
    });
  });

  describe('previewSubtitle', () => {
    it('sends POST and returns lines', async () => {
      const mockResult = { lines: ['Line 1', 'Line 2'], language: 'zh-Hant' };
      mockFetch.mockResolvedValueOnce(mockSuccessResponse(mockResult));

      const result = await subtitleService.previewSubtitle({
        subtitleId: 'sub-1',
        provider: 'assrt',
      });

      expect(result.lines).toHaveLength(2);
      expect(result.language).toBe('zh-Hant');
    });
  });

  // --- Batch (Story 8-11) ---

  describe('startBatch', () => {
    it('sends a snake_case body and parses the 202 camelCase result', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 202,
        json: () => Promise.resolve({ success: true, data: { batch_id: 'b-1', total_items: 42 } }),
      });

      const outcome = await subtitleService.startBatch({ scope: 'library' });

      const [url, options] = mockFetch.mock.calls[0];
      expect(url).toContain('/subtitles/batch');
      expect(options.method).toBe('POST');
      expect(JSON.parse(options.body)).toEqual({ scope: 'library' });

      expect(outcome.conflict).toBe(false);
      if (!outcome.conflict) {
        expect(outcome.result).toEqual({ batchId: 'b-1', totalItems: 42 });
      }
    });

    it('serializes seasonId to season_id for season scope', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 202,
        json: () => Promise.resolve({ success: true, data: { batch_id: 'b-2', total_items: 8 } }),
      });

      await subtitleService.startBatch({ scope: 'season', seasonId: 'season-9' });

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(body).toEqual({ scope: 'season', season_id: 'season-9' });
    });

    it('returns the in-progress snapshot on 409 instead of throwing (AC #7)', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 409,
        json: () =>
          Promise.resolve({
            success: false,
            error: { code: 'SUBTITLE_BATCH_RUNNING', message: 'already running' },
            data: {
              batch_id: 'b-running',
              total_items: 100,
              current_index: 10,
              current_item: '電影 A',
              success_count: 7,
              fail_count: 3,
              status: 'running',
            },
          }),
      });

      const outcome = await subtitleService.startBatch({ scope: 'library' });

      expect(outcome.conflict).toBe(true);
      if (outcome.conflict) {
        expect(outcome.progress.batchId).toBe('b-running');
        expect(outcome.progress.currentIndex).toBe(10);
        expect(outcome.progress.successCount).toBe(7);
        expect(outcome.progress.status).toBe('running');
      }
    });

    it('throws on a non-conflict error response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: () => Promise.resolve({ success: false, error: { message: 'boom' } }),
      });

      await expect(subtitleService.startBatch({ scope: 'library' })).rejects.toThrow('boom');
    });
  });

  describe('getBatchStatus', () => {
    it('parses the { running, progress } wrapper', async () => {
      mockFetch.mockResolvedValueOnce(
        mockSuccessResponse({
          running: true,
          progress: {
            batch_id: 'b-1',
            total_items: 20,
            current_index: 5,
            current_item: '影集 B',
            success_count: 4,
            fail_count: 1,
            status: 'running',
          },
        })
      );

      const status = await subtitleService.getBatchStatus();

      const [url] = mockFetch.mock.calls[0];
      expect(url).toContain('/subtitles/batch/status');
      expect(status.running).toBe(true);
      expect(status.progress?.batchId).toBe('b-1');
      expect(status.progress?.currentItem).toBe('影集 B');
    });

    it('handles the idle response', async () => {
      mockFetch.mockResolvedValueOnce(mockSuccessResponse({ running: false }));
      const status = await subtitleService.getBatchStatus();
      expect(status.running).toBe(false);
      expect(status.progress).toBeUndefined();
    });
  });

  describe('cancelBatch', () => {
    it('POSTs to the cancel endpoint and returns cancelled', async () => {
      mockFetch.mockResolvedValueOnce(mockSuccessResponse({ cancelled: true }));

      const result = await subtitleService.cancelBatch();

      const [url, options] = mockFetch.mock.calls[0];
      expect(url).toContain('/subtitles/batch/cancel');
      expect(options.method).toBe('POST');
      expect(result.cancelled).toBe(true);
    });
  });
});
