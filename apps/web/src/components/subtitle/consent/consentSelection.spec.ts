import { describe, it, expect } from 'vitest';
import {
  applyRouteFilter,
  computeTotals,
  defaultSelection,
  listableCandidates,
  parseBudgetInput,
  groupCandidates,
  groupOrder,
  isWritable,
  selectableIds,
} from './consentSelection';
import type { GenerationCandidate } from '../../../services/subtitleService';

// Media-id fixture convention (9R-18 AC 7): UUID strings only.
const c = (
  mediaId: string,
  route: GenerationCandidate['route'],
  estimatedUsd: number,
  extra?: Partial<GenerationCandidate>
): GenerationCandidate => ({
  mediaId,
  mediaType: 'movie',
  title: `T-${mediaId}`,
  route,
  runtimeMinutes: 120,
  runtimeKnown: true,
  estimatedUsd,
  ...extra,
});

const A = '0a54a9e2-3a67-4f3e-9f8e-a1c2d3e4f501';
const B = '1b65baf3-4b78-4a4f-8a9f-b2d3e4f5a602';
const C = '2c76cba4-5c89-4b5a-9baf-c3e4f5a6b703';
const D = '3d87dcb5-6d9a-4c6b-8cba-d4f5a6b7c804';

describe('listableCandidates', () => {
  it('excludes route=skip (no quote, no list vocabulary in the design)', () => {
    const out = listableCandidates([c(A, 'extract', 0.05), c(B, 'skip', 0), c(C, 'asr', 0.26)]);
    expect(out.map((x) => x.mediaId)).toEqual([A, C]);
  });
});

describe('defaultSelection (準則④ as amended: lowest-cost, never ASR)', () => {
  it('selects every extract candidate and no asr candidate', () => {
    const sel = defaultSelection([c(A, 'extract', 0.05), c(B, 'asr', 0.26), c(C, 'extract', 0.04)]);
    expect(sel).toEqual(new Set([A, C]));
  });
});

describe('applyRouteFilter', () => {
  const list = [c(A, 'extract', 0.05), c(B, 'asr', 0.26)];
  it.each([
    ['all', [A, B]],
    ['extract', [A]],
    ['asr', [B]],
  ] as const)('%s', (filter, ids) => {
    expect(applyRouteFilter(list, filter).map((x) => x.mediaId)).toEqual(ids);
  });
});

describe('computeTotals — the ONE money source (三處同源)', () => {
  const list = [c(A, 'extract', 0.05), c(B, 'extract', 0.04), c(C, 'asr', 0.26), c(D, 'asr', 0.31)];

  it('splits counts and USD by route; total = extract + asr', () => {
    const t = computeTotals(list, new Set([A, B, C, D]), 5);
    expect(t.selectedExtractCount).toBe(2);
    expect(t.selectedAsrCount).toBe(2);
    expect(t.selectedExtractUsd).toBeCloseTo(0.09);
    expect(t.selectedAsrUsd).toBeCloseTo(0.57);
    expect(t.selectedTotalUsd).toBeCloseTo(0.66);
    expect(t.overBudget).toBe(false);
  });

  it('flags overBudget when the total exceeds the ceiling', () => {
    const t = computeTotals(list, new Set([A, B, C, D]), 0.5);
    expect(t.overBudget).toBe(true);
  });

  it('feasibleCount walks selected items in list order with check-BEFORE semantics', () => {
    // Ceiling 0.30: item A runs (0 < 0.30), B runs (0.05 < 0.30), C runs
    // (0.09 < 0.30 — the soft ceiling lets the crossing item start), D paused
    // (0.35 >= 0.30).
    const t = computeTotals(list, new Set([A, B, C, D]), 0.3);
    expect(t.feasibleCount).toBe(3);
  });

  it('deselected items contribute nothing anywhere', () => {
    const t = computeTotals(list, new Set([C]), 5);
    expect(t.selectedCount).toBe(1);
    expect(t.selectedExtractUsd).toBe(0);
    expect(t.selectedTotalUsd).toBeCloseTo(0.26);
  });
});

describe('parseBudgetInput — 0 must NEVER read as unlimited', () => {
  it.each([
    ['5.00', 5],
    ['0.01', 0.01],
    ['12.5', 12.5],
  ])('accepts %s', (text, v) => {
    expect(parseBudgetInput(text)).toBe(v);
  });

  it.each(['0', '-1', 'abc', '', 'Infinity', 'NaN'])('rejects %s as null', (text) => {
    expect(parseBudgetInput(text)).toBeNull();
  });
});

// ─── sub-5-3 AC #2: series/season grouping — 三序同源 ──────────────────────

function seriesCandidate(
  mediaId: string,
  seriesId: string,
  season: number,
  episode: number,
  overrides: Partial<GenerationCandidate> = {}
): GenerationCandidate {
  return {
    mediaId,
    mediaType: 'episode',
    title: `Ep ${mediaId} S${season}E${episode}`,
    route: 'asr',
    runtimeMinutes: 45,
    runtimeKnown: true,
    estimatedUsd: 0.27,
    seriesId,
    seriesTitle: `劇 ${seriesId}`,
    seasonNumber: season,
    episodeNumber: episode,
    ...overrides,
  };
}

function movieCandidate(
  mediaId: string,
  overrides: Partial<GenerationCandidate> = {}
): GenerationCandidate {
  return {
    mediaId,
    mediaType: 'movie',
    title: `Movie ${mediaId}`,
    route: 'extract',
    runtimeMinutes: 100,
    runtimeKnown: true,
    estimatedUsd: 0.04,
    ...overrides,
  };
}

describe('groupCandidates (sub-5-3 AC #2)', () => {
  it('groups episodes by seriesId, movies as one flat section first, series ordered BY TITLE', () => {
    const input = [
      seriesCandidate('b1', 'srs-b', 1, 1),
      movieCandidate('m1'),
      seriesCandidate('a1', 'srs-a', 1, 1),
      seriesCandidate('b2', 'srs-b', 1, 2),
    ];
    const groups = groupCandidates(input);

    expect(groups.map((g) => g.kind)).toEqual(['movies', 'series', 'series']);
    expect(groups[0].items.map((c) => c.mediaId)).toEqual(['m1']);
    // CR M2: 「劇 srs-a」 sorts before 「劇 srs-b」 — section order follows the
    // SERIES title, not which episode the backend's (title,id) sort emitted
    // first (which made the show order look random and jump on new episodes).
    expect(groups[1].seriesId).toBe('srs-a');
    expect(groups[2].seriesId).toBe('srs-b');
    expect(groups[2].items.map((c) => c.mediaId)).toEqual(['b1', 'b2']);
  });

  it('CR M2: a degraded (untitled) series sinks BELOW every named show', () => {
    const groups = groupCandidates([
      seriesCandidate('x1', 'srs-x', 1, 1, { seriesTitle: '' }),
      seriesCandidate('z1', 'srs-z', 1, 1, { seriesTitle: '甲劇' }),
    ]);
    expect(groups.map((g) => g.seriesId)).toEqual(['srs-z', 'srs-x']);
  });

  it('orders within a series by season then episode — NOT by the backend title sort', () => {
    const input = [
      seriesCandidate('x-s2e1', 'srs-x', 2, 1),
      seriesCandidate('x-s1e2', 'srs-x', 1, 2),
      seriesCandidate('x-s1e1', 'srs-x', 1, 1),
    ];
    const [group] = groupCandidates(input);
    expect(group.items.map((c) => c.mediaId)).toEqual(['x-s1e1', 'x-s1e2', 'x-s2e1']);
  });

  it('emits season sections ONLY when a series spans ≥2 seasons (single-season noise rule)', () => {
    const multi = groupCandidates([
      seriesCandidate('a1', 'srs-a', 1, 1),
      seriesCandidate('a2', 'srs-a', 2, 1),
    ])[0];
    const single = groupCandidates([
      seriesCandidate('b1', 'srs-b', 1, 1),
      seriesCandidate('b2', 'srs-b', 1, 2),
    ])[0];

    expect(multi.showSeasonHeaders).toBe(true);
    expect(multi.seasons?.map((s) => s.seasonNumber)).toEqual([1, 2]);
    expect(single.showSeasonHeaders).toBe(false);
  });

  it('S00 specials group and sort as season ZERO (never dropped, sorts first)', () => {
    const [group] = groupCandidates([
      seriesCandidate('a-s1', 'srs-a', 1, 1),
      seriesCandidate('a-s0', 'srs-a', 0, 1),
    ]);
    expect(group.showSeasonHeaders).toBe(true);
    expect(group.seasons?.map((s) => s.seasonNumber)).toEqual([0, 1]);
  });

  it('a degraded (empty) seriesTitle still groups on the id', () => {
    const [group] = groupCandidates([
      seriesCandidate('a1', 'srs-a', 1, 1, { seriesTitle: '' }),
      seriesCandidate('a2', 'srs-a', 1, 2, { seriesTitle: '' }),
    ]);
    expect(group.kind).toBe('series');
    expect(group.items).toHaveLength(2);
    expect(group.seriesTitle).toBe('');
  });
});

describe('groupOrder — 三序同源紅線 (display = submission = feasible walk)', () => {
  it('returns the flattened grouped order, and computeTotals feasibleCount walks THAT order', () => {
    const input = [
      seriesCandidate('b1', 'srs-b', 1, 1, { estimatedUsd: 3 }),
      movieCandidate('m1', { estimatedUsd: 1 }),
      seriesCandidate('b2', 'srs-b', 1, 2, { estimatedUsd: 3 }),
    ];
    const ordered = groupOrder(input);
    expect(ordered.map((c) => c.mediaId)).toEqual(['m1', 'b1', 'b2']);

    // Budget 2: in grouped order the movie ($1) runs, b1 starts under the
    // ceiling, b2 does not — feasibleCount must reflect the DISPLAYED order.
    const totals = computeTotals(ordered, new Set(['m1', 'b1', 'b2']), 2);
    expect(totals.feasibleCount).toBe(2);
  });

  it('is a permutation — nothing added, nothing lost', () => {
    const input = [
      seriesCandidate('a1', 'srs-a', 2, 5),
      movieCandidate('m2'),
      seriesCandidate('a2', 'srs-a', 1, 3),
      movieCandidate('m1'),
    ];
    const ordered = groupOrder(input);
    expect(ordered).toHaveLength(input.length);
    expect(new Set(ordered.map((c) => c.mediaId))).toEqual(new Set(input.map((c) => c.mediaId)));
  });
});

describe('sub-6-1 writability', () => {
  it('defaultSelection skips an extract row the backend marked unwritable', () => {
    const list = [
      c(A, 'extract', 0.02),
      c(B, 'extract', 0.02, { writable: false, blocker: '唯讀' }),
    ];
    expect([...defaultSelection(list)]).toEqual([A]);
  });

  it('isWritable treats a missing field (old server) as writable', () => {
    expect(isWritable(c(A, 'extract', 0.02))).toBe(true);
    expect(isWritable(c(A, 'extract', 0.02, { writable: true }))).toBe(true);
    expect(isWritable(c(A, 'extract', 0.02, { writable: false }))).toBe(false);
  });

  it('selectableIds excludes unwritable rows of every route', () => {
    const list = [
      c(A, 'extract', 0.02),
      c(B, 'asr', 0.3, { writable: false }),
      c(C, 'asr', 0.3),
      c(D, 'extract', 0.02, { writable: false }),
    ];
    expect(selectableIds(list)).toEqual([A, C]);
  });
});
