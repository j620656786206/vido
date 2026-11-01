// Design ref: ux-design.pen — no current screen frame; the AI 花費明細 page was specified in words (disc-2026-11-spend-history-c, ⚖️ Alexyu 2026-11-01: no new .pen frames) and built from the K5 spend card's tokens
/**
 * AI 花費明細 (`/activity/spend`, disc-2026-11-spend-history-d) — where the
 * money went, run by run: the month's total and how good the estimates were,
 * a stacked bar per day, and every run with its models, actual and estimate.
 *
 * Top to bottom (Sally's -c design): header (← 活動, title, ‹ month ›, never
 * into the future) → summary (SpendStat cells, the hub card's family) →
 * daily chart → 處理方式 chips + list + 載入更多.
 *
 * States: one skeleton per block while loading; 「這個月還沒有生成過字幕」 with
 * a way back for an untouched month; a fail-soft 無法載入 + 重試 on error. The
 * header stays in all three so the month can still be switched.
 *
 * The month groups runs by their START time and includes failed and cancelled
 * runs (the API's rule) — the page says so, because the hub card counts by
 * completion and can show a different total.
 *
 * `currentMonth` is a prop (the route reads the clock), so this file never
 * touches the wall clock and its gallery fixtures stay stable (Rule 23).
 */
import { Link } from '@tanstack/react-router';
import { ArrowLeft, ChevronLeft, ChevronRight, Receipt } from 'lucide-react';
import { usd } from '../../lib/currency';
import { useSubtitleRunHistory } from '../../hooks/useSubtitleRunHistory';
import type {
  ChartCategory,
  RunHistoryItem,
  RunHistoryPage,
} from '../../services/subtitleRunHistoryService';
import { ActivitySectionShell } from './ActivitySectionShell';
import { ActivitySectionError, ActivitySkeleton } from './ActivityStates';
import { SpendDailyChart } from './SpendDailyChart';
import { SpendCategoryChips, SpendRunList } from './SpendRunList';
import { SpendStat } from './SpendSection';
import { estimateAccuracyPercent, monthLabel, shiftMonth } from './spendHistoryCopy';

export interface SpendHistoryViewProps {
  /** The month on screen, `YYYY-MM`. */
  month: string;
  /** The latest month the switcher may reach (this month). */
  currentMonth: string;
  onMonthChange: (month: string) => void;
  /** The first page — its `summary` and `series` are the whole month. */
  data?: RunHistoryPage;
  /** Every loaded run, all pages so far. */
  items: RunHistoryItem[];
  isLoading: boolean;
  isError: boolean;
  onRetry: () => void;
  category?: ChartCategory;
  onCategoryChange: (category: ChartCategory | undefined) => void;
  hasMore: boolean;
  isLoadingMore: boolean;
  loadMoreFailed?: boolean;
  onLoadMore: () => void;
}

function MonthSwitcher({
  month,
  currentMonth,
  onMonthChange,
}: Pick<SpendHistoryViewProps, 'month' | 'currentMonth' | 'onMonthChange'>) {
  const atLatest = month >= currentMonth;
  const btn =
    'flex h-11 w-11 items-center justify-center rounded-[var(--radius-md)] text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)] disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent';
  return (
    <div className="flex items-center gap-1" data-testid="spend-history-month-switcher">
      <button
        type="button"
        className={btn}
        aria-label="上個月"
        data-testid="spend-history-prev-month"
        onClick={() => onMonthChange(shiftMonth(month, -1))}
      >
        <ChevronLeft className="h-5 w-5" aria-hidden="true" />
      </button>
      <span
        className="min-w-[7.5rem] text-center text-sm font-medium tabular-nums text-[var(--text-primary)]"
        data-testid="spend-history-month"
      >
        {monthLabel(month)}
      </span>
      <button
        type="button"
        className={btn}
        aria-label="下個月"
        data-testid="spend-history-next-month"
        disabled={atLatest}
        onClick={() => {
          if (!atLatest) onMonthChange(shiftMonth(month, 1));
        }}
      >
        <ChevronRight className="h-5 w-5" aria-hidden="true" />
      </button>
    </div>
  );
}

function SummaryRow({ data }: { data: RunHistoryPage }) {
  const { summary } = data;
  const accuracy = estimateAccuracyPercent(summary);
  return (
    <div
      className="flex flex-col gap-3 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-4"
      data-testid="spend-history-summary"
    >
      <dl className="flex flex-col gap-3 sm:flex-row sm:gap-8">
        <SpendStat label="這個月總共" testId="spend-history-total">
          {usd(summary.spentUsd)} · {summary.runs} 次
        </SpendStat>
        <SpendStat label="預估準度" testId="spend-history-accuracy" muted={accuracy === undefined}>
          {accuracy === undefined ? '還沒有可比的紀錄' : `實際是預估的 ${accuracy}%`}
        </SpendStat>
      </dl>
      <p className="text-xs text-[var(--text-muted)]" data-testid="spend-history-note">
        依開始時間歸到這個月，失敗和中途停止的也算進去
        {summary.unpricedRuns > 0 && `；另有 ${summary.unpricedRuns} 次沒記錄金額，沒算進總額`}
        {accuracy !== undefined && `；準度只算實際和預估都有記的 ${summary.comparableRuns} 次`}
      </p>
    </div>
  );
}

function BlockSkeleton({ className, testId }: { className: string; testId: string }) {
  return (
    <div
      data-testid={testId}
      aria-hidden="true"
      className={`animate-pulse rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] motion-reduce:animate-none ${className}`}
    />
  );
}

function SpendHistoryEmpty() {
  return (
    <div
      data-testid="spend-history-empty"
      className="flex flex-col items-center justify-center gap-4 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-6 py-16 text-center"
    >
      <div className="flex h-16 w-16 items-center justify-center rounded-full bg-[var(--bg-tertiary)]">
        <Receipt className="h-8 w-8 text-[var(--text-secondary)]" aria-hidden="true" />
      </div>
      <h3 className="text-base font-semibold text-[var(--text-primary)]">這個月還沒有生成過字幕</h3>
      <Link
        to="/activity"
        data-testid="spend-history-empty-back"
        className="inline-flex min-h-[44px] items-center gap-2 text-sm font-medium text-[var(--accent-text)] hover:underline"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" />
        回活動頁
      </Link>
    </div>
  );
}

/** Prop-driven page body (gallery fixtures render this directly). */
export function SpendHistoryView(props: SpendHistoryViewProps) {
  const { data, items, isLoading, isError, onRetry, category, onCategoryChange } = props;
  const empty = data !== undefined && data.summary.runs === 0 && data.total === 0 && !category;

  return (
    <div
      data-testid="spend-history-root"
      className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-8 sm:px-6"
    >
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-col gap-1">
          <Link
            to="/activity"
            data-testid="spend-history-back"
            className="flex min-h-[44px] w-fit items-center gap-1 text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
          >
            <ArrowLeft className="h-4 w-4" aria-hidden="true" />
            活動
          </Link>
          <h1 className="text-xl font-bold text-[var(--text-primary)] sm:text-2xl">AI 花費明細</h1>
        </div>
        <MonthSwitcher
          month={props.month}
          currentMonth={props.currentMonth}
          onMonthChange={props.onMonthChange}
        />
      </header>

      {isLoading ? (
        <div className="flex flex-col gap-6" data-testid="spend-history-loading">
          <BlockSkeleton className="h-24" testId="spend-history-summary-skeleton" />
          <BlockSkeleton className="h-56" testId="spend-history-chart-skeleton" />
          <ActivitySkeleton count={4} />
        </div>
      ) : isError || !data ? (
        <ActivitySectionError onRetry={onRetry} testId="spend-history-error" />
      ) : empty ? (
        <SpendHistoryEmpty />
      ) : (
        <>
          <SummaryRow data={data} />
          <div className="rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-4">
            <SpendDailyChart series={data.series} month={data.month} />
          </div>
          <ActivitySectionShell
            title="逐筆紀錄"
            trailing={
              <span className="font-mono tabular-nums" data-testid="spend-history-count">
                {items.length} / {data.total}
              </span>
            }
          >
            <SpendCategoryChips value={category} onChange={onCategoryChange} />
            {items.length === 0 ? (
              <p
                className="rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-4 py-8 text-center text-sm text-[var(--text-secondary)]"
                data-testid="spend-history-filter-empty"
              >
                這個月沒有這種處理方式的紀錄
              </p>
            ) : (
              <SpendRunList items={items} />
            )}
            {props.loadMoreFailed && (
              <ActivitySectionError onRetry={props.onLoadMore} testId="spend-history-more-error" />
            )}
            {props.hasMore && !props.loadMoreFailed && (
              <button
                type="button"
                onClick={props.onLoadMore}
                disabled={props.isLoadingMore}
                data-testid="spend-history-load-more"
                className="flex min-h-[44px] items-center justify-center self-center rounded-[var(--radius-md)] border border-[var(--border-subtle)] px-5 text-sm font-medium text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)] disabled:opacity-60"
              >
                {props.isLoadingMore ? '載入中…' : '載入更多'}
              </button>
            )}
          </ActivitySectionShell>
        </>
      )}
    </div>
  );
}

/** The route's page: wires the query to the view. */
export function SpendHistoryPage({
  month,
  currentMonth,
  onMonthChange,
  category,
  onCategoryChange,
}: {
  /** Undefined = this month (the server's). */
  month?: string;
  currentMonth: string;
  onMonthChange: (month: string) => void;
  category?: ChartCategory;
  onCategoryChange: (category: ChartCategory | undefined) => void;
}) {
  const q = useSubtitleRunHistory({ month, category });
  // The label follows the server's month when the URL names none.
  const shown = month ?? q.first?.month ?? currentMonth;
  return (
    <SpendHistoryView
      month={shown}
      currentMonth={currentMonth}
      onMonthChange={onMonthChange}
      data={q.first}
      items={q.items}
      isLoading={q.isLoading}
      isError={q.isError && !q.first}
      onRetry={() => void q.refetch()}
      category={category}
      onCategoryChange={onCategoryChange}
      hasMore={q.hasNextPage}
      isLoadingMore={q.isFetchingNextPage}
      loadMoreFailed={q.isFetchNextPageError}
      onLoadMore={() => void q.fetchNextPage()}
    />
  );
}
