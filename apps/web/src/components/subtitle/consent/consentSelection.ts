// Implements: <utility — no .pen counterpart>
/**
 * Pure selection/estimate math for the cost-consent screens (sub-4-3 AC #2/#3).
 *
 * ONE selector computes every money figure — the F15 summary bar, the footer
 * left segment, its detail line and the F16/F19 confirm breakdown all render
 * from the SAME ConsentTotals value (三處金額同源; three independent sums are
 * banned by the story AC). Amounts come VERBATIM from the backend's
 * `estimated_usd` (§5-sexies: no "免費" rounding presentation).
 *
 * tech-money-decimal-arithmetic (Alexyu, 2026-09-04): NO money in this file is
 * added, subtracted or compared with `+ - < >`. Those are IEEE-754 double
 * operations, and `0.1 + 0.2 === 0.30000000000000004` is how a confirm screen
 * ends up printing 「$0.01 + $0.01 = $0.01」, or how a $0.80 ceiling fails to
 * trigger on a total that is exactly $0.80. Everything goes through
 * `lib/currency`, which is decimal.js — the same arithmetic the Go backend
 * does with shopspring/decimal, so the quote and the invoice are the same
 * number rather than merely close.
 */
import { addUsd, gtUsd, ltUsd, percentOfUsd, roundUsd, subUsd, usd } from '../../../lib/currency';
import type { KeySource } from '../../../services/keySettingsService';
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
 * Is this row's runtime a GUESS? (sub-6-10b AC #2, moved here by sub-6-12)
 *
 * Prefer `runtimeSource` — it distinguishes "measured from the file" from
 * "TMDb's editorial figure", which `runtimeKnown` cannot. A pre-sub-6-10a
 * server sends no source, so fall back to the old flag and behave exactly as
 * the row did before.
 *
 * It lives in this module because sub-6-12 AC #4 makes it a MONEY fact: the
 * row's own `≈`, the summary total's `≈`, the footer's and the confirm
 * dialog's all have to answer it the same way, and a second copy of this
 * predicate in the panel is how a row would say 「約」 while the total beside
 * it claimed to be exact.
 */
export function isRuntimeApproximate(c: GenerationCandidate): boolean {
  return c.runtimeSource ? c.runtimeSource === 'fallback' : !c.runtimeKnown;
}

/**
 * The ids a bulk action may touch. Callers pass an already-listable array
 * (this module never re-derives listability), so this is the writable subset.
 */
export function selectableIds(candidates: GenerationCandidate[]): string[] {
  return candidates.filter(isWritable).map((c) => c.mediaId);
}

/**
 * The ids a bulk action may touch RIGHT NOW (sub-6-12 AC #1).
 *
 * 全選 and the 整劇/整季 headers select what the user can SEE. Before this
 * story they swept the whole list: filter to 需語音辨識, tick 全選, and the
 * 2,399 hidden extract rows came with it — the critique's「篩選＋全選＝金錢
 * 陷阱」, and at group scope the same trap cost $2.79 for ticking a show whose
 * one visible episode was the only one the user meant (sub-6-11 handed that
 * one over deliberately).
 *
 * `visibleIds` absent = no view filter is running, so the visible set IS the
 * whole list — which keeps every pre-sub-6-12 caller behaving as it did.
 */
export function visibleSelectableIds(
  candidates: GenerationCandidate[],
  visibleIds?: ReadonlySet<string>
): string[] {
  return candidates
    .filter((c) => isWritable(c) && (visibleIds === undefined || visibleIds.has(c.mediaId)))
    .map((c) => c.mediaId);
}

/** What a row should READ as — the backend's honest title, else what shipped before. */
export function displayTitleOf(c: GenerationCandidate): string {
  return c.displayTitle || c.title;
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

// ─── sub-6-11 AC #1: search ─────────────────────────────────────────────────

/**
 * Fold a string into the form both sides of a search comparison use:
 * case-insensitive and whitespace-insensitive.
 *
 * Whitespace is stripped rather than collapsed because the two strings being
 * compared were produced by different machines — TMDb writes 「怪奇物語」, the
 * filename parser writes 「怪奇 物語」 out of a dotted release name — and a user
 * typing either one means the same show. CJK titles carry no word boundaries
 * for a space to be significant at, and for latin titles the cost is only that
 * 「the matrix」 also matches 「thematrix」.
 */
export function normalizeSearch(text: string): string {
  return text.replace(/\s+/g, '').toLowerCase();
}

/**
 * The haystack one row is searched by (sub-6-11 AC #1).
 *
 * THE WIRE CARRIES NO RAW FILENAME. The story's 「片名或檔名」 reads as three
 * fields but `GenerationCandidate` has only these:
 *
 *  - `displayTitle` — what the row READS as (TMDb's title, or, for an unmatched
 *    row, the filename parser's cleaned-up name — which IS the filename, minus
 *    the release junk).
 *  - `title`        — the stored title. On an unmatched row this is the raw,
 *    ugly string the user sees in their file manager, so searching it is the
 *    closest thing to searching the filename that exists here.
 *  - `seriesTitle`  — an episode's own title is 「S04E07」 or 「片名 S04E07」 and
 *    never contains the SHOW name, so without this, typing a show name would
 *    match nothing in a library that is mostly TV.
 *
 * Adding a real `file_path` would be a backend change; this story is frontend
 * only, and inventing a field the API does not send is worse than searching the
 * three it does.
 */
export function candidateSearchText(c: GenerationCandidate): string {
  return normalizeSearch(`${c.displayTitle ?? ''} ${c.title} ${c.seriesTitle ?? ''}`);
}

/**
 * Narrow to rows matching the query. A VIEW filter, exactly like the route
 * chips: it never changes the candidates state, the submission order or what a
 * group header counts.
 */
export function applySearch(
  candidates: GenerationCandidate[],
  query: string
): GenerationCandidate[] {
  const needle = normalizeSearch(query);
  if (needle === '') return candidates;
  return candidates.filter((c) => candidateSearchText(c).includes(needle));
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
  /**
   * sub-6-12 AC #1: the same three counts, restricted to what the view
   * filters (route chip ∘ search) are letting through. `visibleIds` absent ⇒
   * these equal `selectableCount` / `selectedCount`.
   */
  visibleSelectableCount: number;
  visibleSelectedCount: number;
  /**
   * What the visible-and-selected rows cost. The 全選 checkbox and the group
   * headers speak for the VISIBLE set, so when a filter is narrowing them their
   * count and their money have to describe the same rows — a header reading
   * 「已選 0 / 顯示 1 · $1.55」 (the section total, including four episodes off
   * screen) looks like a bug. The whole-list truth never moves: the summary bar
   * and the footer render `selectedTotalUsd` regardless of any view filter.
   */
  visibleSelectedTotalUsd: number;
  /**
   * sub-6-12 AC #4: at least one SELECTED row is priced off the 45-minute
   * assumption, so every total computed from this value must render with `≈`.
   * On the owner's 2026-09-03 production screenshot this was every single row
   * while the total read a flat `$13.92`.
   */
  hasEstimatedRows: boolean;
  /** How many selected rows that is — the F16 line names the number. */
  estimatedRowCount: number;
  /**
   * sub-6-12 AC #3 — where the budget stops paying, as a MEDIA ID rather than
   * an index.
   *
   * The list is sorted for the eye (sub-6-11) while the ceiling is walked in
   * submission order, so an index into the display would point at the wrong
   * film the moment the user picks 金額高→低. The id follows its row wherever
   * that row is drawn, which is what「分隔列跟著提交順序的列走」means.
   *
   * `null` when nothing gets cut — including the case where the ONE selected
   * row is dearer than the whole ceiling: it still runs (the backend checks
   * BEFORE each call, not after), so there is no cut line to draw even though
   * `overBudget` is true.
   */
  cutMediaId: string | null;
  /** Selected rows at or beyond the cut — the ones that will be paused. */
  pausedIds: ReadonlySet<string>;
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
  // Rounded to cents HERE, before it can enter any sum. Exact addition alone
  // is not enough: a row worth 0.005 renders as $0.01, and two of them render
  // a $0.01 total — the breakdown fails to add up even though the arithmetic
  // was perfect, because the rounding happened at DISPLAY time instead. Round
  // the atoms and every level above is consistent by construction: the row,
  // the group subtotal, the two route halves and the grand total.
  //
  // In practice this is a no-op — estimateUSD already rounds every wire value
  // to whole cents — which is exactly what a guard should be.
  return roundUsd(typeof priced === 'number' ? priced : c.estimatedUsd);
}

export function computeTotals(
  candidates: GenerationCandidate[],
  selectedIds: ReadonlySet<string>,
  budgetUsd: number | null,
  prices?: ModelPrices,
  /**
   * sub-6-12 AC #1: the rows the route chip and the search box are letting
   * through. Absent = no view filter, so the visible set is the whole list.
   * It NEVER changes a money figure — only the 「顯示 n」 counts the bulk
   * controls speak for.
   */
  visibleIds?: ReadonlySet<string>
): ConsentTotals {
  let selectedCount = 0;
  let extractCount = 0;
  let asrCount = 0;
  let extractUsd = 0;
  let asrUsd = 0;
  let feasibleCount = 0;
  let cumulative = 0;
  let selectableCount = 0;
  let visibleSelectableCount = 0;
  let visibleSelectedCount = 0;
  let visibleSelectedUsd = 0;
  let estimatedRowCount = 0;
  let cutMediaId: string | null = null;
  const pausedIds = new Set<string>();

  for (const c of candidates) {
    const visible = visibleIds === undefined || visibleIds.has(c.mediaId);
    if (isWritable(c)) {
      selectableCount++;
      if (visible) visibleSelectableCount++;
    }
    if (!selectedIds.has(c.mediaId)) continue;
    selectedCount++;
    if (isRuntimeApproximate(c)) estimatedRowCount++;
    const rowUsd = candidateUsd(c, prices);
    if (visible && isWritable(c)) {
      visibleSelectedCount++;
      visibleSelectedUsd = addUsd(visibleSelectedUsd, rowUsd);
    }
    if (c.route === 'extract') {
      extractCount++;
      extractUsd = addUsd(extractUsd, rowUsd);
    } else {
      asrCount++;
      asrUsd = addUsd(asrUsd, rowUsd);
    }
    if (budgetUsd === null || ltUsd(cumulative, budgetUsd)) {
      feasibleCount++;
    } else {
      // The FIRST row the ceiling refuses is where the divider goes; it and
      // everything after it in submission order is what「之後的項目會暫停」
      // names.
      if (cutMediaId === null) cutMediaId = c.mediaId;
      pausedIds.add(c.mediaId);
    }
    cumulative = addUsd(cumulative, rowUsd);
  }

  // The total is the two DISPLAYED halves added exactly — so the breakdown a
  // user reads always sums to the total beside it.
  const totalUsd = addUsd(extractUsd, asrUsd);
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
    overBudget: budgetUsd !== null && gtUsd(totalUsd, budgetUsd),
    feasibleCount,
    visibleSelectableCount,
    visibleSelectedCount,
    visibleSelectedTotalUsd: visibleSelectedUsd,
    hasEstimatedRows: estimatedRowCount > 0,
    estimatedRowCount,
    cutMediaId,
    pausedIds,
  };
}

/**
 * A total, marked `≈` when the rows under it are priced off an assumed length
 * (sub-6-12 AC #4).
 *
 * It lives beside computeTotals rather than in each component because 三處金額
 * 同源 now covers the MARKER as well as the number: the summary bar, the
 * footer and the F16/F19 confirm dialog must all say 「約」 or all say nothing.
 * A screen where every row reads 「≈ $0.02」 and the total reads a flat
 * 「$13.92」 is the false precision the critique's P2 named.
 */
export function usdWithEstimate(value: number, estimated: boolean): string {
  return estimated ? `≈ ${usd(value)}` : usd(value);
}

/**
 * 「扣誰的錢」 (sub-6-12 AC #6) — the one line that answers the project
 * persona's red flag:「花錢的事先問」asked HOW MUCH and never asked WHOSE key.
 *
 * Both halves come from facts the app already holds — the key resolver's
 * `source` (which is also what the 金鑰設定 page renders) and the sweep's own
 * `self_hosted_asr` — so this never guesses. `undefined` sources mean the key
 * settings request has not answered yet; it reads as 尚未設定, which is what a
 * fresh NAS actually is.
 */
export function spendSourceLabel(input: {
  claudeSource?: KeySource;
  openaiSource?: KeySource;
  /** The sweep's summary flag: ASR runs on the operator's own engine. */
  selfHostedAsr: boolean;
  /** The list has ASR-route rows at all — otherwise the ASR half is noise. */
  hasAsrCandidates: boolean;
}): string {
  const word = (source?: KeySource) =>
    source === 'secret' ? '你的金鑰' : source === 'env' ? '環境變數金鑰' : '尚未設定金鑰';
  const parts = [`使用：Claude（${word(input.claudeSource)}）`];
  if (input.hasAsrCandidates) {
    parts.push(
      input.selfHostedAsr
        ? '語音辨識：自架（不另計費）'
        : `語音辨識：OpenAI（${word(input.openaiSource)}）`
    );
  }
  return parts.join(' · ');
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
  /**
   * sub-6-11 AC #4: which half of the split movies block this is. Absent means
   * the movies were NOT split — one flat, header-less section, the shipped
   * pre-sub-6-11 rendering.
   */
  movieSection?: 'matched' | 'unmatched';
  /** series sections only. seriesTitle '' = backend lookup degraded (未知影集). */
  seriesId?: string;
  seriesTitle?: string;
  showSeasonHeaders?: boolean;
  seasons?: CandidateSeasonSection[];
  /** All of the section's items in display order (= submission order). */
  items: GenerationCandidate[];
}

/**
 * Group candidates for display: the movies first (input order, split into
 * 已匹配 / 未匹配 by movieSections when both exist), then one section per
 * series ordered BY SERIES TITLE; within a series, season
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
/**
 * Split the movies block into 已匹配 / 未匹配 (sub-6-11 AC #4).
 *
 * ORDER — MATCHED FIRST (Alexyu 裁定 2026-09-09). This array is also the
 * SUBMISSION order (see groupOrder), so whichever section leads is the one the
 * budget ceiling spends on first, and money should go to the rows whose
 * identity is settled rather than to the parser's guesses. The unmatched
 * section sinks the way the untitled 未知影集 block already does — still
 * present, still named by its own header, so it cannot be missed.
 *
 * The split only happens when there is something to separate. A library where
 * every movie matched — and EVERY pre-sub-6-10a server, which sends no
 * `tmdb_matched` at all — gets one unlabelled, header-less section: exactly the
 * flat rendering that shipped.
 */
function movieSections(movies: GenerationCandidate[]): CandidateGroup[] {
  if (movies.length === 0) return [];
  // Strictly `=== false`: "the server never told us" is not "TMDb found
  // nothing" — the same rule the row's 未匹配 badge follows.
  const unmatched = movies.filter((c) => c.tmdbMatched === false);
  if (unmatched.length === 0 || unmatched.length === movies.length) {
    return [{ kind: 'movies', items: movies }];
  }
  return [
    {
      kind: 'movies',
      movieSection: 'matched',
      items: movies.filter((c) => c.tmdbMatched !== false),
    },
    { kind: 'movies', movieSection: 'unmatched', items: unmatched },
  ];
}

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
  groups.push(...movieSections(movies));

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

/**
 * Measured grades, best first. CR L6: the first version took `grades.sort()[0]`,
 * which only worked because the catalog happens to use A and B and 'A' < 'B'.
 * A future grade whose letter sorts ahead of a genuinely better one would have
 * handed the 「eval-1 實測品質最穩」 claim to the wrong model — the precise
 * false claim this screen must never make. An UNRECOGNISED grade ranks nowhere
 * and can never be "best": we do not know what it means.
 */
const GRADE_ORDER = ['A', 'B', 'C', 'D', 'F'];

/** Sum the SELECTED, writable rows at one model's prices. */
function sumSelected(
  candidates: GenerationCandidate[],
  selectedIds: ReadonlySet<string>,
  prices?: ModelPrices
): number {
  let total = 0;
  for (const c of candidates) {
    if (!selectedIds.has(c.mediaId) || !isWritable(c)) continue;
    total = addUsd(total, candidateUsd(c, prices));
  }
  return roundUsd(total);
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
 *    selected runtime to sweep runtime carries the model's time share through
 *    without duplicating that constant here — if the backend recalibrates,
 *    this follows. It is NOT exact to the minute (CR L7): the sweep figure
 *    arrives already rounded, so rescaling and rounding again can land a
 *    minute off a from-scratch computation. Acceptable for a display-only
 *    「約 N 分鐘」; it would not be for a money field.
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

  const bestGrade = rows
    .map((r) => r.qualityGrade)
    .filter((g): g is string => !!g && GRADE_ORDER.includes(g))
    .sort((a, b) => GRADE_ORDER.indexOf(a) - GRADE_ORDER.indexOf(b))[0];

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
    if (
      defaultRow &&
      m.id !== defaultRow.id &&
      defaultTotal !== undefined &&
      gtUsd(defaultTotal, 0)
    ) {
      deltaUsd = roundUsd(subUsd(defaultTotal, totalUsd));
      if (deltaUsd !== 0) deltaPercent = percentOfUsd(deltaUsd, defaultTotal);
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
