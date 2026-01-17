import { MediaGrid } from '../media/MediaGrid';
import { Pagination } from '../ui/Pagination';
import type { Movie, TVShow, MovieSearchResponse, TVShowSearchResponse } from '../../types/tmdb';

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
  const movieResults = (type === 'all' || type === 'movie') ? (movies?.results || []) : [];
  const tvResults = (type === 'all' || type === 'tv') ? (tvShows?.results || []) : [];

  // Sort combined results by popularity for 'all' type
  let sortedMovies = movieResults;
  let sortedTvShows = tvResults;

  if (type === 'all') {
    // Create combined array for sorting
    const combined: Array<{ item: Movie | TVShow; mediaType: 'movie' | 'tv' }> = [
      ...movieResults.map((item) => ({ item, mediaType: 'movie' as const })),
      ...tvResults.map((item) => ({ item, mediaType: 'tv' as const })),
    ];
    combined.sort((a, b) => b.item.vote_count - a.item.vote_count);

    // Separate back into movies and tv shows while preserving sort order
    sortedMovies = combined.filter(c => c.mediaType === 'movie').map(c => c.item as Movie);
    sortedTvShows = combined.filter(c => c.mediaType === 'tv').map(c => c.item as TVShow);
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

  const hasResults = sortedMovies.length > 0 || sortedTvShows.length > 0;

  return (
    <div className={className}>
      {/* Results count */}
      {!isLoading && hasResults && (
        <div className="mb-4 text-sm text-slate-400">
          找到 {totalResults} 個結果
        </div>
      )}

      {/* Grid results */}
      <MediaGrid
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
