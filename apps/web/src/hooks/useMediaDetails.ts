import { useQuery } from '@tanstack/react-query';
import { tmdbService } from '../services/tmdb';
import type { MovieDetails, TVShowDetails, Credits } from '../types/tmdb';

// Query key factory for media details
export const detailKeys = {
  all: ['details'] as const,
  movie: (id: number) => [...detailKeys.all, 'movie', id] as const,
  movieCredits: (id: number) => [...detailKeys.movie(id), 'credits'] as const,
  tv: (id: number) => [...detailKeys.all, 'tv', id] as const,
  tvCredits: (id: number) => [...detailKeys.tv(id), 'credits'] as const,
};

/**
 * Hook to fetch movie details from TMDb
 * @param id - TMDb movie ID
 */
export function useMovieDetails(id: number) {
  return useQuery<MovieDetails, Error>({
    queryKey: detailKeys.movie(id),
    queryFn: () => tmdbService.getMovieDetails(id),
    staleTime: 10 * 60 * 1000, // 10 minutes
    enabled: id > 0,
  });
}

/**
 * Hook to fetch TV show details from TMDb
 * @param id - TMDb TV show ID
 */
export function useTVShowDetails(id: number) {
  return useQuery<TVShowDetails, Error>({
    queryKey: detailKeys.tv(id),
    queryFn: () => tmdbService.getTVShowDetails(id),
    staleTime: 10 * 60 * 1000, // 10 minutes
    enabled: id > 0,
  });
}

/**
 * Hook to fetch movie credits (cast and crew) from TMDb
 * @param id - TMDb movie ID
 */
export function useMovieCredits(id: number) {
  return useQuery<Credits, Error>({
    queryKey: detailKeys.movieCredits(id),
    queryFn: () => tmdbService.getMovieCredits(id),
    staleTime: 10 * 60 * 1000, // 10 minutes
    enabled: id > 0,
  });
}

/**
 * Hook to fetch TV show credits (cast and crew) from TMDb
 * @param id - TMDb TV show ID
 */
export function useTVShowCredits(id: number) {
  return useQuery<Credits, Error>({
    queryKey: detailKeys.tvCredits(id),
    queryFn: () => tmdbService.getTVShowCredits(id),
    staleTime: 10 * 60 * 1000, // 10 minutes
    enabled: id > 0,
  });
}

/**
 * Combined hook to fetch media credits based on type
 * @param type - 'movie' or 'tv'
 * @param id - TMDb ID
 */
export function useMediaCredits(type: 'movie' | 'tv', id: number) {
  const movieCredits = useMovieCredits(id);
  const tvCredits = useTVShowCredits(id);

  if (type === 'movie') {
    return {
      ...movieCredits,
      // Disable TV query when type is movie
      data: movieCredits.data,
    };
  }

  return {
    ...tvCredits,
    data: tvCredits.data,
  };
}
