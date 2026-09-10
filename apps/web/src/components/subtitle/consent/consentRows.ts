// Implements: <utility — no .pen counterpart>
/**
 * The F15 list's DISPLAY PROJECTION (sub-6-11) — search/sort/collapse turned
 * into one flat array of rows the panel can virtualise.
 *
 * 三序同源, restated for this story
 * ---------------------------------
 * `consentSelection.groupOrder` re-orders the candidates STATE, and that array
 * remains the ONE order used for submission and for the F18 feasibility walk.
 * Everything in this file is a PROJECTION of that array for the eye only:
 *
 *   state order   →  buildConsentRows  →  what the user scrolls
 *        │
 *        └────────────────────────────→  handleConfirm's id order
 *                                     →  computeTotals' cumulative walk
 *
 * So sorting by 金額高→低 moves rows on screen and moves NOTHING else. That is
 * also why the over-budget banner has to say 「依提交順序」 the moment a
 * non-default sort is active: 「預計可完成約 N 部」 counts down the state array,
 * not the one on screen, and a user reading a cost-sorted list would otherwise
 * assume the top N rows are the N that will run.
 *
 * Rule 23: no wall-clock reads here.
 */
import { gtUsd, ltUsd } from '../../../lib/currency';
import type { GenerationCandidate } from '../../../services/subtitleService';
import {
  candidateUsd,
  displayTitleOf,
  groupCandidates,
  type CandidateGroup,
  type ModelPrices,
} from './consentSelection';

/** F15 sort menu (sub-6-11 AC #2). `group` is the shipped grouped rendering. */
export type ConsentSort = 'group' | 'cost-desc' | 'cost-asc' | 'title-asc' | 'unmatched-first';

/**
 * The menu, in menu order.
 *
 * Every label is kept short enough to READ inside the phone-width control. The
 * sort control is one native <select> at both widths — on a phone the platform
 * already opens it as a picker sheet (AC #5's 「開 sheet」), so hand-rolling a
 * second sheet would only add a component that behaves worse than the OS one.
 */
export const CONSENT_SORTS: ReadonlyArray<{ value: ConsentSort; label: string }> = [
  { value: 'group', label: '群組' },
  { value: 'cost-desc', label: '金額高→低' },
  { value: 'cost-asc', label: '金額低→高' },
  { value: 'title-asc', label: '片名 A→Z' },
  { value: 'unmatched-first', label: '未匹配優先' },
];

/** Money compares through decimal.js, never `<`/`>` — the tech-money red line. */
function cmpUsd(a: number, b: number): number {
  if (gtUsd(a, b)) return 1;
  if (ltUsd(a, b)) return -1;
  return 0;
}

/**
 * One flat, sorted view of the visible rows (every sort but `group`).
 *
 * Every comparator falls back to the row's index in the STATE array, so the
 * order is total and deterministic: two $0.05 films always come back in
 * submission order rather than in whatever order the engine's sort happened to
 * leave them, and re-rendering never reshuffles the list under the user's
 * cursor.
 */
export function sortForDisplay(
  visible: GenerationCandidate[],
  sort: ConsentSort,
  stateOrder: ReadonlyMap<string, number>,
  prices?: ModelPrices
): GenerationCandidate[] {
  if (sort === 'group') return visible;
  const at = (c: GenerationCandidate) => stateOrder.get(c.mediaId) ?? 0;
  const rows = [...visible];
  switch (sort) {
    case 'cost-desc':
      rows.sort(
        (a, b) => cmpUsd(candidateUsd(b, prices), candidateUsd(a, prices)) || at(a) - at(b)
      );
      break;
    case 'cost-asc':
      rows.sort(
        (a, b) => cmpUsd(candidateUsd(a, prices), candidateUsd(b, prices)) || at(a) - at(b)
      );
      break;
    case 'title-asc':
      // zh-Hant collation, because these titles are mostly Chinese and a
      // code-point sort would order 沙丘 and 奧本海默 by their UTF-16 values —
      // an order no reader can predict or scan.
      rows.sort(
        (a, b) => displayTitleOf(a).localeCompare(displayTitleOf(b), 'zh-Hant') || at(a) - at(b)
      );
      break;
    case 'unmatched-first':
      // Strictly `=== false` again: an old server's silence is not a claim.
      rows.sort(
        (a, b) => Number(a.tmdbMatched !== false) - Number(b.tmdbMatched !== false) || at(a) - at(b)
      );
      break;
  }
  return rows;
}

/** A collapsible section header (series, season, or half of the movies block). */
export interface ConsentSectionRow {
  kind: 'section';
  /** React key AND the virtualizer's item key. */
  key: string;
  /** Stable id the collapse state is keyed by; survives search and filtering. */
  sectionId: string;
  /** The shipped `consent-group-*` / `consent-season-*` hook, kept verbatim. */
  testid: string;
  label: string;
  /** Hover copy that explains a label the header alone cannot ("未匹配"). */
  hint?: string;
  /**
   * Accessible name for the section's checkbox. 「選取整部 X」 is right for a
   * show and wrong for a block of films, so each section kind says its own.
   */
  selectLabel: string;
  /**
   * What ONE row of this section is called, for the sub-6-12 narrowed
   * accessible name (「選取顯示的 3 集」 vs 「…3 部」).
   */
  unit: '集' | '部';
  /** ALL the section's items. The 已選 x/n counts and the subtotal still speak
   *  for the whole section (that IS what the show costs); since sub-6-12 the
   *  CHECKBOX speaks for the visible subset instead, and the header prints
   *  「顯示 n」 whenever the two differ. */
  items: GenerationCandidate[];
  /** Season sub-headers render indented and smaller. */
  season: boolean;
  expanded: boolean;
}

export interface ConsentCandidateRow {
  kind: 'candidate';
  key: string;
  candidate: GenerationCandidate;
}

/**
 * The budget cut line (sub-6-12 AC #3) — 「到此為止約 $上限」.
 *
 * It is a ROW, not an overlay, for two reasons: the list is virtualised, so
 * anything drawn between two rows has to be an item the virtualizer can
 * measure and offset; and the fact it states is positional — the reader needs
 * it in the flow, at the boundary, not floating beside it.
 */
export interface ConsentCutRow {
  kind: 'cut';
  key: string;
}

export type ConsentRow = ConsentSectionRow | ConsentCandidateRow | ConsentCutRow;

/** One cut line per list; a fixed key keeps the virtualizer's identity stable. */
const CUT_ROW: ConsentCutRow = { kind: 'cut', key: 'budget-cut' };

/** Section id for each kind of header. One vocabulary, one place. */
export const sectionIds = {
  movies: (half: 'matched' | 'unmatched') => `movies:${half}`,
  series: (seriesId: string) => `series:${seriesId}`,
  season: (seriesId: string, seasonNumber: number) => `season:${seriesId}:${seasonNumber}`,
};

/**
 * Test hooks. The series and season forms are the ones sub-5-3 shipped and the
 * gallery fixtures already point at, so they are reproduced character for
 * character rather than regenerated from the new section ids.
 */
export const sectionTestids = {
  movies: (half: 'matched' | 'unmatched') => `consent-movies-${half}`,
  series: (seriesId: string) => `consent-group-${seriesId}`,
  season: (seriesId: string, seasonNumber: number) => `consent-season-${seriesId}-${seasonNumber}`,
};

/**
 * Is this section open before the user has touched it? (sub-6-11 AC #4)
 *
 * SERIES START CLOSED. A 2,400-episode library is the whole reason this story
 * exists: expanded, it is the unusable wall of rows the critique screenshot
 * showed; collapsed, it is one line per show, and the header already carries
 * 已選 x/n, the subtotal and the route composition, so the user can decide
 * about a whole show without opening it.
 *
 * Movie sections and season sub-sections start OPEN: a movie-only library that
 * opened onto two collapsed headers would be hiding everything it has, and a
 * season inside a show the user just expanded was expanded ON PURPOSE.
 */
export function sectionDefaultExpanded(sectionId: string): boolean {
  return !sectionId.startsWith('series:');
}

export interface BuildConsentRowsInput {
  /** The candidates STATE — already in groupOrder. */
  candidates: GenerationCandidate[];
  /** Rows the route chips AND the search box both let through. */
  visibleIds: ReadonlySet<string>;
  sort: ConsentSort;
  /** User overrides on top of sectionDefaultExpanded. */
  expandedOverride: Readonly<Record<string, boolean>>;
  /**
   * A search is running. Every rendered section opens: a section with no hit is
   * dropped entirely (below), so "sections that matched" and "sections still on
   * screen" are the same set — and a hit the user cannot see is a search that
   * did not work.
   */
  searching: boolean;
  prices?: ModelPrices;
  /**
   * sub-6-12 AC #3: the first SELECTED row the ceiling will not pay for
   * (`ConsentTotals.cutMediaId`). The cut line is emitted immediately before
   * that row wherever it lands in the display order — so under 金額高→低 it
   * travels with its film rather than staying at a position that no longer
   * means anything. `null`/absent, or a row the view filters have hidden, ⇒
   * no cut line: the dimmed rows and the F18 banner still carry the fact, and
   * a divider drawn at an arbitrary place would be a worse lie than no
   * divider.
   */
  cutMediaId?: string | null;
}

/**
 * CR: an explicit click OUTRANKS the search's auto-expand.
 *
 * The first version checked `searching` first, so during a search the
 * disclosure was pinned open while still being clickable — and the click wrote
 * `!(override ?? default)`, i.e. `!false === true` for a show. Clicking
 * 「collapse」 therefore did nothing on screen AND recorded "expanded", so the
 * show came back OPEN once the search was cleared: the exact opposite of what
 * was asked. Reading the override first makes the auto-expand a DEFAULT rather
 * than an override, which is what 「命中的群組自動展開」 means.
 */
function isExpanded(input: BuildConsentRowsInput, sectionId: string): boolean {
  const override = input.expandedOverride[sectionId];
  if (override !== undefined) return override;
  return input.searching || sectionDefaultExpanded(sectionId);
}

function movieSectionLabel(half: 'matched' | 'unmatched'): {
  label: string;
  hint: string;
  selectLabel: string;
} {
  return half === 'matched'
    ? { label: '已匹配', hint: '片名來自 TMDb', selectLabel: '選取所有已匹配的電影' }
    : {
        label: '未匹配',
        hint: 'TMDb 沒有比對到，片名由檔名解析',
        selectLabel: '選取所有未匹配的電影',
      };
}

/**
 * Flatten the grouped candidates into the exact row sequence to render.
 *
 * A section whose every row is filtered out emits NO header (the sub-5-3 CR L5
 * rule, now also covering search): an empty 「第 2 季」 bar under a search for a
 * film is noise, and worse, it reads as a hit.
 */
export function buildConsentRows(input: BuildConsentRowsInput): ConsentRow[] {
  const { candidates, visibleIds, sort, prices, cutMediaId } = input;

  const stateOrder = new Map(candidates.map((c, i) => [c.mediaId, i]));
  const visible = candidates.filter((c) => visibleIds.has(c.mediaId));

  if (sort !== 'group') {
    // A flat sorted list has no group structure left to collapse — 金額高→低
    // interleaves shows and films by definition. The row itself still says
    // which show an episode belongs to (its title carries SxxEyy), and
    // switching back to 群組 restores every header and every collapse state.
    const flat: ConsentRow[] = [];
    for (const candidate of sortForDisplay(visible, sort, stateOrder, prices)) {
      if (candidate.mediaId === cutMediaId) flat.push(CUT_ROW);
      flat.push({ kind: 'candidate', key: candidate.mediaId, candidate });
    }
    return flat;
  }

  const rows: ConsentRow[] = [];
  const pushRows = (items: GenerationCandidate[]) => {
    for (const c of items) {
      if (visibleIds.has(c.mediaId)) {
        if (c.mediaId === cutMediaId) rows.push(CUT_ROW);
        rows.push({ kind: 'candidate', key: c.mediaId, candidate: c });
      }
    }
  };
  const anyVisible = (items: GenerationCandidate[]) => items.some((c) => visibleIds.has(c.mediaId));

  for (const group of groupCandidates(candidates)) {
    if (!anyVisible(group.items)) continue;

    if (group.kind === 'movies') {
      // Unsplit movies keep the flat, header-less rendering that shipped.
      if (!group.movieSection) {
        pushRows(group.items);
        continue;
      }
      const sectionId = sectionIds.movies(group.movieSection);
      const expanded = isExpanded(input, sectionId);
      const { label, hint, selectLabel } = movieSectionLabel(group.movieSection);
      rows.push({
        kind: 'section',
        key: sectionId,
        sectionId,
        testid: sectionTestids.movies(group.movieSection),
        label,
        hint,
        selectLabel,
        unit: '部',
        items: group.items,
        season: false,
        expanded,
      });
      if (expanded) pushRows(group.items);
      continue;
    }

    pushSeriesRows(rows, group, input, pushRows, anyVisible);
  }

  return rows;
}

function pushSeriesRows(
  rows: ConsentRow[],
  group: CandidateGroup,
  input: BuildConsentRowsInput,
  pushRows: (items: GenerationCandidate[]) => void,
  anyVisible: (items: GenerationCandidate[]) => boolean
): void {
  const seriesId = group.seriesId ?? '';
  const sectionId = sectionIds.series(seriesId);
  const expanded = isExpanded(input, sectionId);
  rows.push({
    kind: 'section',
    key: sectionId,
    sectionId,
    testid: sectionTestids.series(seriesId),
    label: group.seriesTitle || '未知影集',
    selectLabel: `選取整部 ${group.seriesTitle || '未知影集'}`,
    unit: '集',
    items: group.items,
    season: false,
    expanded,
  });
  if (!expanded) return;

  if (!group.showSeasonHeaders || !group.seasons) {
    pushRows(group.items);
    return;
  }
  for (const season of group.seasons) {
    if (!anyVisible(season.items)) continue;
    const seasonId = sectionIds.season(seriesId, season.seasonNumber);
    const seasonExpanded = isExpanded(input, seasonId);
    rows.push({
      kind: 'section',
      key: seasonId,
      sectionId: seasonId,
      testid: sectionTestids.season(seriesId, season.seasonNumber),
      label: seasonLabel(season.seasonNumber),
      selectLabel: `選取${seasonLabel(season.seasonNumber)}`,
      unit: '集',
      items: season.items,
      season: true,
      expanded: seasonExpanded,
    });
    if (seasonExpanded) pushRows(season.items);
  }
}

/** S00 is the conventional specials season. */
export function seasonLabel(n: number): string {
  return n === 0 ? '特別篇' : `第 ${n} 季`;
}
