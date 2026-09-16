// Design ref: ux-design.pen Screen B7p-D (Tqy3E) + Screen B6p-D (Z42zy)
/**
 * v2 detail page states (UX Redesign Phase 2 — UX2-3, AC #7, §7).
 * Loading = hero + body skeleton (no spinner; progressive per-section hydrate).
 * Not-found = a back affordance + centered message + return CTA — never a blank or
 * technical page (brief P3).
 * Load error (dsr-2 AC #8) = its own state. ux2-3 AC #7 always asked for not-found
 * and load-error to be distinct; they were merged, so a server failure told the
 * user the item had been removed.
 */
import { AlertTriangle, ArrowLeft, FilmIcon } from 'lucide-react';

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

export function DetailNotFoundV2({
  onBack,
  inLibrary = true,
}: {
  onBack: () => void;
  /**
   * False for a TMDb-only title or a malformed route: those were never in the
   * library, so "removed from the library" would be a false explanation.
   */
  inLibrary?: boolean;
}) {
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
          {inLibrary ? '這個項目可能已從媒體庫移除，或連結已失效。' : '這個連結可能已失效。'}
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

interface DetailLoadErrorV2Props {
  onRetry: () => void;
  onBack: () => void;
  /** Rule-7 error code from the API, shown as a copyable mono pill (A8p-D shape). */
  code?: string;
  /**
   * Library items only. A TMDb-only title is not on disk, so "your files are fine"
   * would reassure about something that does not exist.
   */
  reassureFiles?: boolean;
  /** A retry is in flight — the button says so and ignores further clicks. */
  retrying?: boolean;
}

export function DetailLoadErrorV2({
  onRetry,
  onBack,
  code,
  reassureFiles,
  retrying = false,
}: DetailLoadErrorV2Props) {
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
        data-testid="detail-load-error"
        role="alert"
        className="flex min-h-[60vh] flex-col items-center justify-center px-6 text-center"
      >
        <AlertTriangle className="h-12 w-12 text-[var(--error-text)]" aria-hidden="true" />
        <h1 className="mt-4 text-xl font-semibold text-[var(--text-primary)]">無法載入這部影片</h1>
        {/* Same shape and reasoning as LibraryErrorV2 (dsr-1): the reassurance is the
            point, so it is neutral --text-secondary — alarm red would fight it. */}
        <p className="mt-1 max-w-md text-sm text-[var(--text-secondary)]">
          {reassureFiles
            ? '詳情資料查詢失敗，你的檔案沒有受影響。'
            : '詳情資料暫時無法取得，請稍後再試。'}
        </p>
        {code ? (
          <span
            data-testid="detail-load-error-code"
            className="mt-2 rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)] px-2 py-0.5 font-mono text-[11px] text-[var(--text-muted)]"
          >
            {code}
          </span>
        ) : null}
        <div className="mt-5 flex flex-wrap items-center justify-center gap-2">
          <button
            type="button"
            onClick={onRetry}
            disabled={retrying}
            data-testid="detail-load-error-retry"
            className="min-h-[44px] rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-5 text-sm font-medium text-[var(--text-on-accent)] transition-colors hover:bg-[var(--accent-pressed)] disabled:cursor-wait disabled:opacity-70"
          >
            {retrying ? '重試中…' : '重試'}
          </button>
          <button
            type="button"
            onClick={onBack}
            className="min-h-[44px] rounded-[var(--radius-md)] bg-[var(--bg-secondary)] px-5 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-tertiary)]"
          >
            返回媒體庫
          </button>
        </div>
      </div>
    </div>
  );
}
