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

  // --- Generation batch (ux3-subtitle-v2-batch AC 3, 9R-16 [@contract-v1]) ---

  describe('startGenerationBatch', () => {
    it('sends snake_case media_ids for scope=selected and parses the 202 result', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 202,
        json: () =>
          Promise.resolve({
            success: true,
            data: {
              batch_id: 'gb-1',
              total_items: 2,
              items: [
                { media_id: 42, title: '沙丘：第二部' },
                { media_id: 43, title: '奧本海默' },
              ],
            },
          }),
      });

      const outcome = await subtitleService.startGenerationBatch({
        scope: 'selected',
        mediaIds: [42, 43],
      });

      const [url, options] = mockFetch.mock.calls[0];
      expect(url).toContain('/subtitles/generation-batch');
      expect(options.method).toBe('POST');
      expect(JSON.parse(options.body)).toEqual({ scope: 'selected', media_ids: [42, 43] });
      expect(outcome).toEqual({
        conflict: false,
        result: {
          batchId: 'gb-1',
          totalItems: 2,
          items: [
            { mediaId: 42, title: '沙丘：第二部' },
            { mediaId: 43, title: '奧本海默' },
          ],
        },
      });
    });

    it('omits media_ids for scope=missing', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 202,
        json: () =>
          Promise.resolve({
            success: true,
            data: { batch_id: 'gb-2', total_items: 1, items: [{ media_id: 7, title: 'A' }] },
          }),
      });

      await subtitleService.startGenerationBatch({ scope: 'missing' });

      const [, options] = mockFetch.mock.calls[0];
      expect(JSON.parse(options.body)).toEqual({ scope: 'missing' });
    });

    it('maps the empty-missing 200 to batchId null (not an error)', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ success: true, data: { total_items: 0, items: [] } }),
      });

      const outcome = await subtitleService.startGenerationBatch({ scope: 'missing' });

      expect(outcome).toEqual({
        conflict: false,
        result: { batchId: null, totalItems: 0, items: [] },
      });
    });

    it('returns the in-progress snapshot on 409 instead of throwing', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 409,
        json: () =>
          Promise.resolve({
            success: false,
            error: { code: 'TRANSCRIPTION_BATCH_RUNNING', message: '已有一個字幕生成批次正在執行' },
            data: {
              batch_id: 'gb-run',
              total_items: 38,
              current_index: 12,
              current_media_id: 99,
              current_item: '怪奇物語',
              success_count: 11,
              fail_count: 0,
              paused_count: 0,
              status: 'running',
              spent_usd: 0.42,
              budget_usd: 5,
            },
          }),
      });

      const outcome = await subtitleService.startGenerationBatch({ scope: 'missing' });

      expect(outcome).toEqual({
        conflict: true,
        progress: {
          batchId: 'gb-run',
          totalItems: 38,
          currentIndex: 12,
          currentMediaId: 99,
          currentItem: '怪奇物語',
          successCount: 11,
          failCount: 0,
          pausedCount: 0,
          status: 'running',
          spentUsd: 0.42,
          budgetUsd: 5,
        },
      });
    });

    it('throws on a non-conflict error (400 selection reject)', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        json: () =>
          Promise.resolve({
            success: false,
            error: {
              code: 'VALIDATION_INVALID_FORMAT',
              message: 'media_ids 含無法生成字幕的項目（非電影或沒有媒體檔案）',
            },
          }),
      });

      await expect(
        subtitleService.startGenerationBatch({ scope: 'selected', mediaIds: [1] })
      ).rejects.toThrow('media_ids 含無法生成字幕的項目');
    });
  });

  describe('getGenerationBatchStatus', () => {
    it('parses the { running, progress } wrapper', async () => {
      mockFetch.mockResolvedValueOnce(
        mockSuccessResponse({
          running: true,
          progress: {
            batch_id: 'gb-1',
            total_items: 10,
            current_index: 3,
            current_media_id: 5,
            current_item: 'B',
            success_count: 2,
            fail_count: 0,
            paused_count: 0,
            status: 'running',
            spent_usd: 0.1,
            budget_usd: 5,
          },
        })
      );

      const result = await subtitleService.getGenerationBatchStatus();

      expect(mockFetch.mock.calls[0][0]).toContain('/subtitles/generation-batch/status');
      expect(result.running).toBe(true);
      expect(result.progress?.currentMediaId).toBe(5);
      expect(result.progress?.spentUsd).toBe(0.1);
    });

    it('handles the idle / post-terminal response (progress null)', async () => {
      mockFetch.mockResolvedValueOnce(mockSuccessResponse({ running: false, progress: null }));

      const result = await subtitleService.getGenerationBatchStatus();

      expect(result).toEqual({ running: false, progress: null });
    });
  });

  describe('cancelGenerationBatch', () => {
    it('POSTs to the cancel endpoint and returns { cancelled, running }', async () => {
      mockFetch.mockResolvedValueOnce(mockSuccessResponse({ cancelled: true, running: false }));

      const result = await subtitleService.cancelGenerationBatch();

      const [url, options] = mockFetch.mock.calls[0];
      expect(url).toContain('/subtitles/generation-batch/cancel');
      expect(options.method).toBe('POST');
      expect(result).toEqual({ cancelled: true, running: false });
    });
  });

  describe('previewGenerationBatch', () => {
    it('GETs preview?scope=missing and returns totalItems', async () => {
      mockFetch.mockResolvedValueOnce(mockSuccessResponse({ total_items: 38 }));

      const result = await subtitleService.previewGenerationBatch();

      expect(mockFetch.mock.calls[0][0]).toContain(
        '/subtitles/generation-batch/preview?scope=missing'
      );
      expect(result).toEqual({ totalItems: 38 });
    });
  });
});
