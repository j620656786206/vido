// Implements: <utility — no .pen counterpart>
/**
 * Pure selection/estimate math for the cost-consent screens (sub-4-3 AC #2/#3).
 *
 * ONE selector computes every money figure — the F15 summary bar, the footer
 * left segment, its detail line and the F16/F19 confirm breakdown all render
 * from the SAME ConsentTotals value (三處金額同源; three independent sums are
 * banned by the story AC). Amounts come VERBATIM from the backend's
 * `estimated_usd` (§5-sexies: no "免費" rounding presentation).
 */
import type { GenerationCandidate } from '../../../services/subtitleService';

/** F15 route filter chips. */
export type ConsentRouteFilter = 'all' | 'extract' | 'asr';

/**
 * Candidates the consent list works with: extract+asr only, in backend order
 * (which is also batch submission order). `route=skip` rows carry no quote and
 * have no list vocabulary in the design — they never reach the UI.
 */
export function listableCandidates(candidates: GenerationCandidate[]): GenerationCandidate[] {
  return candidates.filter((c) => c.route === 'extract' || c.route === 'asr');
}

/** 準則④ (as amended by §5-sexies): default selection = the lowest-cost set —
 * every extract-route candidate; paid ASR is NEVER pre-selected. */
export function defaultSelection(candidates: GenerationCandidate[]): Set<string> {
  return new Set(candidates.filter((c) => c.route === 'extract').map((c) => c.mediaId));
}

export function applyRouteFilter(
  candidates: GenerationCandidate[],
  filter: ConsentRouteFilter
): GenerationCandidate[] {
  if (filter === 'all') return candidates;
  return candidates.filter((c) => c.route === filter);
}

export interface ConsentTotals {
  /** Listable candidates (extract+asr). */
  candidateCount: number;
  selectedCount: number;
  selectedExtractCount: number;
  selectedAsrCount: number;
  selectedExtractUsd: number;
  selectedAsrUsd: number;
  /** = selectedExtractUsd + selectedAsrUsd — the ONE total everything renders. */
  selectedTotalUsd: number;
  /** Estimated total exceeds the entered ceiling → F18/F19 presentation. */
  overBudget: boolean;
  /**
   * 「預計可完成約 N 部」(F18/F19): walking the SELECTED candidates in list
   * order (= submission order), an item runs iff the cumulative estimate
   * BEFORE it is still under the ceiling — mirrors the backend's
   * check-before-each-paid-call soft-ceiling semantics. An estimate, never a
   * promise (the copy says 預計/約).
   */
  feasibleCount: number;
}

export function computeTotals(
  candidates: GenerationCandidate[],
  selectedIds: ReadonlySet<string>,
  budgetUsd: number | null
): ConsentTotals {
  let selectedCount = 0;
  let extractCount = 0;
  let asrCount = 0;
  let extractUsd = 0;
  let asrUsd = 0;
  let feasibleCount = 0;
  let cumulative = 0;

  for (const c of candidates) {
    if (!selectedIds.has(c.mediaId)) continue;
    selectedCount++;
    if (c.route === 'extract') {
      extractCount++;
      extractUsd += c.estimatedUsd;
    } else {
      asrCount++;
      asrUsd += c.estimatedUsd;
    }
    if (budgetUsd === null || cumulative < budgetUsd) feasibleCount++;
    cumulative += c.estimatedUsd;
  }

  const totalUsd = extractUsd + asrUsd;
  return {
    candidateCount: candidates.length,
    selectedCount,
    selectedExtractCount: extractCount,
    selectedAsrCount: asrCount,
    selectedExtractUsd: extractUsd,
    selectedAsrUsd: asrUsd,
    selectedTotalUsd: totalUsd,
    overBudget: budgetUsd !== null && totalUsd > budgetUsd,
    feasibleCount,
  };
}

/**
 * Parse the budget input field. Returns the positive number, or null when the
 * text is not a valid ceiling — client-side mirror of the server's
 * `budget_usd must be > 0` 400 rule (0 must NEVER read as "unlimited").
 */
export function parseBudgetInput(text: string): number | null {
  const v = Number(text);
  if (!Number.isFinite(v) || v <= 0) return null;
  return v;
}
