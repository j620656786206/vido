/**
 * TanStack Query hooks for cache management (Story 6.2)
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { cacheService } from '../services/cacheService';
import type { CacheStats, CleanupResult } from '../services/cacheService';

export const cacheKeys = {
  all: ['cache'] as const,
  stats: () => [...cacheKeys.all, 'stats'] as const,
};

export function useCacheStats() {
  return useQuery<CacheStats, Error>({
    queryKey: cacheKeys.stats(),
    queryFn: () => cacheService.getStats(),
    staleTime: 30 * 1000, // 30 seconds
  });
}

export function useClearCacheByType() {
  const queryClient = useQueryClient();

  return useMutation<CleanupResult, Error, string>({
    mutationFn: (cacheType) => cacheService.clearByType(cacheType),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: cacheKeys.stats() });
    },
  });
}

export function useClearCacheByAge() {
  const queryClient = useQueryClient();

  return useMutation<CleanupResult, Error, number>({
    mutationFn: (days) => cacheService.clearByAge(days),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: cacheKeys.stats() });
    },
  });
}

export function useClearAllCache() {
  const queryClient = useQueryClient();

  return useMutation<CleanupResult, Error, void>({
    mutationFn: () => cacheService.clearAll(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: cacheKeys.stats() });
    },
  });
}
