import { describe, it, expect } from 'vitest';
import {
  CHINESE_SUBTITLE_FILTER_OPTIONS,
  chineseSubtitleChipLabel,
  joinChineseSubtitleCsv,
  parseChineseSubtitleCsv,
} from './chineseSubtitleFilter';

describe('chineseSubtitleFilter (disc-2026-10-subtitle-filter-disagrees-with-badges AC #8)', () => {
  it('[P0] D1: exactly three chips, in order, with the ruled wording', () => {
    expect(CHINESE_SUBTITLE_FILTER_OPTIONS.map((o) => [o.value, o.label])).toEqual([
      ['has', '有中文字幕'],
      ['missing', '缺中文字幕'],
      ['unknown', '不知道'],
    ]);
  });

  it('[P0] parse drops unknown values so a hand-edited deep link cannot 400 the list', () => {
    expect(parseChineseSubtitleCsv('missing,bogus,MISSING,not_found')).toEqual(['missing']);
  });

  it('[P0] parse collapses duplicates and trims, preserving first-seen order', () => {
    expect(parseChineseSubtitleCsv('unknown, has ,unknown')).toEqual(['unknown', 'has']);
  });

  it('[P0] parse of empty/absent is [] and join of empty is undefined (param leaves the URL)', () => {
    expect(parseChineseSubtitleCsv(undefined)).toEqual([]);
    expect(parseChineseSubtitleCsv('')).toEqual([]);
    expect(joinChineseSubtitleCsv([])).toBeUndefined();
    expect(joinChineseSubtitleCsv(undefined)).toBeUndefined();
    expect(joinChineseSubtitleCsv(['has', 'missing'])).toBe('has,missing');
  });

  it('[P1] stand-alone labels come from the table; 不知道 says what is unknown; unknown values echo', () => {
    expect(chineseSubtitleChipLabel('missing')).toBe('缺中文字幕');
    expect(chineseSubtitleChipLabel('unknown')).toBe('不知道有沒有中文字幕');
    expect(chineseSubtitleChipLabel('weird')).toBe('weird');
  });
});
