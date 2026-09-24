// Design ref: ux-design.pen Screen C12-D (K28SdR) · C12-M (dOEbF)
import { useState } from 'react';
import { ChevronDown, ChevronRight, Lightbulb } from 'lucide-react';
import { cn } from '../../lib/utils';
import type { SystemLog } from '../../services/logService';
import { formatLocalDateTime } from '../../utils/formatLocalDateTime';

const LEVEL_STYLES: Record<string, string> = {
  ERROR: 'text-[var(--error-text)] bg-[var(--error-tint)]',
  WARN: 'text-[var(--warning-text)] bg-[var(--warning-tint)]',
  // INFO wears --info-*, not gold: gold is 你在這裡, and thousands of log rows
  // wearing it would dilute the one colour that must stay rare.
  INFO: 'text-[var(--info-text)] bg-[var(--info-tint)]',
  DEBUG: 'text-[var(--text-muted)] bg-[var(--bg-tertiary)]',
};

interface LogEntryProps {
  log: SystemLog;
}

export function LogEntry({ log }: LogEntryProps) {
  const [expanded, setExpanded] = useState(false);
  const hasContext = log.context && Object.keys(log.context).length > 0;
  const hasHint = !!log.hint;

  // Local zone, fixed width (never toISOString — that is UTC). The phone row
  // only has room for the time of day; the date is the same for a screenful.
  const timestamp = formatLocalDateTime(log.createdAt, 'second');
  const timeOfDay = timestamp.slice(11);

  return (
    <div
      className="px-4 py-3 transition-colors hover:bg-[var(--bg-tertiary)]/40"
      data-testid="log-entry"
    >
      {/* C12-D: 箭頭 → 等級 → 時間 → 訊息 → [來源], one line.
          C12-M: two lines — 箭頭・等級・時間・來源, then the message on its own
          full-width line. On one line the phone gave the message ~80px after
          the 44px toggle and a 130px timestamp. `order-last basis-full` moves
          ONLY the message down; DOM order stays badge → time → message. */}
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 sm:flex-nowrap sm:items-start">
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

        {/* Level badge — fixed 64 so every row's time starts at the same x. */}
        <span
          className={cn(
            'inline-flex w-16 shrink-0 justify-center rounded py-0.5 font-mono text-xs font-semibold sm:mt-0.5',
            LEVEL_STYLES[log.level]
          )}
          data-testid="log-level"
        >
          {log.level}
        </span>

        <time
          dateTime={log.createdAt}
          className="shrink-0 font-mono text-xs text-[var(--text-muted)] sm:mt-1"
        >
          {/* Not aria-hidden: on a phone this is the only copy on screen (the
              full one is display:none there), so it is what gets read. */}
          <span className="sm:hidden">{timeOfDay}</span>
          <span className="hidden sm:inline" data-testid="log-timestamp">
            {timestamp}
          </span>
        </time>

        {/* 12px on the phone, per C12-M Sc3vW: a log message is a mono-ish data
            row, not Body prose, so DESIGN.md's「內文不縮」does not apply. */}
        <span
          className="order-last min-w-0 basis-full break-words text-xs text-[var(--text-primary)] sm:order-none sm:flex-1 sm:basis-auto sm:text-sm"
          data-testid="log-message"
        >
          {log.message}
        </span>
        {log.source && (
          <span
            className="shrink-0 font-mono text-xs text-[var(--text-muted)] sm:mt-1"
            data-testid="log-source"
          >
            [{log.source}]
          </span>
        )}
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
