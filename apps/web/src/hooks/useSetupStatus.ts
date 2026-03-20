/**
 * Setup status hook using TanStack Query (Story 6.1)
 */

import { useQuery } from '@tanstack/react-query';
import { setupService, type SetupStatus } from '../services/setupService';

export const setupKeys = {
  all: ['setup'] as const,
  status: () => [...setupKeys.all, 'status'] as const,
};

/**
 * Hook to check if setup wizard needs to be shown.
 * Used by the root route to redirect first-time users.
 */
export function useSetupStatus() {
  return useQuery<SetupStatus, Error>({
    queryKey: setupKeys.status(),
    queryFn: () => setupService.getStatus(),
    staleTime: 5 * 60 * 1000, // 5 minutes
    retry: 1,
  });
}
