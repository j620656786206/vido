/**
 * Download monitoring hooks using TanStack Query (Story 4.2, 4.4)
 */

import { useQuery } from '@tanstack/react-query';
import { useSyncExternalStore } from 'react';
import {
  downloadService,
  type Download,
  type DownloadDetails,
  type DownloadCounts,
  type FilterStatus,
  type SortField,
  type SortOrder,
} from '../services/downloadService';

export const downloadKeys = {
  all: ['downloads'] as const,
  list: (filter: FilterStatus, sort: SortField, order: SortOrder) =>
    [...downloadKeys.all, 'list', filter, sort, order] as const,
  counts: () => [...downloadKeys.all, 'counts'] as const,
  detail: (hash: string) => [...downloadKeys.all, 'detail', hash] as const,
};

/**
 * Singleton page visibility detection using useSyncExternalStore.
 * Module-level functions ensure a single shared subscription across all callers.
 */
const subscribeVisibility = (callback: () => void) => {
  document.addEventListener('visibilitychange', callback);
  return () => document.removeEventListener('visibilitychange', callback);
};
const getVisibilitySnapshot = () => document.visibilityState === 'visible';
const getServerSnapshot = () => true;

function usePageVisibility() {
  return useSyncExternalStore(subscribeVisibility, getVisibilitySnapshot, getServerSnapshot);
}

/**
 * Hook for fetching download list with 5-second polling (AC2, NFR-P8)
 * Polling stops when the page is not visible (AC3).
 */
export function useDownloads(
  filter: FilterStatus = 'all',
  sort: SortField = 'added_on',
  order: SortOrder = 'desc'
) {
  const isVisible = usePageVisibility();

  return useQuery<Download[], Error>({
    queryKey: downloadKeys.list(filter, sort, order),
    queryFn: () => downloadService.getDownloads({ filter, sort, order }),
    refetchInterval: isVisible ? 5000 : false,
    refetchOnWindowFocus: true,
  });
}

/**
 * Hook for fetching download counts by status (Story 4.4 AC1, AC3)
 * Polls at the same interval as downloads for consistency.
 */
export function useDownloadCounts(enabled = true) {
  const isVisible = usePageVisibility();

  return useQuery<DownloadCounts, Error>({
    queryKey: downloadKeys.counts(),
    queryFn: () => downloadService.getDownloadCounts(),
    enabled,
    refetchInterval: isVisible ? 5000 : false,
    refetchOnWindowFocus: true,
  });
}

/**
 * Hook for fetching download details (AC4)
 */
export function useDownloadDetails(hash: string) {
  return useQuery<DownloadDetails, Error>({
    queryKey: downloadKeys.detail(hash),
    queryFn: () => downloadService.getDownloadDetails(hash),
    enabled: !!hash,
  });
}
