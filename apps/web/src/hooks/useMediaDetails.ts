import { useQuery } from '@tanstack/react-query';
import { tmdbService } from '../services/tmdb';
import { libraryService } from '../services/libraryService';
import type { MovieDetails, TVShowDetails, Credits } from '../types/tmdb';
import type {
  LibraryMovie,
  LibrarySeries,
  SeasonSummary,
  SeasonEpisodesResponse,
  RecommendationsResponse,
  WatchProvidersResponse,
} from '../types/library';

// Query key factory for media details
export const detailKeys = {
  all: ['details'] as const,
  movie: (id: number) => [...detailKeys.all, 'movie', id] as const,
  movieCredits: (id: number) => [...detailKeys.movie(id), 'credits'] as const,
  tv: (id: number) => [...detailKeys.all, 'tv', id] as const,
  tvCredits: (id: number) => [...detailKeys.tv(id), 'credits'] as const,
  localMovie: (id: string) => [...detailKeys.all, 'local-movie', id] as const,
  localSeries: (id: string) => [...detailKeys.all, 'local-series', id] as const,
  seasons: (seriesId: string) => [...detailKeys.all, 'seasons', seriesId] as const,
  seasonEpisodes: (seriesId: string, seasonNumber: number) =>
    [...detailKeys.all, 'season-episodes', seriesId, seasonNumber] as const,
  recommendations: (tmdbId: number, type: 'movie' | 'tv') =>
    [...detailKeys.all, 'recommendations', type, tmdbId] as const,
  watchProviders: (tmdbId: number, type: 'movie' | 'tv', region: string) =>
    [...detailKeys.all, 'watch-providers', type, tmdbId, region] as const,
};

/**
 * Hook to fetch a series' season summaries (from the cached SeasonsJSON — no
 * TMDb call). Drives the collapsed accordion headers (Story 12-2 AC #2).
 */
export function useSeriesSeasons(seriesId: string, enabled: boolean) {
  return useQuery<SeasonSummary[], Error>({
    queryKey: detailKeys.seasons(seriesId),
    queryFn: () => libraryService.getSeriesSeasons(seriesId),
    staleTime: 60 * 60 * 1000, // 1h — season list rarely changes
    enabled: enabled && !!seriesId,
  });
}

/**
 * Hook to lazily fetch a season's episodes (TMDb metadata merged with local
 * subtitle/file status). Only fetches when `enabled` is true so the accordion
 * fetches on expand (Story 12-2 Tasks 8.3/8.4).
 */
export function useSeasonEpisodes(seriesId: string, seasonNumber: number, enabled: boolean) {
  return useQuery<SeasonEpisodesResponse, Error>({
    queryKey: detailKeys.seasonEpisodes(seriesId, seasonNumber),
    queryFn: () => libraryService.getSeasonEpisodes(seriesId, seasonNumber),
    staleTime: 60 * 60 * 1000, // 1h — episodes change infrequently
    enabled: enabled && !!seriesId,
  });
}

/**
 * Hook to fetch related-content recommendations for a title (Story 12-3).
 * Keyed by the TMDB numeric id (available in both the local and TMDb detail
 * views). 24h staleTime matches the backend cache TTL — recommendations are
 * stable. Gated on a valid TMDB id so library items without a tmdbId never fetch.
 * @param tmdbId - TMDB numeric id
 * @param type - 'movie' | 'tv'
 * @param enabled - extra gate (e.g. only when the section is on a detail page)
 */
export function useRecommendations(tmdbId: number, type: 'movie' | 'tv', enabled: boolean) {
  return useQuery<RecommendationsResponse, Error>({
    queryKey: detailKeys.recommendations(tmdbId, type),
    queryFn: () =>
      type === 'movie'
        ? libraryService.getMovieRecommendations(tmdbId)
        : libraryService.getTVRecommendations(tmdbId),
    staleTime: 24 * 60 * 60 * 1000, // 24h — matches backend cache TTL
    enabled: enabled && tmdbId > 0,
  });
}

/**
 * Hook to fetch streaming-platform availability (TMDB watch providers) for a
 * title in a region (Story 12-4). Keyed by the TMDB numeric id + region; both
 * detail views have the id. 24h staleTime matches the backend cache TTL. Gated
 * on a valid TMDB id so library items without a tmdbId never fetch.
 * @param tmdbId - TMDB numeric id
 * @param type - 'movie' | 'tv'
 * @param enabled - extra gate (e.g. only when the section is on a detail page)
 * @param region - ISO 3166-1 region code (default TW)
 */
export function useWatchProviders(
  tmdbId: number,
  type: 'movie' | 'tv',
  enabled: boolean,
  region = 'TW'
) {
  return useQuery<WatchProvidersResponse, Error>({
    queryKey: detailKeys.watchProviders(tmdbId, type, region),
    queryFn: () =>
      type === 'movie'
        ? libraryService.getMovieWatchProviders(tmdbId, region)
        : libraryService.getTVWatchProviders(tmdbId, region),
    staleTime: 24 * 60 * 60 * 1000, // 24h — matches backend cache TTL
    enabled: enabled && tmdbId > 0,
  });
}

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
