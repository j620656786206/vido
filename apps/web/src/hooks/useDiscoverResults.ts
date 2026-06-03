// Story 11-2 Task 5 — TanStack Query hooks that consume the discover filter
// state and call the Story 11-1 discover endpoints (Rule 5: server state via
// TanStack Query, never Zustand).
import { useQuery } from '@tanstack/react-query';
import { tmdbService } from '../services/tmdb';
import {
  buildDiscoverParams,
  type DiscoverFilters,
  type DiscoverMediaType,
} from '../lib/discoverFilters';
import type { MovieSearchResponse, TVShowSearchResponse } from '../types/tmdb';

export const discoverKeys = {
  all: ['tmdb', 'discover'] as const,
  movies: (filters: DiscoverFilters, page: number) =>
    [
      ...discoverKeys.all,
      'movies',
      buildDiscoverParams(filters, 'movie', page).toString(),
    ] as const,
  tv: (filters: DiscoverFilters, page: number) =>
    [...discoverKeys.all, 'tv', buildDiscoverParams(filters, 'tv', page).toString()] as const,
};

const STALE_TIME = 5 * 60 * 1000; // 5 minutes
const GC_TIME = 30 * 60 * 1000; // 30 minutes

export function useDiscoverMovies(filters: DiscoverFilters, page = 1, enabled = true) {
  return useQuery<MovieSearchResponse, Error>({
    queryKey: discoverKeys.movies(filters, page),
    queryFn: () => tmdbService.discoverMovies(buildDiscoverParams(filters, 'movie', page)),
    enabled,
    staleTime: STALE_TIME,
    gcTime: GC_TIME,
  });
}

export function useDiscoverTVShows(filters: DiscoverFilters, page = 1, enabled = true) {
  return useQuery<TVShowSearchResponse, Error>({
    queryKey: discoverKeys.tv(filters, page),
    queryFn: () => tmdbService.discoverTVShows(buildDiscoverParams(filters, 'tv', page)),
    enabled,
    staleTime: STALE_TIME,
    gcTime: GC_TIME,
  });
}

export interface UseDiscoverResultsResult {
  moviesQuery: ReturnType<typeof useDiscoverMovies>;
  tvQuery: ReturnType<typeof useDiscoverTVShows>;
  isLoading: boolean;
  /** Combined total result count across the active media types. */
  totalResults: number;
}

/**
 * Runs the movie and/or TV discover queries based on the selected media type.
 * Disabled queries report `isLoading === false` (fetchStatus idle), so the
 * combined `isLoading` only reflects the queries that are actually active.
 */
export function useDiscoverResults(
  filters: DiscoverFilters,
  mediaType: DiscoverMediaType,
  page = 1
): UseDiscoverResultsResult {
  const wantMovies = mediaType === 'all' || mediaType === 'movie';
  const wantTV = mediaType === 'all' || mediaType === 'tv';

  const moviesQuery = useDiscoverMovies(filters, page, wantMovies);
  const tvQuery = useDiscoverTVShows(filters, page, wantTV);

  const totalResults =
    (wantMovies ? (moviesQuery.data?.totalResults ?? 0) : 0) +
    (wantTV ? (tvQuery.data?.totalResults ?? 0) : 0);

  return {
    moviesQuery,
    tvQuery,
    isLoading: moviesQuery.isLoading || tvQuery.isLoading,
    totalResults,
  };
}
