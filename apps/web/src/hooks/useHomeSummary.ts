/**
 * TanStack Query hook for the Home v3 readout band (ux3-1-7). Mirrors
 * useActivity's visibility-gated polling, at a slower cadence — band numbers
 * (coverage, today's count, failures) move on the minutes scale, not seconds,
 * and the tech-spec's non-goals forbid a second live channel (no SSE). The
 * endpoint is fail-soft per cell, so a degraded cell arrives as data; a
 * whole-request failure surfaces as isError and the band renders nothing
 * (the page's sections carry on — the band is a readout, not the page).
 */
import { useQuery } from '@tanstack/react-query';
import { useSyncExternalStore } from 'react';
import { homeSummaryService } from '../services/homeSummaryService';
import type { HomeSummary } from '../services/homeSummaryService';

export const homeSummaryKeys = {
  all: ['home-summary'] as const,
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

export function useHomeSummary() {
  const isVisible = usePageVisibility();

  return useQuery<HomeSummary, Error>({
    queryKey: homeSummaryKeys.all,
    queryFn: () => homeSummaryService.getHomeSummary(),
    refetchInterval: isVisible ? 30000 : false,
    staleTime: 25000,
    refetchOnWindowFocus: true,
  });
}
