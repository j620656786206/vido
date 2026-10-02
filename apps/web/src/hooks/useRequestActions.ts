import { useQueryClient, useMutation, useMutationState } from '@tanstack/react-query';
import {
  requestService,
  RequestApiError,
  type MediaRequest,
  type RequestMediaType,
} from '../services/requestService';
import { requestKeys } from './useRequestedMedia';

/** Mutation keys for the row actions — read back by usePendingRequestActions. */
export const requestMutationKeys = {
  cancel: [...requestKeys.all, 'cancel'] as const,
  retry: [...requestKeys.all, 'retry'] as const,
};

/**
 * Ids with a cancel / retry in flight — EVERY pending call, not just the last
 * `mutate()` (a single useMutation only remembers its latest variables). The
 * view uses them to disable buttons and to keep a cancelling row hidden even
 * if an SSE snapshot re-adds it before the DELETE lands.
 */
export function usePendingRequestActions() {
  const cancelling = useMutationState({
    filters: { mutationKey: requestMutationKeys.cancel, status: 'pending' },
    select: (mutation) => mutation.state.variables as string,
  });
  const retrying = useMutationState({
    filters: { mutationKey: requestMutationKeys.retry, status: 'pending' },
    select: (mutation) => mutation.state.variables as string,
  });
  return { cancelling: new Set(cancelling), retrying: new Set(retrying) };
}

export interface CreateRequestVars {
  tmdbId: number;
  mediaType: RequestMediaType;
  /** Display title for the optimistic row (the server re-resolves its own). */
  title: string;
}

/**
 * Mutation hooks for the request system: `create` backs the 想要 button
 * (Story 13-1b AC #4); `cancel` / `retry` back the 想要清單 row actions
 * (Story 13-7b). All three clone the useDownloadActions optimistic template:
 * cancel → snapshot → patch cache → rollback onError → invalidate onSettled. A REQUEST_DUPLICATE 409 is NOT an
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

  // 13-7b — cancel a pending row: optimistic remove. Rollback restores ONLY
  // this row (at its old position): a whole-list snapshot would also undo any
  // other row's action that started after this one (13-7b CR).
  const cancel = useMutation({
    mutationKey: requestMutationKeys.cancel,
    mutationFn: (id: string) => requestService.cancelRequest(id),
    onMutate: async (id) => {
      await queryClient.cancelQueries({ queryKey: requestKeys.all });
      const rows = queryClient.getQueryData<MediaRequest[]>(requestKeys.list()) ?? [];
      const index = rows.findIndex((row) => row.id === id);
      queryClient.setQueryData<MediaRequest[]>(requestKeys.list(), (current) =>
        (current ?? []).filter((row) => row.id !== id)
      );
      return { removed: index >= 0 ? rows[index] : undefined, index };
    },
    onError: (_error, id, context) => {
      const removed = context?.removed;
      if (!removed) return;
      queryClient.setQueryData<MediaRequest[]>(requestKeys.list(), (current) => {
        const rows = current ?? [];
        if (rows.some((row) => row.id === id)) return rows;
        const at = Math.min(context.index, rows.length);
        return [...rows.slice(0, at), removed, ...rows.slice(at)];
      });
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: requestKeys.all });
    },
  });

  // 13-7b — retry a failed row. Deliberately NOT optimistic: a row patched to
  // an active status before the server has written it is "absent + active"
  // to the request_progress SSE merge (applyRequestSnapshot), which drops it —
  // the row would vanish mid-retry. The row stays 失敗 with its button
  // disabled until the server answers, then takes the returned state.
  const retry = useMutation({
    mutationKey: requestMutationKeys.retry,
    mutationFn: (id: string) => requestService.retryRequest(id),
    onSuccess: (updated) => {
      queryClient.setQueryData<MediaRequest[]>(requestKeys.list(), (current) =>
        (current ?? []).map((row) => (row.id === updated.id ? updated : row))
      );
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: requestKeys.all });
    },
  });

  return { create, cancel, retry };
}
