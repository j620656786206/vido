/**
 * Metadata editor hooks using TanStack Query (Story 3.8)
 */

import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  metadataService,
  type UpdateMetadataParams,
  type UpdateMetadataResponse,
  type UploadPosterResponse,
} from '../services/metadata';
import { detailKeys } from './useMediaDetails';
import { libraryKeys } from './useLibrary';

/** What the detail page and the library lists read — refetch them after an edit. */
function invalidateEdited(
  queryClient: ReturnType<typeof useQueryClient>,
  id: string,
  mediaType: 'movie' | 'series'
) {
  queryClient.invalidateQueries({
    queryKey: mediaType === 'movie' ? detailKeys.localMovie(id) : detailKeys.localSeries(id),
  });
  queryClient.invalidateQueries({ queryKey: libraryKeys.all });
}

/**
 * Hook for updating media metadata (AC2)
 * Invalidates media queries on success to refresh UI
 */
export function useUpdateMetadata() {
  const queryClient = useQueryClient();

  return useMutation<UpdateMetadataResponse, Error, UpdateMetadataParams>({
    mutationFn: (params) => metadataService.updateMetadata(params),
    onSuccess: (_data, variables) =>
      invalidateEdited(queryClient, variables.id, variables.mediaType),
  });
}

interface UploadPosterParams {
  mediaId: string;
  mediaType: 'movie' | 'series';
  file: File;
}

/**
 * Hook for uploading poster images (AC3)
 * Invalidates media queries on success to refresh UI
 */
export function useUploadPoster() {
  const queryClient = useQueryClient();

  return useMutation<UploadPosterResponse, Error, UploadPosterParams>({
    mutationFn: ({ mediaId, mediaType, file }) =>
      metadataService.uploadPoster(mediaId, mediaType, file),
    onSuccess: (_data, variables) =>
      invalidateEdited(queryClient, variables.mediaId, variables.mediaType),
  });
}
