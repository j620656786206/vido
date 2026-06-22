// Design ref: ux-design.pen Screen A4-D-v2 (suCiI)
/**
 * Activity hub state components (UX Redesign Phase 3 — ux3-2-3, §7 / N4).
 * Loading skeleton (row-shaped, reduced-motion aware, A4-D-v2), calm empty state with a
 * next-step CTA (A5-D-v2), and a compact per-section fail-soft banner + 重試 (A6-D-v2 —
 * a failed section degrades alone; the page never hard-fails, F3).
 */
import { Link } from '@tanstack/react-router';
import { AlertTriangle, RotateCw, CheckCheck, Radar } from 'lucide-react';

/** Skeleton matching the row shape — icon chip + two text bars + a right bar. */
export function ActivitySkeleton({ count = 4 }: { count?: number }) {
  return (
    <div
      data-testid="activity-skeleton"
      aria-busy="true"
      aria-label="載入中"
      className="flex flex-col gap-3"
    >
      {Array.from({ length: count }).map((_, i) => (
        <div
          key={i}
          className="flex items-center gap-3.5 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-4"
        >
          <div className="h-9 w-9 shrink-0 animate-pulse rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] motion-reduce:animate-none" />
          <div className="flex min-w-0 flex-1 flex-col gap-2">
            <div className="h-3.5 w-1/3 animate-pulse rounded bg-[var(--bg-tertiary)] motion-reduce:animate-none" />
            <div className="h-2.5 w-2/3 animate-pulse rounded bg-[var(--bg-tertiary)] motion-reduce:animate-none" />
          </div>
          <div className="h-3 w-10 shrink-0 animate-pulse rounded bg-[var(--bg-tertiary)] motion-reduce:animate-none" />
        </div>
      ))}
    </div>
  );
}

/** Empty — no activity. Calm card + a next-step CTA (N "always show the next step"). */
export function ActivityEmpty() {
  return (
    <div
      data-testid="activity-empty"
      className="flex flex-col items-center justify-center gap-4 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-6 py-16 text-center"
    >
      <div className="flex h-16 w-16 items-center justify-center rounded-full bg-[var(--bg-tertiary)]">
        <CheckCheck className="h-8 w-8 text-[var(--text-secondary)]" aria-hidden="true" />
      </div>
      <div className="flex flex-col gap-1">
        <h3 className="text-base font-semibold text-[var(--text-primary)]">目前沒有進行中的活動</h3>
        <p className="max-w-sm text-sm text-[var(--text-secondary)]">
          掃描、字幕與下載工作會在這裡顯示。一切都已完成。
        </p>
      </div>
      <Link
        to="/library"
        data-testid="activity-empty-cta"
        className="inline-flex min-h-[44px] items-center gap-2 rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-4 text-sm font-semibold text-[var(--text-on-accent)] transition-colors hover:bg-[var(--accent-hover)]"
      >
        <Radar className="h-4 w-4" aria-hidden="true" />
        前往媒體庫
      </Link>
    </div>
  );
}

/** Per-section fail-soft — compact inline banner + retry; the page still renders (F3). */
export function ActivitySectionError({
  onRetry,
  testId,
}: {
  onRetry: () => void;
  testId?: string;
}) {
  return (
    <div
      data-testid={testId ?? 'activity-section-error'}
      role="alert"
      className="flex items-center justify-between gap-3 rounded-[var(--radius-lg)] bg-[var(--error-tint)] px-4 py-3"
    >
      <p className="flex items-center gap-2 text-sm font-medium text-[var(--error-text)]">
        <AlertTriangle className="h-4 w-4 shrink-0 text-[var(--error)]" aria-hidden="true" />
        無法載入，請稍後再試
      </p>
      <button
        type="button"
        onClick={onRetry}
        data-testid="activity-section-retry"
        className="flex min-h-[44px] shrink-0 items-center gap-1.5 rounded-[var(--radius-md)] border border-[var(--error)] px-3 text-sm font-medium text-[var(--error-text)] transition-colors hover:bg-[var(--error)]/10"
      >
        <RotateCw className="h-4 w-4" aria-hidden="true" />
        重試
      </button>
    </div>
  );
}
