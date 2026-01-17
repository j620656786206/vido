import { PosterCard } from './PosterCard';
import { PosterCardSkeleton } from './PosterCardSkeleton';
import type { Movie, TVShow } from '../../types/tmdb';

export interface MediaGridProps {
  movies?: Movie[];
  tvShows?: TVShow[];
  isLoading?: boolean;
  emptyMessage?: string;
}

export function MediaGrid({
  movies = [],
  tvShows = [],
  isLoading,
  emptyMessage = '沒有找到結果',
}: MediaGridProps) {
  if (isLoading) {
    return (
      <div
        data-testid="media-grid"
        className="grid grid-cols-2 gap-3 sm:grid-cols-[repeat(auto-fill,minmax(160px,1fr))] sm:gap-4 lg:grid-cols-[repeat(auto-fill,minmax(200px,1fr))]"
      >
        {Array.from({ length: 12 }).map((_, i) => (
          <div key={i} data-testid="poster-card-skeleton">
            <PosterCardSkeleton />
          </div>
        ))}
      </div>
    );
  }

  const hasResults = movies.length > 0 || tvShows.length > 0;

  if (!hasResults) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-gray-400">
        <span className="mb-2 text-4xl">🔍</span>
        <p>{emptyMessage}</p>
      </div>
    );
  }

  return (
    <div
      data-testid="media-grid"
      className="grid grid-cols-2 gap-3 sm:grid-cols-[repeat(auto-fill,minmax(160px,1fr))] sm:gap-4 lg:grid-cols-[repeat(auto-fill,minmax(200px,1fr))]"
    >
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
    </div>
  );
}
