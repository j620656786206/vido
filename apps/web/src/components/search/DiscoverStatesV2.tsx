// Design ref: ux-design.pen Screen I6-D-v2 (YYEBd)
/**
 * ux3-3-2 (AC #8): the v2 Discover non-default states — match ux3-3-1 frames
 * I6 (loading skeleton), I7 (no-result, distinct from empty, with active-filter
 * echo) and I8 (per-section fail-soft: a compact inline error + 重試 that lets the
 * rest of the page keep rendering — the page never hard-fails, F3).
 */
import { SearchX, AlertTriangle } from 'lucide-react';

/** Grid-shaped skeleton (auto-fill, mirrors MediaGrid columns; reduced-motion aware). */
export function DiscoverGridSkeletonV2({ count = 12 }: { count?: number }) {
  return (
    <div
      data-testid="discover-grid-skeleton"
      aria-busy="true"
      aria-label="載入中"
      className="grid grid-cols-2 gap-3 md:grid-cols-[repeat(auto-fill,minmax(160px,1fr))] md:gap-4 lg:grid-cols-[repeat(auto-fill,minmax(200px,1fr))]"
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

/**
 * No-result — distinct from empty (I7). Echoes the active filters so it is clear
 * WHY nothing matched, and offers a one-tap clear.
 */
export function DiscoverNoResultV2({
  activeLabels,
  onClearFilters,
}: {
  activeLabels: string[];
  onClearFilters: () => void;
}) {
  return (
    <div
      data-testid="discover-no-result"
      className="flex flex-col items-center justify-center rounded-[var(--radius-lg)] bg-[var(--bg-secondary)] px-6 py-16 text-center"
    >
      <SearchX className="h-10 w-10 text-[var(--text-muted)]" aria-hidden="true" />
      <h3 className="mt-3 text-base font-semibold text-[var(--text-primary)]">找不到相符的結果</h3>
      {activeLabels.length > 0 && (
        <p
          data-testid="discover-no-result-echo"
          className="mt-1 text-sm text-[var(--text-secondary)]"
        >
          目前篩選：{activeLabels.join(' · ')}
        </p>
      )}
      <button
        type="button"
        onClick={onClearFilters}
        data-testid="discover-no-result-clear"
        className="mt-4 min-h-[44px] rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-4 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--accent-subtle)]"
      >
        清除篩選
      </button>
    </div>
  );
}

/**
 * Per-section fail-soft (I8) — a COMPACT inline banner so a failed TMDB section
 * degrades in place while the rest of the page keeps rendering. `message`
 * customizes the copy (e.g. which section failed).
 */
export function DiscoverSectionErrorV2({
  message = 'TMDB 服務暫時無法連線，其他結果不受影響',
  code,
  onRetry,
}: {
  message?: string;
  code?: string;
  onRetry: () => void;
}) {
  return (
    <div
      data-testid="discover-section-error"
      role="alert"
      className="mb-4 flex flex-wrap items-center gap-3 rounded-[var(--radius-md)] bg-[var(--error-tint)] px-4 py-3 text-sm"
    >
      <AlertTriangle className="h-5 w-5 shrink-0 text-[var(--error)]" aria-hidden="true" />
      <span className="text-[var(--error-text)]">
        {message}
        {code ? (
          <span className="ml-1 font-mono text-[11px] text-[var(--text-muted)]">（{code}）</span>
        ) : null}
      </span>
      <button
        type="button"
        onClick={onRetry}
        data-testid="discover-section-error-retry"
        className="ml-auto min-h-[44px] rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-3 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-secondary)]"
      >
        重試
      </button>
    </div>
  );
}
