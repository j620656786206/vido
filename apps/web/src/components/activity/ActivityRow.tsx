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
  success: { icon: 'text-[var(--success)]', wrap: 'bg-[var(--success-tint)]' },
  error: { icon: 'text-[var(--error)]', wrap: 'bg-[var(--error-tint)]' },
};

interface ActivityRowProps {
  icon: LucideIcon;
  iconTone?: RowTone;
  title: string;
  detail?: string;
  /** Right slot — percent/count text, a timestamp, or a CTA link. */
  right?: ReactNode;
  /** 0–100; renders the progress bar when provided. */
  progress?: number;
  testId?: string;
}

export function ActivityRow({
  icon: Icon,
  iconTone = 'neutral',
  title,
  detail,
  right,
  progress,
  testId,
}: ActivityRowProps) {
  const tone = TONE[iconTone];
  return (
    <div
      data-testid={testId}
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
          {detail && <p className="truncate text-[13px] text-[var(--text-secondary)]">{detail}</p>}
        </div>
        {right != null && <div className="shrink-0 text-right">{right}</div>}
      </div>
      {typeof progress === 'number' && (
        <div
          className="h-1.5 overflow-hidden rounded-full bg-[var(--bg-tertiary)]"
          role="progressbar"
          aria-valuenow={Math.round(progress)}
          aria-valuemin={0}
          aria-valuemax={100}
        >
          <div
            className="h-full rounded-full bg-[var(--accent-primary)] transition-[width] duration-500 motion-reduce:transition-none"
            style={{ width: `${Math.min(100, Math.max(0, progress))}%` }}
          />
        </div>
      )}
    </div>
  );
}
