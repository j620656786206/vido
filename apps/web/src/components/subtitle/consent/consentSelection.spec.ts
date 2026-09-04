import { describe, it, expect } from 'vitest';
import {
  applyRouteFilter,
  candidateUsd,
  computeTotals,
  defaultSelection,
  listableCandidates,
  modelChoices,
  parseBudgetInput,
  groupCandidates,
  groupOrder,
  isWritable,
  selectableIds,
} from './consentSelection';
import { usd } from '../../../lib/currency';
import type { GenerationCandidate, TranslationModelInfo } from '../../../services/subtitleService';

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

describe('computeTotals selectable/unwritable counts (sub-6-1 CR M4)', () => {
  it('counts writable rows as selectable and the rest as unwritable', () => {
    const list = [c(A, 'extract', 0.02), c(B, 'asr', 0.3, { writable: false }), c(C, 'asr', 0.3)];
    const t = computeTotals(list, new Set([A]), 5);
    expect(t.candidateCount).toBe(3);
    expect(t.selectableCount).toBe(2);
    expect(t.unwritableCount).toBe(1);
  });
});

// ─── sub-6-8b: the model dimension ──────────────────────────────────────────

const SONNET: TranslationModelInfo = {
  id: 'claude-sonnet-5',
  provider: 'claude',
  displayName: 'Claude Sonnet 5',
  tier: 'balanced',
  isDefault: true,
  qualityGrade: 'A',
  qualityNote: 'Vido 實測 2026-09',
};
const HAIKU: TranslationModelInfo = {
  id: 'claude-haiku-4-5',
  provider: 'claude',
  displayName: 'Claude Haiku 4.5',
  tier: 'fast',
  isDefault: false,
  qualityGrade: 'B',
  qualityNote: 'Vido 實測 2026-09',
};

describe('candidateUsd (the one row-price gate)', () => {
  const row = c(A, 'extract', 0.05);

  it('uses the chosen model price when the row has one', () => {
    expect(candidateUsd(row, { [A]: 0.02 })).toBe(0.02);
  });

  it("falls back to the row's own estimate — a missing per-model price is never invented", () => {
    expect(candidateUsd(row, { [B]: 0.02 })).toBe(0.05);
    expect(candidateUsd(row, undefined)).toBe(0.05);
  });
});

describe('computeTotals under a chosen model', () => {
  const list = [c(A, 'extract', 0.05), c(B, 'asr', 0.31)];
  const prices = { [A]: 0.02, [B]: 0.12 };

  it('re-prices the route split and the total from the same table', () => {
    const t = computeTotals(list, new Set([A, B]), null, prices);
    expect(t.selectedExtractUsd).toBe(0.02);
    expect(t.selectedAsrUsd).toBe(0.12);
    expect(t.selectedTotalUsd).toBeCloseTo(0.14, 5);
  });

  it('re-decides the budget verdict — the ceiling is judged against what will ACTUALLY be spent', () => {
    expect(computeTotals(list, new Set([A, B]), 0.2).overBudget).toBe(true);
    expect(computeTotals(list, new Set([A, B]), 0.2, prices).overBudget).toBe(false);
  });

  it('re-walks the feasible count at the chosen prices', () => {
    // At Sonnet prices the $0.05 ceiling is exactly used up by the first item,
    // so the second never starts; at Haiku prices there is still room for it.
    expect(computeTotals(list, new Set([A, B]), 0.05).feasibleCount).toBe(1);
    expect(computeTotals(list, new Set([A, B]), 0.05, prices).feasibleCount).toBe(2);
  });
});

describe('computeTotals — decimal arithmetic (tech-money-decimal-arithmetic)', () => {
  it('the displayed breakdown always sums to the displayed total', () => {
    // Two halves that each round UP while their float sum rounds DOWN: with
    // `extractUsd + asrUsd` the screen prints 「$0.01 + $0.01 = $0.01」.
    const list = [c(A, 'extract', 0.005), c(B, 'asr', 0.005)];
    const t = computeTotals(list, new Set([A, B]), null);
    expect(usd(t.selectedExtractUsd)).toBe('$0.01');
    expect(usd(t.selectedAsrUsd)).toBe('$0.01');
    expect(usd(t.selectedTotalUsd)).toBe('$0.02');
  });

  it('0.1 + 0.2 is 0.3 — not the number JS gives you', () => {
    const list = [c(A, 'extract', 0.1), c(B, 'asr', 0.2)];
    expect(computeTotals(list, new Set([A, B]), null).selectedTotalUsd).toBe(0.3);
    expect(0.1 + 0.2).not.toBe(0.3);
  });

  it('does not raise a false over-budget alarm on drift alone', () => {
    // CR H2: the first version of this test used a $0.79 ceiling, which
    // 0.7999999999999999 clears by a mile — it passed identically on the old
    // float code and proved nothing. THIS one discriminates.
    //
    // $0.10 + $0.20 is exactly $0.30, the ceiling. In native JS the sum is
    // 0.30000000000000004, so `total > budget` reads TRUE: the user is shown
    // the over-budget screen and a 仍要開始 button for a batch that costs
    // precisely what they authorised.
    const list = [c(A, 'extract', 0.1), c(B, 'asr', 0.2)];
    const t = computeTotals(list, new Set([A, B]), 0.3);

    expect(t.selectedTotalUsd).toBe(0.3);
    expect(t.overBudget).toBe(false);
    // Spelled out so the reason this test exists cannot rot into folklore.
    expect(0.1 + 0.2 > 0.3).toBe(true);
  });

  it('stays exact across a library-sized selection', () => {
    const many = Array.from({ length: 1200 }, (_, i) =>
      c(`${i}`.padStart(8, '0') + '-0000-4000-8000-000000000000', 'extract', 0.01)
    );
    const t = computeTotals(many, new Set(many.map((x) => x.mediaId)), null);
    expect(t.selectedTotalUsd).toBe(12);
  });
});

describe('modelChoices', () => {
  const list = [
    c(A, 'extract', 0.05, { runtimeMinutes: 166 }),
    c(B, 'asr', 0.31, { runtimeMinutes: 52 }),
  ];
  const input = {
    models: [HAIKU, SONNET],
    defaultModelId: 'claude-sonnet-5',
    estimatesByModel: {
      'claude-sonnet-5': { totalUsd: 0.36, perCandidate: { [A]: 0.05, [B]: 0.31 } },
      'claude-haiku-4-5': { totalUsd: 0.14, perCandidate: { [A]: 0.02, [B]: 0.12 } },
    },
    estimatedMinutesByModel: { 'claude-sonnet-5': 37, 'claude-haiku-4-5': 24 },
  };

  it('prices the CURRENT selection, not the whole sweep', () => {
    const rows = modelChoices(list, new Set([A]), input);
    expect(rows.map((r) => [r.id, r.totalUsd])).toEqual([
      ['claude-haiku-4-5', 0.02],
      ['claude-sonnet-5', 0.05],
    ]);
  });

  it('rescales the sweep minutes by the share of runtime actually selected', () => {
    // 166 of 218 runtime minutes selected → 37 * 0.7615 = 28, 24 * 0.7615 = 18.
    const rows = modelChoices(list, new Set([A]), input);
    expect(rows.find((r) => r.id === 'claude-sonnet-5')?.minutes).toBe(28);
    expect(rows.find((r) => r.id === 'claude-haiku-4-5')?.minutes).toBe(18);
  });

  it('omits the time when the server sent no minutes — no duration is invented', () => {
    const rows = modelChoices(list, new Set([A]), {
      ...input,
      estimatedMinutesByModel: undefined,
    });
    expect(rows.every((r) => r.minutes === undefined)).toBe(true);
  });

  it('states the gap against the DEFAULT model, in both directions', () => {
    const rows = modelChoices(list, new Set([A, B]), input);
    const haiku = rows.find((r) => r.id === 'claude-haiku-4-5');
    expect(haiku?.deltaUsd).toBeCloseTo(0.22, 5);
    expect(haiku?.deltaPercent).toBe(61);
    // The default row compares against nothing.
    expect(rows.find((r) => r.id === 'claude-sonnet-5')?.deltaUsd).toBeUndefined();
  });

  it('marks only the top MEASURED grade as best — an ungraded model is not "equal"', () => {
    const gemini: TranslationModelInfo = {
      id: 'gemini-2.5-flash',
      provider: 'gemini',
      displayName: 'Gemini 2.5 Flash',
      tier: 'balanced',
      isDefault: false,
    };
    const rows = modelChoices(list, new Set([A]), {
      ...input,
      models: [gemini, HAIKU, SONNET],
      estimatesByModel: {
        ...input.estimatesByModel,
        'gemini-2.5-flash': { totalUsd: 0.03, perCandidate: { [A]: 0.01, [B]: 0.02 } },
      },
    });
    expect(rows.filter((r) => r.isBestGrade).map((r) => r.id)).toEqual(['claude-sonnet-5']);
    expect(rows.find((r) => r.id === 'gemini-2.5-flash')?.qualityGrade).toBeUndefined();
  });

  it('a pre-sub-6-8a server (no per-model quote) offers the DEFAULT model alone', () => {
    const rows = modelChoices(list, new Set([A]), {
      models: [HAIKU, SONNET],
      defaultModelId: 'claude-sonnet-5',
    });
    // Quoting Haiku at the default rate would be a lie in a money field.
    expect(rows.map((r) => r.id)).toEqual(['claude-sonnet-5']);
    expect(rows[0].totalUsd).toBe(0.05);
  });

  it('an empty catalog is no question at all', () => {
    expect(modelChoices(list, new Set([A]), { models: [], defaultModelId: '' })).toEqual([]);
  });

  it('excludes unwritable rows from every model total (they can never be spent)', () => {
    const withBlocked = [
      c(A, 'extract', 0.05, { runtimeMinutes: 166 }),
      c(D, 'asr', 0.9, { writable: false, runtimeMinutes: 90 }),
    ];
    const rows = modelChoices(withBlocked, new Set([A, D]), input);
    expect(rows.find((r) => r.id === 'claude-sonnet-5')?.totalUsd).toBe(0.05);
  });
});
