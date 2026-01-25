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

/**
 * Hook for updating media metadata (AC2)
 * Invalidates media queries on success to refresh UI
 */
export function useUpdateMetadata() {
  const queryClient = useQueryClient();

  return useMutation<UpdateMetadataResponse, Error, UpdateMetadataParams>({
    mutationFn: (params) => metadataService.updateMetadata(params),
    onSuccess: (data, variables) => {
      // Invalidate media queries to refresh UI
      queryClient.invalidateQueries({ queryKey: ['media', variables.id] });
      queryClient.invalidateQueries({ queryKey: ['library'] });
    },
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
    onSuccess: (data, variables) => {
      // Invalidate media queries to refresh UI
      queryClient.invalidateQueries({ queryKey: ['media', variables.mediaId] });
      queryClient.invalidateQueries({ queryKey: ['library'] });
    },
  });
}
