// Implements: Component/ActivityRow-v2 (fF8nX)
/**
 * Activity hub explain-why row (UX Redesign Phase 3 — ux3-2-3).
 *
 * `[icon-chip · title + why-detail · right-slot]` over an optional progress bar. The
 * backend sends copy-free `kind`/`result` enums (ux3-2-2); the hub maps those to the
 * icon + title + tone here, so all human copy lives on the client (i18n). Token-only
 * colors; Noto for CJK; the right slot carries a JetBrains-Mono percent/count, a status
 * word, or an accent CTA (passed as a node).
 */
import type { LucideIcon } from 'lucide-react';
import type { ReactNode } from 'react';

export type RowTone = 'neutral' | 'success' | 'error';

const TONE: Record<RowTone, { icon: string; wrap: string }> = {
  neutral: { icon: 'text-[var(--text-secondary)]', wrap: 'bg-[var(--bg-tertiary)]' },
  success: { icon: 'text-[var(--success-text)]', wrap: 'bg-[var(--success-tint)]' },
  error: { icon: 'text-[var(--error-text)]', wrap: 'bg-[var(--error-tint)]' },
};

interface ActivityRowProps {
  icon: LucideIcon;
  iconTone?: RowTone;
  title: string;
  /**
   * Why-detail line. A plain string for most rows; a node when segments need their
   * own token color (bugfix-e: the downloads row tints its 錯誤/暫停 counts).
   */
  detail?: ReactNode;
  /** Right slot — percent/count text, a timestamp, or a CTA link. */
  right?: ReactNode;
  /** 0–100; renders the progress bar when provided. */
  progress?: number;
  testId?: string;
  /**
   * A second fact line — the subtitle row's 「3 分鐘 · US$0.27」
   * (disc-2026-10-activity-subtitle-run-time-b, ⚖️ Sally ruling 6). ONE node that
   * the grid moves: above `right` on the right from `sm` up (K1-D-v2 LndN2), a
   * third line under the detail on a phone (K1-M-v2 JLTBV), where `right` keeps
   * only the time. Never two copies toggled with `hidden` — a screen reader and a
   * test would read it twice. Rows without it render exactly as before.
   */
  meta?: ReactNode;
  /** Let the detail wrap below `sm` instead of truncating (a failure reason must be readable on a phone). */
  wrapDetailOnMobile?: boolean;
  /** Rendered as `data-kind` on the row (the activity record's parse / subtitle rows). */
  kind?: string;
}

export function ActivityRow({
  icon: Icon,
  iconTone = 'neutral',
  title,
  detail,
  right,
  progress,
  testId,
  meta,
  wrapDetailOnMobile = false,
  kind,
}: ActivityRowProps) {
  const tone = TONE[iconTone];
  const detailClass = `${wrapDetailOnMobile ? 'break-words sm:truncate' : 'truncate'} text-xs text-[var(--text-secondary)]`;
  const progressBar = typeof progress === 'number' && (
    <div
      className="h-1.5 overflow-hidden rounded-full bg-[var(--bg-tertiary)]"
      role="progressbar"
      aria-valuenow={Math.round(progress)}
      aria-valuemin={0}
      aria-valuemax={100}
    >
      <div
        className="h-full rounded-full bg-[var(--accent-primary)] transition-[width] duration-[var(--motion-move)] motion-reduce:transition-none"
        style={{ width: `${Math.min(100, Math.max(0, progress))}%` }}
      />
    </div>
  );
  // Truthy, not != null: an empty string is no meta line, and keeps the old layout.
  if (meta) {
    return (
      <div
        data-testid={testId}
        data-kind={kind}
        className="flex flex-col gap-2.5 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-4"
      >
        {/*
          Phone:   icon | title+detail | time        Desktop: icon | title+detail | meta
                   icon | meta         | time                 icon | title+detail | time
          Placement only — the DOM order (title, detail, meta, time) is the reading order.
          The meta/time cells are flex so their spans size by their own line-height,
          not the body's 24px strut (which made these rows ~10px taller than the rest).
        */}
        <div className="grid grid-cols-[auto_minmax(0,1fr)_auto] gap-x-3.5 gap-y-1 sm:gap-y-0.5">
          <div
            className={`col-start-1 row-span-2 row-start-1 flex h-9 w-9 shrink-0 items-center justify-center self-center rounded-[var(--radius-md)] ${tone.wrap}`}
          >
            <Icon className={`h-5 w-5 ${tone.icon}`} aria-hidden="true" />
          </div>
          <div className="col-start-2 row-start-1 min-w-0 self-end sm:row-span-2 sm:self-center">
            <h3 className="truncate text-sm font-semibold text-[var(--text-primary)]">{title}</h3>
            {detail && <p className={detailClass}>{detail}</p>}
          </div>
          <div className="col-start-2 row-start-2 flex min-w-0 self-start sm:col-start-3 sm:row-start-1 sm:justify-end sm:self-end sm:whitespace-nowrap">
            {meta}
          </div>
          {right != null && (
            <div className="col-start-3 row-span-2 row-start-1 flex shrink-0 justify-end self-center sm:row-span-1 sm:row-start-2 sm:self-start">
              {right}
            </div>
          )}
        </div>
        {progressBar}
      </div>
    );
  }
  return (
    <div
      data-testid={testId}
      data-kind={kind}
      className="flex flex-col gap-2.5 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-4"
    >
      <div className="flex items-center gap-3.5">
        <div
          className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-[var(--radius-md)] ${tone.wrap}`}
        >
          <Icon className={`h-5 w-5 ${tone.icon}`} aria-hidden="true" />
        </div>
        <div className="min-w-0 flex-1">
          <h3 className="truncate text-sm font-semibold text-[var(--text-primary)]">{title}</h3>
          {detail && <p className={detailClass}>{detail}</p>}
        </div>
        {right != null && <div className="shrink-0 text-right">{right}</div>}
      </div>
      {progressBar}
    </div>
  );
}
