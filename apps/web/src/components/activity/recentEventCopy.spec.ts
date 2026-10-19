import { describe, it, expect } from 'vitest';
import type { RecentEvent } from '../../services/activityService';
import {
  MISSING_TITLE,
  formatRunDuration,
  formatRunSpend,
  recentSubtitleDetail,
  recentSubtitleMeta,
  recentSubtitleMetaLead,
  recentSubtitleName,
} from './recentEventCopy';

function run(over: Partial<RecentEvent> = {}): RecentEvent {
  return {
    kind: 'subtitle',
    result: 'completed',
    at: '2026-10-19T01:25:01Z',
    mediaType: 'episode',
    mediaTitle: '末日光明',
    seasonNumber: 1,
    episodeNumber: 2,
    route: 'translate',
    cueCount: 568,
    ...over,
  };
}

describe('formatRunDuration (AC #4 花多久)', () => {
  it.each([
    [0, '不到 1 秒'],
    [0.4, '不到 1 秒'],
    [4, '4 秒'],
    [59, '59 秒'],
    [60, '1 分鐘'],
    [89, '1 分鐘'],
    [90, '2 分鐘'],
    [180, '3 分鐘'],
    [211, '4 分鐘'],
    [3570, '1 小時'],
    [3900, '1 小時 5 分'],
  ])('%s s → %s', (s, want) => {
    expect(formatRunDuration(s)).toBe(want);
  });

  it('no field (or a nonsense one) → no time part', () => {
    expect(formatRunDuration(undefined)).toBeNull();
    expect(formatRunDuration(null)).toBeNull();
    expect(formatRunDuration(-3)).toBeNull();
    expect(formatRunDuration(Number.NaN)).toBeNull();
  });

  it('59.6 s never reads as 「60 秒」', () => {
    expect(formatRunDuration(59.6)).toBe('59 秒');
  });
});

describe('formatRunSpend (AC #4 花多少錢)', () => {
  it.each([
    [0, '沒花錢'],
    [0.004, '不到 US$0.01'],
    [0.005, 'US$0.01'],
    [0.197439, 'US$0.20'],
    [0.27, 'US$0.27'],
    [1.36365413125, 'US$1.36'],
  ])('%s → %s', (v, want) => {
    expect(formatRunSpend(v)).toBe(want);
  });

  it('no field → no money part (not recorded ≠ $0)', () => {
    expect(formatRunSpend(undefined)).toBeNull();
    expect(formatRunSpend(null)).toBeNull();
  });

  it('a negative amount says nothing rather than a refund', () => {
    expect(formatRunSpend(-0.01)).toBeNull();
  });
});

describe('recentSubtitleName / recentSubtitleDetail (AC #3)', () => {
  it('episode → 「片名 S01E02」, zero-padded', () => {
    expect(recentSubtitleName(run())).toBe('末日光明 S01E02');
    expect(recentSubtitleName(run({ seasonNumber: 12, episodeNumber: 105 }))).toBe(
      '末日光明 S12E105'
    );
  });

  it('movie and whole series carry no episode tag', () => {
    expect(recentSubtitleName(run({ mediaType: 'movie', mediaTitle: '沙丘' }))).toBe('沙丘');
    expect(recentSubtitleName(run({ mediaType: 'series' }))).toBe('末日光明');
  });

  it('an episode the library no longer finds has no season/episode → title only', () => {
    expect(recentSubtitleName(run({ seasonNumber: undefined, episodeNumber: undefined }))).toBe(
      '末日光明'
    );
  });

  it('empty title → 已不在媒體庫的項目 (⚖️ Sally 追認 3)', () => {
    expect(
      recentSubtitleName(
        run({
          mediaType: 'movie',
          mediaTitle: '',
          seasonNumber: undefined,
          episodeNumber: undefined,
        })
      )
    ).toBe(MISSING_TITLE);
    expect(MISSING_TITLE).toBe('已不在媒體庫的項目');
  });

  it.each([
    ['translate', '末日光明 S01E02 · 翻譯 568 句'],
    ['asr', '末日光明 S01E02 · 聽寫＋翻譯 568 句'],
    ['deliver_direct', '末日光明 S01E02 · 片內中文字幕 568 句'],
    ['convert_then_deliver', '末日光明 S01E02 · 簡體轉繁體 568 句'],
    ['online', '末日光明 S01E02 · 線上字幕 568 句'],
    ['', '末日光明 S01E02 · 568 句'],
    ['something_new', '末日光明 S01E02 · 568 句'],
    [undefined, '末日光明 S01E02 · 568 句'],
  ])('route %s → %s', (route, want) => {
    expect(recentSubtitleDetail(run({ route }))).toBe(want);
  });

  it('no cueCount → title only', () => {
    expect(recentSubtitleDetail(run({ cueCount: undefined }))).toBe('末日光明 S01E02');
    expect(recentSubtitleDetail(run({ cueCount: 0 }))).toBe('末日光明 S01E02');
  });

  it.each([
    ['cancelled', '中途停止'],
    ['budget_exceeded', '已達預算上限'],
    ['credit_exhausted', 'AI 帳號沒有額度'],
    ['target_not_writable', '影片所在的資料夾寫不進字幕'],
    ['no_audio_track', '找不到可用的音軌'],
    ['ffmpeg_missing', '伺服器缺少 ffmpeg'],
    ['asr_unconfigured', '語音辨識未設定'],
    ['unauthorized', 'API 金鑰無效'],
    ['timeout', '處理逾時'],
    ['other', '生成失敗'],
    ['brand_new_code', '生成失敗'],
    [undefined, '生成失敗'],
    ['toString', '生成失敗'],
  ])('failed %s → 「片名 · %s」 (AC #6)', (failureReason, text) => {
    expect(
      recentSubtitleDetail(
        run({
          result: 'failed',
          episodeNumber: 1,
          cueCount: undefined,
          route: 'translate',
          failureReason,
        })
      )
    ).toBe(`末日光明 S01E01 · ${text}`);
  });

  it('a failed run never claims 這次沒有花到錢 (it may have paid before failing to write)', () => {
    expect(
      recentSubtitleDetail(run({ result: 'failed', failureReason: 'target_not_writable' }))
    ).not.toContain('沒有花到錢');
  });
});

describe('recentSubtitleMeta (AC #4)', () => {
  it('both parts joined by 「 · 」', () => {
    expect(recentSubtitleMeta(run({ durationSeconds: 180, spentUsd: 0.27 }))).toBe(
      '3 分鐘 · US$0.27'
    );
    expect(recentSubtitleMeta(run({ durationSeconds: 4, spentUsd: 0 }))).toBe('4 秒 · 沒花錢');
  });

  it('a missing part is dropped; both missing → null', () => {
    expect(recentSubtitleMeta(run({ durationSeconds: 211 }))).toBe('4 分鐘');
    expect(recentSubtitleMeta(run({ spentUsd: 0.197439 }))).toBe('US$0.20');
    expect(recentSubtitleMeta(run())).toBeNull();
  });
});

describe('recentSubtitleMetaLead (sr-only 花了)', () => {
  it.each<[string, Partial<RecentEvent>, string | null, string | null]>([
    ['time + money', { durationSeconds: 180, spentUsd: 0.27 }, '3 分鐘 · US$0.27', '花了'],
    ['time + no spend', { durationSeconds: 4, spentUsd: 0 }, '4 秒 · 沒花錢', '花了'],
    ['time only (old row, spend NULL)', { durationSeconds: 180 }, '3 分鐘', '花了'],
    ['money only (clock went back)', { spentUsd: 0.27 }, 'US$0.27', '花了'],
    ['sub-cent money only', { spentUsd: 0.004 }, '不到 US$0.01', '花了'],
    ['沒花錢 only — nothing to follow 花了', { spentUsd: 0 }, '沒花錢', null],
    ['neither part — no meta at all', {}, null, null],
  ])('%s', (_label, over, meta, lead) => {
    const ev = run(over);
    expect(recentSubtitleMeta(ev)).toBe(meta);
    expect(recentSubtitleMetaLead(ev)).toBe(lead);
  });

  it('a value the formatter drops counts as missing (negative spend, bad duration)', () => {
    expect(recentSubtitleMetaLead(run({ spentUsd: -0.27 }))).toBeNull();
    expect(recentSubtitleMetaLead(run({ durationSeconds: Number.NaN, spentUsd: 0 }))).toBeNull();
    expect(recentSubtitleMetaLead(run({ durationSeconds: -5, spentUsd: 0.27 }))).toBe('花了');
  });

  // Adversarial review: the two are computed apart, so pin that 「花了」 is
  // never read with nothing after it, and the meta never starts with 沒花錢 under it.
  it('「花了」 only ever leads a meta that starts with an amount', () => {
    const durations = [undefined, Number.NaN, -5, 0, 4, 180, 3600];
    const spends = [undefined, Number.NaN, -0.27, 0, 0.004, 0.27];
    for (const durationSeconds of durations) {
      for (const spentUsd of spends) {
        const ev = run({ durationSeconds, spentUsd });
        const meta = recentSubtitleMeta(ev);
        if (recentSubtitleMetaLead(ev) === null) {
          expect(meta === null || meta === '沒花錢').toBe(true);
        } else {
          expect(meta).not.toBeNull();
          expect(meta?.startsWith('沒花錢')).toBe(false);
        }
      }
    }
  });
});

// Adversarial review (2026-10-19): wire data the contract does not promise.
describe('recentEventCopy — hostile wire data', () => {
  it.each(['constructor', '__proto__', 'toString', 'hasOwnProperty'])(
    'route %s never prints an Object.prototype member',
    (route) => {
      expect(recentSubtitleDetail(run({ route }))).toBe('末日光明 S01E02 · 568 句');
    }
  );

  it('a non-integer cue count is no count', () => {
    expect(recentSubtitleDetail(run({ cueCount: Number.NaN }))).toBe('末日光明 S01E02');
    expect(recentSubtitleDetail(run({ cueCount: 1.5 }))).toBe('末日光明 S01E02');
    expect(recentSubtitleDetail(run({ cueCount: -3 }))).toBe('末日光明 S01E02');
  });

  it('a nonsense season/episode drops the tag', () => {
    expect(recentSubtitleName(run({ seasonNumber: -1 }))).toBe('末日光明');
    expect(recentSubtitleName(run({ episodeNumber: 2.5 }))).toBe('末日光明');
  });

  it('the title is trimmed; a blank or missing title names no episode', () => {
    expect(recentSubtitleName(run({ mediaTitle: '  末日光明 ' }))).toBe('末日光明 S01E02');
    expect(recentSubtitleName(run({ mediaTitle: '   ' }))).toBe(MISSING_TITLE);
    expect(recentSubtitleName(run({ mediaTitle: undefined }))).toBe(MISSING_TITLE);
  });

  it('a failed run that paid before failing shows what it paid (和平使者 S01E01)', () => {
    const ev = run({
      result: 'failed',
      mediaTitle: '和平使者',
      episodeNumber: 1,
      cueCount: undefined,
      failureReason: 'target_not_writable',
      durationSeconds: 251,
      spentUsd: 0.196393,
    });
    expect(recentSubtitleDetail(ev)).toBe('和平使者 S01E01 · 影片所在的資料夾寫不進字幕');
    expect(recentSubtitleMeta(ev)).toBe('4 分鐘 · US$0.20');
  });
});
