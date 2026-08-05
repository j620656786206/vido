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

// ─── Story sub-1-7b — the 5 pipeline states (sub-1-2 [@contract-v1] 9-value set) ───

describe('deriveSubtitleStatus — subtitle-pipeline states (sub-1-7b)', () => {
  it.each([
    ['probing', 'transient — the Activity hub + SSE own in-flight progress'],
    ['extracting', 'transient'],
    ['translating', 'transient'],
  ])('returns null for the transient state %s (%s)', (subtitleStatus) => {
    expect(deriveSubtitleStatus(m('success', { subtitleStatus }))).toBeNull();
  });

  it('maps no_text_source → 無字幕源 (neutral, non-steady)', () => {
    const s = deriveSubtitleStatus(m('success', { subtitleStatus: 'no_text_source' }));
    expect(s).toEqual({ label: '無字幕源', className: expect.stringContaining('--bg-tertiary') });
    expect(s?.steadyState).toBeFalsy();
  });

  it('maps skipped → 已略過 (neutral, non-steady)', () => {
    const s = deriveSubtitleStatus(m('success', { subtitleStatus: 'skipped' }));
    expect(s).toEqual({ label: '已略過', className: expect.stringContaining('--bg-tertiary') });
    expect(s?.steadyState).toBeFalsy();
  });

  it('keeps 無字幕源 distinct from 缺字幕 — different recoveries (sub-1-7a AC #4)', () => {
    // 缺字幕 = searched online, found nothing → re-search may help.
    // 無字幕源 = no text track to extract → only P2 ASR can help.
    expect(deriveSubtitleStatus(m('success', { subtitleStatus: 'not_found' }))?.label).toBe(
      '缺字幕'
    );
    expect(deriveSubtitleStatus(m('success', { subtitleStatus: 'no_text_source' }))?.label).toBe(
      '無字幕源'
    );
  });
});

// THE ordering regression tests. These fail against the natural "append the new
// cases at the bottom" implementation: step 2 infers a badge from subtitleTracks
// and returns on ANY track, so a terminal engine verdict placed below it loses to
// a naive track count and the user sees 有字幕 on a file that will never get one.
describe('deriveSubtitleStatus — terminal verdicts outrank track inference (sub-1-7b)', () => {
  // PGS is an IMAGE track: the pipeline cannot extract text from it, which is
  // exactly why the item is no_text_source — yet it IS a track in subtitleTracks.
  const imageOnlyTracks = JSON.stringify([{ language: 'eng', codec: 'hdmv_pgs_subtitle' }]);
  // An `und`-tagged text track is what P0 refuses to treat as English → skipped.
  const undTracks = JSON.stringify([{ language: 'und' }]);

  it('no_text_source wins over an image-only embedded track', () => {
    expect(
      deriveSubtitleStatus(
        m('success', { subtitleStatus: 'no_text_source', subtitleTracks: imageOnlyTracks })
      )?.label
    ).toBe('無字幕源');
  });

  it('skipped wins over an und-tagged embedded track', () => {
    expect(
      deriveSubtitleStatus(m('success', { subtitleStatus: 'skipped', subtitleTracks: undTracks }))
        ?.label
    ).toBe('已略過');
  });

  it('the transient states reach null without falling into track inference', () => {
    for (const subtitleStatus of ['probing', 'extracting', 'translating']) {
      expect(
        deriveSubtitleStatus(m('success', { subtitleStatus, subtitleTracks: undTracks }))
      ).toBeNull();
    }
  });
});

describe('pickPosterBadge — the new pipeline states (sub-1-7b AC #2)', () => {
  it('surfaces the terminal verdicts on the grid (they are exceptions)', () => {
    expect(pickPosterBadge(m('success', { subtitleStatus: 'no_text_source' }))?.label).toBe(
      '無字幕源'
    );
    expect(pickPosterBadge(m('success', { subtitleStatus: 'skipped' }))?.label).toBe('已略過');
  });

  it('renders NO badge for the three transient states (normal progress is not an exception)', () => {
    for (const subtitleStatus of ['probing', 'extracting', 'translating']) {
      expect(pickPosterBadge(m('success', { subtitleStatus }))).toBeNull();
    }
  });

  it('a lifecycle exception still wins over a terminal subtitle verdict', () => {
    expect(pickPosterBadge(m('pending', { subtitleStatus: 'skipped' }))?.label).toBe('整理中');
  });
});

// sub-2-2b: the untranslated badge + the found-fallback ladder hole.
// Murat's mandate — the "authoritative verdict vs EMPTY track list" class had
// ZERO coverage; step 1's `deriveFromTracks(media) ?? 有字幕` fallback let
// deriveFromTracks' 缺字幕-on-empty-array override an engine verdict.
describe('deriveSubtitleStatus — authoritative verdicts vs an EMPTY track list (sub-2-2b)', () => {
  const emptyTracks = JSON.stringify([]);

  it('found + en + empty tracks → 有字幕, never 缺字幕 (the ladder hole)', () => {
    const s = deriveSubtitleStatus(
      m('success', { subtitleStatus: 'found', subtitleLanguage: 'en', subtitleTracks: emptyTracks })
    );
    expect(s?.label).toBe('有字幕');
  });

  it('untranslated + empty tracks → 未翻譯', () => {
    expect(
      deriveSubtitleStatus(
        m('success', { subtitleStatus: 'untranslated', subtitleTracks: emptyTracks })
      )?.label
    ).toBe('未翻譯');
  });

  it('no_text_source + empty tracks → 無字幕源', () => {
    expect(
      deriveSubtitleStatus(
        m('success', { subtitleStatus: 'no_text_source', subtitleTracks: emptyTracks })
      )?.label
    ).toBe('無字幕源');
  });

  it('skipped + empty tracks → 已略過', () => {
    expect(
      deriveSubtitleStatus(m('success', { subtitleStatus: 'skipped', subtitleTracks: emptyTracks }))
        ?.label
    ).toBe('已略過');
  });

  it('found + zh track present still refines to 繁中 (track refinement survives the fix)', () => {
    const s = deriveSubtitleStatus(
      m('success', {
        subtitleStatus: 'found',
        subtitleLanguage: 'en',
        subtitleTracks: JSON.stringify([{ language: 'zh-Hant' }]),
      })
    );
    expect(s?.label).toBe('繁中');
  });
});

describe('deriveSubtitleStatus — 未翻譯 (sub-2-2b AC #2)', () => {
  it('renders neutral tint — not an error, and accent stays reserved for in-progress', () => {
    const s = deriveSubtitleStatus(m('success', { subtitleStatus: 'untranslated' }));
    expect(s?.label).toBe('未翻譯');
    expect(s?.className).toContain('--bg-tertiary');
    expect(s?.className).not.toContain('--error');
    expect(s?.className).not.toContain('--accent');
  });

  it('wins over an embedded English track (the verdict already accounts for it)', () => {
    expect(
      deriveSubtitleStatus(
        m('success', {
          subtitleStatus: 'untranslated',
          subtitleTracks: JSON.stringify([{ language: 'en' }]),
        })
      )?.label
    ).toBe('未翻譯');
  });

  it('SCOPE FENCE: an embedded English track WITHOUT the pipeline verdict never badges 未翻譯', () => {
    // A foreign film with an embedded en track was never owed a translation —
    // untranslated derives from subtitleStatus exclusively (Sally fence).
    for (const subtitleStatus of [undefined, 'not_searched']) {
      const s = deriveSubtitleStatus(
        m('success', { subtitleStatus, subtitleTracks: JSON.stringify([{ language: 'en' }]) })
      );
      expect(s?.label).toBe('有字幕');
    }
  });

  it('is an exception (non-steady): pickPosterBadge surfaces it on the grid', () => {
    const badge = pickPosterBadge(m('success', { subtitleStatus: 'untranslated' }));
    expect(badge?.label).toBe('未翻譯');
  });
});
