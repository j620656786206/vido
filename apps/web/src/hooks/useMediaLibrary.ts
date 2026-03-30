/**
 * TanStack Query hooks for media library CRUD (Story 7b-4)
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  mediaLibraryService,
  type CreateLibraryRequest,
  type UpdateLibraryRequest,
} from '../services/mediaLibraryService';

export const libraryKeys = {
  all: ['media-libraries'] as const,
  detail: (id: string) => ['media-libraries', id] as const,
};

export function useMediaLibraries() {
  return useQuery({
    queryKey: libraryKeys.all,
    queryFn: () => mediaLibraryService.getAll(),
  });
}

export function useMediaLibrary(id: string) {
  return useQuery({
    queryKey: libraryKeys.detail(id),
    queryFn: () => mediaLibraryService.getById(id),
    enabled: !!id,
  });
}

export function useCreateLibrary() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: CreateLibraryRequest) => mediaLibraryService.create(req),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: libraryKeys.all }),
  });
}

export function useUpdateLibrary() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...req }: UpdateLibraryRequest & { id: string }) =>
      mediaLibraryService.update(id, req),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: libraryKeys.all }),
  });
}

export function useDeleteLibrary() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, removeMedia }: { id: string; removeMedia?: boolean }) =>
      mediaLibraryService.delete(id, removeMedia),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: libraryKeys.all }),
  });
}

export function useAddLibraryPath() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ libraryId, path }: { libraryId: string; path: string }) =>
      mediaLibraryService.addPath(libraryId, path),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: libraryKeys.all }),
  });
}

export function useRemoveLibraryPath() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ libraryId, pathId }: { libraryId: string; pathId: string }) =>
      mediaLibraryService.removePath(libraryId, pathId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: libraryKeys.all }),
  });
}

export function useRefreshLibraryPaths() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (libraryId: string) => mediaLibraryService.refreshPaths(libraryId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: libraryKeys.all }),
  });
}
