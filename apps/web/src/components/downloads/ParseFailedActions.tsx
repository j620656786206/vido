import { RotateCcw, Search } from 'lucide-react';
import { cn } from '../../lib/utils';

interface ParseFailedActionsProps {
  torrentHash: string;
  errorMessage?: string;
  onRetry: () => void;
  onManualSearch?: (hash: string) => void;
  isRetrying?: boolean;
  className?: string;
}

export function ParseFailedActions({
  torrentHash,
  errorMessage,
  onRetry,
  onManualSearch,
  isRetrying = false,
  className,
}: ParseFailedActionsProps) {
  return (
    <div className={cn('flex flex-col gap-2', className)} data-testid="parse-failed-actions">
      {errorMessage && <p className="text-xs text-[var(--error)]">{errorMessage}</p>}
      <div className="flex gap-2">
        <button
          type="button"
          className={cn(
            'inline-flex items-center gap-1 rounded px-2 py-1 text-xs',
            'border border-[var(--border-subtle)] text-[var(--text-secondary)] transition-colors',
            'hover:border-[var(--text-muted)] hover:text-white',
            'disabled:cursor-not-allowed disabled:opacity-50'
          )}
          onClick={onRetry}
          disabled={isRetrying}
          aria-label="重試解析"
        >
          <RotateCcw className={cn('h-3 w-3', isRetrying && 'animate-spin')} />
          重試
        </button>
        {onManualSearch && (
          <button
            type="button"
            className={cn(
              'inline-flex items-center gap-1 rounded px-2 py-1 text-xs',
              'border border-[var(--border-subtle)] text-[var(--text-secondary)] transition-colors',
              'hover:border-[var(--text-muted)] hover:text-white'
            )}
            onClick={() => onManualSearch(torrentHash)}
            aria-label="手動搜尋"
          >
            <Search className="h-3 w-3" />
            手動搜尋
          </button>
        )}
      </div>
    </div>
  );
}
