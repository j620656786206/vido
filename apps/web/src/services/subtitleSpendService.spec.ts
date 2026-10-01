import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { subtitleSpendService } from './subtitleSpendService';

/** The sub-7-6b wire shape, snake_case, with the honesty nulls in place. */
const WIRE = {
  period: 'month',
  from: '2026-10-01T00:00:00+08:00',
  to: '2026-11-01T00:00:00+08:00',
  translated_runs: 12,
  translated_usd: 3.48,
  asr_runs: 2,
  asr_usd: 1.9,
  unpriced_runs: 1,
  unrouted_runs: 0,
  unrouted_usd: 0,
  skipped_deliver_count: 8,
  skipped_saved_usd_estimate: 2.14,
  skipped_saved_runtime_assumed: true,
  cache_hit_cues: 0,
  cache_measured_runs: 0,
  cache_saved_usd_estimate: null,
  by_model: [{ model_id: 'claude-sonnet-5', runs: 9, usd: 4.2 }],
  by_batch: {
    batch_id: 'gb-1',
    runs: 5,
    completed_runs: 5,
    usd: 0.53,
    cue_count: 844,
    cache_hit_cues: 101,
    cache_measured_runs: 5,
    model_id: 'claude-sonnet-5',
  },
};

describe('subtitleSpendService (GET /subtitles/spend — sub-7-6b [@contract-v1])', () => {
  const fetchMock = vi.fn();
  beforeEach(() => {
    fetchMock.mockReset();
    vi.stubGlobal('fetch', fetchMock);
  });
  afterEach(() => vi.unstubAllGlobals());

  it('asks for period=month, camelCases the body and keeps the nulls null', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ success: true, data: WIRE }),
    });
    const out = await subtitleSpendService.getMonthSummary();
    expect(fetchMock.mock.calls[0][0]).toMatch(/\/subtitles\/spend\?period=month$/);
    expect(out.translatedUsd).toBe(3.48);
    expect(out.skippedSavedRuntimeAssumed).toBe(true);
    // null stays null — "nobody measured it" must not become 0 on the way in.
    expect(out.cacheSavedUsdEstimate).toBeNull();
    expect(out.byModel[0]).toEqual({ modelId: 'claude-sonnet-5', runs: 9, usd: 4.2 });
    expect(out.byBatch?.cueCount).toBe(844);
    expect(out.byBatch?.cacheHitCues).toBe(101);
  });

  it('adds batch_id when a batch receipt is wanted', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ success: true, data: WIRE }),
    });
    await subtitleSpendService.getMonthSummary('gb-1');
    expect(fetchMock.mock.calls[0][0]).toMatch(/\/subtitles\/spend\?period=month&batch_id=gb-1$/);
  });

  it('surfaces the API error message', async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 500,
      json: async () => ({
        success: false,
        error: { code: 'INTERNAL', message: '無法讀取字幕花費紀錄，請稍後再試。' },
      }),
    });
    await expect(subtitleSpendService.getMonthSummary()).rejects.toThrow(
      '無法讀取字幕花費紀錄，請稍後再試。'
    );
  });
});
