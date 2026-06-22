/**
 * TanStack Query hook for the v2 Activity hub (/activity — ux3-2-3 / D4-1).
 * Polls GET /api/v1/activity while the tab is visible (Rule 8 — plain query, no eager
 * SSE). The endpoint is fail-soft per section, so a degraded section arrives as data,
 * not an error; a whole-request failure (network/500) surfaces as isError for the
 * page-level retry.
 */
import { useQuery } from '@tanstack/react-query';
import { useSyncExternalStore } from 'react';
import { activityService } from '../services/activityService';
import type { ActivitySummary } from '../services/activityService';

export const activityKeys = {
  all: ['activity'] as const,
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

export function useActivity() {
  const isVisible = usePageVisibility();

  return useQuery<ActivitySummary, Error>({
    queryKey: activityKeys.all,
    queryFn: () => activityService.getActivity(),
    refetchInterval: isVisible ? 15000 : false,
    staleTime: 12000,
    refetchOnWindowFocus: true,
  });
}
