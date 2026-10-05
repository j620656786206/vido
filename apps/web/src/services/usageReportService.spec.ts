import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { usageReportService, UsageReportApiError } from './usageReportService';

const fetchMock = vi.fn();

beforeEach(() => {
  fetchMock.mockReset();
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

const PAYLOAD =
  '{"type":"event","payload":{"website":"11111111-2222-3333-4444-555555555555","hostname":"vido","url":"/usage-report","name":"weekly_usage","id":"3f2a6c1e-8f0b-4d5e-9a7c-1b2c3d4e5f60","ip":"127.0.0.1","data":{"version":"0.1.2","subtitles_auto_7d":7,"subtitles_embedded_7d":4,"subtitles_online_7d":2,"subtitles_asr_7d":1}}}';

function ok(data: unknown) {
  return { ok: true, status: 200, json: async () => ({ success: true, data }) };
}

describe('usageReportService (a2 wire shape)', () => {
  it('GET maps snake_case keys and leaves the payload string untouched', async () => {
    fetchMock.mockResolvedValue(
      ok({
        available: true,
        enabled: true,
        last_sent_at: '2026-10-04T12:00:00Z',
        last_payload: PAYLOAD,
      })
    );

    const st = await usageReportService.get();

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/settings/usage-report', undefined);
    expect(st).toEqual({
      available: true,
      enabled: true,
      lastSentAt: '2026-10-04T12:00:00Z',
      lastPayload: PAYLOAD,
    });
  });

  it('GET never-sent keeps nulls', async () => {
    fetchMock.mockResolvedValue(
      ok({ available: false, enabled: false, last_sent_at: null, last_payload: null })
    );

    const st = await usageReportService.get();

    expect(st.lastSentAt).toBeNull();
    expect(st.lastPayload).toBeNull();
  });

  it('PUT sends {enabled} and returns the fresh state', async () => {
    fetchMock.mockResolvedValue(
      ok({ available: true, enabled: true, last_sent_at: null, last_payload: null })
    );

    const st = await usageReportService.setEnabled(true);

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe('/api/v1/settings/usage-report');
    expect(init.method).toBe('PUT');
    expect(JSON.parse(init.body)).toEqual({ enabled: true });
    expect(st.enabled).toBe(true);
  });

  it('a backend error becomes UsageReportApiError with its code', async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 400,
      json: async () => ({
        success: false,
        error: { code: 'VALIDATION_ERROR', message: 'enabled is required' },
      }),
    });

    await expect(usageReportService.setEnabled(true)).rejects.toMatchObject({
      name: 'UsageReportApiError',
      code: 'VALIDATION_ERROR',
    });
    await expect(usageReportService.setEnabled(true)).rejects.toBeInstanceOf(UsageReportApiError);
  });
});
