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

// ─── sub-5-3 AC #2: series/season grouping ──────────────────────────────────

/** One season's rows inside a series section. */
export interface CandidateSeasonSection {
  seasonNumber: number;
  items: GenerationCandidate[];
}

/**
 * One rendered section of the F15 list. `movies` renders as flat rows (the
 * pre-sub-5-3 shipped form — no header); `series` gets a header row with the
 * 整劇 checkbox, and season sub-headers only when the show spans ≥2 seasons
 * (single-season shows would just repeat the series header's meaning).
 */
export interface CandidateGroup {
  kind: 'movies' | 'series';
  /** series sections only. seriesTitle '' = backend lookup degraded (未知影集). */
  seriesId?: string;
  seriesTitle?: string;
  showSeasonHeaders?: boolean;
  seasons?: CandidateSeasonSection[];
  /** All of the section's items in display order (= submission order). */
  items: GenerationCandidate[];
}

/**
 * Group candidates for display: one flat movies section first (input order),
 * then one section per series ordered BY SERIES TITLE; within a series, season
 * asc then episode asc (the backend's global (title,id) sort shuffles episodes
 * — episode_number exists on the wire precisely for this).
 *
 * CR M2: series sections were originally emitted in first-APPEARANCE order,
 * which is a function of the backend's per-EPISODE (title, id) sort — so the
 * show order looked random next to the alphabetical movie rows, and adding one
 * early-sorting episode title could jump a whole series to the top. Sorting by
 * the series' own title makes the section order mean something and stay put.
 * Degraded (empty) titles sort LAST — an unlabelled 未知影集 block belongs at
 * the bottom, not ahead of every named show.
 */
export function groupCandidates(candidates: GenerationCandidate[]): CandidateGroup[] {
  const movies: GenerationCandidate[] = [];
  const bySeries = new Map<string, GenerationCandidate[]>();

  for (const c of candidates) {
    if (c.seriesId) {
      const bucket = bySeries.get(c.seriesId);
      if (bucket) bucket.push(c);
      else bySeries.set(c.seriesId, [c]);
    } else {
      movies.push(c);
    }
  }

  const groups: CandidateGroup[] = [];
  if (movies.length > 0) groups.push({ kind: 'movies', items: movies });

  const seriesSections: CandidateGroup[] = [];
  for (const [seriesId, items] of bySeries) {
    const sorted = [...items].sort(
      (a, b) =>
        (a.seasonNumber ?? 0) - (b.seasonNumber ?? 0) ||
        (a.episodeNumber ?? 0) - (b.episodeNumber ?? 0)
    );
    const seasons: CandidateSeasonSection[] = [];
    for (const item of sorted) {
      const season = item.seasonNumber ?? 0;
      const last = seasons[seasons.length - 1];
      if (last && last.seasonNumber === season) last.items.push(item);
      else seasons.push({ seasonNumber: season, items: [item] });
    }
    seriesSections.push({
      kind: 'series',
      seriesId,
      seriesTitle: sorted[0].seriesTitle ?? '',
      showSeasonHeaders: seasons.length >= 2,
      seasons,
      items: sorted,
    });
  }

  seriesSections.sort((a, b) => {
    const at = a.seriesTitle ?? '';
    const bt = b.seriesTitle ?? '';
    // Untitled (backend lookup degraded) sinks below every named show; the id
    // is the deterministic tie-break so the order never depends on Map order.
    if (at === '' || bt === '') return at === bt ? 0 : at === '' ? 1 : -1;
    if (at !== bt) return at < bt ? -1 : 1;
    return (a.seriesId ?? '') < (b.seriesId ?? '') ? -1 : 1;
  });

  return [...groups, ...seriesSections];
}

/**
 * 三序同源紅線: the flattened grouped order. `seedList` re-orders the
 * candidates STATE with this, so the display order, the submitted id order and
 * the F18 feasibleCount cumulative walk are one and the same array — three
 * independently-maintained orders are the drift this function exists to ban.
 */
export function groupOrder(candidates: GenerationCandidate[]): GenerationCandidate[] {
  return groupCandidates(candidates).flatMap((g) => g.items);
}
