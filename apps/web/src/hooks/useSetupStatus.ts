/**
 * Setup status hook using TanStack Query (Story 6.1)
 */

import { useQuery } from '@tanstack/react-query';
import { setupService, type SetupStatus } from '../services/setupService';

export const setupKeys = {
  all: ['setup'] as const,
  status: () => [...setupKeys.all, 'status'] as const,
};

interface UseSetupStatusOptions {
  /**
   * False while the auth gate has not cleared yet. `/setup/status` sits BEHIND
   * the gate, so asking before login is a guaranteed 401 — one that lands in the
   * console of every visitor to the login screen and tells nobody anything.
   * The auth gate runs first by design; this makes the queries agree.
   */
  enabled?: boolean;
}

/**
 * Hook to check if setup wizard needs to be shown.
 * Used by the root route to redirect first-time users.
 */
export function useSetupStatus({ enabled = true }: UseSetupStatusOptions = {}) {
  return useQuery<SetupStatus, Error>({
    queryKey: setupKeys.status(),
    queryFn: () => setupService.getStatus(),
    staleTime: 5 * 60 * 1000, // 5 minutes
    retry: 1,
    enabled,
  });
}
