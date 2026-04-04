/**
 * RetryQueueSection Component (Story 3.11 - HIGH #4 Integration)
 * Shows retry queue status inline within the parse progress card
 */

import { RefreshCw, AlertTriangle, CheckCircle2 } from 'lucide-react';
import { cn } from '../../lib/utils';
import { usePendingRetries } from '../../hooks/useRetry';
import { CountdownTimer } from '../retry/CountdownTimer';

export interface RetryQueueSectionProps {
  /** Additional CSS classes */
  className?: string;
  /** Whether to show detailed view */
  detailed?: boolean;
}

/**
 * Inline section showing retry queue status
 * Shows when there are pending retries or recently completed retries
 */
export function RetryQueueSection({ className, detailed = false }: RetryQueueSectionProps) {
  const { data, isLoading, isError } = usePendingRetries();

  const items = data?.items || [];
  const stats = data?.stats;

  // Don't show if loading, error, or no items
  if (isLoading || isError || items.length === 0) {
    return null;
  }

  return (
    <div
      className={cn('border-t border-[var(--border-subtle)] pt-3 mt-3', className)}
      data-testid="retry-queue-section"
    >
      {/* Header */}
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2 text-sm">
          <RefreshCw className="h-4 w-4 text-[var(--warning)]" />
          <span className="text-[var(--text-secondary)]">重試隊列</span>
          <span className="text-xs px-1.5 py-0.5 rounded bg-[var(--warning)]/20 text-[var(--warning)]">
            {items.length}
          </span>
        </div>
        {stats && stats.totalSucceeded > 0 && (
          <span className="text-xs text-[var(--success)] flex items-center gap-1">
            <CheckCircle2 className="h-3 w-3" />
            {stats.totalSucceeded} 已成功
          </span>
        )}
      </div>

      {/* Compact item list (show first 2 items) */}
      <div className="space-y-2">
        {items.slice(0, detailed ? items.length : 2).map((item) => (
          <div
            key={item.id}
            className="flex items-center justify-between text-sm bg-[var(--bg-tertiary)]/50 rounded px-2 py-1.5"
          >
            <div className="flex items-center gap-2 min-w-0 flex-1">
              <span className="text-xs px-1.5 py-0.5 rounded bg-[var(--bg-tertiary)] text-[var(--text-secondary)]">
                {item.taskType === 'parse' ? '解析' : '元資料'}
              </span>
              <span className="text-[var(--text-secondary)] truncate text-xs" title={item.taskId}>
                {item.attemptCount}/{item.maxAttempts}
              </span>
            </div>
            <CountdownTimer targetTime={item.nextAttemptAt} className="text-xs" />
          </div>
        ))}
        {!detailed && items.length > 2 && (
          <div className="text-xs text-[var(--text-muted)] text-center">
            還有 {items.length - 2} 項...
          </div>
        )}
      </div>

      {/* Warning for exhausted retries */}
      {stats && stats.totalFailed > 0 && (
        <div className="mt-2 flex items-center gap-1.5 text-xs text-[var(--warning)]">
          <AlertTriangle className="h-3 w-3" />
          <span>{stats.totalFailed} 項需要手動處理</span>
        </div>
      )}
    </div>
  );
}

export default RetryQueueSection;
