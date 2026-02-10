/**
 * Dashboard data hooks using TanStack Query (Story 4.3)
 */

import { useQuery } from '@tanstack/react-query';
import { mediaService, type RecentMedia } from '../services/mediaService';

export const mediaKeys = {
  all: ['media'] as const,
  recent: (limit: number) => [...mediaKeys.all, 'recent', limit] as const,
};

/**
 * Hook for fetching recent media items (AC1, AC2)
 */
export function useRecentMedia(limit: number = 8) {
  return useQuery<RecentMedia[], Error>({
    queryKey: mediaKeys.recent(limit),
    queryFn: () => mediaService.getRecentMedia(limit),
    staleTime: 60 * 1000, // 1 minute
  });
}
