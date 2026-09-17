// Design ref: ux-design.pen — no current screen frame; /search（TMDb 搜尋結果頁）沒有設計稿，見 disc-2026-09-search-page-no-design
import { MediaGrid, type MediaItem } from '../media/MediaGrid';
import { Pagination } from '../ui/Pagination';
import type { MovieSearchResponse, TVShowSearchResponse } from '../../types/tmdb';

interface SearchResultsProps {
  movies?: MovieSearchResponse;
  tvShows?: TVShowSearchResponse;
  isLoading: boolean;
  type: 'all' | 'movie' | 'tv';
  currentPage: number;
  onPageChange: (page: number) => void;
  className?: string;
  /** dsr-8 AC #5: every query the current tab needs failed and has nothing to show. */
  isError?: boolean;
  /** One side failed while the other answered (type=all only). */
  failedSide?: 'movie' | 'tv';
  /** Rule-7 code of the failure, shown as a mono pill. */
  errorCode?: string;
  /** Refetch only the failed query/queries. */
  onRetry?: () => void;
  retrying?: boolean;
}

const retryClass =
  'min-h-[44px] rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-3 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-secondary)] disabled:cursor-wait disabled:opacity-70';
const codePillClass =
  'rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)] px-2 py-0.5 font-mono text-[11px] text-[var(--text-muted)]';

export function SearchResults({
  movies,
  tvShows,
  isLoading,
  type,
  currentPage,
  onPageChange,
  className,
  isError = false,
  failedSide,
  errorCode,
  onRetry,
  retrying = false,
}: SearchResultsProps) {
  // dsr-8 AC #5: the whole search failed — say so, never 「找不到符合的結果」 (that
  // blames the user's words for a server fault).
  if (isError) {
    return (
      <div className={className}>
        <div
          role="alert"
          data-testid="search-error"
          className="flex flex-col items-center gap-3 rounded-[var(--radius-lg)] bg-[var(--error-tint)] px-6 py-12 text-center"
        >
          <p className="text-sm font-medium text-[var(--error-text)]">
            搜尋暫時無法使用，請稍後再試
          </p>
          {errorCode ? (
            <span data-testid="search-error-code" className={codePillClass}>
              {errorCode}
            </span>
          ) : null}
          {onRetry && (
            <button
              type="button"
              onClick={onRetry}
              disabled={retrying}
              data-testid="search-error-retry"
              className={retryClass}
            >
              {retrying ? '重試中…' : '重試'}
            </button>
          )}
        </div>
      </div>
    );
  }

  // Filter results based on type
  const movieResults = type === 'all' || type === 'movie' ? movies?.results || [] : [];
  const tvResults = type === 'all' || type === 'tv' ? tvShows?.results || [] : [];

  // For 'all' type, create unified sorted array to preserve interleaved order
  let sortedItems: MediaItem[] | undefined;
  let sortedMovies = movieResults;
  let sortedTvShows = tvResults;

  if (type === 'all') {
    // Create combined array for sorting - preserves interleaved order
    sortedItems = [
      ...movieResults.map((item): MediaItem => ({ item, mediaType: 'movie' })),
      ...tvResults.map((item): MediaItem => ({ item, mediaType: 'tv' })),
    ];
    sortedItems.sort((a, b) => b.item.voteCount - a.item.voteCount);
    // Clear individual arrays since we're using unified items
    sortedMovies = [];
    sortedTvShows = [];
  }

  // Calculate totals
  const totalResults =
    (type === 'all' || type === 'movie' ? movies?.totalResults || 0 : 0) +
    (type === 'all' || type === 'tv' ? tvShows?.totalResults || 0 : 0);

  // Calculate total pages based on type filter
  let totalPages = 1;
  if (type === 'movie') {
    totalPages = movies?.totalPages || 1;
  } else if (type === 'tv') {
    totalPages = tvShows?.totalPages || 1;
  } else {
    // For 'all' type, use the max of both
    totalPages = Math.max(movies?.totalPages || 1, tvShows?.totalPages || 1);
  }

  const hasResults = sortedItems
    ? sortedItems.length > 0
    : sortedMovies.length > 0 || sortedTvShows.length > 0;

  return (
    <div className={className}>
      {/* One side failed: keep the good results, say which side is missing. */}
      {failedSide && (
        <div
          role="alert"
          data-testid="search-partial-error"
          className="mb-4 flex flex-wrap items-center gap-3 rounded-[var(--radius-md)] bg-[var(--error-tint)] px-4 py-3 text-sm"
        >
          <span className="text-[var(--error-text)]">
            {failedSide === 'movie' ? '電影' : '影集'}結果暫時無法載入，其他結果不受影響
          </span>
          {errorCode ? <span className={codePillClass}>{errorCode}</span> : null}
          {onRetry && (
            <button
              type="button"
              onClick={onRetry}
              disabled={retrying}
              data-testid="search-partial-error-retry"
              className={`ml-auto ${retryClass}`}
            >
              {retrying ? '重試中…' : '重試'}
            </button>
          )}
        </div>
      )}

      {/* Results count — not while a side is missing: it would count only half. */}
      {!isLoading && hasResults && !failedSide && (
        <div className="mb-4 text-sm text-[var(--text-secondary)]">找到 {totalResults} 個結果</div>
      )}

      {/* Grid results — no "try other keywords" when the empty half is only the half
          that answered: the banner above already says the other half is missing. */}
      {!(failedSide && !isLoading && !hasResults) && (
        <MediaGrid
          items={sortedItems}
          movies={sortedMovies}
          tvShows={sortedTvShows}
          isLoading={isLoading}
          emptyMessage="找不到符合的結果，請嘗試使用不同的關鍵字搜尋"
        />
      )}

      {/* Pagination */}
      {!isLoading && hasResults && totalPages > 1 && (
        <Pagination
          currentPage={currentPage}
          totalPages={totalPages}
          onPageChange={onPageChange}
          className="mt-8"
        />
      )}
    </div>
  );
}
