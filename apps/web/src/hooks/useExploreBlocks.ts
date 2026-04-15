/**
 * TanStack Query hooks for explore blocks — Story 10.3.
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  exploreBlockService,
  type CreateExploreBlockRequest,
  type UpdateExploreBlockRequest,
} from '../services/exploreBlockService';

const FIVE_MINUTES = 5 * 60 * 1000;

export const exploreBlockKeys = {
  all: ['explore-blocks'] as const,
  list: () => [...exploreBlockKeys.all, 'list'] as const,
  detail: (id: string) => [...exploreBlockKeys.all, 'detail', id] as const,
  content: (id: string) => [...exploreBlockKeys.all, 'content', id] as const,
};

export function useExploreBlocks() {
  return useQuery({
    queryKey: exploreBlockKeys.list(),
    queryFn: () => exploreBlockService.getAll(),
    staleTime: FIVE_MINUTES,
  });
}

export function useExploreBlockContent(id: string | undefined) {
  return useQuery({
    queryKey: id ? exploreBlockKeys.content(id) : ['explore-blocks', 'content', 'disabled'],
    queryFn: () => exploreBlockService.getContent(id as string),
    enabled: !!id,
    staleTime: FIVE_MINUTES,
    retry: 1,
  });
}

export function useCreateExploreBlock() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: CreateExploreBlockRequest) => exploreBlockService.create(req),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: exploreBlockKeys.all }),
  });
}

export function useUpdateExploreBlock() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...req }: UpdateExploreBlockRequest & { id: string }) =>
      exploreBlockService.update(id, req),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: exploreBlockKeys.all });
      queryClient.invalidateQueries({ queryKey: exploreBlockKeys.content(variables.id) });
    },
  });
}

export function useDeleteExploreBlock() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => exploreBlockService.remove(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: exploreBlockKeys.all }),
  });
}

export function useReorderExploreBlocks() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (orderedIds: string[]) => exploreBlockService.reorder(orderedIds),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: exploreBlockKeys.all }),
  });
}
