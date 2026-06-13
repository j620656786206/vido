import { describe, it, expect } from 'vitest';
import { deriveLifecycleStatus, deriveSubtitleStatus } from './libraryStatus';

type Media = { parseStatus: string; subtitleTracks?: string };
const m = (parseStatus: string, subtitleTracks?: string): Media => ({
  parseStatus,
  subtitleTracks,
});

describe('deriveLifecycleStatus', () => {
  it('maps success → 已入庫 (success tint)', () => {
    const s = deriveLifecycleStatus(m('success'));
    expect(s?.label).toBe('已入庫');
    expect(s?.className).toContain('--success-tint');
  });
  it('maps pending → 整理中 (warning) and failed → 失敗 (error)', () => {
    expect(deriveLifecycleStatus(m('pending'))?.label).toBe('整理中');
    expect(deriveLifecycleStatus(m('failed'))?.label).toBe('失敗');
    expect(deriveLifecycleStatus(m('failed'))?.className).toContain('--error-tint');
  });
  it('returns null for unknown status or missing media (F3)', () => {
    expect(deriveLifecycleStatus(m('weird'))).toBeNull();
    expect(deriveLifecycleStatus(undefined)).toBeNull();
  });
});

describe('deriveSubtitleStatus', () => {
  it('flags 繁中 when a zh-Hant track is present', () => {
    const s = deriveSubtitleStatus(m('success', JSON.stringify([{ language: 'zh-Hant' }])));
    expect(s?.label).toBe('繁中');
    expect(s?.className).toContain('--success-tint');
  });
  it('flags 簡中 when only zh-Hans is present', () => {
    expect(deriveSubtitleStatus(m('success', JSON.stringify([{ lang: 'zh-Hans' }])))?.label).toBe(
      '簡中'
    );
  });
  it('flags 缺字幕 for an empty track list', () => {
    expect(deriveSubtitleStatus(m('success', JSON.stringify([])))?.label).toBe('缺字幕');
  });
  it('flags 有字幕 for non-zh tracks only', () => {
    expect(deriveSubtitleStatus(m('success', JSON.stringify([{ language: 'en' }])))?.label).toBe(
      '有字幕'
    );
  });
  it('returns null when subtitleTracks is absent (unknown, not known-missing)', () => {
    expect(deriveSubtitleStatus(m('success', undefined))).toBeNull();
  });
  it('returns null on non-JSON legacy values (never throws)', () => {
    expect(deriveSubtitleStatus(m('success', 'srt'))).toBeNull();
  });
});
