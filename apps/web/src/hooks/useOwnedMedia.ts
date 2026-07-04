import { useCallback, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { availabilityService } from '../services/availabilityService';
import { useRequestedMedia } from './useRequestedMedia';

/**
 * Query key factory for ownership queries. Keys include the sorted ID tuple
 * so repeat visits to the same homepage view reuse the cached answer instead
 * of re-hitting the backend on every render.
 */
export const ownedMediaKeys = {
  all: ['owned-media'] as const,
  lookup: (tmdbIds: readonly number[]) =>
    [...ownedMediaKeys.all, 'lookup', normaliseIds(tmdbIds)] as const,
};

function normaliseIds(tmdbIds: readonly number[]): readonly number[] {
  const unique = Array.from(new Set(tmdbIds.filter((id) => Number.isInteger(id) && id > 0)));
  unique.sort((a, b) => a - b);
  return unique;
}

/**
 * Lookup result for a set of TMDb IDs. `isOwned` is exposed as a set so the
 * rendering path stays O(1) per card; `isRequested` is LIVE since Story 13-1b
 * (Epic 13) — backed by GET /api/v1/requests via useRequestedMedia. The
 * Story-10-4 signature carries no media type, so it matches an active request
 * of ANY type for that TMDb id (badge-level semantic; the 想要 button uses the
 * exact (tmdbId, mediaType) check from useRequestedMedia directly).
 */
export interface OwnedMediaState {
  owned: Set<number>;
  isOwned(tmdbId: number | null | undefined): boolean;
  isRequested(tmdbId: number | null | undefined): boolean;
  isLoading: boolean;
  error: Error | null;
}

const EMPTY: readonly number[] = Object.freeze([]);

/**
 * useOwnedMedia batches all visible TMDb IDs into a single POST to avoid N+1
 * queries (AC #4). The request auto-normalises and deduplicates the input so
 * consumers can pass raw trending lists without preprocessing.
 *
 * Returns a stable empty result while disabled or loading — callers can render
 * without null-checking the return value. The returned object, its Set, and
 * its predicate functions preserve reference identity across renders when the
 * underlying data is unchanged, so `React.memo`'d consumers stay memoised.
 *
 * Story 10-4 (P2-006).
 */
export function useOwnedMedia(tmdbIds: readonly number[] = EMPTY): OwnedMediaState {
  // Memoise the normalised id list against the raw input — sorted+deduped so
  // reorderings don't bust the cache.
  const normalised = useMemo(() => normaliseIds(tmdbIds), [tmdbIds]);
  const enabled = normalised.length > 0;

  const query = useQuery({
    queryKey: ownedMediaKeys.lookup(normalised),
    queryFn: () => availabilityService.checkOwned([...normalised]),
    enabled,
    // Ownership flips only when the user scans, removes, or adds media — none
    // of those happen on the homepage itself. 60s keeps the display fresh
    // across route transitions without thrashing the backend.
    staleTime: 60 * 1000,
    retry: 1,
  });

  // Rebuild the Set only when the query payload changes, not every render.
  const owned = useMemo(() => new Set<number>(query.data ?? []), [query.data]);

  const isOwned = useCallback(
    (tmdbId: number | null | undefined) =>
      typeof tmdbId === 'number' && Number.isInteger(tmdbId) && tmdbId > 0 && owned.has(tmdbId),
    [owned]
  );

  // Story 13-1b: the Story-10-4 stub flips live — delegate to the request
  // system. Query only fires when this hook has ids to check (same gating
  // philosophy as the ownership POST above).
  const requested = useRequestedMedia(enabled);
  const isRequested = requested.isRequested;

  const error = (query.error as Error | null) ?? null;

  return useMemo<OwnedMediaState>(
    () => ({
      owned,
      isOwned,
      isRequested,
      isLoading: query.isLoading,
      error,
    }),
    [owned, isOwned, isRequested, query.isLoading, error]
  );
}
