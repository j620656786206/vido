import { useQuery } from '@tanstack/react-query';
import { tmdbService } from '../services/tmdb';
import { libraryService } from '../services/libraryService';
import type { MovieDetails, TVShowDetails, Credits } from '../types/tmdb';
import type { LibraryMovie, LibrarySeries } from '../types/library';

// Query key factory for media details
export const detailKeys = {
  all: ['details'] as const,
  movie: (id: number) => [...detailKeys.all, 'movie', id] as const,
  movieCredits: (id: number) => [...detailKeys.movie(id), 'credits'] as const,
  tv: (id: number) => [...detailKeys.all, 'tv', id] as const,
  tvCredits: (id: number) => [...detailKeys.tv(id), 'credits'] as const,
  localMovie: (id: string) => [...detailKeys.all, 'local-movie', id] as const,
  localSeries: (id: string) => [...detailKeys.all, 'local-series', id] as const,
};

/**
 * Hook to fetch movie details from local DB API
 * @param id - Internal UUID
 */
export function useLocalMovieDetails(id: string) {
  return useQuery<LibraryMovie, Error>({
    queryKey: detailKeys.localMovie(id),
    queryFn: () => libraryService.getMovieById(id),
    staleTime: 5 * 60 * 1000,
    enabled: !!id,
  });
}

/**
 * Hook to fetch series details from local DB API
 * @param id - Internal UUID
 */
export function useLocalSeriesDetails(id: string) {
  return useQuery<LibrarySeries, Error>({
    queryKey: detailKeys.localSeries(id),
    queryFn: () => libraryService.getSeriesById(id),
    staleTime: 5 * 60 * 1000,
    enabled: !!id,
  });
}

/**
 * Hook to fetch movie details from TMDb (for progressive enhancement)
 * @param id - TMDb movie ID
 */
export function useMovieDetails(id: number) {
  return useQuery<MovieDetails, Error>({
    queryKey: detailKeys.movie(id),
    queryFn: () => tmdbService.getMovieDetails(id),
    staleTime: 10 * 60 * 1000,
    enabled: id > 0,
  });
}

/**
 * Hook to fetch TV show details from TMDb (for progressive enhancement)
 * @param id - TMDb TV show ID
 */
export function useTVShowDetails(id: number) {
  return useQuery<TVShowDetails, Error>({
    queryKey: detailKeys.tv(id),
    queryFn: () => tmdbService.getTVShowDetails(id),
    staleTime: 10 * 60 * 1000,
    enabled: id > 0,
  });
}

/**
 * Hook to fetch movie credits from TMDb
 * @param id - TMDb movie ID
 */
export function useMovieCredits(id: number) {
  return useQuery<Credits, Error>({
    queryKey: detailKeys.movieCredits(id),
    queryFn: () => tmdbService.getMovieCredits(id),
    staleTime: 10 * 60 * 1000,
    enabled: id > 0,
  });
}

/**
 * Hook to fetch TV show credits from TMDb
 * @param id - TMDb TV show ID
 */
export function useTVShowCredits(id: number) {
  return useQuery<Credits, Error>({
    queryKey: detailKeys.tvCredits(id),
    queryFn: () => tmdbService.getTVShowCredits(id),
    staleTime: 10 * 60 * 1000,
    enabled: id > 0,
  });
}
