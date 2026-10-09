import { describe, it, expect } from 'vitest';
import {
  deriveLifecycleStatus,
  deriveSubtitleStatus,
  pickPosterBadge,
  subtitleLangLabel,
} from './libraryStatus';
import type { ChineseSubtitle } from '../types/library';

type Media = {
  parseStatus: string;
  subtitleTracks?: string;
  subtitleStatus?: string;
  subtitleLanguage?: string;
  chineseSubtitle?: ChineseSubtitle;
};
const m = (parseStatus: string, over: Partial<Media> = {}): Media => ({ parseStatus, ...over });

describe('deriveLifecycleStatus', () => {
  it('maps success → 已入庫 (success tint, steady)', () => {
    const s = deriveLifecycleStatus(m('success'));
    expect(s?.label).toBe('已入庫');
    expect(s?.className).toContain('--success-tint');
    expect(s?.steadyState).toBe(true);
  });
  // ⚖️ Alexyu 2026-09-11（dsr-11）：整理中改成泥金。赭色的定義是「你要求了，但它沒發生」，
  // 而整理中就是正在發生；泥金才是「正在跑」。七月的「暫態處理中家族」被這條裁定取代。
  it('maps pending → 整理中 (accent) and failed → 失敗 (error), neither steady', () => {
    expect(deriveLifecycleStatus(m('pending'))?.label).toBe('整理中');
    expect(deriveLifecycleStatus(m('pending'))?.className).toContain('--accent-tint');
    expect(deriveLifecycleStatus(m('pending'))?.className).not.toContain('--warning');
    expect(deriveLifecycleStatus(m('pending'))?.steadyState).toBeFalsy();
    expect(deriveLifecycleStatus(m('failed'))?.label).toBe('失敗');
    expect(deriveLifecycleStatus(m('failed'))?.className).toContain('--error-tint');
  });
  it('returns null for unknown status or missing media (F3)', () => {
    expect(deriveLifecycleStatus(m('weird'))).toBeNull();
    expect(deriveLifecycleStatus(undefined)).toBeNull();
  });
});

// disc-2026-10-subtitle-filter-disagrees-with-badges AC #7: the badge reads the
// backend's `chineseSubtitle` verdict — the same rule the library filter uses.
// Each row is an AC #4 scenario as the frontend receives it (the backend verdict
// for that shape + the raw subtitleStatus). `group` is the filter chip the
// backend puts it in; the badge must agree with that chip.
describe('deriveSubtitleStatus — AC #4 scenarios as the frontend receives them', () => {
  const tracks = (...langs: string[]) => JSON.stringify(langs.map((language) => ({ language })));
  it.each<[string, Partial<Media>, string | null, 'has' | 'missing' | 'unknown']>([
    [
      'chi + eng',
      {
        chineseSubtitle: 'zh',
        subtitleStatus: 'not_searched',
        subtitleTracks: tracks('chi', 'eng'),
      },
      '中文',
      'has',
    ],
    [
      'only eng',
      { chineseSubtitle: 'none', subtitleStatus: 'not_searched', subtitleTracks: tracks('eng') },
      '缺中文',
      'missing',
    ],
    [
      '[]',
      { chineseSubtitle: 'none', subtitleStatus: 'not_searched', subtitleTracks: '[]' },
      '缺中文',
      'missing',
    ],
    [
      'NULL + not_searched',
      { chineseSubtitle: 'unknown', subtitleStatus: 'not_searched' },
      null,
      'unknown',
    ],
    [
      'NULL + not_found',
      { chineseSubtitle: 'none', subtitleStatus: 'not_found' },
      '缺中文',
      'missing',
    ],
    [
      'NULL + untranslated + en',
      { chineseSubtitle: 'none', subtitleStatus: 'untranslated', subtitleLanguage: 'en' },
      '未翻譯',
      'missing',
    ],
    [
      'found + zh-Hant',
      { chineseSubtitle: 'zh_hant', subtitleStatus: 'found', subtitleLanguage: 'zh-Hant' },
      '繁中',
      'has',
    ],
    [
      'found + zh',
      { chineseSubtitle: 'zh', subtitleStatus: 'found', subtitleLanguage: 'zh' },
      '中文',
      'has',
    ],
    [
      'NFO chi,eng',
      { chineseSubtitle: 'zh', subtitleStatus: 'not_searched', subtitleTracks: 'chi,eng' },
      '中文',
      'has',
    ],
    [
      'sidecar zh-TW',
      {
        chineseSubtitle: 'zh_hant',
        subtitleStatus: 'not_searched',
        subtitleTracks: tracks('zh-TW'),
      },
      '繁中',
      'has',
    ],
    [
      'sidecar chi.forced',
      {
        chineseSubtitle: 'zh',
        subtitleStatus: 'not_searched',
        subtitleTracks: tracks('chi.forced'),
      },
      '中文',
      'has',
    ],
    [
      'only und',
      { chineseSubtitle: 'unknown', subtitleStatus: 'not_searched', subtitleTracks: tracks('und') },
      null,
      'unknown',
    ],
    [
      'eng + und',
      {
        chineseSubtitle: 'unknown',
        subtitleStatus: 'not_searched',
        subtitleTracks: tracks('eng', 'und'),
      },
      null,
      'unknown',
    ],
    [
      'chi 繁體中文',
      { chineseSubtitle: 'zh_hant', subtitleStatus: 'not_searched', subtitleTracks: tracks('chi') },
      '繁中',
      'has',
    ],
    [
      'chi 简体',
      { chineseSubtitle: 'zh_hans', subtitleStatus: 'not_searched', subtitleTracks: tracks('chi') },
      '簡中',
      'has',
    ],
    [
      'only chi 粵語',
      { chineseSubtitle: 'none', subtitleStatus: 'not_searched', subtitleTracks: tracks('chi') },
      '缺中文',
      'missing',
    ],
    [
      'only yue',
      { chineseSubtitle: 'none', subtitleStatus: 'not_searched', subtitleTracks: tracks('yue') },
      '缺中文',
      'missing',
    ],
    [
      'yue + chi',
      {
        chineseSubtitle: 'zh',
        subtitleStatus: 'not_searched',
        subtitleTracks: tracks('yue', 'chi'),
      },
      '中文',
      'has',
    ],
    [
      'PGS chi + no_text_source',
      { chineseSubtitle: 'zh', subtitleStatus: 'no_text_source', subtitleTracks: tracks('chi') },
      '中文',
      'has',
    ],
    [
      'found + en + NULL',
      { chineseSubtitle: 'unknown', subtitleStatus: 'found', subtitleLanguage: 'en' },
      null,
      'unknown',
    ],
    [
      'garbage tracks',
      { chineseSubtitle: 'unknown', subtitleStatus: 'not_searched', subtitleTracks: '{oops' },
      null,
      'unknown',
    ],
  ])('%s → %s', (_name, over, label, group) => {
    const s = deriveSubtitleStatus(m('success', over));
    expect(s?.label ?? null).toBe(label);
    // The chip and the badge say the same thing (AC #7 / D1).
    if (group === 'has') expect(['繁中', '簡中', '中文']).toContain(s?.label);
    if (group === 'missing') expect(['缺中文', '無字幕源', '已略過', '未翻譯']).toContain(s?.label);
  });

  it('繁中 and 中文 are steady (poster stays quiet); 簡中 is info, 缺中文 warning (j2-d)', () => {
    const hant = deriveSubtitleStatus(m('success', { chineseSubtitle: 'zh_hant' }));
    expect(hant?.className).toContain('--success-tint');
    expect(hant?.steadyState).toBe(true);
    const zh = deriveSubtitleStatus(m('success', { chineseSubtitle: 'zh' }));
    expect(zh?.label).toBe('中文');
    expect(zh?.steadyState).toBe(true);
    const hans = deriveSubtitleStatus(m('success', { chineseSubtitle: 'zh_hans' }));
    expect(hans?.className).toContain('--info-tint');
    expect(hans?.className).not.toContain('--accent-tint');
    const none = deriveSubtitleStatus(m('success', { chineseSubtitle: 'none' }));
    expect(none?.label).toBe('缺中文');
    expect(none?.className).toContain('--warning-tint');
    expect(none?.className).toContain('--warning-text');
    expect(none?.className).not.toContain('--bg-tertiary');
    expect(none?.steadyState).toBeFalsy();
  });

  it('never re-derives "has Chinese" from tracks: no verdict → no badge (AC #3)', () => {
    expect(deriveSubtitleStatus(m('success', { subtitleTracks: tracks('zh-Hant') }))).toBeNull();
    expect(
      deriveSubtitleStatus(m('success', { subtitleStatus: 'found', subtitleLanguage: 'zh-Hant' }))
    ).toBeNull();
    expect(deriveSubtitleStatus(m('success', { subtitleTracks: tracks('eng') }))).toBeNull();
    expect(deriveSubtitleStatus(m('success'))).toBeNull();
    expect(deriveSubtitleStatus(undefined)).toBeNull();
  });
});

describe('deriveSubtitleStatus — pipeline states (sub-1-7b, kept under the new verdict)', () => {
  it.each(['probing', 'extracting', 'translating'])(
    'returns null for the transient state %s when the title has no Chinese yet',
    (subtitleStatus) => {
      expect(
        deriveSubtitleStatus(m('success', { subtitleStatus, chineseSubtitle: 'none' }))
      ).toBeNull();
      expect(
        deriveSubtitleStatus(m('success', { subtitleStatus, chineseSubtitle: 'unknown' }))
      ).toBeNull();
    }
  );

  it('a title that already has Chinese keeps its 有 badge while the pipeline runs', () => {
    expect(
      deriveSubtitleStatus(m('success', { subtitleStatus: 'extracting', chineseSubtitle: 'zh' }))
        ?.label
    ).toBe('中文');
  });

  it.each([
    ['no_text_source', '無字幕源'],
    ['skipped', '已略過'],
    ['untranslated', '未翻譯'],
  ])('%s → %s (neutral, non-steady) outranks the plain 缺中文', (subtitleStatus, label) => {
    for (const chineseSubtitle of ['none', 'unknown'] as const) {
      const s = deriveSubtitleStatus(m('success', { subtitleStatus, chineseSubtitle }));
      expect(s?.label).toBe(label);
      expect(s?.className).toContain('--bg-tertiary');
      expect(s?.className).not.toContain('--error');
      expect(s?.className).not.toContain('--accent');
      expect(s?.steadyState).toBeFalsy();
    }
  });

  it('keeps 無字幕源 distinct from 缺中文 — different recoveries (sub-1-7a AC #4)', () => {
    expect(
      deriveSubtitleStatus(m('success', { subtitleStatus: 'not_found', chineseSubtitle: 'none' }))
        ?.label
    ).toBe('缺中文');
    expect(
      deriveSubtitleStatus(
        m('success', { subtitleStatus: 'no_text_source', chineseSubtitle: 'none' })
      )?.label
    ).toBe('無字幕源');
  });

  it('SCOPE FENCE: an English-only title without the pipeline verdict never badges 未翻譯', () => {
    for (const subtitleStatus of [undefined, 'not_searched']) {
      expect(
        deriveSubtitleStatus(m('success', { subtitleStatus, chineseSubtitle: 'none' }))?.label
      ).toBe('缺中文');
    }
  });
});

describe('pickPosterBadge — exception signal (ux3-0-2)', () => {
  it('suppresses the happy steady states (已入庫 + 繁中 / 中文) → no badge', () => {
    expect(pickPosterBadge(m('success', { chineseSubtitle: 'zh_hant' }))).toBeNull();
    expect(pickPosterBadge(m('success', { chineseSubtitle: 'zh' }))).toBeNull();
  });
  it('shows a subtitle exception for an in-library item (缺中文 / 簡中 / pipeline verdicts)', () => {
    expect(pickPosterBadge(m('success', { chineseSubtitle: 'none' }))?.label).toBe('缺中文');
    expect(pickPosterBadge(m('success', { chineseSubtitle: 'zh_hans' }))?.label).toBe('簡中');
    expect(
      pickPosterBadge(m('success', { subtitleStatus: 'no_text_source', chineseSubtitle: 'none' }))
        ?.label
    ).toBe('無字幕源');
    expect(pickPosterBadge(m('success', { subtitleStatus: 'skipped' }))?.label).toBe('已略過');
    expect(pickPosterBadge(m('success', { subtitleStatus: 'untranslated' }))?.label).toBe('未翻譯');
  });
  it('renders NO badge for the three transient states (normal progress is not an exception)', () => {
    for (const subtitleStatus of ['probing', 'extracting', 'translating']) {
      expect(pickPosterBadge(m('success', { subtitleStatus, chineseSubtitle: 'none' }))).toBeNull();
    }
  });
  it('a lifecycle exception (整理中 / 失敗) wins over subtitle', () => {
    expect(pickPosterBadge(m('pending', { chineseSubtitle: 'zh_hant' }))?.label).toBe('整理中');
    expect(pickPosterBadge(m('pending', { subtitleStatus: 'skipped' }))?.label).toBe('整理中');
    expect(pickPosterBadge(m('failed'))?.label).toBe('失敗');
  });
  it('shows no badge for unknown state (F3)', () => {
    expect(pickPosterBadge(m('success', { chineseSubtitle: 'unknown' }))).toBeNull();
    expect(pickPosterBadge(m('success'))).toBeNull();
    expect(pickPosterBadge(undefined)).toBeNull();
  });
});

// AC #9 (frontend): the NAS distribution as the API returns it — 37 movies with
// embedded chi+eng, 3 English-only, 12 never-read, 3 found zh-Hant, 2 series.
// In list-row mode (deriveSubtitleStatus) the 40 "has" titles read 繁中/中文 and
// the 3 "missing" titles read 缺中文; the 14 unknown ones carry no badge.
describe('NAS-shaped library (AC #9)', () => {
  const chiEng = JSON.stringify([{ language: 'chi' }, { language: 'eng' }]);
  const eng = JSON.stringify([{ language: 'eng' }]);
  const rows: Media[] = [
    ...Array.from({ length: 37 }, () =>
      m('success', {
        chineseSubtitle: 'zh',
        subtitleStatus: 'not_searched',
        subtitleTracks: chiEng,
      })
    ),
    ...Array.from({ length: 3 }, () =>
      m('success', { chineseSubtitle: 'none', subtitleStatus: 'not_searched', subtitleTracks: eng })
    ),
    ...Array.from({ length: 12 }, () =>
      m('failed', { chineseSubtitle: 'unknown', subtitleStatus: 'not_searched' })
    ),
    ...Array.from({ length: 3 }, () =>
      m('success', {
        chineseSubtitle: 'zh_hant',
        subtitleStatus: 'found',
        subtitleLanguage: 'zh-Hant',
      })
    ),
    ...Array.from({ length: 2 }, () =>
      m('success', { chineseSubtitle: 'unknown', subtitleStatus: 'not_searched' })
    ),
  ];

  it('has 40 / missing 3 / unknown 14, and every badge matches its chip', () => {
    const labels = rows.map((r) => deriveSubtitleStatus(r)?.label ?? null);
    const has = labels.filter((l) => l === '繁中' || l === '中文');
    const missing = labels.filter((l) => l === '缺中文');
    const none = labels.filter((l) => l === null);
    expect(has).toHaveLength(40);
    expect(missing).toHaveLength(3);
    expect(none).toHaveLength(14);
    // Nothing reads the retired 有字幕 / 缺字幕.
    expect(labels).not.toContain('有字幕');
    expect(labels).not.toContain('缺字幕');
  });
});

// dsr-6b AC #3 — one mapping for subtitle-language labels, shared by the detail
// page's 檔案資訊 row and the 管理字幕 dialog's track pills.
describe('subtitleLangLabel', () => {
  it('classifies scripts with the shared HANT/HANS sets, case-insensitively', () => {
    expect(subtitleLangLabel('zh-Hant')).toEqual({ label: '繁中', family: 'hant' });
    expect(subtitleLangLabel('zh-tw')).toEqual({ label: '繁中', family: 'hant' });
    expect(subtitleLangLabel('zh')).toEqual({ label: '繁中', family: 'hant' });
    expect(subtitleLangLabel('zh-CN')).toEqual({ label: '簡中', family: 'hans' });
    expect(subtitleLangLabel('zh-Hans')).toEqual({ label: '簡中', family: 'hans' });
  });

  it('reads English from en, en-*, and the ISO 639-2 eng tag', () => {
    expect(subtitleLangLabel('en')).toEqual({ label: '英文', family: 'en' });
    expect(subtitleLangLabel('en-US')).toEqual({ label: '英文', family: 'en' });
    expect(subtitleLangLabel('eng')).toEqual({ label: '英文', family: 'en' });
  });

  it('says 未標示 for an empty or und tag', () => {
    expect(subtitleLangLabel('')).toEqual({ label: '未標示', family: 'other' });
    expect(subtitleLangLabel('und')).toEqual({ label: '未標示', family: 'other' });
  });

  it('keeps anything else as the file stated it — chi/zho never invent a script', () => {
    expect(subtitleLangLabel('chi')).toEqual({ label: 'chi', family: 'other' });
    expect(subtitleLangLabel('JPN')).toEqual({ label: 'JPN', family: 'other' });
  });
});

// disc-2026-10-subtitle-filter-series-phase-2 AC #6: a SERIES row carries no
// tracks and is never searched online — its chineseSubtitle is now rolled up
// from its episodes by the backend, and the badge reads only that.
describe('deriveSubtitleStatus — series rolled up from episodes', () => {
  const series = (chineseSubtitle: ChineseSubtitle) =>
    m('success', { chineseSubtitle, subtitleStatus: 'not_searched' });

  it('《末日光明》with one English-only episode → 缺中文, a poster exception', () => {
    expect(deriveSubtitleStatus(series('none'))?.label).toBe('缺中文');
    expect(pickPosterBadge(series('none'))?.label).toBe('缺中文');
  });

  it('every episode Traditional → 繁中, steady (no poster badge)', () => {
    expect(deriveSubtitleStatus(series('zh_hant'))?.label).toBe('繁中');
    expect(pickPosterBadge(series('zh_hant'))).toBeNull();
  });

  it('one unread episode → unknown, no badge', () => {
    expect(deriveSubtitleStatus(series('unknown'))).toBeNull();
  });
});
