/**
 * TanStack Query hook for the AI 花費明細 page (disc-2026-11-spend-history-d):
 * one month of GET /api/v1/subtitles/runs, 50 runs a page, 「載入更多」 appends
 * the next page. `summary` and `series` are the whole month whatever the page
 * or category, so they are read from the first page.
 */
import { useInfiniteQuery } from '@tanstack/react-query';
import { subtitleRunHistoryService } from '../services/subtitleRunHistoryService';
import type {
  ChartCategory,
  RunHistoryItem,
  RunHistoryPage,
} from '../services/subtitleRunHistoryService';

export const RUN_HISTORY_PAGE_SIZE = 50;

export const subtitleRunHistoryKeys = {
  all: ['subtitle-run-history'] as const,
  /** '' month = the server's current month; '' category = all. */
  list: (month = '', category = '') => ['subtitle-run-history', month, category] as const,
};

export interface UseSubtitleRunHistoryOptions {
  month?: string;
  category?: ChartCategory;
}

export function useSubtitleRunHistory({ month, category }: UseSubtitleRunHistoryOptions = {}) {
  const query = useInfiniteQuery<RunHistoryPage, Error>({
    queryKey: subtitleRunHistoryKeys.list(month ?? '', category ?? ''),
    queryFn: ({ pageParam }) =>
      subtitleRunHistoryService.getRuns({
        month,
        category,
        page: pageParam as number,
        pageSize: RUN_HISTORY_PAGE_SIZE,
      }),
    initialPageParam: 1,
    getNextPageParam: (last) =>
      last.page * last.pageSize < last.total ? last.page + 1 : undefined,
    // Switching the category chip keeps the month's summary and chart on screen
    // (they do not depend on it) while the filtered list loads; switching the
    // MONTH does not — the old month's numbers under the new month's label
    // would be a lie, so that shows the skeleton.
    placeholderData: (previous, previousQuery) =>
      previousQuery?.queryKey[1] === (month ?? '') ? previous : undefined,
    staleTime: 50_000,
    retry: 1,
  });

  const first = query.data?.pages[0];
  const items: RunHistoryItem[] = query.data?.pages.flatMap((p) => p.items) ?? [];

  return { ...query, first, items };
}
