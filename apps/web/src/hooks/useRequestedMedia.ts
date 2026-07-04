import { useCallback, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  requestService,
  ACTIVE_REQUEST_STATUSES,
  type MediaRequest,
  type RequestMediaType,
} from '../services/requestService';

/**
 * Query key factory for the request system (Epic 13). 13-3b's SSE cache
 * patching (`applyRequestSnapshot`) targets `[...requestKeys.all, 'list']` —
 * keep the shape stable.
 */
export const requestKeys = {
  all: ['requests'] as const,
  list: () => [...requestKeys.all, 'list'] as const,
};

export interface RequestedMediaState {
  requests: MediaRequest[];
  /**
   * True when an ACTIVE request (pending/searching/downloading) exists.
   * With `mediaType` the check is exact; without it, ANY media type matches —
   * the badge-level semantic `useOwnedMedia.isRequested` needs (its Story-10-4
   * signature has no type param).
   */
  isRequested(tmdbId: number | null | undefined, mediaType?: RequestMediaType): boolean;
  isLoading: boolean;
}

/**
 * useRequestedMedia mirrors useOwnedMedia for the request system (Story 13-1b):
 * one cached GET /api/v1/requests, O(1) lookups per card. Also consumed
 * internally by useOwnedMedia to flip its Story-10-4 `isRequested` stub live.
 */
export function useRequestedMedia(enabled = true): RequestedMediaState {
  const query = useQuery({
    queryKey: requestKeys.list(),
    queryFn: () => requestService.listRequests(),
    enabled,
    // Requests change through the user's own clicks (mutations invalidate
    // immediately); 30s covers cross-surface freshness until 13-3b's SSE.
    staleTime: 30 * 1000,
    retry: 1,
  });

  const activeKeys = useMemo(() => {
    const keys = new Set<string>();
    for (const req of query.data ?? []) {
      if (ACTIVE_REQUEST_STATUSES.includes(req.status)) {
        keys.add(`${req.mediaType}:${req.tmdbId}`);
        keys.add(`any:${req.tmdbId}`);
      }
    }
    return keys;
  }, [query.data]);

  const isRequested = useCallback(
    (tmdbId: number | null | undefined, mediaType?: RequestMediaType) => {
      if (typeof tmdbId !== 'number' || !Number.isInteger(tmdbId) || tmdbId <= 0) return false;
      return activeKeys.has(mediaType ? `${mediaType}:${tmdbId}` : `any:${tmdbId}`);
    },
    [activeKeys]
  );

  return useMemo<RequestedMediaState>(
    () => ({
      requests: query.data ?? [],
      isRequested,
      isLoading: query.isLoading,
    }),
    [query.data, isRequested, query.isLoading]
  );
}
