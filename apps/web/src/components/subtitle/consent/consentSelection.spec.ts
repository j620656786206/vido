import { describe, it, expect } from 'vitest';
import {
  applyRouteFilter,
  computeTotals,
  defaultSelection,
  listableCandidates,
  parseBudgetInput,
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
