import { useQuery, keepPreviousData } from '@tanstack/react-query';
import { tmdbService } from '../services/tmdb';
import type { MovieSearchResponse, TVShowSearchResponse, UnifiedSearchResult } from '../types/tmdb';

export const tmdbKeys = {
  all: ['tmdb'] as const,
  searches: () => [...tmdbKeys.all, 'search'] as const,
  searchMovies: (query: string, page: number) =>
    [...tmdbKeys.searches(), 'movies', query, page] as const,
  searchTV: (query: string, page: number) => [...tmdbKeys.searches(), 'tv', query, page] as const,
  instant: (query: string) => [...tmdbKeys.searches(), 'instant', query] as const,
};

export function useSearchMovies(query: string, page = 1) {
  return useQuery<MovieSearchResponse, Error>({
    queryKey: tmdbKeys.searchMovies(query, page),
    queryFn: () => tmdbService.searchMovies(query, page),
    enabled: query.length >= 2,
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 30 * 60 * 1000, // 30 minutes (formerly cacheTime)
  });
}

export function useSearchTVShows(query: string, page = 1) {
  return useQuery<TVShowSearchResponse, Error>({
    queryKey: tmdbKeys.searchTV(query, page),
    queryFn: () => tmdbService.searchTVShows(query, page),
    enabled: query.length >= 2,
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 30 * 60 * 1000, // 30 minutes (formerly cacheTime)
  });
}

// Story 11-3 — unified instant search powering the suggestions dropdown.
// Gated on query.length >= 2 (AC #1). The caller is responsible for passing an
// already-debounced query so we don't fire a request per keystroke.
export function useInstantSearch(query: string) {
  return useQuery<UnifiedSearchResult, Error>({
    queryKey: tmdbKeys.instant(query),
    queryFn: () => tmdbService.unifiedSearch(query),
    enabled: query.length >= 2,
    // Keep the prior result visible while the next debounced query loads so the
    // dropdown does not flicker back to the "搜尋中…" state on every new query.
    placeholderData: keepPreviousData,
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 30 * 60 * 1000, // 30 minutes
  });
}
