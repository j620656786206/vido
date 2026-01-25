import { useQuery } from '@tanstack/react-query';
import { tmdbService } from '../services/tmdb';
import type { MovieSearchResponse, TVShowSearchResponse } from '../types/tmdb';

export const tmdbKeys = {
  all: ['tmdb'] as const,
  searches: () => [...tmdbKeys.all, 'search'] as const,
  searchMovies: (query: string, page: number) =>
    [...tmdbKeys.searches(), 'movies', query, page] as const,
  searchTV: (query: string, page: number) => [...tmdbKeys.searches(), 'tv', query, page] as const,
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
