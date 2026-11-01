import { describe, it, expect } from 'vitest';
import {
  estimateAccuracyPercent,
  isMonth,
  mediaTitle,
  monthLabel,
  overEstimatePercent,
  routeLabel,
  runModels,
  runStatus,
  shiftMonth,
} from './spendHistoryCopy';

describe('routeLabel — the -c design table', () => {
  it.each([
    [
      { route: 'deliver_direct', source: 'embedded_text', category: 'chinese_track' },
      '直接用片內繁中',
    ],
    [{ route: 'convert_then_deliver', source: null, category: 'chinese_track' }, '片內簡中轉繁體'],
    [
      { route: 'translate', source: 'embedded_text', category: 'translate_english' },
      '翻譯片內英文字幕',
    ],
    [{ route: 'translate', source: 'sidecar', category: 'translate_english' }, '翻譯片旁英文字幕'],
    [{ route: 'translate', source: null, category: 'translate_english' }, '翻譯英文字幕'],
    [{ route: 'asr', source: 'audio', category: 'asr' }, '語音辨識＋翻譯'],
    [{ route: null, source: null, category: 'legacy' }, '舊紀錄（沒分類）'],
    [{ route: 'no_text_source', source: null, category: 'no_subtitle' }, '沒有可用字幕'],
    [{ route: 'skip', source: null, category: 'no_subtitle' }, '沒有可用字幕'],
  ] as const)('%o → %s', (item, label) => {
    expect(routeLabel(item)).toBe(label);
  });
});

describe('runStatus', () => {
  it('completed / skipped / cancelled / failed with its reason', () => {
    expect(runStatus({ status: 'completed', failureReason: '' }).label).toBe('完成');
    expect(runStatus({ status: 'skipped', failureReason: '' }).label).toBe('略過');
    expect(runStatus({ status: 'failed', failureReason: 'cancelled' }).label).toBe('中途停止');
    const failed = runStatus({ status: 'failed', failureReason: 'timeout' });
    expect(failed.label).toBe('失敗');
    expect(failed.detail).toBe('處理逾時');
  });
});

describe('overEstimatePercent — ⚠ only when MORE than 1.5×', () => {
  it('over 50% → the overshoot in whole percent', () => {
    expect(overEstimatePercent({ spentUsd: 0.8, estimateUsd: 0.5 })).toBe(60);
    expect(overEstimatePercent({ spentUsd: 0.31, estimateUsd: 0.2 })).toBe(55);
  });
  it('exactly 1.5× (decimal, not float) or less → nothing', () => {
    expect(overEstimatePercent({ spentUsd: 0.75, estimateUsd: 0.5 })).toBeUndefined();
    expect(overEstimatePercent({ spentUsd: 0.3, estimateUsd: 0.2 })).toBeUndefined();
    expect(overEstimatePercent({ spentUsd: 0.4, estimateUsd: 0.5 })).toBeUndefined();
  });
  it('either side unrecorded, or a $0 estimate → nothing', () => {
    expect(overEstimatePercent({ spentUsd: null, estimateUsd: 0.5 })).toBeUndefined();
    expect(overEstimatePercent({ spentUsd: 0.5, estimateUsd: null })).toBeUndefined();
    expect(overEstimatePercent({ spentUsd: 0.5, estimateUsd: 0 })).toBeUndefined();
  });
});

describe('estimateAccuracyPercent', () => {
  it('comparable spent / estimate; undefined when nothing to compare', () => {
    const base = { spentUsd: 1, runs: 1, unpricedRuns: 0 };
    expect(
      estimateAccuracyPercent({
        ...base,
        comparableRuns: 5,
        comparableSpentUsd: 1.2,
        comparableEstimateUsd: 1.35,
      })
    ).toBe(89);
    expect(
      estimateAccuracyPercent({
        ...base,
        comparableRuns: 0,
        comparableSpentUsd: null,
        comparableEstimateUsd: null,
      })
    ).toBeUndefined();
  });
});

describe('runModels / mediaTitle', () => {
  it('translation model, then 「ElevenLabs scribe_v2」; unknown engine keeps its id', () => {
    expect(
      runModels({
        translationModel: 'claude-sonnet-5',
        asrProvider: 'elevenlabs',
        asrModel: 'scribe_v2',
      })
    ).toEqual(['claude-sonnet-5', 'ElevenLabs scribe_v2']);
    expect(runModels({ translationModel: null, asrProvider: 'acme', asrModel: 'x1' })).toEqual([
      'acme x1',
    ]);
    expect(runModels({ translationModel: null, asrProvider: null, asrModel: null })).toEqual([]);
  });
  it('episode gets SxEy; deleted media is null', () => {
    expect(
      mediaTitle({ media: { id: '1', type: 'episode', title: '母獅', season: 2, episode: 1 } })
    ).toBe('母獅 S2E1');
    expect(
      mediaTitle({ media: { id: '1', type: 'movie', title: '沙丘', season: null, episode: null } })
    ).toBe('沙丘');
    expect(
      mediaTitle({ media: { id: '1', type: 'movie', title: null, season: null, episode: null } })
    ).toBeNull();
  });
});

describe('months', () => {
  it('shiftMonth crosses years both ways', () => {
    expect(shiftMonth('2026-01', -1)).toBe('2025-12');
    expect(shiftMonth('2025-12', 1)).toBe('2026-01');
    expect(shiftMonth('2026-11', -2)).toBe('2026-09');
  });
  it('isMonth / monthLabel', () => {
    expect(isMonth('2026-10')).toBe(true);
    expect(isMonth('2026-13')).toBe(false);
    expect(isMonth(202610)).toBe(false);
    expect(monthLabel('2026-09')).toBe('2026 年 9 月');
  });
});
