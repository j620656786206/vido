// Design ref: ux-design.pen Screen F15-D-v2 (pwMzT) · F15-M-v2 (fdu4y) · F18-D-v2 (zBik1)
/**
 * F15 產生字幕．候選清單 (sub-4-3 AC #2/#3) — the core consent screen; F18 is
 * the same panel in its over-budget state; F15-M is the responsive variant
 * (the host dialog is already a mobile bottom sheet).
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
import { useMemo } from 'react';
import { CircleAlert, Loader2 } from 'lucide-react';
import { cn } from '../../../lib/utils';
import { usd } from '../../../lib/currency';
import type { GenerationCandidate } from '../../../services/subtitleService';
import {
  applyRouteFilter,
  computeTotals,
  groupCandidates,
  type ConsentRouteFilter,
  type ConsentTotals,
} from './consentSelection';

export interface CandidateListPanelProps {
  /** Listable candidates (extract+asr, backend order = submission order). */
  candidates: GenerationCandidate[];
  selectedIds: ReadonlySet<string>;
  filter: ConsentRouteFilter;
  /** The ONE totals value every money figure renders from. */
  totals: ConsentTotals;
  /** Raw budget input text (controlled). */
  budgetText: string;
  /** parseBudgetInput(budgetText) — null = invalid (<=0 / not a number). */
  budgetUsd: number | null;
  starting?: boolean;
  startError?: string | null;
  onToggle: (mediaId: string) => void;
  /**
   * 整劇/整季 group toggle (sub-5-3 AC #2) — operates on the group's ALL
   * listable items, NOT the filtered-visible subset: the route chips are a
   * view filter only, same semantics as 全選. `next` is the target state.
   */
  onToggleGroup: (mediaIds: string[], next: boolean) => void;
  onToggleAll: () => void;
  onSelectAllExtract: () => void;
  onClearSelection: () => void;
  onFilterChange: (filter: ConsentRouteFilter) => void;
  onBudgetTextChange: (text: string) => void;
  onStartClick: () => void;
}

function routeSubtitle(c: GenerationCandidate): string {
  if (!c.runtimeKnown) return '片長未知，以 45 分鐘估算';
  return c.route === 'extract' ? '內嵌英文字幕 → 翻譯' : '無文字字幕軌 → 語音辨識 + 翻譯';
}

function CandidateRow({
  candidate,
  checked,
  onToggle,
}: {
  candidate: GenerationCandidate;
  checked: boolean;
  onToggle: (mediaId: string) => void;
}) {
  const isExtract = candidate.route === 'extract';
  return (
    <li
      data-testid={`consent-row-${candidate.mediaId}`}
      data-route={candidate.route}
      className="flex items-center gap-3 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3.5 py-3"
    >
      <input
        type="checkbox"
        aria-label={`選取 ${candidate.title}`}
        checked={checked}
        onChange={() => onToggle(candidate.mediaId)}
        className="h-4 w-4 shrink-0 accent-[var(--accent-primary)]"
      />
      <span
        aria-hidden="true"
        className="h-[54px] w-[38px] shrink-0 rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)]"
      />
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm text-[var(--text-primary)]">{candidate.title}</span>
        <span className="block truncate text-xs text-[var(--text-muted)]">
          {routeSubtitle(candidate)}
        </span>
      </span>
      <span className="flex shrink-0 items-center gap-2">
        <span
          className={cn(
            'rounded-[var(--radius-sm)] px-2 py-0.5 text-[11px]',
            isExtract
              ? 'bg-[var(--success-tint)] text-[var(--success)]'
              : 'bg-[var(--warning-tint)] text-[var(--warning)]'
          )}
        >
          {isExtract ? '抽取' : '語音辨識'}
        </span>
        <span
          data-testid={`consent-row-usd-${candidate.mediaId}`}
          className={cn(
            'font-mono text-[13px] font-semibold tabular-nums',
            isExtract ? 'text-[var(--success)]' : 'text-[var(--warning)]'
          )}
        >
          {candidate.runtimeKnown
            ? usd(candidate.estimatedUsd)
            : `≈ ${usd(candidate.estimatedUsd)}`}
        </span>
      </span>
    </li>
  );
}

/**
 * Series / season header row (sub-5-3 AC #2). Selection state and the 已選
 * subtotal are computed over the group's ALL items (chips are a view filter);
 * amounts come verbatim from estimated_usd — no second totals engine.
 */
function GroupHeaderRow({
  testid,
  label,
  items,
  selectedIds,
  onToggleGroup,
  season = false,
}: {
  testid: string;
  label: string;
  items: GenerationCandidate[];
  selectedIds: ReadonlySet<string>;
  onToggleGroup: (mediaIds: string[], next: boolean) => void;
  season?: boolean;
}) {
  // CR H1: the ONE totals engine (三處金額同源, extended to group scope) —
  // route counts AND the subtotal both come from computeTotals over this
  // group's items, never from a second hand-rolled sum. `null` ceiling: the
  // budget verdict is a whole-list concern the footer owns.
  const groupTotals = computeTotals(items, selectedIds, null);
  const all = items.length > 0 && groupTotals.selectedCount === items.length;
  const some = groupTotals.selectedCount > 0 && !all;
  const ids = items.map((i) => i.mediaId);
  return (
    <li
      data-testid={testid}
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
        aria-label={season ? `選取${label}` : `選取整部 ${label}`}
        checked={all}
        ref={(el) => {
          if (el) el.indeterminate = some;
        }}
        onChange={() => onToggleGroup(ids, !all)}
        className="h-4 w-4 shrink-0 accent-[var(--accent-primary)]"
      />
      <span
        className={cn(
          'min-w-0 flex-1 truncate',
          season
            ? 'text-[13px] text-[var(--text-secondary)]'
            : 'text-sm font-semibold text-[var(--text-primary)]'
        )}
      >
        {label}
      </span>
      {/* Route composition of what is SELECTED here — the "will ticking this
          cost me money?" signal, in the row vocabulary (抽取 / 語音辨識). */}
      <span
        data-testid={`${testid}-routes`}
        className="flex shrink-0 items-center gap-1.5 text-[11px]"
      >
        {groupTotals.selectedExtractCount > 0 && (
          <span className="rounded-[var(--radius-sm)] bg-[var(--success-tint)] px-1.5 py-0.5 text-[var(--success)]">
            抽取 {groupTotals.selectedExtractCount}
          </span>
        )}
        {groupTotals.selectedAsrCount > 0 && (
          <span className="rounded-[var(--radius-sm)] bg-[var(--warning-tint)] px-1.5 py-0.5 text-[var(--warning)]">
            語音辨識 {groupTotals.selectedAsrCount}
          </span>
        )}
      </span>
      <span
        data-testid={`${testid}-selected`}
        className="shrink-0 font-mono text-xs tabular-nums text-[var(--text-muted)]"
      >
        已選 {groupTotals.selectedCount}/{items.length} · {usd(groupTotals.selectedTotalUsd)}
      </span>
    </li>
  );
}

/** S00 is the conventional specials season. */
function seasonLabel(n: number): string {
  return n === 0 ? '特別篇' : `第 ${n} 季`;
}

export function CandidateListPanel({
  candidates,
  selectedIds,
  filter,
  totals,
  budgetText,
  budgetUsd,
  starting = false,
  startError = null,
  onToggle,
  onToggleGroup,
  onToggleAll,
  onSelectAllExtract,
  onClearSelection,
  onFilterChange,
  onBudgetTextChange,
  onStartClick,
}: CandidateListPanelProps) {
  const visible = applyRouteFilter(candidates, filter);
  // CR L2: grouping sorts every series bucket, and this panel re-renders on
  // each budget-input keystroke — without the memo a 1,200-item library would
  // regroup + rebuild the visibility set per character typed.
  const visibleIds = useMemo(() => new Set(visible.map((c) => c.mediaId)), [visible]);
  const groups = useMemo(() => groupCandidates(candidates), [candidates]);
  const extractTotal = candidates.filter((c) => c.route === 'extract').length;
  const asrTotal = candidates.length - extractTotal;
  const allSelected = totals.selectedCount === candidates.length && candidates.length > 0;
  const someSelected = totals.selectedCount > 0 && !allSelected;
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
        <span className="rounded-[var(--radius-sm)] bg-[var(--success-tint)] px-1.5 py-0.5 text-[10px] font-semibold text-[var(--success)]">
          僅翻譯費
        </span>
      )}
      {marker === 'paid' && (
        <span className="rounded-[var(--radius-sm)] bg-[var(--warning-tint)] px-1.5 py-0.5 text-[10px] font-semibold text-[var(--warning)]">
          付費
        </span>
      )}
    </button>
  );

  return (
    <>
      {/* Body */}
      <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto p-6 pb-3">
        {/* Summary bar */}
        <p className="flex flex-wrap items-center gap-[3px] text-[13px] text-[var(--text-secondary)]">
          候選 <span className="font-mono tabular-nums">{totals.candidateCount}</span> 部 · 已選
          <span className="font-mono tabular-nums">{totals.selectedCount}</span> 部 · 預估
          <span
            data-testid="consent-summary-usd"
            className={cn(
              'font-mono font-semibold tabular-nums',
              overBudget ? 'text-[var(--warning)]' : 'text-[var(--text-primary)]'
            )}
          >
            {usd(totals.selectedTotalUsd)}
          </span>
        </p>

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
              aria-label={allSelected ? '取消全選' : '全選'}
              checked={allSelected}
              ref={(el) => {
                if (el) el.indeterminate = someSelected;
              }}
              onChange={onToggleAll}
              className="h-4 w-4 accent-[var(--accent-primary)]"
            />
            已選
            <span className="font-mono tabular-nums">
              {totals.selectedCount} / {totals.candidateCount}
            </span>
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

        {/* Candidate list — grouped sections (sub-5-3 AC #2). Display order is
            the groupOrder the container already re-sorted the candidates STATE
            into, so rows here render in submission (= feasible-walk) order;
            the chips only decide row VISIBILITY, never group membership. */}
        <ul className="flex flex-col gap-2" data-testid="consent-candidate-list">
          {groups.flatMap((group) => {
            const rowsOf = (items: GenerationCandidate[]) =>
              items
                .filter((c) => visibleIds.has(c.mediaId))
                .map((c) => (
                  <CandidateRow
                    key={c.mediaId}
                    candidate={c}
                    checked={selectedIds.has(c.mediaId)}
                    onToggle={onToggle}
                  />
                ));

            if (group.kind === 'movies') return rowsOf(group.items);
            if (!group.items.some((c) => visibleIds.has(c.mediaId))) return [];

            const nodes = [
              <GroupHeaderRow
                key={`series-${group.seriesId}`}
                testid={`consent-group-${group.seriesId}`}
                label={group.seriesTitle || '未知影集'}
                items={group.items}
                selectedIds={selectedIds}
                onToggleGroup={onToggleGroup}
              />,
            ];
            if (group.showSeasonHeaders && group.seasons) {
              for (const season of group.seasons) {
                if (!season.items.some((c) => visibleIds.has(c.mediaId))) continue;
                nodes.push(
                  <GroupHeaderRow
                    key={`season-${group.seriesId}-${season.seasonNumber}`}
                    testid={`consent-season-${group.seriesId}-${season.seasonNumber}`}
                    label={seasonLabel(season.seasonNumber)}
                    items={season.items}
                    selectedIds={selectedIds}
                    onToggleGroup={onToggleGroup}
                    season
                  />,
                  ...rowsOf(season.items)
                );
              }
            } else {
              nodes.push(...rowsOf(group.items));
            }
            return nodes;
          })}
        </ul>

        <p className="text-xs text-[var(--text-muted)]">金額為預估值，實際費用依內容長度而定。</p>
        {startError && (
          <p data-testid="consent-start-error" className="text-sm text-[var(--error)]">
            {startError}
          </p>
        )}
      </div>

      {/* Over-budget banner (F18) — warning, NOT error: an informed choice. */}
      {overBudget && budgetUsd !== null && (
        <div
          data-testid="consent-over-budget-banner"
          className="mx-6 mb-2 flex items-center gap-2.5 rounded-[var(--radius-md)] bg-[var(--warning-tint)] p-3"
        >
          <CircleAlert className="h-4 w-4 shrink-0 text-[var(--warning)]" aria-hidden="true" />
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
            部後暫停，其餘保留在佇列，可提高上限或稍後續跑。
          </p>
        </div>
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
                overBudget ? 'text-[var(--warning)]' : 'text-[var(--text-primary)]'
              )}
            >
              {usd(totals.selectedTotalUsd)}
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
            <span className="text-[11px] text-[var(--text-muted)]">
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
