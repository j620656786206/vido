import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  nfoLocalizerService,
  NfoLocalizeApiError,
  NFO_ERROR_CODES,
  isSeriesResult,
} from './nfoLocalizerService';

function mockJson(body: unknown, ok = true, status = 200) {
  return vi.fn().mockResolvedValue({
    ok,
    status,
    json: () => Promise.resolve(body),
  } as unknown as Response);
}

const okEnvelope = (data: unknown) => ({ success: true, data });

function lastCall() {
  const [url, init] = (global.fetch as unknown as ReturnType<typeof vi.fn>).mock.calls[0];
  return { url: url as string, init: init as RequestInit | undefined };
}

function parsedBody() {
  const { init } = lastCall();
  return init?.body ? JSON.parse(init.body as string) : undefined;
}

describe('nfoLocalizerService', () => {
  beforeEach(() => {
    global.fetch = mockJson(okEnvelope({ path: '/media/x.nfo', backup_path: '', replaced: false }));
  });
  afterEach(() => vi.restoreAllMocks());

  describe('localizeMovieNfo', () => {
    // Movies are ADDITIVE — the backend has no confirm gate on this route, and
    // sending one would imply a risk that does not exist.
    it('POSTs to the movie route with NO body', async () => {
      await nfoLocalizerService.localizeMovieNfo('movie-1');

      const { url, init } = lastCall();
      expect(url).toBe('/api/v1/movies/movie-1/localize-nfo');
      expect(init?.method).toBe('POST');
      expect(init?.body).toBeUndefined();
    });

    it('snakeToCamel maps backup_path onto backupPath', async () => {
      global.fetch = mockJson(
        okEnvelope({ path: '/media/x.nfo', backup_path: '/media/x.nfo.orig', replaced: true })
      );

      const res = await nfoLocalizerService.localizeMovieNfo('movie-1');

      expect(res).toEqual({
        path: '/media/x.nfo',
        backupPath: '/media/x.nfo.orig',
        replaced: true,
      });
    });
  });

  describe('localizeSeriesNfo', () => {
    // 🔴 The confirm flag is a PARAMETER, never a constant: the backend gate
    // exists to protect the user's curated tvshow.nfo, and a hard-coded `true`
    // here would silently defeat it.
    it('sends confirm_replace: true when the user confirmed', async () => {
      await nfoLocalizerService.localizeSeriesNfo('series-1', { confirmReplace: true });

      expect(lastCall().url).toBe('/api/v1/series/series-1/localize-nfo');
      expect(parsedBody()).toEqual({ confirm_replace: true });
    });

    it('sends confirm_replace: false verbatim — it does not "helpfully" coerce', async () => {
      global.fetch = mockJson(
        { success: false, error: { code: NFO_ERROR_CODES.notConfirmed, message: 'nope' } },
        false,
        409
      );

      await expect(
        nfoLocalizerService.localizeSeriesNfo('series-1', { confirmReplace: false })
      ).rejects.toThrow(NfoLocalizeApiError);

      expect(parsedBody()).toEqual({ confirm_replace: false });
    });

    it('omits the query string when includeEpisodes is not asked for', async () => {
      await nfoLocalizerService.localizeSeriesNfo('series-1', { confirmReplace: true });
      expect(lastCall().url).not.toContain('include_episodes');
    });

    it('appends include_episodes=true when asked', async () => {
      await nfoLocalizerService.localizeSeriesNfo('series-1', {
        confirmReplace: true,
        includeEpisodes: true,
      });
      expect(lastCall().url).toBe('/api/v1/series/series-1/localize-nfo?include_episodes=true');
    });

    it('returns the whole-show shape with its three counters', async () => {
      global.fetch = mockJson(
        okEnvelope({
          show: { path: '/media/Buffy/tvshow.nfo', backup_path: '', replaced: false },
          episodes: [{ path: '/media/Buffy/S01/e1.nfo', backup_path: '', replaced: false }],
          succeeded: 22,
          failed: 0,
          skipped: 2,
        })
      );

      const res = await nfoLocalizerService.localizeSeriesNfo('series-1', {
        confirmReplace: true,
        includeEpisodes: true,
      });

      expect(isSeriesResult(res)).toBe(true);
      if (!isSeriesResult(res)) throw new Error('unreachable');
      expect(res.succeeded).toBe(22);
      expect(res.skipped).toBe(2);
      expect(res.show.path).toBe('/media/Buffy/tvshow.nfo');
    });
  });

  describe('localizeEpisodeNfo', () => {
    it('POSTs the confirm flag to the episode route', async () => {
      await nfoLocalizerService.localizeEpisodeNfo('episode-1', { confirmReplace: true });

      expect(lastCall().url).toBe('/api/v1/episodes/episode-1/localize-nfo');
      expect(parsedBody()).toEqual({ confirm_replace: true });
    });
  });

  describe('error envelope', () => {
    // The generic parseError helper other services use throws a bare Error and
    // drops error.code — which would make all four failures look identical to
    // the UI. This surface needs to tell them apart.
    it.each([
      [503, NFO_ERROR_CODES.disabled],
      [500, NFO_ERROR_CODES.failed],
      [400, NFO_ERROR_CODES.missingPath],
      [409, NFO_ERROR_CODES.notConfirmed],
    ])('preserves error.code for HTTP %i (%s)', async (status, code) => {
      global.fetch = mockJson({ success: false, error: { code, message: 'boom' } }, false, status);

      await expect(nfoLocalizerService.localizeMovieNfo('m')).rejects.toMatchObject({
        name: 'NfoLocalizeApiError',
        code,
        message: 'boom',
      });
    });

    it('falls back to INTERNAL_ERROR when the body is not a parseable envelope', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 502,
        json: () => Promise.reject(new Error('not json')),
      } as unknown as Response);

      await expect(nfoLocalizerService.localizeMovieNfo('m')).rejects.toMatchObject({
        code: 'INTERNAL_ERROR',
      });
    });

    // A 200 carrying success:false is still a failure — the envelope, not the
    // status line, is the contract (Rule 3).
    it('treats success:false as an error even on HTTP 200', async () => {
      global.fetch = mockJson({ success: false, error: { code: 'X_Y', message: 'nope' } });

      await expect(nfoLocalizerService.localizeMovieNfo('m')).rejects.toMatchObject({
        code: 'X_Y',
      });
    });
  });

  describe('isSeriesResult', () => {
    it('separates the two response shapes', () => {
      expect(isSeriesResult({ path: 'a', backupPath: '', replaced: false })).toBe(false);
      expect(
        isSeriesResult({
          show: { path: 'a', backupPath: '', replaced: false },
          episodes: [],
          succeeded: 0,
          failed: 0,
          skipped: 0,
        })
      ).toBe(true);
    });
  });
});
