import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { requestService, RequestApiError } from './requestService';

const okEnvelope = (data: unknown) => ({
  ok: true,
  json: () => Promise.resolve({ success: true, data }),
});

describe('requestService', () => {
  const fetchMock = vi.fn();

  beforeEach(() => {
    vi.stubGlobal('fetch', fetchMock);
    fetchMock.mockReset();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('listRequests hits GET /requests and snake→camel transforms rows ([@contract-v1] 13-1a AC #3)', async () => {
    fetchMock.mockResolvedValue(
      okEnvelope({
        requests: [
          {
            id: 'r1',
            tmdb_id: 550,
            media_type: 'movie',
            title: '鬥陣俱樂部',
            status: 'pending',
            fulfilment_source: null,
            external_id: null,
            seasons: null,
            episodes: null,
            error_message: null,
            requested_at: '2026-07-04T12:00:00Z',
            updated_at: '2026-07-04T12:00:00Z',
          },
        ],
      })
    );

    const requests = await requestService.listRequests();

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/requests', undefined);
    expect(requests).toHaveLength(1);
    expect(requests[0].tmdbId).toBe(550);
    expect(requests[0].mediaType).toBe('movie');
    expect(requests[0].requestedAt).toBe('2026-07-04T12:00:00Z');
  });

  it('listRequests returns [] when the payload has no rows', async () => {
    fetchMock.mockResolvedValue(okEnvelope({ requests: [] }));
    await expect(requestService.listRequests()).resolves.toEqual([]);
  });

  it('createRequest POSTs a camel→snake body (Rule 18) and returns the created resource', async () => {
    fetchMock.mockResolvedValue(
      okEnvelope({
        id: 'new',
        tmdb_id: 1399,
        media_type: 'tv',
        title: '權力遊戲',
        status: 'pending',
      })
    );

    const created = await requestService.createRequest({ tmdbId: 1399, mediaType: 'tv' });

    const [url, options] = fetchMock.mock.calls[0];
    expect(url).toBe('/api/v1/requests');
    expect(options.method).toBe('POST');
    expect(JSON.parse(options.body)).toEqual({ tmdb_id: 1399, media_type: 'tv' });
    expect(created.status).toBe('pending');
  });

  it('errors preserve the Rule-7 code so callers can branch (REQUEST_DUPLICATE)', async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 409,
      json: () =>
        Promise.resolve({
          success: false,
          error: { code: 'REQUEST_DUPLICATE', message: '已有進行中的請求' },
        }),
    });

    const err = await requestService
      .createRequest({ tmdbId: 550, mediaType: 'movie' })
      .catch((e) => e);

    expect(err).toBeInstanceOf(RequestApiError);
    expect(err.code).toBe('REQUEST_DUPLICATE');
    expect(err.message).toBe('已有進行中的請求');
  });

  it('non-JSON failures fall back to a generic error with INTERNAL_ERROR code', async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 502,
      json: () => Promise.reject(new Error('not json')),
    });

    const err = await requestService.listRequests().catch((e) => e);
    expect(err).toBeInstanceOf(RequestApiError);
    expect(err.code).toBe('INTERNAL_ERROR');
  });
});
