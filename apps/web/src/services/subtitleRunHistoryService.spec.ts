import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { subtitleRunHistoryService } from './subtitleRunHistoryService';

/** The disc-2026-11-spend-history-b wire shape, snake_case, nulls in place. */
const WIRE = {
  month: '2026-11',
  summary: {
    spent_usd: 3.91,
    runs: 12,
    unpriced_runs: 0,
    comparable_runs: 5,
    comparable_spent_usd: 1.2,
    comparable_estimate_usd: 1.35,
  },
  series: [
    {
      date: '2026-11-01',
      spent_usd: 3.91,
      runs: 9,
      by_category: { chinese_track: 0, translate_english: 0.81, asr: 3.1, legacy: 0 },
    },
  ],
  items: [
    {
      id: 'r1',
      started_at: '2026-10-28T13:33:46Z',
      completed_at: '2026-10-28T13:38:25Z',
      status: 'completed',
      failure_reason: '',
      media: { id: 'm1', type: 'episode', title: 'Lioness', season: 1, episode: 1 },
      route: 'asr',
      source: 'audio',
      category: 'asr',
      spent_usd: 0.52,
      estimate_usd: null,
      translation_model: 'claude-sonnet-5',
      asr_provider: null,
      asr_model: null,
      batch_id: null,
      triggered_by: 'manual',
    },
  ],
  page: 1,
  page_size: 50,
  total: 12,
};

describe('subtitleRunHistoryService (GET /subtitles/runs — spend-history-b [@contract-v1])', () => {
  const fetchMock = vi.fn();
  beforeEach(() => {
    fetchMock.mockReset();
    vi.stubGlobal('fetch', fetchMock);
  });
  afterEach(() => vi.unstubAllGlobals());

  it('sends month / category / page / page_size and camelCases the body, nulls kept', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ success: true, data: WIRE }),
    });
    const res = await subtitleRunHistoryService.getRuns({
      month: '2026-11',
      category: 'asr',
      page: 2,
      pageSize: 50,
    });
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/subtitles/runs?month=2026-11&category=asr&page=2&page_size=50',
      expect.anything()
    );
    expect(res.summary.comparableEstimateUsd).toBe(1.35);
    expect(res.items[0].estimateUsd).toBeNull();
    expect(res.items[0].asrProvider).toBeNull();
    expect(res.items[0].startedAt).toBe('2026-10-28T13:33:46Z');
    expect(res.pageSize).toBe(50);
    // by_category's keys are category values — they stay in the enum's spelling.
    expect(res.series[0].byCategory).toEqual({
      chinese_track: 0,
      translate_english: 0.81,
      asr: 3.1,
      legacy: 0,
    });
  });

  it('omits every unset param (this month, all categories)', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ success: true, data: WIRE }),
    });
    await subtitleRunHistoryService.getRuns();
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/subtitles/runs', expect.anything());
  });

  it('throws the backend message on an error envelope', async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 400,
      json: async () => ({
        success: false,
        error: { code: 'VALIDATION_INVALID_FORMAT', message: '月份格式錯誤' },
      }),
    });
    await expect(subtitleRunHistoryService.getRuns({ month: 'bad' })).rejects.toThrow(
      '月份格式錯誤'
    );
  });
});
