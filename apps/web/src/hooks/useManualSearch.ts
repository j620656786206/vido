/**
 * Manual search hook using TanStack Query (Story 3.7)
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  metadataService,
  type ManualSearchParams,
  type ManualSearchResponse,
  type ApplyMetadataParams,
  type ApplyMetadataResponse,
} from '../services/metadata';
import { detailKeys } from './useMediaDetails';
import { libraryKeys } from './useLibrary';

// Query keys following project conventions
export const metadataKeys = {
  all: ['metadata'] as const,
  manualSearch: (params: ManualSearchParams) =>
    [...metadataKeys.all, 'manual-search', params] as const,
};

/**
 * Hook for manual metadata search (AC1, AC4)
 * Enables search when query has at least 2 characters
 */
export function useManualSearch(params: ManualSearchParams) {
  return useQuery<ManualSearchResponse, Error>({
    queryKey: metadataKeys.manualSearch(params),
    queryFn: () => metadataService.manualSearch(params),
    enabled: params.query.length >= 2,
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 30 * 60 * 1000, // 30 minutes
  });
}

/**
 * Hook for applying metadata to a media item (AC3)
 * Invalidates media queries on success to refresh UI
 */
export function useApplyMetadata() {
  const queryClient = useQueryClient();

  return useMutation<ApplyMetadataResponse, Error, ApplyMetadataParams>({
    mutationFn: (params) => metadataService.applyMetadata(params),
    onSuccess: (_data, variables) => {
      // dsr-2b-b AC #4: the keys the detail page and the library actually read.
      // This used to invalidate ['media', id], which nothing reads, so an applied
      // match did not show until a reload.
      queryClient.invalidateQueries({
        queryKey:
          variables.mediaType === 'series'
            ? detailKeys.localSeries(variables.mediaId)
            : detailKeys.localMovie(variables.mediaId),
      });
      queryClient.invalidateQueries({ queryKey: libraryKeys.all });
    },
  });
}
