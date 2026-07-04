import { useQueryClient, useMutation } from '@tanstack/react-query';
import {
  requestService,
  RequestApiError,
  type MediaRequest,
  type RequestMediaType,
} from '../services/requestService';
import { requestKeys } from './useRequestedMedia';

export interface CreateRequestVars {
  tmdbId: number;
  mediaType: RequestMediaType;
  /** Display title for the optimistic row (the server re-resolves its own). */
  title: string;
}

/**
 * Mutation hook for the 想要 button (Story 13-1b AC #4) — clones the
 * useDownloadActions optimistic template: cancel → snapshot → patch cache →
 * rollback onError → invalidate onSettled. A REQUEST_DUPLICATE 409 is NOT an
 * error: the requested state is true, so the optimistic row stands and the
 * settle-invalidate reconciles with the server's actual row.
 */
export function useRequestActions() {
  const queryClient = useQueryClient();

  const create = useMutation({
    mutationFn: (vars: CreateRequestVars) =>
      requestService.createRequest({ tmdbId: vars.tmdbId, mediaType: vars.mediaType }),
    onMutate: async (vars) => {
      await queryClient.cancelQueries({ queryKey: requestKeys.all });
      const key = requestKeys.list();
      const previous = queryClient.getQueryData<MediaRequest[]>(key);

      const optimistic: MediaRequest = {
        id: `optimistic-${vars.mediaType}-${vars.tmdbId}`,
        tmdbId: vars.tmdbId,
        mediaType: vars.mediaType,
        title: vars.title,
        status: 'pending',
        fulfilmentSource: null,
        externalId: null,
        seasons: null,
        episodes: null,
        errorMessage: null,
        requestedAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      };
      queryClient.setQueryData<MediaRequest[]>(key, [optimistic, ...(previous ?? [])]);

      return { previous };
    },
    onError: (error, _vars, context) => {
      // Duplicate = the requested state is REAL — keep the optimistic row and
      // let onSettled's invalidate swap in the server's actual row (AC #4).
      if (error instanceof RequestApiError && error.code === 'REQUEST_DUPLICATE') return;
      queryClient.setQueryData(requestKeys.list(), context?.previous);
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: requestKeys.all });
    },
  });

  return { create };
}
