// Implements: <screen-section — pending epic-19-8 mapping>
/**
 * v2 Browse state components (UX Redesign Phase 2 — UX2-2, AC #6, §7).
 * Loading skeleton (grid-shaped, reduced-motion aware), no-result (distinct from
 * empty — acknowledges the filter), and a per-section fail-soft error (compact
 * inline + error code + retry; the page never hard-fails — F3). The Empty state
 * reuses the existing EmptyLibrary 3-state classifier in the container.
 */
import { SearchX, AlertTriangle } from 'lucide-react';

/** Skeleton matching the grid shape — poster blocks + two text bars. */
export function LibraryGridSkeletonV2({ count = 12 }: { count?: number }) {
  return (
    <div
      data-testid="library-grid-skeleton"
      aria-busy="true"
      aria-label="載入中"
      className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:gap-4 lg:grid-cols-4 xl:grid-cols-6"
    >
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="flex flex-col gap-2">
          <div className="aspect-[2/3] animate-pulse rounded-[var(--radius-lg)] bg-[var(--bg-secondary)] motion-reduce:animate-none" />
          <div className="h-3.5 w-4/5 animate-pulse rounded bg-[var(--bg-secondary)] motion-reduce:animate-none" />
          <div className="h-2.5 w-2/5 animate-pulse rounded bg-[var(--bg-tertiary)] motion-reduce:animate-none" />
        </div>
      ))}
    </div>
  );
}

/** No-result — distinct from Empty; acknowledges the active filter. */
export function LibraryNoResultV2({ onClearFilters }: { onClearFilters: () => void }) {
  return (
    <div
      data-testid="library-no-result"
      className="flex flex-col items-center justify-center rounded-[var(--radius-lg)] bg-[var(--bg-secondary)] px-6 py-16 text-center"
    >
      <SearchX className="h-10 w-10 text-[var(--text-muted)]" aria-hidden="true" />
      <h3 className="mt-3 text-base font-semibold text-[var(--text-primary)]">
        沒有符合條件的項目
      </h3>
      <p className="mt-1 text-sm text-[var(--text-secondary)]">試著調整或清除目前的篩選條件。</p>
      <button
        type="button"
        onClick={onClearFilters}
        data-testid="clear-all-filters"
        className="mt-4 min-h-[44px] rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-4 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--accent-subtle)]"
      >
        清除全部篩選
      </button>
    </div>
  );
}

/** Per-section fail-soft error — compact, carries a code, offers retry. */
export function LibraryErrorV2({ code, onRetry }: { code?: string; onRetry: () => void }) {
  return (
    <div
      data-testid="library-error"
      role="alert"
      className="flex flex-col items-center justify-center rounded-[var(--radius-lg)] bg-[var(--error-tint)] px-6 py-16 text-center"
    >
      <AlertTriangle className="h-10 w-10 text-[var(--error)]" aria-hidden="true" />
      <h3 className="mt-3 text-base font-semibold text-[var(--text-primary)]">無法載入媒體庫</h3>
      <p className="mt-1 text-sm text-[var(--error-text)]">
        請稍後再試
        {code ? (
          <span className="ml-1 font-mono text-[11px] text-[var(--text-muted)]">（{code}）</span>
        ) : null}
      </p>
      <button
        type="button"
        onClick={onRetry}
        data-testid="library-error-retry"
        className="mt-4 min-h-[44px] rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-4 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-secondary)]"
      >
        重試
      </button>
    </div>
  );
}
