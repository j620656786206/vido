/**
 * RetryQueuePanel Component (Story 3.11 - Task 8.1)
 * Displays pending retry items with countdown timers and action buttons
 * AC4: UI visibility for pending retries
 */

import { RefreshCw, X, AlertCircle, CheckCircle2, Loader2 } from 'lucide-react';
import { cn } from '../../lib/utils';
import { usePendingRetries, useTriggerRetry, useCancelRetry } from '../../hooks/useRetry';
import { CountdownTimer } from './CountdownTimer';
import type { RetryItem } from '../../services/retry';

export interface RetryQueuePanelProps {
  /** Additional CSS classes */
  className?: string;
}

/**
 * Panel showing all pending retry items
 * Shows countdown timers, retry/cancel buttons, and attempt counts
 */
export function RetryQueuePanel({ className }: RetryQueuePanelProps) {
  const { data, isLoading, isError, error, refetch } = usePendingRetries();
  const triggerMutation = useTriggerRetry();
  const cancelMutation = useCancelRetry();

  const handleTriggerRetry = (id: string) => {
    triggerMutation.mutate(id);
  };

  const handleCancelRetry = (id: string) => {
    cancelMutation.mutate(id);
  };

  if (isLoading) {
    return (
      <div className={cn('p-4', className)} data-testid="retry-queue-loading">
        <div className="flex items-center justify-center gap-2 text-slate-400">
          <Loader2 className="h-5 w-5 animate-spin" />
          <span>載入中...</span>
        </div>
      </div>
    );
  }

  if (isError) {
    return (
      <div className={cn('p-4', className)} data-testid="retry-queue-error">
        <div className="flex items-center gap-2 text-red-400 mb-2">
          <AlertCircle className="h-5 w-5" />
          <span>載入失敗</span>
        </div>
        <p className="text-sm text-slate-400 mb-3">{error?.message || '無法取得重試隊列'}</p>
        <button
          onClick={() => refetch()}
          className="text-sm text-blue-400 hover:text-blue-300 flex items-center gap-1"
        >
          <RefreshCw className="h-4 w-4" />
          重試
        </button>
      </div>
    );
  }

  const items = data?.items || [];
  const stats = data?.stats;

  return (
    <div className={cn('space-y-4', className)} data-testid="retry-queue-panel">
      {/* Header with stats */}
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium text-white flex items-center gap-2">
          <RefreshCw className="h-5 w-5" />
          重試隊列
        </h3>
        {stats && (
          <span className="text-sm text-slate-400" data-testid="retry-stats">
            {stats.totalPending} 個待重試
          </span>
        )}
      </div>

      {/* Empty state */}
      {items.length === 0 && (
        <div
          className="py-8 text-center text-slate-400"
          data-testid="retry-queue-empty"
        >
          <CheckCircle2 className="h-8 w-8 mx-auto mb-2 text-green-500" />
          <p>目前沒有待重試項目</p>
        </div>
      )}

      {/* Retry items list */}
      {items.length > 0 && (
        <div className="space-y-3" data-testid="retry-items-list">
          {items.map((item) => (
            <RetryItemCard
              key={item.id}
              item={item}
              onTrigger={handleTriggerRetry}
              onCancel={handleCancelRetry}
              isTriggering={triggerMutation.isPending && triggerMutation.variables === item.id}
              isCanceling={cancelMutation.isPending && cancelMutation.variables === item.id}
            />
          ))}
        </div>
      )}
    </div>
  );
}

interface RetryItemCardProps {
  item: RetryItem;
  onTrigger: (id: string) => void;
  onCancel: (id: string) => void;
  isTriggering?: boolean;
  isCanceling?: boolean;
}

/**
 * Individual retry item card with actions
 */
function RetryItemCard({
  item,
  onTrigger,
  onCancel,
  isTriggering,
  isCanceling,
}: RetryItemCardProps) {
  const isActioning = isTriggering || isCanceling;

  return (
    <div
      className="rounded-lg border border-slate-700 bg-slate-800/50 p-3 space-y-2"
      data-testid={`retry-item-${item.id}`}
    >
      {/* Task info row */}
      <div className="flex items-start justify-between">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-xs px-1.5 py-0.5 rounded bg-slate-700 text-slate-300">
              {getTaskTypeLabel(item.taskType)}
            </span>
            <span className="text-xs text-slate-500">
              {item.attemptCount}/{item.maxAttempts} 次
            </span>
          </div>
          <p className="text-sm text-slate-300 truncate mt-1" title={item.taskId}>
            {item.taskId}
          </p>
        </div>
        <CountdownTimer targetTime={item.nextAttemptAt} />
      </div>

      {/* Error message if present */}
      {item.lastError && (
        <div className="text-xs text-red-400 bg-red-500/10 rounded px-2 py-1">
          {item.lastError}
        </div>
      )}

      {/* Action buttons */}
      <div className="flex items-center gap-2">
        <button
          onClick={() => onTrigger(item.id)}
          disabled={isActioning}
          className={cn(
            'flex-1 flex items-center justify-center gap-1.5',
            'rounded px-3 py-1.5 text-sm font-medium',
            'bg-blue-600 hover:bg-blue-700 text-white',
            'disabled:opacity-50 disabled:cursor-not-allowed',
            'transition-colors'
          )}
          data-testid={`trigger-retry-${item.id}`}
        >
          {isTriggering ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <RefreshCw className="h-4 w-4" />
          )}
          立即重試
        </button>
        <button
          onClick={() => onCancel(item.id)}
          disabled={isActioning}
          className={cn(
            'flex items-center justify-center gap-1.5',
            'rounded px-3 py-1.5 text-sm',
            'border border-slate-600 text-slate-400 hover:text-slate-300 hover:border-slate-500',
            'disabled:opacity-50 disabled:cursor-not-allowed',
            'transition-colors'
          )}
          data-testid={`cancel-retry-${item.id}`}
        >
          {isCanceling ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <X className="h-4 w-4" />
          )}
          取消
        </button>
      </div>
    </div>
  );
}

function getTaskTypeLabel(taskType: string): string {
  const labels: Record<string, string> = {
    parse: '解析',
    metadata_fetch: '取得元資料',
  };
  return labels[taskType] || taskType;
}

export default RetryQueuePanel;
