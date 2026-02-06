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
}

export function SearchResults({
  movies,
  tvShows,
  isLoading,
  type,
  currentPage,
  onPageChange,
  className,
}: SearchResultsProps) {
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
    sortedItems.sort((a, b) => b.item.vote_count - a.item.vote_count);
    // Clear individual arrays since we're using unified items
    sortedMovies = [];
    sortedTvShows = [];
  }

  // Calculate totals
  const totalResults =
    (type === 'all' || type === 'movie' ? movies?.total_results || 0 : 0) +
    (type === 'all' || type === 'tv' ? tvShows?.total_results || 0 : 0);

  // Calculate total pages based on type filter
  let totalPages = 1;
  if (type === 'movie') {
    totalPages = movies?.total_pages || 1;
  } else if (type === 'tv') {
    totalPages = tvShows?.total_pages || 1;
  } else {
    // For 'all' type, use the max of both
    totalPages = Math.max(movies?.total_pages || 1, tvShows?.total_pages || 1);
  }

  const hasResults = sortedItems
    ? sortedItems.length > 0
    : sortedMovies.length > 0 || sortedTvShows.length > 0;

  return (
    <div className={className}>
      {/* Results count */}
      {!isLoading && hasResults && (
        <div className="mb-4 text-sm text-slate-400">找到 {totalResults} 個結果</div>
      )}

      {/* Grid results */}
      <MediaGrid
        items={sortedItems}
        movies={sortedMovies}
        tvShows={sortedTvShows}
        isLoading={isLoading}
        emptyMessage="找不到符合的結果，請嘗試使用不同的關鍵字搜尋"
      />

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
