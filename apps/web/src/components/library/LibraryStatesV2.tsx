// Design ref: ux-design.pen Screen A2p-D (EsoIv) · A7p-D (R3FqJc) · A8p-D (dVGIa)
// Skeleton / no-result / error, one frame each. (Was still carrying the 19-8
// `pending` placeholder, which Rule 21 says no components/ file should have.)
/**
 * v2 Browse state components (UX Redesign Phase 2 — UX2-2, AC #6, §7).
 * Loading skeleton (grid-shaped, reduced-motion aware), no-result (distinct from
 * empty — acknowledges the filter), and a per-section fail-soft error (compact
 * inline + error code + retry; the page never hard-fails — F3). The Empty state
 * reuses the existing EmptyLibrary 3-state classifier in the container.
 */
import { SearchX, AlertTriangle } from 'lucide-react';
import type { LibraryMediaType } from '../../types/library';

/**
 * 沒有「電影」符合… — the subject of the no-result sentence.
 *
 * `all` is 「內容」, NOT 「項目」: the page header four lines above counts the same
 * rows in 部 (⚖️ 2026-09-16), and one screen saying 「媒體庫 0 部」 over
 * 「沒有項目符合…」 is the same two-voices problem that ruling was made to end.
 * (Whether 部 is right for the 影集 and mixed views at all is open —
 * disc-2026-09-count-noun-only-evidenced-on-movies.)
 */
const TYPE_NOUN: Record<LibraryMediaType, string> = {
  all: '內容',
  movie: '電影',
  tv: '影集',
};

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

/**
 * No-result — distinct from Empty; names what excluded everything.
 *
 * A7p-D says 「沒有電影符合目前的篩選條件（4K）。」 rather than a subjectless
 * 「試著調整或清除目前的篩選條件。」, and the difference matters: the generic line
 * sends you hunting for which filter did it, when the answer is already known here.
 * With nothing filtering there is no subject to name, so the generic line is the
 * honest one and stays as the fallback.
 */
export function LibraryNoResultV2({
  onClearFilters,
  mediaType = 'all',
  activeFilters = [],
}: {
  onClearFilters: () => void;
  mediaType?: LibraryMediaType;
  /** Human labels of the filters currently narrowing the query, in chip order. */
  activeFilters?: string[];
}) {
  const detail =
    activeFilters.length > 0
      ? `沒有${TYPE_NOUN[mediaType]}符合目前的篩選條件（${activeFilters.join('、')}）。試著調整或清除篩選。`
      : '試著調整或清除目前的篩選條件。';
  return (
    <div
      data-testid="library-no-result"
      className="flex flex-col items-center justify-center rounded-[var(--radius-lg)] bg-[var(--bg-secondary)] px-6 py-16 text-center"
    >
      <SearchX className="h-10 w-10 text-[var(--text-muted)]" aria-hidden="true" />
      <h2 className="mt-3 text-base font-semibold text-[var(--text-primary)]">找不到符合的結果</h2>
      <p className="mt-1 max-w-md text-sm text-[var(--text-secondary)]">{detail}</p>
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
      <AlertTriangle className="h-10 w-10 text-[var(--error-text)]" aria-hidden="true" />
      <h2 className="mt-3 text-base font-semibold text-[var(--text-primary)]">無法載入媒體庫</h2>
      {/* A8p-D. The reassurance is the point: when the library will not load, the
          first thought is 「我的檔案還在嗎」, and a bare 「請稍後再試」 leaves it
          hanging. The query failed; the files on disk were never touched. */}
      {/* --text-secondary, NOT --error-text. A8p-D renders this line neutral, and
          for good reason: the sentence exists to REASSURE ("你的檔案沒有受影響"),
          and alarm-red works against the only thing it is there to do. The red
          belongs to the icon and the heading above it. */}
      <p className="mt-1 max-w-md text-sm text-[var(--text-secondary)]">
        媒體庫資料查詢失敗，你的檔案沒有受影響。
      </p>
      {/* A8p-D puts the code on its own line as a mono pill. Inline it produced
          「…沒有受影響。（DB_QUERY_FAILED）」 — a sentence-final 。 with a
          parenthetical hanging off it, which reads as an afterthought rather than
          the thing you paste into a bug report. */}
      {code ? (
        <span
          data-testid="library-error-code"
          className="mt-2 rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)] px-2 py-0.5 font-mono text-[11px] text-[var(--text-muted)]"
        >
          {code}
        </span>
      ) : null}
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
