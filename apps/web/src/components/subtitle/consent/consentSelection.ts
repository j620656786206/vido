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
import type {
  GenerationCandidate,
  ModelEstimate,
  TranslationModelInfo,
} from '../../../services/subtitleService';

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
  return new Set(
    candidates.filter((c) => c.route === 'extract' && isWritable(c)).map((c) => c.mediaId)
  );
}

/**
 * sub-6-1: a candidate whose folder failed the backend's write probe. The
 * pipeline would refuse it before spending, so the UI never lets it into a
 * selection — not by default, not by 全選, not by group toggle. `undefined`
 * (pre-sub-6-1 server) reads as writable.
 */
export function isWritable(c: GenerationCandidate): boolean {
  return c.writable !== false;
}

/**
 * The ids a bulk action may touch. Callers pass an already-listable array
 * (this module never re-derives listability), so this is the writable subset.
 */
export function selectableIds(candidates: GenerationCandidate[]): string[] {
  return candidates.filter(isWritable).map((c) => c.mediaId);
}

/** zh-TW copy for an unwritable row, composed from the code + folder name. */
export function blockerLabel(c: GenerationCandidate): string {
  return c.blockerDir ? `資料夾無法寫入：${c.blockerDir}` : '資料夾無法寫入';
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
  /** Listable AND writable — the only rows a selection may contain (sub-6-1). */
  selectableCount: number;
  /** Listable rows the backend's write probe refused. */
  unwritableCount: number;
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

/**
 * Per-media-id price under ONE chosen model (sub-6-8b AC #3) — the backend's
 * `estimates_by_model[<id>].per_candidate`. Undefined, or a row missing from
 * it, means "no per-model quote for this row": the row's own `estimatedUsd`
 * (the server's DEFAULT-model price) stands.
 */
export type ModelPrices = Readonly<Record<string, number>>;

/**
 * The price of ONE row under the chosen model. Every money figure on these
 * screens goes through here — the row, the group subtotal, the summary bar,
 * the footer, the confirm dialog and the F18 feasibility walk — so switching
 * model can never move four of them and leave the fifth behind.
 */
export function candidateUsd(c: GenerationCandidate, prices?: ModelPrices): number {
  const priced = prices?.[c.mediaId];
  return typeof priced === 'number' ? priced : c.estimatedUsd;
}

export function computeTotals(
  candidates: GenerationCandidate[],
  selectedIds: ReadonlySet<string>,
  budgetUsd: number | null,
  prices?: ModelPrices
): ConsentTotals {
  let selectedCount = 0;
  let extractCount = 0;
  let asrCount = 0;
  let extractUsd = 0;
  let asrUsd = 0;
  let feasibleCount = 0;
  let cumulative = 0;
  let selectableCount = 0;

  for (const c of candidates) {
    if (isWritable(c)) selectableCount++;
    if (!selectedIds.has(c.mediaId)) continue;
    selectedCount++;
    const rowUsd = candidateUsd(c, prices);
    if (c.route === 'extract') {
      extractCount++;
      extractUsd += rowUsd;
    } else {
      asrCount++;
      asrUsd += rowUsd;
    }
    if (budgetUsd === null || cumulative < budgetUsd) feasibleCount++;
    cumulative += rowUsd;
  }

  const totalUsd = extractUsd + asrUsd;
  return {
    candidateCount: candidates.length,
    selectableCount,
    unwritableCount: candidates.length - selectableCount,
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

// ─── sub-6-8b: the model choice ─────────────────────────────────────────────

/** One row of the F16/F19 「選擇翻譯模型」 radio list. */
export interface ModelChoice {
  id: string;
  displayName: string;
  isDefault: boolean;
  /** MEASURED grade; absent = 尚未評測 (never "equivalent"). */
  qualityGrade?: string;
  /** Provenance for the grade — which eval, which corpus. */
  qualityNote?: string;
  /** No graded row scores better. Drives the 「品質最穩」 copy without asserting it. */
  isBestGrade: boolean;
  /** THIS batch (the current selection) under this model. */
  totalUsd: number;
  /** 約 N 分鐘 for the current selection; undefined → the time column is hidden. */
  minutes?: number;
  /**
   * vs the DEFAULT model: positive = cheaper, negative = dearer. Undefined on
   * the default row itself and whenever the default is not quotable.
   */
  deltaUsd?: number;
  /** |deltaUsd| as a whole percent of the default's total; undefined when 0. */
  deltaPercent?: number;
}

export interface ModelChoiceInput {
  /** `GET /settings/models` — display order is the backend's. */
  models: TranslationModelInfo[];
  defaultModelId: string;
  /** Absent on a pre-sub-6-8a server → single default-model row (AC #2). */
  estimatesByModel?: Record<string, ModelEstimate>;
  estimatedMinutesByModel?: Record<string, number>;
}

/** Sum the SELECTED, writable rows at one model's prices. */
function sumSelected(
  candidates: GenerationCandidate[],
  selectedIds: ReadonlySet<string>,
  prices?: ModelPrices
): number {
  let total = 0;
  for (const c of candidates) {
    if (!selectedIds.has(c.mediaId) || !isWritable(c)) continue;
    total += candidateUsd(c, prices);
  }
  return Math.round(total * 100) / 100;
}

/**
 * Build the model rows for the CURRENT selection (sub-6-8b AC #1/#4).
 *
 * Every figure is derived from the backend's own numbers — the FE owns no
 * rate table (sub-7-6's 「不得另寫一份費率」 red line):
 *
 *  - money   = the selected rows summed at `per_candidate` prices.
 *  - minutes = the sweep's `estimated_minutes_by_model` RESCALED by the share
 *    of runtime the user actually selected. The backend's estimate is
 *    `runtime × share(model)` for every non-skipped row, so the ratio of
 *    selected runtime to sweep runtime carries it exactly — no share constant
 *    is duplicated here, and if the backend recalibrates, this follows.
 *
 * Returns [] when there is nothing to choose between (no key configured, or
 * the catalog request failed) — the caller then renders no picker at all
 * rather than a one-option question.
 */
export function modelChoices(
  candidates: GenerationCandidate[],
  selectedIds: ReadonlySet<string>,
  input: ModelChoiceInput
): ModelChoice[] {
  const { models, defaultModelId, estimatesByModel, estimatedMinutesByModel } = input;
  if (models.length === 0) return [];

  const quoted = (id: string) => estimatesByModel?.[id];
  const anyQuote = models.some((m) => quoted(m.id) !== undefined);

  // Pre-sub-6-8a server: only the default model has a price we can stand
  // behind (it is what `estimated_usd` already quotes). Offering the others
  // priced at the default's rate would be a lie in a money field.
  const rows = anyQuote
    ? models.filter((m) => quoted(m.id) !== undefined)
    : models.filter((m) => m.id === defaultModelId || m.isDefault);
  if (rows.length === 0) return [];

  // Denominator for the minutes rescale: the same set the backend summed —
  // every writable listable candidate, selected or not.
  let sweepRuntime = 0;
  let selectedRuntime = 0;
  for (const c of candidates) {
    if (!isWritable(c)) continue;
    sweepRuntime += c.runtimeMinutes;
    if (selectedIds.has(c.mediaId)) selectedRuntime += c.runtimeMinutes;
  }
  const runtimeShare = sweepRuntime > 0 ? selectedRuntime / sweepRuntime : 0;

  const grades = rows.map((r) => r.qualityGrade).filter((g): g is string => !!g);
  const bestGrade = grades.length > 0 ? [...grades].sort()[0] : undefined;

  const totalOf = (id: string) => sumSelected(candidates, selectedIds, quoted(id)?.perCandidate);
  const defaultRow = rows.find((r) => r.id === defaultModelId) ?? rows.find((r) => r.isDefault);
  const defaultTotal = defaultRow ? totalOf(defaultRow.id) : undefined;

  return rows.map((m) => {
    const totalUsd = totalOf(m.id);
    const sweepMinutes = estimatedMinutesByModel?.[m.id];
    const minutes =
      typeof sweepMinutes === 'number' ? Math.round(sweepMinutes * runtimeShare) : undefined;

    let deltaUsd: number | undefined;
    let deltaPercent: number | undefined;
    if (defaultRow && m.id !== defaultRow.id && defaultTotal !== undefined && defaultTotal > 0) {
      deltaUsd = Math.round((defaultTotal - totalUsd) * 100) / 100;
      if (deltaUsd !== 0) deltaPercent = Math.round((Math.abs(deltaUsd) / defaultTotal) * 100);
    }

    return {
      id: m.id,
      displayName: m.displayName,
      // The row the deployment pre-selects — driven by the server's effective
      // model, so a CLAUDE_MODEL override marks ITS model as the default here.
      isDefault: m.id === defaultModelId || m.isDefault,
      qualityGrade: m.qualityGrade,
      qualityNote: m.qualityNote,
      isBestGrade: !!m.qualityGrade && m.qualityGrade === bestGrade,
      totalUsd,
      minutes,
      deltaUsd,
      deltaPercent,
    };
  });
}
