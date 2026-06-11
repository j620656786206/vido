import { useQuery } from '@tanstack/react-query';
import { libraryService } from '../services/libraryService';
import type { DoubanRatingResponse } from '../types/library';

// 24h — matches the server-side Douban cache TTL (Story 12-1 Task 7.4).
const DOUBAN_RATING_STALE_TIME = 24 * 60 * 60 * 1000;

/**
 * useDoubanRating lazily fetches the Douban rating for a local media record
 * (Story 12-1). Pass `enabled` (typically `tmdbId > 0`) to gate the request —
 * a record with no TMDb match cannot be reliably matched on Douban either, and
 * gating avoids a pointless scrape. Returns null data on graceful degradation.
 */
export function useDoubanRating(id: string, type: 'movie' | 'series', enabled: boolean) {
  return useQuery<DoubanRatingResponse, Error>({
    queryKey: ['douban-rating', type, id],
    queryFn: () =>
      type === 'movie'
        ? libraryService.getMovieDoubanRating(id)
        : libraryService.getSeriesDoubanRating(id),
    staleTime: DOUBAN_RATING_STALE_TIME,
    enabled: enabled && !!id,
  });
}
