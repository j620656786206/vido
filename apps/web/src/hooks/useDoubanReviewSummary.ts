import { useQuery } from '@tanstack/react-query';
import { libraryService } from '../services/libraryService';
import type { DoubanReviewSummaryResponse } from '../types/library';

// 24h client-side freshness window, mirroring useDoubanRating's staleTime. (The
// server-side douban_cache TTL is a separate, longer 7-day window.)
const DOUBAN_REVIEW_SUMMARY_STALE_TIME = 24 * 60 * 60 * 1000;

/**
 * useDoubanReviewSummary lazily fetches the Douban short-comment summary (短評)
 * for a local media record (Story 12-6). Gate it with `enabled` on a RESOLVED
 * doubanId (e.g. `Boolean(doubanQuery.data?.doubanId)`) so a record with no Douban
 * match never triggers a wasted scrape (CR M1) — the summary only renders once a
 * doubanId is known. Returns null data on graceful degradation (the review block is
 * omitted, the direct link still renders).
 */
export function useDoubanReviewSummary(id: string, type: 'movie' | 'series', enabled: boolean) {
  return useQuery<DoubanReviewSummaryResponse, Error>({
    queryKey: ['douban-review-summary', type, id],
    queryFn: () =>
      type === 'movie'
        ? libraryService.getMovieDoubanReviewSummary(id)
        : libraryService.getSeriesDoubanReviewSummary(id),
    staleTime: DOUBAN_REVIEW_SUMMARY_STALE_TIME,
    enabled: enabled && !!id,
  });
}
