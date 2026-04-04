import { useState } from 'react';
import { ChevronDown, ChevronRight, Lightbulb } from 'lucide-react';
import { cn } from '../../lib/utils';
import type { SystemLog } from '../../services/logService';

const LEVEL_STYLES: Record<string, string> = {
  ERROR: 'text-[var(--error)] bg-red-400/10',
  WARN: 'text-[var(--warning)] bg-yellow-400/10',
  INFO: 'text-[var(--accent-primary)] bg-blue-400/10',
  DEBUG: 'text-[var(--text-secondary)] bg-[var(--text-muted)]/10',
};

interface LogEntryProps {
  log: SystemLog;
}

export function LogEntry({ log }: LogEntryProps) {
  const [expanded, setExpanded] = useState(false);
  const hasContext = log.context && Object.keys(log.context).length > 0;
  const hasHint = !!log.hint;

  const timestamp = new Date(log.createdAt).toLocaleString('zh-TW', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });

  return (
    <div
      className="border-b border-[var(--border-subtle)]/50 px-4 py-2.5 transition-colors hover:bg-[var(--bg-secondary)]/50"
      data-testid="log-entry"
    >
      <div className="flex items-start gap-3">
        {/* Expand toggle */}
        <button
          onClick={() => setExpanded(!expanded)}
          className="mt-0.5 text-[var(--text-muted)] hover:text-[var(--text-secondary)]"
          disabled={!hasContext && !hasHint}
          aria-label={expanded ? '收合' : '展開'}
          data-testid="log-expand-btn"
        >
          {expanded ? (
            <ChevronDown className="h-4 w-4" />
          ) : (
            <ChevronRight className={cn('h-4 w-4', !hasContext && !hasHint && 'invisible')} />
          )}
        </button>

        {/* Level badge */}
        <span
          className={cn(
            'mt-0.5 rounded px-1.5 py-0.5 text-xs font-semibold',
            LEVEL_STYLES[log.level]
          )}
          data-testid="log-level"
        >
          {log.level}
        </span>

        {/* Main content */}
        <div className="min-w-0 flex-1">
          <div className="flex items-baseline gap-2">
            <span className="text-sm text-[var(--text-primary)]" data-testid="log-message">
              {log.message}
            </span>
            {log.source && (
              <span className="shrink-0 text-xs text-[var(--text-muted)]" data-testid="log-source">
                [{log.source}]
              </span>
            )}
          </div>
        </div>

        {/* Timestamp */}
        <span className="shrink-0 text-xs text-[var(--text-muted)]" data-testid="log-timestamp">
          {timestamp}
        </span>
      </div>

      {/* Expanded details */}
      {expanded && (
        <div className="ml-11 mt-2 space-y-2" data-testid="log-details">
          {hasHint && (
            <div className="flex items-start gap-2 rounded bg-yellow-900/20 px-3 py-2 text-sm text-yellow-300">
              <Lightbulb className="mt-0.5 h-4 w-4 shrink-0" />
              <span data-testid="log-hint">{log.hint}</span>
            </div>
          )}
          {hasContext && (
            <pre
              className="overflow-x-auto rounded bg-[var(--bg-primary)] px-3 py-2 text-xs text-[var(--text-secondary)]"
              data-testid="log-context"
            >
              {JSON.stringify(log.context, null, 2)}
            </pre>
          )}
        </div>
      )}
    </div>
  );
}
