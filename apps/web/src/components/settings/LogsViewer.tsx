// Design ref: ux-design.pen Screen C12-D (K28SdR) · C12-M (dOEbF)；篩到沒結果見 C17-D (Gw61P) · C17-M (J186P)
import { useState, useCallback, useEffect, useRef } from 'react';
import {
  FileText,
  Trash2,
  Loader2,
  ChevronLeft,
  ChevronRight,
  SearchX,
  ScrollText,
} from 'lucide-react';
import { useLogs, useClearLogs } from '../../hooks/useLogs';
import { LogEntry } from './LogEntry';
import { LogFilters } from './LogFilters';
import type { LogClearResult } from '../../services/logService';
import { SettingsErrorState } from './SettingsErrorState';

const PER_PAGE = 50;

export function LogsViewer() {
  const [level, setLevel] = useState('');
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [lastResult, setLastResult] = useState<LogClearResult | null>(null);

  const { data, isFetched, isFetching, isPlaceholderData, error, refetch } = useLogs({
    level: level || undefined,
    keyword: keyword || undefined,
    page,
    perPage: PER_PAGE,
  });

  const clearLogs = useClearLogs();

  const handleLevelChange = useCallback((newLevel: string) => {
    setLevel(newLevel);
    setPage(1);
  }, []);

  const handleKeywordChange = useCallback((newKeyword: string) => {
    // A keyword of spaces is no filter: it would read「關鍵字「 」」.
    setKeyword(newKeyword.trim());
    setPage(1);
  }, []);

  // LogFilters keeps the typed keyword in its own state; remounting it is the
  // one reset path that clears the box as well as the applied filter.
  const [filtersKey, setFiltersKey] = useState(0);
  const filtersRef = useRef<HTMLDivElement>(null);
  const clearFilters = useCallback(() => {
    setLevel('');
    setKeyword('');
    setPage(1);
    setFiltersKey((k) => k + 1);
    // 清除篩選 unmounts itself and LogFilters remounts: without this, focus
    // falls to <body> and a keyboard user starts over from the top.
    window.requestAnimationFrame(() =>
      filtersRef.current?.querySelector<HTMLElement>('[data-testid="log-filter-all"]')?.focus()
    );
  }, []);
  const filtered = level !== '' || keyword !== '';

  // Two-step confirm — one click on a 14k-row purge is a real loss with no
  // undo, and the cache page's per-type clears already taught users that this
  // app confirms destructive actions.
  const [confirmingClearOld, setConfirmingClearOld] = useState(false);

  const handleClearOld = () => {
    if (!confirmingClearOld) {
      setConfirmingClearOld(true);
      return;
    }
    clearLogs.mutate(30, {
      onSuccess: (result) => {
        setLastResult(result);
        // The page we were on may no longer exist once old logs are gone.
        setPage(1);
      },
      onSettled: () => setConfirmingClearOld(false),
    });
  };

  const totalPages = data ? Math.ceil(data.total / PER_PAGE) : 0;
  // Belt and braces: never sit past the last page (an empty page with a
  // non-zero total would read as both「共 60 筆」and「還沒有日誌記錄」).
  useEffect(() => {
    if (totalPages > 0 && page > totalPages) setPage(totalPages);
  }, [page, totalPages]);

  // Keyed on data (dsr-3c lesson): a failed read with nothing cached drops back
  // to pending on every refetch, which would swap the error page for the spinner.
  if (!data && !error && !isFetched) {
    return (
      <div className="flex items-center justify-center py-20" data-testid="logs-loading">
        <Loader2 className="h-8 w-8 animate-spin text-[var(--text-secondary)]" />
      </div>
    );
  }

  // No data at all → the error page. (A failed refetch of the SAME query keeps
  // its data; after a filter change the new query has none, so it lands here
  // too.) Logs are written by the backend itself, so the ones already
  // recorded are safe whatever happened to this request.
  if (!data) {
    return (
      <SettingsErrorState
        testId="logs-error"
        title="無法載入系統日誌"
        description="與後端的連線中斷了。已記錄的日誌不受影響。"
        isRetrying={isFetching}
        onRetry={() => void refetch()}
      />
    );
  }

  return (
    <div className="space-y-4" data-testid="logs-viewer">
      {/* Header */}
      <div className="flex items-center justify-between">
        {/* Title at the route level; this is the live record count. */}
        <div className="flex items-center gap-2 text-sm text-[var(--text-secondary)]">
          <FileText className="h-4 w-4" aria-hidden="true" />
          {/* The backend's total is AFTER filtering; there is no unfiltered
              count (disc-2026-09-logs-unfiltered-total). With a filter on,
              「共 0 筆記錄」read as "the log is empty" — so say what it is. */}
          <span data-testid="logs-count">
            {/* While a new filter loads, `data` is still the PREVIOUS query's
                (keepPreviousData): its total is not「符合條件」of anything. */}
            {isPlaceholderData
              ? '載入中…'
              : filtered
                ? `符合條件 ${data.total.toLocaleString()} 筆`
                : `共 ${data.total.toLocaleString()} 筆記錄`}
          </span>
        </div>

        <div className="flex items-center gap-2">
          {confirmingClearOld && !clearLogs.isPending && (
            <button
              onClick={() => setConfirmingClearOld(false)}
              className="rounded-lg px-3 py-2 text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
              data-testid="clear-old-logs-cancel-btn"
            >
              取消
            </button>
          )}
          <button
            onClick={handleClearOld}
            disabled={clearLogs.isPending}
            className={`flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-colors disabled:opacity-50 ${
              confirmingClearOld
                ? 'bg-[var(--error)] text-[var(--text-on-scrim)] hover:bg-[var(--error-pressed)]'
                : 'bg-[var(--bg-tertiary)] text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)]'
            }`}
            data-testid="clear-old-logs-btn"
          >
            {clearLogs.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Trash2 className="h-4 w-4" />
            )}
            {confirmingClearOld ? '確認清除 30 天前' : '清除 30 天前'}
          </button>
        </div>
      </div>

      {/* Filters */}
      <div ref={filtersRef}>
        <LogFilters
          key={filtersKey}
          level={level}
          keyword={keyword}
          onLevelChange={handleLevelChange}
          onKeywordChange={handleKeywordChange}
        />
      </div>

      {/* Log entries */}
      <div
        className="overflow-hidden rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)]"
        data-testid="logs-list"
      >
        {data.logs && data.logs.length > 0 ? (
          <div className={isPlaceholderData ? 'opacity-60' : undefined}>
            {data.logs.map((log) => (
              <LogEntry key={log.id} log={log} />
            ))}
          </div>
        ) : isPlaceholderData ? (
          // The previous query was empty; do not claim anything about this one yet.
          <div className="flex justify-center py-14" data-testid="logs-refetching">
            <Loader2
              className="h-6 w-6 motion-safe:animate-spin text-[var(--text-muted)]"
              aria-hidden="true"
            />
          </div>
        ) : (
          <LogsEmpty level={level} keyword={keyword} onClear={clearFilters} />
        )}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between" data-testid="logs-pagination">
          <button
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={page <= 1}
            className="flex items-center gap-1 rounded-lg border border-[var(--border-subtle)] px-3 py-1.5 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] disabled:opacity-50"
            data-testid="logs-prev-btn"
          >
            <ChevronLeft className="h-4 w-4" />
            上一頁
          </button>

          <span className="text-sm text-[var(--text-secondary)]" data-testid="logs-page-info">
            第 {page} / {totalPages} 頁
          </span>

          <button
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            disabled={page >= totalPages}
            className="flex items-center gap-1 rounded-lg border border-[var(--border-subtle)] px-3 py-1.5 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] disabled:opacity-50"
            data-testid="logs-next-btn"
          >
            下一頁
            <ChevronRight className="h-4 w-4" />
          </button>
        </div>
      )}

      {/* Clear result feedback */}
      {lastResult && (
        <div
          className="flex items-center gap-2 rounded-lg bg-[var(--bg-tertiary)] px-4 py-3 text-sm text-[var(--text-secondary)]"
          data-testid="logs-clear-result"
        >
          <Trash2 className="h-4 w-4 flex-shrink-0" />
          <span>已清除 {lastResult.entriesRemoved.toLocaleString()} 筆日誌記錄</span>
        </div>
      )}
    </div>
  );
}

/**
 * C17: two different empties. Filtered to nothing → say which filter, and offer
 * to clear it. Nothing logged at all → say so, with nothing to press.
 */
export function describeLogFilter(level: string, keyword: string): string {
  const parts: string[] = [];
  if (level) parts.push(level);
  if (keyword) parts.push(`關鍵字「${keyword}」`);
  const advice = !keyword
    ? '放寬等級再試一次。'
    : !level
      ? '清除關鍵字再試一次。'
      : '放寬等級或清除關鍵字再試一次。';
  return `目前篩選：${parts.join(' ＋ ')}。${advice}`;
}

export function LogsEmpty({
  level,
  keyword,
  onClear,
}: {
  level: string;
  keyword: string;
  onClear: () => void;
}) {
  const filtered = level !== '' || keyword !== '';
  const Icon = filtered ? SearchX : ScrollText;
  return (
    <div
      className="flex flex-col items-center gap-3 px-4 py-14 text-center"
      data-testid="logs-empty"
      data-state={filtered ? 'filtered' : 'none'}
    >
      <span className="flex size-12 items-center justify-center rounded-full bg-[var(--bg-tertiary)] sm:size-14">
        <Icon className="size-[22px] text-[var(--text-muted)] sm:size-6" aria-hidden="true" />
      </span>
      <p className="text-base font-semibold text-[var(--text-primary)] sm:text-lg">
        {filtered ? '沒有符合條件的日誌記錄' : '還沒有日誌記錄'}
      </p>
      {filtered && (
        <>
          <p
            className="max-w-xl text-xs text-[var(--text-secondary)] sm:text-sm"
            data-testid="logs-empty-filter"
          >
            {describeLogFilter(level, keyword)}
          </p>
          <button
            type="button"
            onClick={onClear}
            className="h-11 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-4 text-sm font-semibold text-[var(--text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)] sm:h-10"
            data-testid="logs-clear-filters"
          >
            清除篩選
          </button>
        </>
      )}
    </div>
  );
}
