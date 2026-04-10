import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { libraryService } from '../services/libraryService';
import type {
  LibraryListParams,
  BatchResult,
} from '../types/library';

export const libraryKeys = {
  all: ['library'] as const,
  lists: () => [...libraryKeys.all, 'list'] as const,
  list: (params: LibraryListParams) => [...libraryKeys.lists(), params] as const,
  recent: (limit: number) => [...libraryKeys.all, 'recent', limit] as const,
  searches: () => [...libraryKeys.all, 'search'] as const,
  search: (query: string, params: LibraryListParams) =>
    [...libraryKeys.searches(), query, params] as const,
  genres: () => [...libraryKeys.all, 'genres'] as const,
  stats: () => [...libraryKeys.all, 'stats'] as const,
  movieStats: () => [...libraryKeys.all, 'movie-stats'] as const,
  seriesStats: () => [...libraryKeys.all, 'series-stats'] as const,
  videos: (type: 'movie' | 'series', id: string) =>
    [...libraryKeys.all, type, id, 'videos'] as const,
};

export function useLibraryList(params: LibraryListParams) {
  return useQuery({
    queryKey: libraryKeys.list(params),
    queryFn: () => libraryService.listLibrary(params),
    staleTime: 30 * 1000, // NFR-P9: 30s freshness
  });
}

export function useRecentlyAdded(limit: number = 20) {
  return useQuery({
    queryKey: libraryKeys.recent(limit),
    queryFn: () => libraryService.getRecentlyAdded(limit),
    staleTime: 30 * 1000, // 30s (NFR-P9)
    refetchInterval: 30_000, // Auto-refresh every 30s
  });
}

export function useLibrarySearch(query: string, params: LibraryListParams = {}) {
  return useQuery({
    queryKey: libraryKeys.search(query, params),
    queryFn: () => libraryService.searchLibrary(query, params),
    enabled: query.length >= 2,
    staleTime: 60 * 1000, // 60s
    gcTime: 5 * 60 * 1000, // 5min
  });
}

export function useLibraryGenres() {
  return useQuery({
    queryKey: libraryKeys.genres(),
    queryFn: () => libraryService.getGenres(),
    staleTime: 5 * 60 * 1000, // 5min — genres change infrequently
  });
}

export function useLibraryStats() {
  return useQuery({
    queryKey: libraryKeys.stats(),
    queryFn: () => libraryService.getStats(),
    staleTime: 60 * 1000, // 1min
  });
}

export function useMovieStats() {
  return useQuery({
    queryKey: libraryKeys.movieStats(),
    queryFn: () => libraryService.getMovieStats(),
    staleTime: 60 * 1000,
  });
}

export function useSeriesStats() {
  return useQuery({
    queryKey: libraryKeys.seriesStats(),
    queryFn: () => libraryService.getSeriesStats(),
    staleTime: 60 * 1000,
  });
}

export function useDeleteLibraryItem() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ type, id }: { type: 'movie' | 'series'; id: string }) => {
      return type === 'movie' ? libraryService.deleteMovie(id) : libraryService.deleteSeries(id);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: libraryKeys.all });
    },
  });
}

export function useReparseItem() {
  return useMutation({
    mutationFn: ({ type, id }: { type: 'movie' | 'series'; id: string }) => {
      return type === 'movie' ? libraryService.reparseMovie(id) : libraryService.reparseSeries(id);
    },
  });
}

export function useMediaTrailers(type: 'movie' | 'series', id: string, enabled = false) {
  return useQuery({
    queryKey: libraryKeys.videos(type, id),
    queryFn: () =>
      type === 'movie' ? libraryService.getMovieVideos(id) : libraryService.getSeriesVideos(id),
    enabled,
    staleTime: 10 * 60 * 1000, // 10min — trailers rarely change
  });
}

export function useExportItem() {
  return useMutation({
    mutationFn: ({ type, id }: { type: 'movie' | 'series'; id: string }) => {
      return type === 'movie' ? libraryService.exportMovie(id) : libraryService.exportSeries(id);
    },
  });
}

export function useBatchDelete() {
  const queryClient = useQueryClient();

  return useMutation<BatchResult, Error, { ids: string[]; type: 'movie' | 'series' }>({
    mutationFn: ({ ids, type }) => libraryService.batchDelete(ids, type),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: libraryKeys.all });
    },
  });
}

export function useBatchReparse() {
  const queryClient = useQueryClient();

  return useMutation<BatchResult, Error, { ids: string[]; type: 'movie' | 'series' }>({
    mutationFn: ({ ids, type }) => libraryService.batchReparse(ids, type),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: libraryKeys.all });
    },
  });
}

export function useBatchExport() {
  return useMutation<unknown[], Error, { ids: string[]; type: 'movie' | 'series' }>({
    mutationFn: ({ ids, type }) => libraryService.batchExport(ids, type),
  });
}
