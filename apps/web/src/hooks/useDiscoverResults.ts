// Story 11-2 Task 5 — TanStack Query hooks that consume the discover filter
// state and call the Story 11-1 discover endpoints (Rule 5: server state via
// TanStack Query, never Zustand).
import { keepPreviousData, useQuery } from '@tanstack/react-query';
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

export function useDiscoverMovies(
  filters: DiscoverFilters,
  page = 1,
  enabled = true,
  keepPrevious = false
) {
  return useQuery<MovieSearchResponse, Error>({
    queryKey: discoverKeys.movies(filters, page),
    queryFn: () => tmdbService.discoverMovies(buildDiscoverParams(filters, 'movie', page)),
    enabled,
    staleTime: STALE_TIME,
    gcTime: GC_TIME,
    // ux3-3-2 AC #6: keep the previous page's results visible across a filter toggle
    // (only when the caller opts in — the v2 rail) so the grid does not flash a
    // skeleton on every chip toggle.
    placeholderData: keepPrevious ? keepPreviousData : undefined,
  });
}

export function useDiscoverTVShows(
  filters: DiscoverFilters,
  page = 1,
  enabled = true,
  keepPrevious = false
) {
  return useQuery<TVShowSearchResponse, Error>({
    queryKey: discoverKeys.tv(filters, page),
    queryFn: () => tmdbService.discoverTVShows(buildDiscoverParams(filters, 'tv', page)),
    enabled,
    staleTime: STALE_TIME,
    gcTime: GC_TIME,
    placeholderData: keepPrevious ? keepPreviousData : undefined,
  });
}

export interface UseDiscoverResultsResult {
  moviesQuery: ReturnType<typeof useDiscoverMovies>;
  tvQuery: ReturnType<typeof useDiscoverTVShows>;
  /** Coalesced first-load state — true while ANY enabled query has no data yet. */
  isLoading: boolean;
  /** Coalesced background-refresh state — true while ANY enabled query is fetching. */
  isFetching: boolean;
  /** Combined total result count across the active media types. */
  totalResults: number;
}

export interface UseDiscoverResultsOptions {
  /**
   * Gate the whole hook off (no network). Default `true`. The mobile filter sheet
   * passes `isOpen` so it live-counts a draft only while open and stays idle closed.
   */
  enabled?: boolean;
  /**
   * Keep the previous results visible across a filter/key change (no skeleton
   * flash). Default `false`. The v2 rail opts in (AC #6).
   */
  keepPrevious?: boolean;
}

/**
 * Runs the movie and/or TV discover queries based on the selected media type.
 * Disabled queries report `isLoading === false` (fetchStatus idle), so the
 * combined `isLoading` only reflects the queries that are actually active.
 */
export function useDiscoverResults(
  filters: DiscoverFilters,
  mediaType: DiscoverMediaType,
  page = 1,
  options: UseDiscoverResultsOptions = {}
): UseDiscoverResultsResult {
  const { enabled = true, keepPrevious = false } = options;
  const wantMovies = enabled && (mediaType === 'all' || mediaType === 'movie');
  const wantTV = enabled && (mediaType === 'all' || mediaType === 'tv');

  const moviesQuery = useDiscoverMovies(filters, page, wantMovies, keepPrevious);
  const tvQuery = useDiscoverTVShows(filters, page, wantTV, keepPrevious);

  const totalResults =
    (wantMovies ? (moviesQuery.data?.totalResults ?? 0) : 0) +
    (wantTV ? (tvQuery.data?.totalResults ?? 0) : 0);

  // ux3-3-2 AC #6: treat the movies+tv pair as ONE logical refresh. Only the
  // queries enabled for the current media type contribute (a disabled query is
  // idle, not loading), so a `type='all'` toggle resolves to a single skeleton
  // rather than a movie-then-tv double flash.
  return {
    moviesQuery,
    tvQuery,
    isLoading: (wantMovies && moviesQuery.isLoading) || (wantTV && tvQuery.isLoading),
    isFetching: (wantMovies && moviesQuery.isFetching) || (wantTV && tvQuery.isFetching),
    totalResults,
  };
}
