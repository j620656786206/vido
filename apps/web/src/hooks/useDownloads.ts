/**
 * Download monitoring hooks using TanStack Query (Story 4.2, 4.4)
 */

import { useQuery } from '@tanstack/react-query';
import { useSyncExternalStore } from 'react';
import {
  downloadService,
  type PaginatedDownloads,
  type DownloadDetails,
  type DownloadCounts,
  type FilterStatus,
  type SortField,
  type SortOrder,
} from '../services/downloadService';
import { useQBittorrentConfig } from './useQBittorrent';

export const downloadKeys = {
  all: ['downloads'] as const,
  list: (filter: FilterStatus, sort: SortField, order: SortOrder, page: number, pageSize: number) =>
    [...downloadKeys.all, 'list', filter, sort, order, page, pageSize] as const,
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
 * Hook for fetching download list with 5-second polling (AC2, NFR-P8).
 * Polling stops when the page is not visible (AC3).
 *
 * Gate (bugfix-10-2): all three download hooks fail CLOSED on the qBT config
 * signal — `configured !== true` (including `undefined` while loading OR if
 * `/api/v1/settings/qbittorrent` itself errors) suppresses the fetch with no
 * `error` surfaced from the hook. Intentional: an init-race or config-endpoint
 * failure should NOT burst /downloads requests; the user-visible surface for
 * a broken config endpoint lives in `useQBittorrentConfig` consumers
 * (DownloadPanel, QBittorrentForm, etc.).
 */
export function useDownloads(
  filter: FilterStatus = 'all',
  sort: SortField = 'added_on',
  order: SortOrder = 'desc',
  page: number = 1,
  pageSize: number = 100
) {
  const isVisible = usePageVisibility();
  const { data: qbtConfig } = useQBittorrentConfig();
  const isConfigured = qbtConfig?.configured === true;

  return useQuery<PaginatedDownloads, Error>({
    queryKey: downloadKeys.list(filter, sort, order, page, pageSize),
    queryFn: () => downloadService.getDownloads({ filter, sort, order, page, pageSize }),
    // bugfix-10-2: skip polling until qBT config check confirms configured; prevents init-race 503 burst
    enabled: isConfigured,
    refetchInterval: isVisible && isConfigured ? 5000 : false,
    refetchOnWindowFocus: true,
  });
}

/**
 * Hook for fetching download counts by status (Story 4.4 AC1, AC3)
 * Polls at the same interval as downloads for consistency.
 */
export function useDownloadCounts(enabled = true) {
  const isVisible = usePageVisibility();
  const { data: qbtConfig } = useQBittorrentConfig();
  const isConfigured = qbtConfig?.configured === true;
  const effectiveEnabled = enabled && isConfigured;

  return useQuery<DownloadCounts, Error>({
    queryKey: downloadKeys.counts(),
    queryFn: () => downloadService.getDownloadCounts(),
    // bugfix-10-2: skip polling until qBT config check confirms configured; prevents init-race 503 burst
    enabled: effectiveEnabled,
    refetchInterval: isVisible && effectiveEnabled ? 5000 : false,
    refetchOnWindowFocus: true,
  });
}

/**
 * Hook for fetching download details (AC4)
 */
export function useDownloadDetails(hash: string) {
  const { data: qbtConfig } = useQBittorrentConfig();
  const isConfigured = qbtConfig?.configured === true;

  return useQuery<DownloadDetails, Error>({
    queryKey: downloadKeys.detail(hash),
    queryFn: () => downloadService.getDownloadDetails(hash),
    // bugfix-10-2: skip polling until qBT config check confirms configured; prevents init-race 503 burst
    enabled: !!hash && isConfigured,
  });
}
