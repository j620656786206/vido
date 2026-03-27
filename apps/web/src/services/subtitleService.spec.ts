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
        mediaId: 'movie-1',
        mediaType: 'movie',
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
      expect(body.convertToTraditional).toBe(false);
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
});
