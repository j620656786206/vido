/**
 * TanStack Query hook for the sidebar-footer status strip (ux3-0-4 / D4-2).
 * Polls GET /api/v1/status/summary while the tab is visible (Rule 8 — plain query,
 * no eager SSE in this always-mounted footer). The endpoint is fail-soft, so a
 * degraded section arrives as data, not an error.
 */
import { useQuery } from '@tanstack/react-query';
import { useSyncExternalStore } from 'react';
import { statusSummaryService } from '../services/statusSummaryService';
import type { StatusSummary } from '../services/statusSummaryService';

export const statusSummaryKeys = {
  all: ['statusSummary'] as const,
};

const subscribeVisibility = (callback: () => void) => {
  document.addEventListener('visibilitychange', callback);
  return () => document.removeEventListener('visibilitychange', callback);
};
const getVisibilitySnapshot = () => document.visibilityState === 'visible';
const getServerSnapshot = () => true;

function usePageVisibility() {
  return useSyncExternalStore(subscribeVisibility, getVisibilitySnapshot, getServerSnapshot);
}

export function useStatusSummary() {
  const isVisible = usePageVisibility();

  return useQuery<StatusSummary, Error>({
    queryKey: statusSummaryKeys.all,
    queryFn: () => statusSummaryService.getSummary(),
    refetchInterval: isVisible ? 30000 : false,
    staleTime: 25000,
    refetchOnWindowFocus: true,
  });
}
