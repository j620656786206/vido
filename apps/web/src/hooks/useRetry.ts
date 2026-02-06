/**
 * Retry hooks using TanStack Query (Story 3.11)
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  retryService,
  type PendingRetriesResponse,
  type RetryItem,
  type TriggerResponse,
} from '../services/retry';

// Query keys following project conventions
export const retryKeys = {
  all: ['retry'] as const,
  pending: () => [...retryKeys.all, 'pending'] as const,
  detail: (id: string) => [...retryKeys.all, 'detail', id] as const,
};

/**
 * Hook for fetching pending retries with stats (AC4)
 * Auto-refreshes every 5 seconds to update countdown timers
 */
export function usePendingRetries() {
  return useQuery<PendingRetriesResponse, Error>({
    queryKey: retryKeys.pending(),
    queryFn: () => retryService.getPending(),
    refetchInterval: 5000, // 5 seconds for countdown updates
    staleTime: 2000,
  });
}

/**
 * Hook for fetching a specific retry item by ID
 */
export function useRetryItem(id: string) {
  return useQuery<RetryItem, Error>({
    queryKey: retryKeys.detail(id),
    queryFn: () => retryService.getById(id),
    enabled: !!id,
  });
}

/**
 * Hook for triggering immediate retry (AC4)
 * Invalidates pending list on success
 */
export function useTriggerRetry() {
  const queryClient = useQueryClient();

  return useMutation<TriggerResponse, Error, string>({
    mutationFn: (id) => retryService.triggerImmediate(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: retryKeys.pending() });
    },
  });
}

/**
 * Hook for canceling a retry (AC4)
 * Invalidates pending list on success
 */
export function useCancelRetry() {
  const queryClient = useQueryClient();

  return useMutation<void, Error, string>({
    mutationFn: (id) => retryService.cancel(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: retryKeys.pending() });
    },
  });
}
