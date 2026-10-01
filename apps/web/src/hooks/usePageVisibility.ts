import { useSyncExternalStore } from 'react';

const subscribeVisibility = (callback: () => void) => {
  document.addEventListener('visibilitychange', callback);
  return () => document.removeEventListener('visibilitychange', callback);
};
const getVisibilitySnapshot = () => document.visibilityState === 'visible';
const getServerSnapshot = () => true;

/**
 * `true` while the tab is visible. The polling hooks (useActivity,
 * useHomeSummary, useSubtitleSpend) gate their refetchInterval on this so a
 * background tab stops hitting the API (Rule 8 — plain queries, no eager SSE).
 */
export function usePageVisibility(): boolean {
  return useSyncExternalStore(subscribeVisibility, getVisibilitySnapshot, getServerSnapshot);
}
