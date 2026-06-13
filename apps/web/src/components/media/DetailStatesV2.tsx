// Implements: Component/Detail-Loading (Tqy3E) + Component/Detail-NotFound (Z42zy)
/**
 * v2 detail page states (UX Redesign Phase 2 — UX2-3, AC #7, §7).
 * Loading = hero + body skeleton (no spinner; progressive per-section hydrate).
 * Not-found / error = a back affordance + centered message + return CTA — never a
 * blank or technical page (brief P3).
 */
import { ArrowLeft, FilmIcon } from 'lucide-react';

export function DetailSkeletonV2() {
  return (
    <div data-testid="detail-skeleton" aria-busy="true" aria-label="載入中">
      <div className="relative">
        <div className="absolute inset-x-0 top-0 h-[300px] animate-pulse bg-[var(--bg-secondary)] motion-reduce:animate-none sm:h-[420px]" />
        <div className="relative px-4 pt-[180px] sm:px-8 sm:pt-[260px]">
          <div className="flex gap-4 sm:gap-6">
            <div className="aspect-[2/3] w-24 shrink-0 animate-pulse rounded-[var(--radius-lg)] bg-[var(--bg-tertiary)] motion-reduce:animate-none sm:w-40" />
            <div className="flex-1 space-y-3 pt-4">
              <div className="h-7 w-2/3 animate-pulse rounded bg-[var(--bg-tertiary)] motion-reduce:animate-none" />
              <div className="h-4 w-1/3 animate-pulse rounded bg-[var(--bg-secondary)] motion-reduce:animate-none" />
              <div className="h-10 w-48 animate-pulse rounded-[var(--radius-md)] bg-[var(--bg-secondary)] motion-reduce:animate-none" />
            </div>
          </div>
        </div>
      </div>
      <div className="space-y-3 px-4 py-8 sm:px-8">
        <div className="h-4 w-full animate-pulse rounded bg-[var(--bg-secondary)] motion-reduce:animate-none" />
        <div className="h-4 w-5/6 animate-pulse rounded bg-[var(--bg-secondary)] motion-reduce:animate-none" />
        <div className="h-4 w-3/4 animate-pulse rounded bg-[var(--bg-secondary)] motion-reduce:animate-none" />
      </div>
    </div>
  );
}

export function DetailNotFoundV2({ onBack }: { onBack: () => void }) {
  return (
    <div className="relative min-h-[60vh]">
      <button
        type="button"
        onClick={onBack}
        aria-label="返回媒體庫"
        data-testid="detail-back"
        className="absolute left-4 top-4 flex h-11 w-11 items-center justify-center rounded-full bg-[var(--bg-secondary)] text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)]"
      >
        <ArrowLeft className="h-5 w-5" aria-hidden="true" />
      </button>
      <div
        data-testid="detail-not-found"
        className="flex min-h-[60vh] flex-col items-center justify-center px-6 text-center"
      >
        <FilmIcon className="h-12 w-12 text-[var(--text-muted)]" aria-hidden="true" />
        <h1 className="mt-4 text-xl font-semibold text-[var(--text-primary)]">找不到這部影片</h1>
        <p className="mt-1 text-sm text-[var(--text-secondary)]">
          這個項目可能已被移除，或連結有誤。
        </p>
        <button
          type="button"
          onClick={onBack}
          className="mt-5 min-h-[44px] rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-5 text-sm font-medium text-[var(--text-on-accent)] transition-colors hover:bg-[var(--accent-pressed)]"
        >
          返回媒體庫
        </button>
      </div>
    </div>
  );
}
