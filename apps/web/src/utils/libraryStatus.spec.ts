import { describe, it, expect } from 'vitest';
import { deriveLifecycleStatus, deriveSubtitleStatus, pickPosterBadge } from './libraryStatus';

type Media = {
  parseStatus: string;
  subtitleTracks?: string;
  subtitleStatus?: string;
  subtitleLanguage?: string;
};
const m = (parseStatus: string, over: Partial<Media> = {}): Media => ({ parseStatus, ...over });

describe('deriveLifecycleStatus', () => {
  it('maps success → 已入庫 (success tint, steady)', () => {
    const s = deriveLifecycleStatus(m('success'));
    expect(s?.label).toBe('已入庫');
    expect(s?.className).toContain('--success-tint');
    expect(s?.steadyState).toBe(true);
  });
  it('maps pending → 整理中 (warning) and failed → 失敗 (error), neither steady', () => {
    expect(deriveLifecycleStatus(m('pending'))?.label).toBe('整理中');
    expect(deriveLifecycleStatus(m('pending'))?.steadyState).toBeFalsy();
    expect(deriveLifecycleStatus(m('failed'))?.label).toBe('失敗');
    expect(deriveLifecycleStatus(m('failed'))?.className).toContain('--error-tint');
  });
  it('returns null for unknown status or missing media (F3)', () => {
    expect(deriveLifecycleStatus(m('weird'))).toBeNull();
    expect(deriveLifecycleStatus(undefined)).toBeNull();
  });
});

describe('deriveSubtitleStatus — embedded tracks (fallback)', () => {
  it('flags 繁中 (steady) when a zh-Hant track is present', () => {
    const s = deriveSubtitleStatus(
      m('success', { subtitleTracks: JSON.stringify([{ language: 'zh-Hant' }]) })
    );
    expect(s?.label).toBe('繁中');
    expect(s?.className).toContain('--success-tint');
    expect(s?.steadyState).toBe(true);
  });
  it('flags 簡中 (info tint — Sally gate 2026-07-05, accent reserved for in-progress) when only zh-Hans is present', () => {
    const s = deriveSubtitleStatus(
      m('success', { subtitleTracks: JSON.stringify([{ lang: 'zh-Hans' }]) })
    );
    expect(s?.label).toBe('簡中');
    expect(s?.className).toContain('--info-tint');
    expect(s?.className).not.toContain('--accent-tint');
  });
  it('flags 缺字幕 for an empty track list', () => {
    expect(deriveSubtitleStatus(m('success', { subtitleTracks: JSON.stringify([]) }))?.label).toBe(
      '缺字幕'
    );
  });
  it('flags 有字幕 for non-zh tracks only', () => {
    expect(
      deriveSubtitleStatus(m('success', { subtitleTracks: JSON.stringify([{ language: 'en' }]) }))
        ?.label
    ).toBe('有字幕');
  });
  it('returns null when no subtitle info at all (unknown, not known-missing)', () => {
    expect(deriveSubtitleStatus(m('success'))).toBeNull();
  });
  it('returns null on non-JSON legacy values (never throws)', () => {
    expect(deriveSubtitleStatus(m('success', { subtitleTracks: 'srt' }))).toBeNull();
  });
});

describe('deriveSubtitleStatus — authoritative engine result (ux3-0-1)', () => {
  it('prefers subtitleStatus=found + zh-Hant → 繁中 (steady), no tracks needed', () => {
    const s = deriveSubtitleStatus(
      m('success', { subtitleStatus: 'found', subtitleLanguage: 'zh-Hant' })
    );
    expect(s?.label).toBe('繁中');
    expect(s?.steadyState).toBe(true);
  });
  it('subtitleStatus=found + zh-Hans → 簡中 (info tint per F1-D-v2 pill C8lUe)', () => {
    const s = deriveSubtitleStatus(
      m('success', { subtitleStatus: 'found', subtitleLanguage: 'zh-Hans' })
    );
    expect(s?.label).toBe('簡中');
    expect(s?.className).toContain('--info-tint');
    expect(s?.className).not.toContain('--accent-tint');
  });
  it('subtitleStatus=not_found (no tracks) → 缺字幕', () => {
    expect(deriveSubtitleStatus(m('success', { subtitleStatus: 'not_found' }))?.label).toBe(
      '缺字幕'
    );
  });
  it('falls back to embedded tracks when subtitleStatus is not_searched', () => {
    const s = deriveSubtitleStatus(
      m('success', {
        subtitleStatus: 'not_searched',
        subtitleTracks: JSON.stringify([{ language: 'zh-Hant' }]),
      })
    );
    expect(s?.label).toBe('繁中');
  });
});

describe('pickPosterBadge — exception signal (ux3-0-2)', () => {
  it('suppresses the happy steady state (已入庫 + 繁中) → no badge', () => {
    expect(
      pickPosterBadge(m('success', { subtitleStatus: 'found', subtitleLanguage: 'zh-Hant' }))
    ).toBeNull();
    expect(
      pickPosterBadge(m('success', { subtitleTracks: JSON.stringify([{ language: 'zh-Hant' }]) }))
    ).toBeNull();
  });
  it('shows a subtitle exception for an in-library item (缺字幕 / 簡中)', () => {
    expect(pickPosterBadge(m('success', { subtitleStatus: 'not_found' }))?.label).toBe('缺字幕');
    expect(
      pickPosterBadge(m('success', { subtitleTracks: JSON.stringify([{ lang: 'zh-Hans' }]) }))
        ?.label
    ).toBe('簡中');
  });
  it('a lifecycle exception (整理中 / 失敗) wins over subtitle', () => {
    expect(
      pickPosterBadge(m('pending', { subtitleStatus: 'found', subtitleLanguage: 'zh-Hant' }))?.label
    ).toBe('整理中');
    expect(pickPosterBadge(m('failed'))?.label).toBe('失敗');
  });
  it('shows no badge for unknown state (F3)', () => {
    expect(pickPosterBadge(m('success'))).toBeNull();
    expect(pickPosterBadge(undefined)).toBeNull();
  });
});
