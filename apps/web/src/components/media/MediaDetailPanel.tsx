import { getImageUrl } from '../../lib/image';
import type { MovieDetails, TVShowDetails, Credits } from '../../types/tmdb';

export interface MediaDetailPanelProps {
  type: 'movie' | 'tv';
  details: MovieDetails | TVShowDetails | null;
  credits?: Credits | null;
  isLoading?: boolean;
}

/**
 * MediaDetailPanel - Displays detailed information about a movie or TV show
 * AC #1, #2, #3: Full details display with loading skeleton
 */
export function MediaDetailPanel({ type, details, credits, isLoading }: MediaDetailPanelProps) {
  // Task 4.8: Loading skeleton state
  if (isLoading || !details) {
    return <MediaDetailSkeleton />;
  }

  const isMovie = type === 'movie';
  const movie = isMovie ? (details as MovieDetails) : null;
  const tvShow = !isMovie ? (details as TVShowDetails) : null;

  // Task 4.4: Title (zh-TW) and original title
  const title = isMovie ? movie!.title : tvShow!.name;
  const originalTitle = isMovie ? movie!.original_title : tvShow!.original_name;

  // Task 4.5: Year, runtime, rating
  const year = isMovie ? movie!.release_date?.slice(0, 4) : tvShow!.first_air_date?.slice(0, 4);
  const runtime = isMovie ? movie!.runtime : tvShow!.episode_run_time?.[0];

  // Task 4.2: High-resolution poster (w500 size)
  const posterUrl = getImageUrl(details.poster_path, 'w500');
  // Task 4.3: Backdrop image as header
  const backdropUrl = getImageUrl(details.backdrop_path, 'w780');

  // Find director from crew (for movies)
  const director = credits?.crew?.find((c) => c.job === 'Director');

  return (
    <div className="flex flex-col" data-testid="media-detail-panel">
      {/* Task 4.3: Backdrop header */}
      {backdropUrl && (
        <div className="relative h-48 w-full">
          <img src={backdropUrl} alt="" className="h-full w-full object-cover" loading="lazy" />
          <div className="absolute inset-0 bg-gradient-to-t from-slate-900 to-transparent" />
        </div>
      )}

      <div className="p-4">
        {/* Poster and basic info */}
        <div className="flex gap-4">
          {/* Task 4.2: High-resolution poster */}
          {posterUrl && (
            <img
              src={posterUrl}
              alt={title}
              className="h-48 w-32 flex-shrink-0 rounded-lg object-cover shadow-lg"
              loading="lazy"
              data-testid="detail-poster"
            />
          )}
          <div className="flex-1 min-w-0">
            {/* Task 4.4: Title and original title */}
            <h1 className="text-xl font-bold text-white" data-testid="detail-title">
              {title}
            </h1>
            {originalTitle && originalTitle !== title && (
              <p className="text-sm text-gray-400 truncate" data-testid="detail-original-title">
                {originalTitle}
              </p>
            )}

            {/* Task 4.5: Year, runtime, rating */}
            <div className="mt-2 flex flex-wrap items-center gap-3 text-sm text-gray-300">
              {year && <span data-testid="detail-year">{year}</span>}
              {runtime && runtime > 0 && <span data-testid="detail-runtime">{runtime} 分鐘</span>}
              {details.vote_average > 0 && (
                <span
                  className="flex items-center gap-1 text-yellow-400"
                  data-testid="detail-rating"
                >
                  ⭐ {details.vote_average.toFixed(1)}
                </span>
              )}
            </div>

            {/* Task 4.6: Genre tags as chips */}
            <div className="mt-3 flex flex-wrap gap-2" data-testid="detail-genres">
              {details.genres?.map((genre) => (
                <span
                  key={genre.id}
                  className="rounded-full bg-slate-800 px-3 py-1 text-xs text-gray-300"
                >
                  {genre.name}
                </span>
              ))}
            </div>
          </div>
        </div>

        {/* Task 4.7: Plot overview (zh-TW) */}
        <div className="mt-6">
          <h3 className="mb-2 text-sm font-semibold text-gray-400">劇情簡介</h3>
          <p className="text-sm leading-relaxed text-gray-300" data-testid="detail-overview">
            {details.overview || '暫無簡介'}
          </p>
        </div>

        {/* Director (for movies) */}
        {director && (
          <div className="mt-4">
            <span className="text-sm text-gray-400">導演：</span>
            <span className="ml-2 text-sm text-white">{director.name}</span>
          </div>
        )}

        {/* Created by (for TV shows) */}
        {tvShow?.created_by && tvShow.created_by.length > 0 && (
          <div className="mt-4">
            <span className="text-sm text-gray-400">創作者：</span>
            <span className="ml-2 text-sm text-white">
              {tvShow.created_by.map((c) => c.name).join(', ')}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}

/**
 * Task 4.8: Loading skeleton for MediaDetailPanel
 */
function MediaDetailSkeleton() {
  return (
    <div className="animate-pulse" data-testid="media-detail-skeleton">
      {/* Backdrop skeleton */}
      <div className="h-48 w-full bg-slate-700" />

      <div className="p-4">
        <div className="flex gap-4">
          {/* Poster skeleton */}
          <div className="h-48 w-32 flex-shrink-0 rounded-lg bg-slate-700" />
          <div className="flex-1 space-y-3">
            {/* Title skeleton */}
            <div className="h-6 w-3/4 rounded bg-slate-700" />
            {/* Original title skeleton */}
            <div className="h-4 w-1/2 rounded bg-slate-700" />
            {/* Meta info skeleton */}
            <div className="flex gap-3">
              <div className="h-4 w-12 rounded bg-slate-700" />
              <div className="h-4 w-16 rounded bg-slate-700" />
              <div className="h-4 w-10 rounded bg-slate-700" />
            </div>
            {/* Genre chips skeleton */}
            <div className="flex gap-2">
              <div className="h-6 w-16 rounded-full bg-slate-700" />
              <div className="h-6 w-20 rounded-full bg-slate-700" />
              <div className="h-6 w-14 rounded-full bg-slate-700" />
            </div>
          </div>
        </div>

        {/* Overview skeleton */}
        <div className="mt-6 space-y-2">
          <div className="h-4 w-20 rounded bg-slate-700" />
          <div className="h-4 w-full rounded bg-slate-700" />
          <div className="h-4 w-full rounded bg-slate-700" />
          <div className="h-4 w-2/3 rounded bg-slate-700" />
        </div>
      </div>
    </div>
  );
}

export default MediaDetailPanel;
