import { describe, it, expect } from 'vitest';
import {
  buildConsentRows,
  sectionDefaultExpanded,
  sectionIds,
  sortForDisplay,
  type ConsentRow,
  type ConsentSort,
} from './consentRows';
import { groupCandidates, groupOrder } from './consentSelection';
import type { GenerationCandidate } from '../../../services/subtitleService';

const SRS = 'b1c2d3e4-f5a6-4b7c-8d9e-0f1a2b3c4d5e';

function movie(
  mediaId: string,
  title: string,
  extra: Partial<GenerationCandidate> = {}
): GenerationCandidate {
  return {
    mediaId,
    mediaType: 'movie',
    title,
    route: 'extract',
    runtimeMinutes: 100,
    runtimeKnown: true,
    estimatedUsd: 0.05,
    ...extra,
  };
}

function episode(
  mediaId: string,
  season: number,
  ep: number,
  extra: Partial<GenerationCandidate> = {}
): GenerationCandidate {
  return {
    mediaId,
    mediaType: 'episode',
    title: `S0${season}E0${ep}`,
    route: 'asr',
    runtimeMinutes: 50,
    runtimeKnown: true,
    estimatedUsd: 0.3,
    seriesId: SRS,
    seriesTitle: '怪奇物語',
    seasonNumber: season,
    episodeNumber: ep,
    ...extra,
  };
}

/** Every id visible — the state the list is in with no chip and no search. */
const allVisible = (cs: GenerationCandidate[]) => new Set(cs.map((c) => c.mediaId));

function rowsOf(
  candidates: GenerationCandidate[],
  overrides: Partial<Parameters<typeof buildConsentRows>[0]> = {}
): ConsentRow[] {
  return buildConsentRows({
    candidates,
    visibleIds: allVisible(candidates),
    sort: 'group',
    expandedOverride: {},
    searching: false,
    ...overrides,
  });
}

const keys = (rows: ConsentRow[]) => rows.map((r) => r.key);
const candidateKeys = (rows: ConsentRow[]) =>
  rows.filter((r) => r.kind === 'candidate').map((r) => r.key);

describe('movies section split (sub-6-11 AC #4)', () => {
  const matchedA = movie('m1', '沙丘：第二部', { tmdbMatched: true });
  const matchedB = movie('m2', '奧本海默', { tmdbMatched: true });
  const unmatched = movie('m3', 'Interstellar.2014.x265-GRP', { tmdbMatched: false });

  it('MATCHED comes first — the leading section is also the one the budget spends on first', () => {
    // Alexyu 裁定 2026-09-09. groupOrder IS the submission order, so this
    // assertion is about money, not about layout.
    const groups = groupCandidates([unmatched, matchedA, matchedB]);
    expect(groups.map((g) => g.movieSection)).toEqual(['matched', 'unmatched']);
    expect(groupOrder([unmatched, matchedA, matchedB]).map((c) => c.mediaId)).toEqual([
      'm1',
      'm2',
      'm3',
    ]);
  });

  it('all matched → ONE header-less section, the shipped flat rendering', () => {
    const groups = groupCandidates([matchedA, matchedB]);
    expect(groups).toHaveLength(1);
    expect(groups[0].movieSection).toBeUndefined();
    expect(rowsOf([matchedA, matchedB]).every((r) => r.kind === 'candidate')).toBe(true);
  });

  it('all unmatched → still ONE section: there is nothing to separate it from', () => {
    const groups = groupCandidates([unmatched]);
    expect(groups[0].movieSection).toBeUndefined();
  });

  it('a pre-sub-6-10a server (no tmdb_matched at all) never splits', () => {
    const old = [movie('m1', '沙丘'), movie('m2', '奧本海默')];
    expect(groupCandidates(old)[0].movieSection).toBeUndefined();
    expect(rowsOf(old).every((r) => r.kind === 'candidate')).toBe(true);
  });

  it('the two sections get their own collapsible headers', () => {
    const rows = rowsOf([matchedA, matchedB, unmatched]);
    expect(keys(rows)).toEqual([
      sectionIds.movies('matched'),
      'm1',
      'm2',
      sectionIds.movies('unmatched'),
      'm3',
    ]);
  });
});

describe('collapse (sub-6-11 AC #4)', () => {
  const CANDIDATES = [movie('m1', '沙丘'), episode('e1', 1, 1), episode('e2', 1, 2)];

  it('a series starts CLOSED — a 2,400-episode library opens as one line per show', () => {
    expect(sectionDefaultExpanded(sectionIds.series(SRS))).toBe(false);
    const rows = rowsOf(CANDIDATES);
    expect(keys(rows)).toEqual(['m1', sectionIds.series(SRS)]);
  });

  it('movie sections and seasons start OPEN', () => {
    expect(sectionDefaultExpanded(sectionIds.movies('unmatched'))).toBe(true);
    expect(sectionDefaultExpanded(sectionIds.season(SRS, 1))).toBe(true);
  });

  it('an override opens the show and its episodes come back', () => {
    const rows = rowsOf(CANDIDATES, {
      expandedOverride: { [sectionIds.series(SRS)]: true },
    });
    expect(candidateKeys(rows)).toEqual(['m1', 'e1', 'e2']);
  });

  it('a search opens a section the user has not touched', () => {
    const rows = rowsOf(CANDIDATES, { searching: true });
    expect(candidateKeys(rows)).toEqual(['m1', 'e1', 'e2']);
  });

  it('[CR] an explicit collapse OUTRANKS the search auto-expand', () => {
    // The auto-expand is a DEFAULT, not an override. Before this fix the
    // searching branch was checked first, so a show the user collapsed during a
    // search stayed open on screen while recording "expanded" — and came back
    // open once the search was cleared, the opposite of the click.
    const rows = rowsOf(CANDIDATES, {
      searching: true,
      expandedOverride: { [sectionIds.series(SRS)]: false },
    });
    expect(candidateKeys(rows)).toEqual(['m1']);
  });

  it('a season the user closed hides only that season', () => {
    const multi = [episode('e1', 1, 1), episode('e2', 2, 1)];
    const rows = rowsOf(multi, {
      expandedOverride: {
        [sectionIds.series(SRS)]: true,
        [sectionIds.season(SRS, 1)]: false,
      },
    });
    expect(keys(rows)).toEqual([
      sectionIds.series(SRS),
      sectionIds.season(SRS, 1),
      sectionIds.season(SRS, 2),
      'e2',
    ]);
  });

  it('a section with no visible row emits NO header — an empty bar reads as a hit', () => {
    const multi = [episode('e1', 1, 1), episode('e2', 2, 1)];
    const rows = buildConsentRows({
      candidates: multi,
      visibleIds: new Set(['e2']),
      sort: 'group',
      expandedOverride: { [sectionIds.series(SRS)]: true },
      searching: true,
    });
    expect(keys(rows)).toEqual([sectionIds.series(SRS), sectionIds.season(SRS, 2), 'e2']);
  });

  it('a collapsed header still carries the WHOLE section, so its counts speak for it', () => {
    const rows = rowsOf(CANDIDATES);
    const header = rows.find((r) => r.kind === 'section');
    expect(header?.kind === 'section' && header.items).toHaveLength(2);
    expect(header?.kind === 'section' && header.expanded).toBe(false);
  });
});

describe('sort is a projection (sub-6-11 AC #2)', () => {
  const cheap = movie('m1', '便宜片', { estimatedUsd: 0.04, tmdbMatched: true });
  const dear = movie('m2', '貴片', { estimatedUsd: 0.9, tmdbMatched: true });
  const mid = movie('m3', 'Alpha', { estimatedUsd: 0.2, tmdbMatched: false });
  const ALL = [cheap, dear, mid];
  const order = new Map(ALL.map((c, i) => [c.mediaId, i]));

  const ids = (sort: ConsentSort) => sortForDisplay(ALL, sort, order).map((c) => c.mediaId);

  it('金額高→低 and 金額低→高 are exact reverses of each other', () => {
    expect(ids('cost-desc')).toEqual(['m2', 'm3', 'm1']);
    expect(ids('cost-asc')).toEqual(['m1', 'm3', 'm2']);
  });

  it('未匹配優先 lifts only rows the server actually called unmatched', () => {
    expect(ids('unmatched-first')).toEqual(['m3', 'm1', 'm2']);
  });

  it('片名 A→Z collates zh-Hant (stroke order) rather than sorting by code point', () => {
    // The two orders genuinely disagree here, which is the point: by UTF-16
    // value 阿里 (U+963F…) sorts LAST and Alpha first; by the zh-Hant
    // collation a Taiwanese reader expects, 阿里 (8 strokes) comes first,
    // 賓士 (14) later, and the latin title last.
    const titles = [movie('t1', 'Alpha'), movie('t2', '賓士'), movie('t3', '阿里')];
    const titleOrder = new Map(titles.map((c, i) => [c.mediaId, i]));
    expect(sortForDisplay(titles, 'title-asc', titleOrder).map((c) => c.mediaId)).toEqual([
      't3',
      't2',
      't1',
    ]);
  });

  it('equal amounts fall back to STATE order, so the list never reshuffles itself', () => {
    const tie = [
      movie('a', 'A', { estimatedUsd: 0.05 }),
      movie('b', 'B', { estimatedUsd: 0.05 }),
      movie('c', 'C', { estimatedUsd: 0.05 }),
    ];
    const tieOrder = new Map(tie.map((c, i) => [c.mediaId, i]));
    expect(sortForDisplay(tie, 'cost-desc', tieOrder).map((c) => c.mediaId)).toEqual([
      'a',
      'b',
      'c',
    ]);
  });

  it('the model prices, not the default quote, decide a cost sort', () => {
    // sub-6-8b: switching model re-prices every row. If the sort ignored that,
    // 「最貴的先」 would point at a row that is no longer the most expensive.
    const prices = { m1: 0.9, m2: 0.04, m3: 0.2 };
    expect(sortForDisplay(ALL, 'cost-desc', order, prices).map((c) => c.mediaId)).toEqual([
      'm1',
      'm3',
      'm2',
    ]);
  });

  it('sorting FLATTENS: no section headers survive a non-group sort', () => {
    const rows = buildConsentRows({
      candidates: [movie('m1', '沙丘'), episode('e1', 1, 1)],
      visibleIds: new Set(['m1', 'e1']),
      sort: 'cost-desc',
      expandedOverride: {},
      searching: false,
    });
    expect(rows.every((r) => r.kind === 'candidate')).toBe(true);
    // …and a collapsed show no longer hides its episodes, because there is no
    // show to collapse in a flat list.
    expect(candidateKeys(rows)).toEqual(['e1', 'm1']);
  });

  it('sorting never touches groupOrder — the submission order is untouched', () => {
    const before = groupOrder(ALL).map((c) => c.mediaId);
    sortForDisplay(ALL, 'cost-desc', order);
    expect(groupOrder(ALL).map((c) => c.mediaId)).toEqual(before);
  });
});

describe('section accessible names (sub-6-11 CR)', () => {
  it('a movie section is not 「整部」 — it is a block of films, not one film', () => {
    const rows = buildConsentRows({
      candidates: [
        movie('m1', '沙丘', { tmdbMatched: true }),
        movie('m2', 'Blah.2014.x265', { tmdbMatched: false }),
      ],
      visibleIds: new Set(['m1', 'm2']),
      sort: 'group',
      expandedOverride: {},
      searching: false,
    });
    const labels = rows
      .filter((r) => r.kind === 'section')
      .map((r) => (r.kind === 'section' ? r.selectLabel : ''));
    expect(labels).toEqual(['選取所有已匹配的電影', '選取所有未匹配的電影']);
  });

  it('a series keeps 「選取整部 X」 and a season keeps 「選取第 n 季」', () => {
    const rows = rowsOf([episode('e1', 1, 1), episode('e2', 2, 1)], {
      expandedOverride: { [sectionIds.series(SRS)]: true },
    });
    const labels = rows
      .filter((r) => r.kind === 'section')
      .map((r) => (r.kind === 'section' ? r.selectLabel : ''));
    expect(labels).toEqual(['選取整部 怪奇物語', '選取第 1 季', '選取第 2 季']);
  });
});
