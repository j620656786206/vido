// Story ux3-discover-facet-aggregation-fe Task 2 — TanStack Query hook that
// fetches contextual per-facet result counts for the Discover rail (Rule 5:
// server state via TanStack Query). Consumes the ux3-discover-facet-aggregation-be
// [@contract-v1] endpoint via tmdbService.discoverFacetCounts.
//
// Behaviour:
//   - Gated to the v2 shell AND a caller opt-in (desktop rail only — the mobile
//     bottom sheet never calls it, AC7). Off → no network.
//   - The committed-filter change is debounced ~350ms before it becomes the query
//     key (matches FilterPanel's numeric-year debounce), so a burst of chip toggles
//     fires ONE counts query; placeholderData keeps the prior counts stable across
//     the debounce + refetch (no full-rail "everything loading", AC4).
//   - When the BE returns partial:true, re-poll with exponential backoff and a
//     single in-flight request (conservative concurrency, AR-F7) until partial
//     clears — so the re-polls don't re-drain the shared upstream limiter (AC5).
import { keepPreviousData, useQuery, type Query } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { tmdbService, type FacetCounts } from '../services/tmdb';
import { buildFacetCountParams, type DiscoverFilters } from '../lib/discoverFilters';

const STALE_TIME = 5 * 60 * 1000; // 5 minutes
const GC_TIME = 30 * 60 * 1000; // 30 minutes
const DEBOUNCE_MS = 350; // match FilterPanel debounceMs (AC4)
const REPOLL_BASE_MS = 2000; // AR-F7 backoff base
const REPOLL_MAX_MS = 30000; // AR-F7 backoff ceiling
const REPOLL_MAX_ATTEMPTS = 6; // AR-F7: give up if `partial` never clears (avoid perpetual polling)

export const facetCountKeys = {
  all: ['tmdb', 'discover', 'facet-counts'] as const,
  for: (paramsString: string) => [...facetCountKeys.all, paramsString] as const,
};

export interface UseDiscoverFacetCountsOptions {
  /**
   * Caller opt-in. Default `false` so a non-desktop consumer that forgets to
   * gate stays silent. The desktop rail passes `true` (AC7). Combined with the
   * internal v2-shell gate (defense-in-depth) to decide whether to fetch.
   */
  enabled?: boolean;
}

export interface UseDiscoverFacetCountsResult {
  /** Keyed contextual counts, or undefined until the first response (or on hard fail → rail falls back to the single total, AC6). */
  counts: FacetCounts['counts'] | undefined;
  /** True while the BE could not resolve every requested facet (AC5). */
  partial: boolean;
  /** First-load state (no data yet) for the enabled query. */
  isLoading: boolean;
  /** Background-refresh state for the enabled query. */
  isFetching: boolean;
}

/**
 * AR-F7 partial re-poll cadence: while the latest response is partial, schedule
 * the next refetch with exponential backoff capped at REPOLL_MAX_MS. Stops when
 * partial clears, when there is no data, OR after REPOLL_MAX_ATTEMPTS — so a
 * persistently-partial endpoint cannot drive perpetual background polling; the
 * last partial counts simply remain (chips keep their resolved values, the rest
 * stay "–").
 */
export function repollInterval(query: Query<FacetCounts, Error>): number | false {
  if (!query.state.data?.partial) return false;
  const attempts = query.state.dataUpdateCount;
  if (attempts >= REPOLL_MAX_ATTEMPTS) return false;
  return Math.min(REPOLL_BASE_MS * 2 ** attempts, REPOLL_MAX_MS);
}

export function useDiscoverFacetCounts(
  filters: DiscoverFilters,
  options: UseDiscoverFacetCountsOptions = {}
): UseDiscoverFacetCountsResult {
  // ux3-cutover-3: shell gate removed — the caller opt-in is the only gate.
  const { enabled = false } = options;

  // Debounce the committed-filter → query-key transition. The string is stable
  // across sort/page changes (buildFacetCountParams strips them), so those never
  // re-query; only a base-filter change does (AR-F2).
  const paramsString = buildFacetCountParams(filters).toString();
  const [debouncedParams, setDebouncedParams] = useState(paramsString);
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedParams(paramsString), DEBOUNCE_MS);
    return () => clearTimeout(timer);
  }, [paramsString]);

  const query = useQuery<FacetCounts, Error>({
    queryKey: facetCountKeys.for(debouncedParams),
    queryFn: () => tmdbService.discoverFacetCounts(new URLSearchParams(debouncedParams)),
    enabled,
    staleTime: STALE_TIME,
    gcTime: GC_TIME,
    // Keep the prior counts visible across a filter change / debounce / refetch so
    // the rail never flashes "everything computing" (AC4).
    placeholderData: keepPreviousData,
    refetchInterval: repollInterval,
  });

  return {
    counts: query.data?.counts,
    partial: query.data?.partial ?? false,
    isLoading: enabled && query.isLoading,
    isFetching: enabled && query.isFetching,
  };
}
