/**
 * Download monitoring hooks using TanStack Query (Story 4.2)
 */

import { useQuery } from '@tanstack/react-query';
import { useState, useEffect } from 'react';
import {
  downloadService,
  type Download,
  type DownloadDetails,
  type SortField,
  type SortOrder,
} from '../services/downloadService';

export const downloadKeys = {
  all: ['downloads'] as const,
  list: (sort: SortField, order: SortOrder) => [...downloadKeys.all, 'list', sort, order] as const,
  detail: (hash: string) => [...downloadKeys.all, 'detail', hash] as const,
};

/**
 * Hook for fetching download list with 5-second polling (AC2, NFR-P8)
 * Polling stops when the page is not visible (AC3).
 */
export function useDownloads(sort: SortField = 'added_on', order: SortOrder = 'desc') {
  const [isVisible, setIsVisible] = useState(true);

  useEffect(() => {
    const handleVisibilityChange = () => {
      setIsVisible(document.visibilityState === 'visible');
    };
    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => document.removeEventListener('visibilitychange', handleVisibilityChange);
  }, []);

  return useQuery<Download[], Error>({
    queryKey: downloadKeys.list(sort, order),
    queryFn: () => downloadService.getDownloads({ sort, order }),
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
