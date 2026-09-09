// Design ref: ux-design.pen Screen F15-D-v2 (pwMzT) · F15-M-v2 (fdu4y) · F18-D-v2 (zBik1)
/**
 * F15 產生字幕．候選清單 (sub-4-3 AC #2/#3) — the core consent screen; F18 is
 * the same panel in its over-budget state; F15-M is the responsive variant
 * (the host dialog is already a mobile bottom sheet, and since
 * bugfix-f15-row-mobile-identity-collapse the ROW itself re-flows below 36rem
 * of list width — see CandidateRow).
 *
 * Money honesty (§5-sexies 2026-08-11): every amount renders the backend's
 * `estimated_usd` VERBATIM through the shared usd() formatter — the extract
 * route shows its small translation fee (success green), never "免費". The
 * summary bar, footer segment and its detail line all render from ONE
 * ConsentTotals value (三處金額同源).
 *
 * Soft ceiling: the over-budget banner's 「預計可完成約 N 部」 is the
 * list-order prefix-sum estimate (consentSelection.feasibleCount) and the copy
 * never promises the ceiling cannot be exceeded.
 */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useVirtualizer } from '@tanstack/react-virtual';
import { ArrowUpDown, ChevronRight, CircleAlert, Loader2, Search, X } from 'lucide-react';
import { cn } from '../../../lib/utils';
import { usd } from '../../../lib/currency';
import { getImageUrl } from '../../../lib/image';
import { formatRuntime } from '../../../lib/formatMedia';
import type { GenerationCandidate } from '../../../services/subtitleService';
import {
  blockerLabel,
  candidateUsd,
  computeTotals,
  displayTitleOf,
  isRuntimeApproximate,
  isWritable,
  spendSourceLabel,
  usdWithEstimate,
  visibleSelectableIds,
  type ConsentRouteFilter,
  type ConsentTotals,
  type ModelPrices,
} from './consentSelection';
import { buildConsentRows, CONSENT_SORTS, type ConsentRow, type ConsentSort } from './consentRows';
import type { KeySource } from '../../../services/keySettingsService';

/**
 * Rows below which the list renders WHOLE, with no virtualizer (sub-6-11 AC #3).
 *
 * Virtualization is not free: it takes the browser's own Ctrl-F, printing and
 * "scroll to a row you just left" away from the user, and it re-measures on
 * every resize. Under a hundred rows it buys nothing — the DOM was never the
 * problem there — so the small library keeps the plain list it shipped with,
 * and the 2,400-row library gets the window. Both paths render the SAME row
 * components from the SAME ConsentRow array; only the slice differs.
 */
const VIRTUALIZE_FROM_ROWS = 80;

/** Row-height seeds for the virtualizer. Real heights come from measureElement. */
const ESTIMATED_SECTION_PX = 48;
const ESTIMATED_ROW_PX = 86;
/** The sub-6-12 budget divider — a rule and one line of 12px text. */
const ESTIMATED_CUT_PX = 24;
/** The `gap-2` between list items, told to the virtualizer so its offsets agree
 *  with what flexbox actually draws (measureElement reports height, not gap). */
const LIST_GAP_PX = 8;

export interface CandidateListPanelProps {
  /** Listable candidates (extract+asr, backend order = submission order). */
  candidates: GenerationCandidate[];
  selectedIds: ReadonlySet<string>;
  filter: ConsentRouteFilter;
  /** The ONE totals value every money figure renders from. */
  totals: ConsentTotals;
  /**
   * sub-6-8b: per-row prices under the model the user picked in F16/F19.
   * Absent = the backend's default-model quote (`estimatedUsd`) stands. This
   * panel does not choose the model; it renders whatever the container's ONE
   * selector priced, so the rows, the group subtotals, the summary bar and the
   * footer always agree.
   */
  prices?: ModelPrices;
  /** Raw budget input text (controlled). */
  budgetText: string;
  /** parseBudgetInput(budgetText) — null = invalid (<=0 / not a number). */
  budgetUsd: number | null;
  starting?: boolean;
  startError?: string | null;
  /**
   * The rows the route chip AND the search box are both letting through
   * (sub-6-11 Dev Notes), computed ONCE by the container.
   *
   * It lives up there rather than here because since sub-6-12 it decides what
   * the bulk controls SELECT, not merely what the list draws — and 全選 is a
   * container action. One set, one meaning: the same ids feed the projection,
   * the 顯示 n counts and every toggle.
   */
  visibleIds: ReadonlySet<string>;
  onToggle: (mediaId: string) => void;
  /**
   * 整劇/整季 group toggle (sub-5-3 AC #2, re-scoped by sub-6-12 AC #1) — the
   * header hands up the ids it is standing next to: this section's writable
   * rows that the chips and the search are letting through. `next` is the
   * target state.
   *
   * sub-6-11 shipped it over the section's ALL items and handed the hazard
   * over on purpose: search 「S04E07」, see one episode under a nine-episode
   * show, tick the header, consent to about $2.79 of episodes never displayed.
   * Alexyu ruled on 2026-09-09 that the checkbox selects what the user can see,
   * the same rule 全選 now follows.
   */
  onToggleGroup: (mediaIds: string[], next: boolean) => void;
  onToggleAll: () => void;
  onSelectAllExtract: () => void;
  onClearSelection: () => void;
  onFilterChange: (filter: ConsentRouteFilter) => void;
  onBudgetTextChange: (text: string) => void;
  onStartClick: () => void;
  /**
   * sub-6-11 AC #1 — the two halves of the search box.
   *
   * `search` is what the user has typed THIS KEYSTROKE and is what the input
   * renders; `searchQuery` is the same text after the container's 200ms
   * debounce and is what the list actually filters by. Keeping them apart is
   * what stops a 2,400-row list re-projecting on every character while still
   * letting the caret behave like a normal text field.
   */
  search: string;
  searchQuery: string;
  sort: ConsentSort;
  onSearchChange: (text: string) => void;
  onSortChange: (sort: ConsentSort) => void;
  /**
   * sub-6-12 AC #6 —「扣誰的錢」. Where the Claude and cloud-ASR keys resolved
   * from (the 金鑰設定 page's own `source`), and whether this deployment runs
   * ASR on its own hardware (the sweep's `self_hosted_asr`). Undefined sources
   * = the key settings request has not answered; the line then reads 尚未設定,
   * which is what a fresh NAS actually is.
   */
  claudeKeySource?: KeySource;
  openaiKeySource?: KeySource;
  selfHostedAsr?: boolean;
}

/** The runtime half of the subtitle. Empty when there is nothing honest to say. */
function runtimeLabel(c: GenerationCandidate): string {
  const minutes = Math.round(c.runtimeMinutes);
  if (minutes <= 0) return '';
  // Sally 裁定 1 (2026-09-05): NOT `≈` — 45 minutes is an ASSUMPTION, not an
  // approximate measurement, and the amount's own ≈ already means "this figure
  // rests on an assumed length". One ≈ per row, one meaning.
  if (isRuntimeApproximate(c)) return `片長未知（估 ${minutes} 分）`;
  // formatRuntime is the shipped zh-TW form ("1 小時 52 分") — the PosterCard
  // and the detail page already say it that way, and a second runtime format
  // in the same product is how two screens end up disagreeing about a film.
  return formatRuntime(minutes);
}

/**
 * Route AND runtime, side by side (sub-6-10b AC #2).
 *
 * Before this story the two were MUTUALLY EXCLUSIVE: an unknown runtime
 * replaced the route line entirely, so 「片長未知，以 45 分鐘估算」 erased the
 * one sentence that told the user WHY the row costs money. On the 2026-09-03
 * production screenshot that was every single row.
 */
function routeSubtitle(c: GenerationCandidate): string {
  const route = c.route === 'extract' ? '內嵌英文字幕 → 翻譯' : '無文字字幕軌 → 語音辨識 + 翻譯';
  const runtime = runtimeLabel(c);
  return runtime ? `${route} · ${runtime}` : route;
}

/**
 * The row's artwork (sub-6-10b AC #1).
 *
 * A grey rectangle is not an identity — 2,399 of them is what the critique
 * screenshot showed. With no poster we draw the title's first character
 * instead, which at least distinguishes one row from the next.
 */
function CandidatePoster({ candidate }: { candidate: GenerationCandidate }) {
  const [failed, setFailed] = useState(false);
  const url = getImageUrl(candidate.posterPath ?? null, 'w92');
  const boxClass = 'h-[54px] w-[38px] shrink-0 rounded-[var(--radius-sm)]';

  if (!url || failed) {
    const initial = displayTitleOf(candidate).trim().charAt(0) || '？';
    return (
      <span
        aria-hidden="true"
        data-testid={`consent-row-poster-fallback-${candidate.mediaId}`}
        className={cn(
          boxClass,
          // Sally 裁定 3 (2026-09-05): Title tier (18px/600) on --text-secondary,
          // a DELIBERATE exception to the Small-By-Default rule — this is a
          // graphic stand-in for a poster, not a text label, and at 12px muted
          // on --bg-tertiary it cannot do its only job (telling row 47 from
          // row 48). An existing tier, so no unauthorised font size.
          'flex items-center justify-center bg-[var(--bg-tertiary)] text-lg font-semibold text-[var(--text-secondary)]'
        )}
      >
        {initial}
      </span>
    );
  }

  return (
    <img
      src={url}
      // Decorative: the title sits right next to it, so announcing the poster
      // would make a screen reader read every row's name twice.
      alt=""
      loading="lazy"
      onError={() => setFailed(true)}
      data-testid={`consent-row-poster-${candidate.mediaId}`}
      className={cn(boxClass, 'bg-[var(--bg-tertiary)] object-cover')}
    />
  );
}

/**
 * What a virtualised row needs on its <li> so the virtualizer can find and
 * measure it. Absent on the non-virtual path, where nothing measures anything.
 */
interface VirtualRowAttrs {
  rowRef?: (el: HTMLElement | null) => void;
  index?: number;
}

function CandidateRow({
  candidate,
  checked,
  onToggle,
  prices,
  paused,
  rowRef,
  index,
}: {
  candidate: GenerationCandidate;
  checked: boolean;
  onToggle: (mediaId: string) => void;
  prices?: ModelPrices;
  /**
   * sub-6-12 AC #3: selected, but past the point where the ceiling stops
   * paying — it stays in the queue rather than running. Dimmed, never
   * disabled: the user can still untick it, and 「暫停」 is not 「拒絕」.
   */
  paused?: boolean;
} & VirtualRowAttrs) {
  const isExtract = candidate.route === 'extract';
  // sub-6-1: the backend's write probe refused this folder. The pipeline
  // would fail the item before spending anyway; here the user learns it
  // BEFORE consenting, and cannot tick a row that can never be placed.
  const writable = isWritable(candidate);
  // sub-6-10b AC #3. Strictly `=== false`: a pre-sub-6-10a server omits the
  // field, and "the server never told us" is not "TMDb found nothing".
  const unmatched = candidate.tmdbMatched === false;
  return (
    <li
      ref={rowRef}
      data-index={index}
      data-testid={`consent-row-${candidate.mediaId}`}
      data-route={candidate.route}
      data-writable={writable ? 'true' : 'false'}
      data-paused={paused ? 'true' : undefined}
      className={cn(
        'flex items-center gap-3 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3.5 py-3',
        !writable && 'opacity-70',
        paused && 'opacity-60'
      )}
    >
      <input
        type="checkbox"
        aria-label={`選取 ${displayTitleOf(candidate)}`}
        checked={checked}
        disabled={!writable}
        onChange={() => onToggle(candidate.mediaId)}
        className="h-4 w-4 shrink-0 accent-[var(--accent-primary)] disabled:cursor-not-allowed"
      />
      <CandidatePoster candidate={candidate} />
      {/* The identity column is a two-line GRID that also owns the state→kind→cost
          cluster, so the SAME nodes lay out two ways depending on the width the
          row actually gets — a container query on the <ul>, not the viewport,
          so the drawn phone row also holds in a narrow desktop dialog and can be
          pinned by a 390px gallery fixture (bugfix-f15-row-mobile-identity-collapse):

            list ≥ 36rem (F15-D-v2 pwMzT)
              [title · 未匹配 ………………………][status · kind · cost]   ← cluster spans both lines
              [route · runtime, one line…][                    ]
            list < 36rem (F15-M-v2 fdu4y)
              [title · 未匹配 ……][status · cost]   ← cluster moves onto the title line
              [route · runtime, wrapping across both columns ……]

          Why: on a 390px phone the identity column was getting 115–131px of
          the 227–303px the subtitle needs, because the cluster is shrink-0 —
          「· 片長」 was invisible on EVERY row, and the worst row lost half its
          route too. Moving the cost up beside the title and letting the
          subtitle take both columns (and wrap) gives it the full 256px the
          drawing shows. The kind badge (抽取／語音辨識) goes below 36rem: the
          amount's colour already says extract (green) vs ASR (amber).

          Why the grid is NESTED here rather than being the <li>: the outer flex
          row (checkbox · poster · identity) is untouched, so the desktop
          geometry is exactly what shipped. A whole-row grid with the 54px
          poster spanning both text rows made Chromium distribute the poster's
          extra height into the text tracks (first attempt: +16px per row, a 1%
          diff on every desktop baseline); inside this grid nothing is taller
          than the two text lines, so the rows stay 20px + 16px. Placement uses
          the col-start / col-end / row-start / row-end longhands only —
          Tailwind's span utilities are the grid shorthands and hand the cell
          back to auto-placement (that is how the first phone shot put the
          subtitle in the wrong columns). */}
      <div className="grid min-w-0 flex-1 grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3">
        {/* Sally 裁定 2 (2026-09-05): the identity column carries the doubt
            about the identity. The right-hand cluster is state → kind → cost;
            「未匹配」 is none of those, and a grey badge half a row away does
            not read as being ABOUT the title. */}
        <span className="col-start-1 row-start-1 flex min-w-0 items-center gap-2">
          <span
            className="truncate text-sm text-[var(--text-primary)]"
            // sub-6-10b AC #3: an unmatched row shows the parser's cleaned-up
            // guess, so hovering must still reveal what the file is really
            // called — that is the string the user will look for on disk.
            title={unmatched ? candidate.title : undefined}
          >
            {displayTitleOf(candidate)}
          </span>
          {unmatched && (
            <span
              data-testid={`consent-row-unmatched-${candidate.mediaId}`}
              title="TMDb 沒有比對到，片名由檔名解析"
              className="shrink-0 rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)] px-2 py-0.5 text-xs text-[var(--text-muted)]"
            >
              未匹配
            </span>
          )}
        </span>
        {/* Subtitle: one truncated line beside the cluster on desktop; below
            36rem it takes both columns and wraps — the drawn phone row has the
            whole width, and a route sentence cut in half tells the user nothing
            about why the row costs money. */}
        <span
          data-testid={`consent-row-subtitle-${candidate.mediaId}`}
          className="col-start-1 row-start-2 block truncate text-xs text-[var(--text-muted)] @max-xl:col-end-3 @max-xl:whitespace-normal"
        >
          {routeSubtitle(candidate)}
        </span>
        <span
          data-testid={`consent-row-cluster-${candidate.mediaId}`}
          className="col-start-2 row-start-1 flex shrink-0 items-center gap-2 @xl:row-end-3"
        >
          {!writable && (
            <span
              data-testid={`consent-row-unwritable-${candidate.mediaId}`}
              title={blockerLabel(candidate)}
              className="rounded-[var(--radius-sm)] bg-[var(--error-tint)] px-2 py-0.5 text-xs text-[var(--error-text)]"
            >
              資料夾無法寫入
            </span>
          )}
          <span
            data-testid={`consent-row-kind-${candidate.mediaId}`}
            className={cn(
              'hidden rounded-[var(--radius-sm)] px-2 py-0.5 text-xs @xl:inline',
              isExtract
                ? 'bg-[var(--success-tint)] text-[var(--success-text)]'
                : 'bg-[var(--warning-tint)] text-[var(--warning-text)]'
            )}
          >
            {isExtract ? '抽取' : '語音辨識'}
          </span>
          <span
            data-testid={`consent-row-usd-${candidate.mediaId}`}
            className={cn(
              'font-mono text-[13px] font-semibold tabular-nums',
              isExtract ? 'text-[var(--success-text)]' : 'text-[var(--warning-text)]'
            )}
          >
            {isRuntimeApproximate(candidate)
              ? `≈ ${usd(candidateUsd(candidate, prices))}`
              : usd(candidateUsd(candidate, prices))}
          </span>
        </span>
      </div>
    </li>
  );
}

/**
 * The budget cut line (sub-6-12 AC #3).
 *
 * Before this story the F18 banner said 「預計可完成約 251 部後暫停」 and the
 * list gave the reader no way to map 251 onto anything — the critique's
 * 「砍線不可見」. Drawing it IN the list turns an abstract count into a place:
 * everything above runs, everything below waits.
 *
 * It is `aria-hidden` because the same fact is already announced by the
 * banner above the footer; a screen reader hitting a decorative rule between
 * two rows would hear the ceiling twice.
 */
function BudgetCutRow({ budgetUsd, rowRef, index }: { budgetUsd: number } & VirtualRowAttrs) {
  return (
    <li
      ref={rowRef}
      data-index={index}
      data-testid="consent-budget-cut"
      aria-hidden="true"
      className="flex items-center gap-3 py-1"
    >
      <span className="h-px flex-1 bg-[var(--warning)]" />
      <span className="shrink-0 text-xs text-[var(--warning-text)]">
        到此為止約 <span className="font-mono tabular-nums">{usd(budgetUsd)}</span>
        ，之後的項目會暫停
      </span>
      <span className="h-px flex-1 bg-[var(--warning)]" />
    </li>
  );
}

/**
 * Section header row — a series, a season, or one half of the split movies
 * block (sub-5-3 AC #2, extended by sub-6-11 AC #4).
 *
 * Selection state and the 已選 subtotal are computed over the section's ALL
 * items (chips and search are view filters); amounts come verbatim from
 * estimated_usd — no second totals engine.
 */
function GroupHeaderRow({
  testid,
  label,
  hint,
  selectLabel,
  unit,
  items,
  selectedIds,
  visibleIds,
  onToggleGroup,
  prices,
  season = false,
  expanded,
  onToggleExpanded,
  rowRef,
  index,
}: {
  testid: string;
  label: string;
  hint?: string;
  selectLabel: string;
  unit: '集' | '部';
  items: GenerationCandidate[];
  selectedIds: ReadonlySet<string>;
  /** The rows the chips and the search box are letting through (sub-6-12). */
  visibleIds: ReadonlySet<string>;
  onToggleGroup: (mediaIds: string[], next: boolean) => void;
  prices?: ModelPrices;
  season?: boolean;
  expanded: boolean;
  onToggleExpanded: () => void;
} & VirtualRowAttrs) {
  // CR H1: the ONE totals engine (三處金額同源, extended to group scope) —
  // route counts AND the subtotal both come from computeTotals over this
  // group's items, never from a second hand-rolled sum. `null` ceiling: the
  // budget verdict is a whole-list concern the footer owns.
  const groupTotals = computeTotals(items, selectedIds, null, prices, visibleIds);
  // sub-6-1 CR H3: the toggled ids span the SELECTABLE members — an unwritable
  // episode can never be ticked, so it must not keep the header permanently
  // indeterminate (which would make the group un-deselectable).
  //
  // sub-6-12 (Alexyu 裁定 2026-09-09): and only the VISIBLE ones. sub-6-11
  // shipped this toggle over the whole section and handed the hazard over with
  // a real number — search 「S04E07」, see one episode under a nine-episode
  // show, tick the header, consent to $2.79 of episodes you cannot see. The
  // checkbox now selects what the checkbox is next to.
  const ids = useMemo(() => visibleSelectableIds(items, visibleIds), [items, visibleIds]);
  // sub-6-11 AC #4: the route composition is what TICKING THIS HEADER would
  // pull in, and it is ALWAYS on screen. A collapsed show is the only thing the
  // user can see of it, and 「這部劇要花錢嗎」 has to be answerable BEFORE
  // ticking — the old badges appeared only once something was already ticked,
  // which answered the question after it stopped mattering. Still the same one
  // engine (CR H1): computeTotals over the ids the checkbox owns.
  const sectionTotals = computeTotals(items, new Set(ids), null, prices);
  const all = ids.length > 0 && groupTotals.visibleSelectedCount === ids.length;
  const some = groupTotals.visibleSelectedCount > 0 && !all;
  // A view filter is hiding part of this section, so the checkbox and the
  // section's own counts no longer speak for the same set — and the header has
  // to say which is which rather than let the reader assume.
  const narrowed = groupTotals.visibleSelectableCount < groupTotals.selectableCount;
  return (
    <li
      ref={rowRef}
      data-index={index}
      data-testid={testid}
      data-expanded={expanded ? 'true' : 'false'}
      className={cn(
        'flex items-center gap-3 rounded-[var(--radius-lg)] px-3.5',
        season
          ? 'ml-6 min-h-[40px] bg-[var(--bg-tertiary)]/60'
          : 'min-h-[44px] bg-[var(--bg-tertiary)]'
      )}
    >
      {/* Tri-state via the NATIVE checked + indeterminate pair — the shipped
          consent-select-all idiom. CR L3: an extra aria-checked on a native
          checkbox is redundant ARIA (the browser already exposes mixed from
          `indeterminate`) and risks desyncing from the real state. */}
      <input
        type="checkbox"
        aria-label={narrowed ? `選取顯示的 ${ids.length} ${unit}` : selectLabel}
        checked={all}
        ref={(el) => {
          if (el) el.indeterminate = some;
        }}
        onChange={() => onToggleGroup(ids, !all)}
        className="h-4 w-4 shrink-0 accent-[var(--accent-primary)]"
      />
      {/* Disclosure. A BUTTON, not a click handler on the row: the checkbox
          lives in the same row, and a row-wide handler would make "tick this
          show" and "open this show" the same gesture. The label is inside the
          button so the whole name is a target, not just the 16px chevron. */}
      <button
        type="button"
        data-testid={`${testid}-disclosure`}
        aria-expanded={expanded}
        onClick={onToggleExpanded}
        title={hint}
        className="flex min-h-[40px] min-w-0 flex-1 items-center gap-2 text-left"
      >
        <ChevronRight
          aria-hidden="true"
          className={cn(
            'h-4 w-4 shrink-0 text-[var(--text-muted)] transition-transform motion-reduce:transition-none',
            expanded && 'rotate-90'
          )}
        />
        <span
          className={cn(
            'min-w-0 truncate',
            season
              ? 'text-[13px] text-[var(--text-secondary)]'
              : 'text-sm font-semibold text-[var(--text-primary)]'
          )}
        >
          {label}
        </span>
      </button>
      <span data-testid={`${testid}-routes`} className="flex shrink-0 items-center gap-1.5 text-xs">
        {sectionTotals.selectedExtractCount > 0 && (
          <span className="rounded-[var(--radius-sm)] bg-[var(--success-tint)] px-1.5 py-0.5 text-[var(--success-text)]">
            抽取 {sectionTotals.selectedExtractCount}
          </span>
        )}
        {sectionTotals.selectedAsrCount > 0 && (
          <span className="rounded-[var(--radius-sm)] bg-[var(--warning-tint)] px-1.5 py-0.5 text-[var(--warning-text)]">
            語音辨識 {sectionTotals.selectedAsrCount}
          </span>
        )}
      </span>
      {/* CR: the denominator is the SELECTABLE count, the same set the route
          badges and the header checkbox speak for. It used to be items.length,
          so a 9-episode show with 2 unwritable folders rendered 「語音辨識 7」
          next to 「已選 0/9」 — two counts of the same show, disagreeing, on the
          one line a collapsed show gets. The unwritable rows are still counted,
          once, by the toolbar's 「N 部資料夾無法寫入」. */}
      {/* The header speaks for exactly the set its CHECKBOX owns: the whole
          section normally, and only the visible rows once a chip or a search is
          narrowing it — count and money together, so 「已選 0 · $1.55」 (a
          subtotal built from four episodes that are off screen) can never
          happen. The whole-list figures never move: they are in the summary bar
          and the footer, which no view filter touches. */}
      <span
        data-testid={`${testid}-selected`}
        className="shrink-0 font-mono text-xs tabular-nums text-[var(--text-muted)]"
      >
        {narrowed
          ? `已選 ${groupTotals.visibleSelectedCount} / 顯示 ${groupTotals.visibleSelectableCount}（全部 ${groupTotals.selectableCount}） · ${usd(groupTotals.visibleSelectedTotalUsd)}`
          : `已選 ${groupTotals.selectedCount}/${groupTotals.selectableCount} · ${usd(groupTotals.selectedTotalUsd)}`}
      </span>
    </li>
  );
}

export function CandidateListPanel({
  candidates,
  selectedIds,
  filter,
  totals,
  prices,
  budgetText,
  budgetUsd,
  starting = false,
  startError = null,
  visibleIds,
  onToggle,
  onToggleGroup,
  onToggleAll,
  onSelectAllExtract,
  onClearSelection,
  onFilterChange,
  onBudgetTextChange,
  onStartClick,
  search,
  searchQuery,
  sort,
  onSearchChange,
  onSortChange,
  claudeKeySource,
  openaiKeySource,
  selfHostedAsr = false,
}: CandidateListPanelProps) {
  /**
   * Which sections the user has opened or closed by hand. Anything absent
   * falls back to sectionDefaultExpanded (shows closed, movies and seasons
   * open), and the whole map lives and dies with this panel — AC #5's
   * 「記在 dialog 生命週期內」: reopening 產生字幕 starts from the defaults again
   * rather than restoring a collapse state from a library that has since been
   * rescanned.
   */
  const [expandedOverride, setExpandedOverride] = useState<Record<string, boolean>>({});
  // CR: invert the state the user can SEE, not the stored override. During a
  // search every rendered section is force-open, so `!(override ?? default)`
  // would record "expand" for a show the user just asked to collapse.
  const toggleSection = useCallback((sectionId: string, expanded: boolean) => {
    setExpandedOverride((prev) => ({ ...prev, [sectionId]: !expanded }));
  }, []);

  const searching = searchQuery.trim() !== '';

  /**
   * sub-6-12 AC #3: draw the cut line, and dim what falls past it, ONLY in the
   * state the F18 banner also fires in.
   *
   * `cutMediaId` is truthful on its own — the ceiling refuses a row the moment
   * the running total has reached it, which can happen on a selection whose
   * total lands EXACTLY on the ceiling and is therefore not `overBudget`. The
   * banner has never spoken in that case, and a divider appearing with no
   * banner and no warning colour would read as a bug rather than a warning. One
   * flag, so the banner, the divider and the dimming appear together or not at
   * all.
   */
  const showCut = totals.overBudget && totals.cutMediaId !== null;
  const rows = useMemo(
    () =>
      buildConsentRows({
        candidates,
        visibleIds,
        sort,
        expandedOverride,
        searching,
        prices,
        cutMediaId: showCut ? totals.cutMediaId : null,
      }),
    [candidates, visibleIds, sort, expandedOverride, searching, prices, showCut, totals.cutMediaId]
  );

  // ── Virtualization (AC #3) ────────────────────────────────────────────────
  const scrollRef = useRef<HTMLDivElement | null>(null);
  const virtualized = rows.length > VIRTUALIZE_FROM_ROWS;
  const virtualizer = useVirtualizer({
    count: rows.length,
    enabled: virtualized,
    getScrollElement: () => scrollRef.current,
    estimateSize: (i) =>
      rows[i]?.kind === 'section'
        ? ESTIMATED_SECTION_PX
        : rows[i]?.kind === 'cut'
          ? ESTIMATED_CUT_PX
          : ESTIMATED_ROW_PX,
    // Rows are NOT a fixed height (an unwritable badge, a wrapped subtitle on a
    // phone), so every rendered row reports its real height back — via the
    // library's OWN measureElement. CR: a hand-rolled
    // `getBoundingClientRect().height` reads the TRANSFORMED box, and the host
    // dialog opens with a `scale(0.96 → 1)` enter animation, so the first
    // measurement pass would cache every visible row ~4% short and never
    // re-fire (a transform does not change the layout box). The default
    // prefers the ResizeObserver's borderBoxSize and falls back to
    // offsetHeight, neither of which a transform touches.
    // The <ul>'s flex gap is invisible to measureElement; declaring it here is
    // what keeps the computed offsets and the drawn list from drifting 8px per
    // row apart.
    gap: LIST_GAP_PX,
    overscan: 8,
    // Stable identity: an inline lambda invalidates getMeasurements on every
    // render, which on 2,400 rows means re-deriving 2,400 offsets per keystroke
    // in the budget field.
    getItemKey: useCallback((i: number) => rows[i]?.key ?? i, [rows]),
  });

  // AC #3: a new filter, a new search or a new sort is a NEW LIST. Staying at
  // scroll offset 14,000 in it means landing somewhere arbitrary — or, once the
  // list is shorter than the offset, on an empty screen that looks like a bug.
  useEffect(() => {
    if (scrollRef.current) scrollRef.current.scrollTop = 0;
  }, [filter, searchQuery, sort]);

  const virtualItems = virtualized ? virtualizer.getVirtualItems() : [];
  const padTop = virtualItems.length > 0 ? virtualItems[0].start : 0;
  const padBottom =
    virtualItems.length > 0
      ? virtualizer.getTotalSize() - virtualItems[virtualItems.length - 1].end
      : 0;
  const rendered: { row: ConsentRow; index: number }[] = virtualized
    ? virtualItems.map((v) => ({ row: rows[v.index], index: v.index }))
    : rows.map((row, index) => ({ row, index }));

  const extractTotal = candidates.filter((c) => c.route === 'extract').length;
  const asrTotal = candidates.length - extractTotal;
  /**
   * 全選 spans the rows that are SELECTABLE (sub-6-1: an unwritable row can
   * never be ticked) AND VISIBLE (sub-6-12 AC #1).
   *
   * The critique's money trap: filter to 需語音辨識, tick 全選, and the 2,399
   * hidden extract rows came along — a total that jumped by an order of
   * magnitude from a control the user read as 「select these」. The box now
   * selects what the box is looking at.
   */
  const allSelected =
    totals.visibleSelectedCount === totals.visibleSelectableCount &&
    totals.visibleSelectableCount > 0;
  const someSelected = totals.visibleSelectedCount > 0 && !allSelected;
  /** A chip or the search box is hiding part of the list. */
  const narrowed = totals.visibleSelectableCount < totals.selectableCount;
  const overBudget = totals.overBudget;
  const budgetInvalid = budgetUsd === null;

  const chip = (
    key: ConsentRouteFilter,
    label: string,
    count: number,
    marker?: 'free' | 'paid'
  ) => (
    <button
      type="button"
      data-testid={`consent-chip-${key}`}
      aria-pressed={filter === key}
      onClick={() => onFilterChange(key)}
      className={cn(
        'flex h-11 items-center gap-1.5 rounded-[var(--radius-sm)] px-3 text-[13px] transition-colors',
        filter === key
          ? 'bg-[var(--accent-subtle)] font-semibold text-[var(--accent-text)]'
          : 'bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
      )}
    >
      {label}
      <span className="font-mono font-semibold tabular-nums">{count}</span>
      {marker === 'free' && (
        <span className="rounded-[var(--radius-sm)] bg-[var(--success-tint)] px-1.5 py-0.5 text-xs font-semibold text-[var(--success-text)]">
          僅翻譯費
        </span>
      )}
      {marker === 'paid' && (
        <span className="rounded-[var(--radius-sm)] bg-[var(--warning-tint)] px-1.5 py-0.5 text-xs font-semibold text-[var(--warning-text)]">
          付費
        </span>
      )}
    </button>
  );

  return (
    <>
      {/* Body. sub-6-11 AC #1: the controls no longer scroll away with the
          list. Before this story the whole body was one scroller, so on a
          2,400-row library the search box, the chips and 全選 were 40 screens
          above wherever the user was reading. Now the controls are a fixed
          block and ONLY the list scrolls — which is also what gives the
          virtualizer a scroll element it can measure. */}
      <div className="flex min-h-0 flex-1 flex-col overflow-y-auto">
        <div className="flex shrink-0 flex-col gap-3 px-6 pb-3 pt-6">
          {/* Summary bar */}
          <div className="flex flex-col gap-0.5">
            <p className="flex flex-wrap items-center gap-[3px] text-[13px] text-[var(--text-secondary)]">
              候選 <span className="font-mono tabular-nums">{totals.candidateCount}</span> 部 · 已選
              <span className="font-mono tabular-nums">{totals.selectedCount}</span> 部 · 預估
              <span
                data-testid="consent-summary-usd"
                className={cn(
                  'font-mono font-semibold tabular-nums',
                  overBudget ? 'text-[var(--warning-text)]' : 'text-[var(--text-primary)]'
                )}
              >
                {usdWithEstimate(totals.selectedTotalUsd, totals.hasEstimatedRows)}
              </span>
            </p>
            {/* sub-6-12 AC #6 —「花錢的事先問」asked HOW MUCH and never asked
                WHOSE key. This is the answer, on the one line that is already
                about the money. */}
            <p data-testid="consent-spend-source" className="text-xs text-[var(--text-muted)]">
              {spendSourceLabel({
                claudeSource: claudeKeySource,
                openaiSource: openaiKeySource,
                selfHostedAsr,
                hasAsrCandidates: asrTotal > 0,
              })}
            </p>
          </div>

          {/* Search + sort (AC #1/#2/#5). One row at every width: on a phone the
            sort control shrinks but stays beside the box, because stacking them
            would cost a whole line of an 85vh sheet. */}
          <div className="flex items-center gap-2">
            <div className="relative flex h-11 min-w-0 flex-1 items-center rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)] px-3">
              <Search
                aria-hidden="true"
                className="mr-2 h-4 w-4 shrink-0 text-[var(--text-muted)]"
              />
              <input
                type="search"
                value={search}
                onChange={(e) => onSearchChange(e.target.value)}
                placeholder="搜尋片名或影集"
                aria-label="搜尋候選"
                data-testid="consent-search-input"
                className="min-w-0 flex-1 bg-transparent text-[13px] text-[var(--text-primary)] outline-none placeholder:text-[var(--text-muted)] [&::-webkit-search-cancel-button]:appearance-none"
              />
              {search !== '' && (
                <button
                  type="button"
                  onClick={() => onSearchChange('')}
                  aria-label="清除搜尋"
                  data-testid="consent-search-clear"
                  className="ml-1 flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-[var(--text-muted)] transition-colors hover:text-[var(--text-primary)]"
                >
                  <X aria-hidden="true" className="h-3.5 w-3.5" />
                </button>
              )}
            </div>
            {/* One native <select> at both widths. On a phone the platform opens
              it as its own picker sheet, which is AC #5's sheet — and a better
              one than anything hand-rolled here would be. */}
            <span className="flex h-11 shrink-0 items-center gap-1.5 rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)] pl-2.5 pr-1">
              <ArrowUpDown
                aria-hidden="true"
                className="h-4 w-4 shrink-0 text-[var(--text-muted)]"
              />
              <select
                value={sort}
                onChange={(e) => onSortChange(e.target.value as ConsentSort)}
                aria-label="排序方式"
                data-testid="consent-sort-select"
                className="h-11 bg-transparent pr-1 text-[13px] text-[var(--text-primary)] outline-none"
              >
                {CONSENT_SORTS.map((s) => (
                  <option key={s.value} value={s.value}>
                    {s.label}
                  </option>
                ))}
              </select>
            </span>
          </div>

          {/* Route filter chips */}
          <div className="flex flex-wrap items-center gap-2 overflow-x-auto">
            {chip('all', '全部', candidates.length)}
            {chip('extract', '可抽取內嵌', extractTotal, 'free')}
            {chip('asr', '需語音辨識', asrTotal, 'paid')}
          </div>

          {/* Selection toolbar */}
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <label className="flex min-h-[44px] items-center gap-2 text-[13px] text-[var(--text-secondary)]">
              <input
                type="checkbox"
                data-testid="consent-select-all"
                // sub-6-12 AC #1: the name says the SCOPE, because that is the
                // thing the shipped 「全選」 got wrong — it read as「these」and
                // meant「all 2,399」.
                aria-label={
                  narrowed
                    ? allSelected
                      ? `取消選取顯示的 ${totals.visibleSelectableCount} 部`
                      : `選取顯示的 ${totals.visibleSelectableCount} 部`
                    : allSelected
                      ? '取消全選'
                      : '全選'
                }
                checked={allSelected}
                ref={(el) => {
                  if (el) el.indeterminate = someSelected;
                }}
                onChange={onToggleAll}
                className="h-4 w-4 accent-[var(--accent-primary)]"
              />
              已選
              {/* The numerator is what the CHECKBOX has ticked, so the pair
                  always reads as one fraction. The library-wide count sits in
                  the summary bar two lines above and is unaffected by any
                  filter — 「已選 2 / 顯示 3」 with only one of the three shown
                  rows ticked would be two different facts wearing one slash. */}
              <span data-testid="consent-select-all-count" className="font-mono tabular-nums">
                {narrowed
                  ? `${totals.visibleSelectedCount} / 顯示 ${totals.visibleSelectableCount}（全部 ${totals.selectableCount}）`
                  : `${totals.selectedCount} / ${totals.selectableCount}`}
              </span>
              {totals.unwritableCount > 0 && (
                <span
                  data-testid="consent-unwritable-count"
                  className="text-xs text-[var(--error-text)]"
                >
                  （{totals.unwritableCount} 部資料夾無法寫入）
                </span>
              )}
            </label>
            <span className="flex-1" />
            <button
              type="button"
              data-testid="consent-select-extract"
              onClick={onSelectAllExtract}
              className="flex min-h-[44px] items-center text-[13px] text-[var(--accent-text)] transition-colors hover:opacity-80"
            >
              <span className="sm:hidden">全部可抽取</span>
              <span className="hidden sm:inline">選取全部可抽取（僅翻譯費）</span>
            </button>
            <button
              type="button"
              data-testid="consent-clear-selection"
              onClick={onClearSelection}
              className="flex min-h-[44px] items-center text-[13px] text-[var(--accent-text)] transition-colors hover:opacity-80"
            >
              <span className="sm:hidden">清除</span>
              <span className="hidden sm:inline">清除選取</span>
            </button>
          </div>
        </div>

        {/* The list — the ONLY thing that scrolls (AC #1/#3). Its rows come
            from buildConsentRows, a projection of the candidates STATE the
            container already re-sorted into groupOrder: the chips, the search
            box and the sort menu decide what is DRAWN and in what order it is
            drawn, never what is submitted. */}
        <div
          ref={scrollRef}
          data-testid="consent-list-scroll"
          // CR: min-h floor + a scrollable body. On a short viewport (a laptop
          // window ~450px tall, a phone in landscape) the fixed control block
          // alone can exceed 85vh; without the floor the list clamped to 0 and
          // the clipped controls had no way to scroll into view.
          className="min-h-[10rem] flex-1 overflow-y-auto px-6"
        >
          {/* @container: CandidateRow re-flows on THIS list's width (36rem),
              not the viewport — see the row for why. */}
          <ul
            className="flex flex-col gap-2 @container"
            data-testid="consent-candidate-list"
            data-virtualized={virtualized ? 'true' : 'false'}
            style={virtualized ? { paddingTop: padTop, paddingBottom: padBottom } : undefined}
          >
            {rendered.map(({ row, index }) =>
              row.kind === 'section' ? (
                <GroupHeaderRow
                  key={row.key}
                  testid={row.testid}
                  label={row.label}
                  hint={row.hint}
                  selectLabel={row.selectLabel}
                  unit={row.unit}
                  items={row.items}
                  selectedIds={selectedIds}
                  visibleIds={visibleIds}
                  onToggleGroup={onToggleGroup}
                  prices={prices}
                  season={row.season}
                  expanded={row.expanded}
                  onToggleExpanded={() => toggleSection(row.sectionId, row.expanded)}
                  rowRef={virtualized ? virtualizer.measureElement : undefined}
                  index={virtualized ? index : undefined}
                />
              ) : row.kind === 'cut' ? (
                <BudgetCutRow
                  key={row.key}
                  budgetUsd={budgetUsd ?? 0}
                  rowRef={virtualized ? virtualizer.measureElement : undefined}
                  index={virtualized ? index : undefined}
                />
              ) : (
                <CandidateRow
                  key={row.key}
                  candidate={row.candidate}
                  checked={selectedIds.has(row.candidate.mediaId)}
                  onToggle={onToggle}
                  prices={prices}
                  paused={showCut && totals.pausedIds.has(row.candidate.mediaId)}
                  rowRef={virtualized ? virtualizer.measureElement : undefined}
                  index={virtualized ? index : undefined}
                />
              )
            )}
          </ul>

          {/* AC #1: a search that matched nothing has to SAY so and offer the
              way back. An empty list under a full-looking screen reads as a
              loading failure. */}
          {rows.length === 0 && searching && (
            <div
              data-testid="consent-search-empty"
              className="flex flex-col items-center gap-2 py-10 text-center"
            >
              <p className="text-sm text-[var(--text-secondary)]">沒有符合的候選</p>
              <button
                type="button"
                onClick={() => onSearchChange('')}
                data-testid="consent-search-empty-clear"
                className="flex min-h-[44px] items-center text-[13px] text-[var(--accent-text)] transition-colors hover:opacity-80"
              >
                清除搜尋
              </button>
            </div>
          )}
        </div>

        <div className="flex shrink-0 flex-col gap-1 px-6 pb-3 pt-3">
          <p className="text-xs text-[var(--text-muted)]">金額為預估值，實際費用依內容長度而定。</p>
        </div>
      </div>

      {/* Over-budget banner (F18) — warning, NOT error: an informed choice. */}
      {overBudget && budgetUsd !== null && (
        <div
          data-testid="consent-over-budget-banner"
          className="mx-6 mb-2 flex items-center gap-2.5 rounded-[var(--radius-md)] bg-[var(--warning-tint)] p-3"
        >
          <CircleAlert className="h-4 w-4 shrink-0 text-[var(--warning-text)]" aria-hidden="true" />
          <p className="flex flex-wrap items-center gap-[3px] text-[13px] text-[var(--text-primary)]">
            預估
            <span className="font-mono font-semibold tabular-nums">
              {usd(totals.selectedTotalUsd)}
            </span>
            已超過上限
            <span className="font-mono font-semibold tabular-nums">{usd(budgetUsd)}</span>
            —— 預計可完成約
            <span
              data-testid="consent-feasible-count"
              className="font-mono font-semibold tabular-nums"
            >
              {totals.feasibleCount}
            </span>
            {/* One flex child, because the banner's `gap-[3px]` would otherwise
                push the 、and 。away from the words they belong to — zh-TW
                punctuation floating a space off its clause reads as a typo.
                sub-6-12 AC #3: the parenthetical only claims the list shows the
                cut when a divider was actually drawn; a selection whose one row
                is dearer than the whole ceiling still runs, so there is nothing
                to mark. */}
            <span>
              部後暫停
              {showCut && (
                <span data-testid="consent-feasible-marked" className="text-[var(--text-muted)]">
                  （清單中已標示）
                </span>
              )}
              ，其餘保留在佇列，可提高上限或稍後續跑。
            </span>
            {/* sub-6-11 AC #2: once the list is sorted by anything but 群組,
                the N counts down the SUBMISSION order, which is no longer the
                order on screen. Saying so is cheaper than letting the user
                believe the top N rows are the N that will run. */}
            {sort !== 'group' && (
              <span data-testid="consent-feasible-order-note" className="text-[var(--text-muted)]">
                （依提交順序計算，不是目前的排序）
              </span>
            )}
          </p>
        </div>
      )}

      {/* sub-6-12 AC #2: 開始 failed — say so WHERE THE BUTTON IS.
          It used to be the last child of the scroll region, so on the owner's
          2,400-row library a 502 from 開始產生 rendered roughly forty screens
          below the button that had just failed: the user saw a spinner stop and
          nothing else. Here it sits with the over-budget banner, directly above
          the sticky footer, outside everything that scrolls. */}
      {startError && (
        <p
          role="alert"
          data-testid="consent-start-error"
          className="mx-6 mb-2 flex items-center gap-2.5 rounded-[var(--radius-md)] bg-[var(--error-tint)] p-3 text-sm text-[var(--error-text)]"
        >
          <CircleAlert className="h-4 w-4 shrink-0" aria-hidden="true" />
          {startError}
        </p>
      )}

      {/* Sticky footer: 已選/預估 · 預算上限 · 開始產生 */}
      <div className="flex shrink-0 flex-col gap-3 border-t border-[var(--border-subtle)] px-6 py-3.5 sm:flex-row sm:items-center">
        <div className="flex flex-col">
          <p className="flex items-center gap-[3px] text-[13px] text-[var(--text-primary)]">
            已選 <span className="font-mono tabular-nums">{totals.selectedCount}</span> 部 · 預估
            <span
              data-testid="consent-footer-usd"
              className={cn(
                'font-mono font-semibold tabular-nums',
                overBudget ? 'text-[var(--warning-text)]' : 'text-[var(--text-primary)]'
              )}
            >
              {usdWithEstimate(totals.selectedTotalUsd, totals.hasEstimatedRows)}
            </span>
          </p>
          <p
            data-testid="consent-footer-detail"
            className="flex flex-wrap items-center gap-[3px] text-xs text-[var(--text-muted)]"
          >
            抽取 <span className="font-mono tabular-nums">{totals.selectedExtractCount}</span> 部
            <span className="font-mono tabular-nums">{usd(totals.selectedExtractUsd)}</span> ·
            語音辨識
            <span className="font-mono tabular-nums">{totals.selectedAsrCount}</span> 部
            <span className="font-mono tabular-nums">{usd(totals.selectedAsrUsd)}</span>
          </p>
        </div>
        <span className="hidden flex-1 sm:block" />
        <label className="flex flex-col gap-1">
          <span className="flex items-center gap-2 text-[13px] text-[var(--text-secondary)]">
            預算上限
            <span
              className="flex items-center rounded-[var(--radius-sm)] border bg-[var(--bg-secondary)] px-2 py-1.5 font-mono text-[13px] tabular-nums text-[var(--text-primary)]"
              style={{
                borderColor:
                  overBudget || budgetInvalid ? 'var(--warning)' : 'var(--border-subtle)',
              }}
            >
              $
              <input
                type="number"
                min="0.01"
                step="0.01"
                inputMode="decimal"
                value={budgetText}
                onChange={(e) => onBudgetTextChange(e.target.value)}
                aria-label="預算上限（美元）"
                aria-invalid={budgetInvalid}
                data-testid="consent-budget-input"
                className="w-16 bg-transparent font-mono tabular-nums outline-none"
              />
            </span>
          </span>
          {/* F18 third design round: in the over-budget state the banner already
              says it — the small hint would be a semantic duplicate (deleted
              from the drawn f18). It renders only in the normal state. */}
          {(budgetInvalid || !overBudget) && (
            <span className="text-xs text-[var(--text-muted)]">
              {budgetInvalid ? '上限必須大於 0' : '達到上限會自動暫停，可稍後續跑'}
            </span>
          )}
        </label>
        <button
          type="button"
          onClick={onStartClick}
          disabled={starting || totals.selectedCount === 0 || budgetInvalid}
          data-testid="consent-start-btn"
          className="flex min-h-[44px] items-center justify-center gap-2 rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-6 text-sm font-medium text-[var(--text-on-accent)] transition-colors hover:bg-[var(--accent-pressed)] disabled:cursor-not-allowed disabled:opacity-50"
        >
          {starting && (
            <Loader2
              className="h-4 w-4 animate-spin motion-reduce:animate-none"
              aria-hidden="true"
            />
          )}
          {overBudget ? '開始產生（將於上限暫停）' : '開始產生'}
        </button>
      </div>
    </>
  );
}
