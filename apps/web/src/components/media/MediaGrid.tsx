import { PosterCard } from './PosterCard';
import { PosterCardSkeleton } from './PosterCardSkeleton';
import type { Movie, TVShow } from '../../types/tmdb';

export interface MediaItem {
  item: Movie | TVShow;
  mediaType: 'movie' | 'tv';
}

export interface MediaGridProps {
  /** Unified sorted array of items (preferred for mixed results) */
  items?: MediaItem[];
  /** @deprecated Use items prop for sorted mixed results */
  movies?: Movie[];
  /** @deprecated Use items prop for sorted mixed results */
  tvShows?: TVShow[];
  isLoading?: boolean;
  emptyMessage?: string;
}

export function MediaGrid({
  items,
  movies = [],
  tvShows = [],
  isLoading,
  emptyMessage = '沒有找到結果',
}: MediaGridProps) {
  if (isLoading) {
    return (
      <div
        data-testid="media-grid"
        className="grid grid-cols-2 gap-3 md:grid-cols-[repeat(auto-fill,minmax(160px,1fr))] md:gap-4 lg:grid-cols-[repeat(auto-fill,minmax(200px,1fr))]"
      >
        {Array.from({ length: 12 }).map((_, i) => (
          <div key={i} data-testid="poster-card-skeleton">
            <PosterCardSkeleton />
          </div>
        ))}
      </div>
    );
  }

  // Use unified items if provided, otherwise fall back to separate arrays
  const hasResults = items ? items.length > 0 : movies.length > 0 || tvShows.length > 0;

  if (!hasResults) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-gray-400">
        <span className="mb-2 text-4xl">🔍</span>
        <p>{emptyMessage}</p>
      </div>
    );
  }

  // Helper to render a single item
  const renderItem = (mediaItem: MediaItem) => {
    const { item, mediaType } = mediaItem;
    if (mediaType === 'movie') {
      const movie = item as Movie;
      return (
        <PosterCard
          key={`movie-${movie.id}`}
          id={movie.id}
          type="movie"
          title={movie.title}
          originalTitle={movie.original_title}
          posterPath={movie.poster_path}
          releaseDate={movie.release_date}
          voteAverage={movie.vote_average}
          overview={movie.overview}
          genreIds={movie.genre_ids}
        />
      );
    } else {
      const show = item as TVShow;
      return (
        <PosterCard
          key={`tv-${show.id}`}
          id={show.id}
          type="tv"
          title={show.name}
          originalTitle={show.original_name}
          posterPath={show.poster_path}
          releaseDate={show.first_air_date}
          voteAverage={show.vote_average}
          overview={show.overview}
          genreIds={show.genre_ids}
        />
      );
    }
  };

  return (
    <div
      data-testid="media-grid"
      className="grid grid-cols-2 gap-3 md:grid-cols-[repeat(auto-fill,minmax(160px,1fr))] md:gap-4 lg:grid-cols-[repeat(auto-fill,minmax(200px,1fr))]"
    >
      {items ? (
        items.map(renderItem)
      ) : (
        <>
          {movies.map((movie) => (
            <PosterCard
              key={`movie-${movie.id}`}
              id={movie.id}
              type="movie"
              title={movie.title}
              originalTitle={movie.original_title}
              posterPath={movie.poster_path}
              releaseDate={movie.release_date}
              voteAverage={movie.vote_average}
              overview={movie.overview}
              genreIds={movie.genre_ids}
            />
          ))}
          {tvShows.map((show) => (
            <PosterCard
              key={`tv-${show.id}`}
              id={show.id}
              type="tv"
              title={show.name}
              originalTitle={show.original_name}
              posterPath={show.poster_path}
              releaseDate={show.first_air_date}
              voteAverage={show.vote_average}
              overview={show.overview}
              genreIds={show.genre_ids}
            />
          ))}
        </>
      )}
    </div>
  );
}
