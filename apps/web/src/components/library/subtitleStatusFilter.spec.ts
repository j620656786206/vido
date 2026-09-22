import { describe, it, expect } from 'vitest';
import {
  KNOWN_SUBTITLE_STATUSES,
  SUBTITLE_STATUS_FILTER_OPTIONS,
  joinSubtitleStatusCsv,
  parseSubtitleStatusCsv,
  subtitleStatusLabel,
} from './subtitleStatusFilter';

describe('subtitleStatusFilter (dsr-1b-b)', () => {
  it('[P0] parse drops unknown values so a hand-edited deep link cannot 400 the list', () => {
    expect(parseSubtitleStatusCsv('not_found,bogus,NOT_FOUND')).toEqual(['not_found']);
  });

  it('[P0] parse keeps every backend-legal value, not just the three chips', () => {
    expect(parseSubtitleStatusCsv('skipped,untranslated')).toEqual(['skipped', 'untranslated']);
    expect(KNOWN_SUBTITLE_STATUSES).toHaveLength(10);
    for (const o of SUBTITLE_STATUS_FILTER_OPTIONS) {
      expect(KNOWN_SUBTITLE_STATUSES).toContain(o.value);
    }
  });

  it('[P0] parse collapses duplicates and trims, preserving first-seen order', () => {
    expect(parseSubtitleStatusCsv('found, found ,not_found,found')).toEqual(['found', 'not_found']);
  });

  it('[P0] parse of empty/absent is [] and join of empty is undefined (param leaves the URL)', () => {
    expect(parseSubtitleStatusCsv(undefined)).toEqual([]);
    expect(parseSubtitleStatusCsv('')).toEqual([]);
    expect(joinSubtitleStatusCsv([])).toBeUndefined();
    expect(joinSubtitleStatusCsv(undefined)).toBeUndefined();
    expect(joinSubtitleStatusCsv(['a', 'b'])).toBe('a,b');
  });

  it('[P1] labels come from the table; an unknown value echoes itself', () => {
    expect(subtitleStatusLabel('not_found')).toBe('缺字幕');
    expect(subtitleStatusLabel('skipped')).toBe('skipped');
  });
});
