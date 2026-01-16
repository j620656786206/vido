import { cn } from '../../lib/utils';
import { getImageUrl } from '../../services/tmdb';
import { SearchResultSkeleton } from './SearchResultSkeleton';
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

interface MediaItemProps {
  item: Movie | TVShow;
  mediaType: 'movie' | 'tv';
}

function isMovie(item: Movie | TVShow): item is Movie {
  return 'title' in item;
}

function MediaItem({ item, mediaType }: MediaItemProps) {
  const title = isMovie(item) ? item.title : item.name;
  const originalTitle = isMovie(item) ? item.original_title : item.original_name;
  const releaseDate = isMovie(item) ? item.release_date : item.first_air_date;
  const year = releaseDate ? new Date(releaseDate).getFullYear() : null;
  const posterUrl = getImageUrl(item.poster_path, 'w92');

  return (
    <div className="flex space-x-4 p-4 bg-slate-800 rounded-lg hover:bg-slate-750 transition-colors cursor-pointer">
      {/* Poster */}
      <div className="w-16 h-24 bg-slate-700 rounded flex-shrink-0 overflow-hidden">
        {posterUrl ? (
          <img
            src={posterUrl}
            alt={title}
            className="w-full h-full object-cover"
            loading="lazy"
          />
        ) : (
          <div className="w-full h-full flex items-center justify-center text-slate-500 text-xs">
            無圖片
          </div>
        )}
      </div>

      {/* Content */}
      <div className="flex-1 min-w-0">
        {/* Primary title (zh-TW) */}
        <h3 className="text-white font-medium truncate" title={title}>
          {title}
        </h3>

        {/* Original title (if different) */}
        {originalTitle && originalTitle !== title && (
          <p className="text-slate-400 text-sm truncate" title={originalTitle}>
            {originalTitle}
          </p>
        )}

        {/* Year and media type */}
        <div className="flex items-center space-x-2 mt-2">
          {year && <span className="text-slate-400 text-sm">{year}</span>}
          <span
            className={cn(
              'px-2 py-0.5 text-xs font-medium rounded-full',
              mediaType === 'movie'
                ? 'bg-blue-500/20 text-blue-400'
                : 'bg-purple-500/20 text-purple-400'
            )}
          >
            {mediaType === 'movie' ? '電影' : '影集'}
          </span>
        </div>
      </div>
    </div>
  );
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
  if (isLoading) {
    return <SearchResultSkeleton count={5} className={className} />;
  }

  const movieResults = movies?.results || [];
  const tvResults = tvShows?.results || [];

  // Combine and filter results based on type
  let results: Array<{ item: Movie | TVShow; mediaType: 'movie' | 'tv' }> = [];

  if (type === 'all' || type === 'movie') {
    results = results.concat(
      movieResults.map((item) => ({ item, mediaType: 'movie' as const }))
    );
  }

  if (type === 'all' || type === 'tv') {
    results = results.concat(
      tvResults.map((item) => ({ item, mediaType: 'tv' as const }))
    );
  }

  // Sort by popularity (vote_count) for 'all' type
  if (type === 'all') {
    results.sort((a, b) => b.item.vote_count - a.item.vote_count);
  }

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

  if (results.length === 0) {
    return (
      <div className={cn('text-center py-12', className)}>
        <p className="text-slate-400 text-lg">找不到符合的結果</p>
        <p className="text-slate-500 text-sm mt-2">
          請嘗試使用不同的關鍵字搜尋
        </p>
      </div>
    );
  }

  return (
    <div className={className}>
      {/* Results count */}
      <div className="text-slate-400 text-sm mb-4">
        找到 {totalResults} 個結果
      </div>

      {/* Results list */}
      <div className="space-y-3">
        {results.map(({ item, mediaType }) => (
          <MediaItem key={`${mediaType}-${item.id}`} item={item} mediaType={mediaType} />
        ))}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
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
