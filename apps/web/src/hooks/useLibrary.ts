import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { libraryService } from '../services/libraryService';
import type { LibraryListParams } from '../types/library';

export const libraryKeys = {
  all: ['library'] as const,
  lists: () => [...libraryKeys.all, 'list'] as const,
  list: (params: LibraryListParams) => [...libraryKeys.lists(), params] as const,
};

export function useLibraryList(params: LibraryListParams) {
  return useQuery({
    queryKey: libraryKeys.list(params),
    queryFn: () => libraryService.listLibrary(params),
    staleTime: 30 * 1000, // NFR-P9: 30s freshness
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

export function useExportItem() {
  return useMutation({
    mutationFn: ({ type, id }: { type: 'movie' | 'series'; id: string }) => {
      return type === 'movie' ? libraryService.exportMovie(id) : libraryService.exportSeries(id);
    },
  });
}
