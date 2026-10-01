/**
 * TanStack Query hook for this month's subtitle AI spend (sub-7-6c), read by
 * the activity hub's 本月 AI 花費 card, the home readout band and — with a
 * `batchId` — the batch dialog's completion receipt.
 *
 * Polls GET /api/v1/subtitles/spend while the tab is visible, at a minute: the
 * number moves only when a run finishes, and the batch dialog invalidates the
 * month key on every terminal so a finished batch shows up at once rather
 * than on the next tick.
 */
import { useQuery } from '@tanstack/react-query';
import { subtitleSpendService } from '../services/subtitleSpendService';
import type { SubtitleSpendSummary } from '../services/subtitleSpendService';
import { usePageVisibility } from './usePageVisibility';

export const subtitleSpendKeys = {
  all: ['subtitle-spend'] as const,
  /** '' = the plain month summary; a batch id = the summary + that batch's receipt. */
  month: (batchId = '') => ['subtitle-spend', 'month', batchId] as const,
};

export interface UseSubtitleSpendOptions {
  /** Adds `byBatch` (that batch's receipt) to the response. */
  batchId?: string;
  /** Default true. The batch dialog enables this only once a batch has ended. */
  enabled?: boolean;
}

export function useSubtitleSpend({ batchId = '', enabled = true }: UseSubtitleSpendOptions = {}) {
  const isVisible = usePageVisibility();

  return useQuery<SubtitleSpendSummary, Error>({
    queryKey: subtitleSpendKeys.month(batchId),
    queryFn: () => subtitleSpendService.getMonthSummary(batchId || undefined),
    enabled,
    refetchInterval: isVisible ? 60000 : false,
    staleTime: 50000,
    refetchOnWindowFocus: true,
    // One retry, not the default three: the activity hub holds its skeleton
    // until this settles, and an endpoint that is down should degrade to the
    // card's own 無法載入 in seconds, not keep the whole page blank.
    retry: 1,
  });
}
