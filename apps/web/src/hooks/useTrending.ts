import { useQuery } from '@tanstack/react-query';
import tmdbService from '../services/tmdb';
import type { HeroBannerItem, Movie, TVShow } from '../types/tmdb';

const HERO_BANNER_LIMIT = 5;

export const trendingKeys = {
  all: ['trending'] as const,
  hero: (timeWindow: 'day' | 'week') => [...trendingKeys.all, 'hero', timeWindow] as const,
};

function movieToHeroItem(movie: Movie): HeroBannerItem {
  return {
    id: movie.id,
    mediaType: 'movie',
    title: movie.title,
    overview: movie.overview,
    backdropPath: movie.backdropPath,
    releaseDate: movie.releaseDate,
    voteAverage: movie.voteAverage,
  };
}

function tvShowToHeroItem(tv: TVShow): HeroBannerItem {
  return {
    id: tv.id,
    mediaType: 'tv',
    title: tv.name,
    overview: tv.overview,
    backdropPath: tv.backdropPath,
    releaseDate: tv.firstAirDate,
    voteAverage: tv.voteAverage,
  };
}

// Story 10-2 AC #1, #5 — merges trending movies + TV into a single banner feed.
// Items missing a backdrop are filtered out (would render as broken banner).
// Errors and empty results surface via TanStack Query state; the consumer is
// responsible for hiding the banner section gracefully (AC #5).
export function useTrendingHero(timeWindow: 'day' | 'week' = 'week') {
  return useQuery<HeroBannerItem[], Error>({
    queryKey: trendingKeys.hero(timeWindow),
    queryFn: async () => {
      const [moviesRes, tvRes] = await Promise.all([
        tmdbService.getTrendingMovies(timeWindow, 1),
        tmdbService.getTrendingTVShows(timeWindow, 1),
      ]);

      const movies = moviesRes.results.map(movieToHeroItem);
      const tvShows = tvRes.results.map(tvShowToHeroItem);

      // Interleave by popularity (already sorted by TMDb), keep only items with a backdrop,
      // and cap at HERO_BANNER_LIMIT to keep the carousel tight.
      const merged: HeroBannerItem[] = [];
      const max = Math.max(movies.length, tvShows.length);
      for (let i = 0; i < max && merged.length < HERO_BANNER_LIMIT; i++) {
        if (movies[i] && movies[i].backdropPath) merged.push(movies[i]);
        if (merged.length >= HERO_BANNER_LIMIT) break;
        if (tvShows[i] && tvShows[i].backdropPath) merged.push(tvShows[i]);
      }
      return merged.slice(0, HERO_BANNER_LIMIT);
    },
    staleTime: 60 * 60 * 1000, // 1h — matches backend cache TTL for trending
    // Banner must hide gracefully (AC #5) within ~1s, not after the default
    // 3-retry exponential backoff (~4s+ silent failure). (Code review L1.)
    retry: 1,
  });
}
