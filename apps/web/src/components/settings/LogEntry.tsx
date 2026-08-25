// Design ref: ux-design.pen — no current screen frame; 系統日誌 tab was never given a frame — rides the designed settings shell (Screen C4-D, 6UCtX)
import { useState } from 'react';
import { ChevronDown, ChevronRight, Lightbulb } from 'lucide-react';
import { cn } from '../../lib/utils';
import type { SystemLog } from '../../services/logService';

const LEVEL_STYLES: Record<string, string> = {
  ERROR: 'text-[var(--error-text)] bg-[var(--error-tint)]',
  WARN: 'text-[var(--warning-text)] bg-[var(--warning-tint)]',
  // INFO wears --info-*, not gold: gold is 你在這裡, and thousands of log rows
  // wearing it would dilute the one colour that must stay rare.
  INFO: 'text-[var(--info-text)] bg-[var(--info-tint)]',
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
          // 44px hit area on touch (the glyph stays 16px); desktop reverts to compact.
          className="-m-1.5 flex min-h-[44px] min-w-[44px] items-center justify-center p-1.5 text-[var(--text-muted)] hover:text-[var(--text-secondary)] sm:mt-0.5 sm:block sm:min-h-0 sm:min-w-0 sm:p-0 sm:-m-0"
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
            <div className="flex items-start gap-2 rounded bg-[var(--warning-tint)] px-3 py-2 text-sm text-[var(--warning-text)]">
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
