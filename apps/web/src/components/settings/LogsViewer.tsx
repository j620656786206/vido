import { useState, useCallback } from 'react';
import { FileText, Trash2, Loader2, ChevronLeft, ChevronRight } from 'lucide-react';
import { useLogs, useClearLogs } from '../../hooks/useLogs';
import { LogEntry } from './LogEntry';
import { LogFilters } from './LogFilters';
import type { LogClearResult } from '../../services/logService';

const PER_PAGE = 50;

export function LogsViewer() {
  const [level, setLevel] = useState('');
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [lastResult, setLastResult] = useState<LogClearResult | null>(null);

  const { data, isLoading, error } = useLogs({
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
    setKeyword(newKeyword);
    setPage(1);
  }, []);

  const handleClearOld = () => {
    clearLogs.mutate(30, {
      onSuccess: (result) => setLastResult(result),
    });
  };

  const totalPages = data ? Math.ceil(data.total / PER_PAGE) : 0;

  if (isLoading && !data) {
    return (
      <div className="flex items-center justify-center py-20" data-testid="logs-loading">
        <Loader2 className="h-8 w-8 animate-spin text-[var(--text-secondary)]" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="py-10 text-center" data-testid="logs-error">
        <p className="text-[var(--error)]">無法載入系統日誌</p>
        <p className="mt-1 text-sm text-[var(--text-muted)]">{error.message}</p>
      </div>
    );
  }

  return (
    <div className="space-y-4" data-testid="logs-viewer">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <FileText className="h-5 w-5 text-[var(--text-secondary)]" />
          <div>
            <h2 className="text-lg font-semibold text-[var(--text-primary)]">系統日誌</h2>
            <p className="text-sm text-[var(--text-secondary)]">
              共 {data?.total.toLocaleString() ?? 0} 筆記錄
            </p>
          </div>
        </div>

        <button
          onClick={handleClearOld}
          disabled={clearLogs.isPending}
          className="flex items-center gap-2 rounded-lg bg-[var(--bg-tertiary)] px-4 py-2 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-tertiary)] disabled:opacity-50"
          data-testid="clear-old-logs-btn"
        >
          {clearLogs.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Trash2 className="h-4 w-4" />
          )}
          清除 30 天前
        </button>
      </div>

      {/* Filters */}
      <LogFilters
        level={level}
        keyword={keyword}
        onLevelChange={handleLevelChange}
        onKeywordChange={handleKeywordChange}
      />

      {/* Log entries */}
      <div
        className="overflow-hidden rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-secondary)]/50"
        data-testid="logs-list"
      >
        {data?.logs && data.logs.length > 0 ? (
          data.logs.map((log) => <LogEntry key={log.id} log={log} />)
        ) : (
          <div className="py-12 text-center text-[var(--text-muted)]" data-testid="logs-empty">
            沒有符合條件的日誌記錄
          </div>
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
          className="flex items-center gap-2 rounded-lg border border-green-700 bg-green-900/30 px-4 py-3 text-sm text-green-300"
          data-testid="logs-clear-result"
        >
          <Trash2 className="h-4 w-4 flex-shrink-0" />
          <span>已清除 {lastResult.entriesRemoved.toLocaleString()} 筆日誌記錄</span>
        </div>
      )}
    </div>
  );
}
