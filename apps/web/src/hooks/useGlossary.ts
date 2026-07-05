/**
 * Glossary server-state hooks (ux3-subtitle-v2 AC 4 — Rule 5: TanStack Query for
 * ALL server state; list = query, every write = mutation + list invalidation).
 * `mediaId` is the STRING local media id (9R-15 route contract).
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  glossaryService,
  type GlossaryAddParams,
  type GlossaryEditParams,
} from '../services/glossaryService';

export const glossaryKeys = {
  all: ['glossary'] as const,
  list: (mediaId: string) => [...glossaryKeys.all, 'list', mediaId] as const,
};

export function useGlossaryTerms(mediaId: string, enabled = true) {
  return useQuery({
    queryKey: glossaryKeys.list(mediaId),
    queryFn: () => glossaryService.listTerms(mediaId),
    enabled: enabled && mediaId !== '',
  });
}

/** All five write mutations, each invalidating the media's glossary list. */
export function useGlossaryMutations(mediaId: string) {
  const queryClient = useQueryClient();
  const invalidate = () => queryClient.invalidateQueries({ queryKey: glossaryKeys.list(mediaId) });

  const add = useMutation({
    mutationFn: (params: GlossaryAddParams) => glossaryService.addTerm(mediaId, params),
    onSuccess: invalidate,
  });

  const edit = useMutation({
    mutationFn: ({ termId, ...params }: GlossaryEditParams & { termId: string }) =>
      glossaryService.editTerm(mediaId, termId, params),
    onSuccess: invalidate,
  });

  const confirm = useMutation({
    mutationFn: (termId: string) => glossaryService.confirmTerm(mediaId, termId),
    onSuccess: invalidate,
  });

  const confirmAll = useMutation({
    mutationFn: () => glossaryService.confirmAll(mediaId),
    onSuccess: invalidate,
  });

  const remove = useMutation({
    mutationFn: (termId: string) => glossaryService.deleteTerm(mediaId, termId),
    onSuccess: invalidate,
  });

  return { add, edit, confirm, confirmAll, remove };
}
