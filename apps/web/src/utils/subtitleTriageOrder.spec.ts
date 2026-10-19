/**
 * The FRONTEND half of the 字幕狀態 sort contract
 * (disc-2026-10-library-list-triage-b2 AC #2; confirmed against [@contract-v1]
 * Story disc-2026-10-library-list-triage-a AC #1).
 *
 * The backend sorts `sort_by=subtitle_status` by `models.SubtitleTriageRank`
 * (apps/api/internal/models/chinese_subtitle.go), pinned there by
 * `TestSubtitleTriageRank_EveryCombination`
 * (apps/api/internal/models/subtitle_triage_rank_test.go). That rank is meant to be
 * the rank of the badge the list SHOWS — and the badge is drawn here, by
 * `deriveSubtitleStatus`. This file walks the SAME (verdict × subtitle_status) grid
 * as the Go table and checks that the badge this side draws for each pair sits in
 * the rank the Go side sorts it into. If either side changes, both tables change
 * together.
 *
 *   0 缺中文 → 1 無字幕源／未翻譯／已略過 → 2 不知道 → 3 簡中 → 4 中文 → 5 繁中
 */
import { describe, it, expect } from 'vitest';
import { deriveSubtitleStatus } from './libraryStatus';
import type { ChineseSubtitle } from '../types/library';

/** The rank of each word the list can show in its 字幕 column (`null` badge → 不知道). */
const RANK_OF_LABEL: Record<string, number> = {
  缺中文: 0,
  無字幕源: 1,
  未翻譯: 1,
  已略過: 1,
  不知道: 2,
  簡中: 3,
  中文: 4,
  繁中: 5,
};

// '' = NULL subtitle_status, then models.AllSubtitleStatuses() in the Go table's order.
const STATUSES = [
  '',
  'not_searched',
  'searching',
  'found',
  'not_found',
  'probing',
  'extracting',
  'translating',
  'no_text_source',
  'skipped',
  'untranslated',
] as const;

// want[verdict][status] — copied by hand from TestSubtitleTriageRank_EveryCombination.
const GO_TABLE: Record<ChineseSubtitle, Record<(typeof STATUSES)[number], number>> = {
  zh_hant: {
    '': 5,
    not_searched: 5,
    searching: 5,
    found: 5,
    not_found: 5,
    probing: 5,
    extracting: 5,
    translating: 5,
    no_text_source: 5,
    skipped: 5,
    untranslated: 5,
  },
  zh_hans: {
    '': 3,
    not_searched: 3,
    searching: 3,
    found: 3,
    not_found: 3,
    probing: 3,
    extracting: 3,
    translating: 3,
    no_text_source: 3,
    skipped: 3,
    untranslated: 3,
  },
  zh: {
    '': 4,
    not_searched: 4,
    searching: 4,
    found: 4,
    not_found: 4,
    probing: 4,
    extracting: 4,
    translating: 4,
    no_text_source: 4,
    skipped: 4,
    untranslated: 4,
  },
  none: {
    '': 0,
    not_searched: 0,
    searching: 0,
    found: 0,
    not_found: 0,
    probing: 2,
    extracting: 2,
    translating: 2,
    no_text_source: 1,
    skipped: 1,
    untranslated: 1,
  },
  unknown: {
    '': 2,
    not_searched: 2,
    searching: 2,
    found: 2,
    not_found: 2,
    probing: 2,
    extracting: 2,
    translating: 2,
    no_text_source: 1,
    skipped: 1,
    untranslated: 1,
  },
};

const shownLabel = (chineseSubtitle: ChineseSubtitle, status: string) =>
  deriveSubtitleStatus({
    parseStatus: 'success',
    chineseSubtitle,
    subtitleStatus: status || undefined,
  })?.label ?? '不知道';

describe('字幕狀態 sort — the badge the list shows ranks where the backend sorts it', () => {
  it('[P0] covers every verdict the backend has, every status in the Go table', () => {
    expect(Object.keys(GO_TABLE).sort()).toEqual(
      ['none', 'unknown', 'zh', 'zh_hans', 'zh_hant'].sort()
    );
    for (const byStatus of Object.values(GO_TABLE)) {
      expect(Object.keys(byStatus)).toHaveLength(STATUSES.length);
    }
  });

  for (const [verdict, byStatus] of Object.entries(GO_TABLE) as [
    ChineseSubtitle,
    Record<string, number>,
  ][]) {
    it(`[P0] verdict ${verdict}: every subtitle_status lands in the Go rank`, () => {
      const got = STATUSES.map((st) => [st, RANK_OF_LABEL[shownLabel(verdict, st)]]);
      const want = STATUSES.map((st) => [st, byStatus[st]]);
      expect(got).toEqual(want);
    });
  }

  it('[P0] sorted by that rank, the words read 缺中文 → reason → 不知道 → 簡中 → 中文 → 繁中', () => {
    const labels = new Set<string>();
    for (const verdict of Object.keys(GO_TABLE) as ChineseSubtitle[]) {
      for (const st of STATUSES) labels.add(shownLabel(verdict, st));
    }
    const ordered = [...labels].sort(
      (a, b) => RANK_OF_LABEL[a] - RANK_OF_LABEL[b] || a.localeCompare(b)
    );
    // Every word the badge can draw has a rank (no undefined slipping to the end).
    expect(ordered.every((l) => l in RANK_OF_LABEL)).toBe(true);
    expect(ordered.map((l) => RANK_OF_LABEL[l])).toEqual([0, 1, 1, 1, 2, 3, 4, 5]);
    expect(ordered[0]).toBe('缺中文');
    expect(new Set(ordered.slice(1, 4))).toEqual(new Set(['無字幕源', '未翻譯', '已略過']));
    expect(ordered.slice(4)).toEqual(['不知道', '簡中', '中文', '繁中']);
  });
});
