/**
 * Auth status hook using TanStack Query (V0.1.1 password gate).
 *
 * Mirrors useSetupStatus: the root route reads this to redirect unauthenticated
 * visitors to /login. The /auth/status endpoint is public (not behind the gate),
 * so this query succeeds even when the rest of the API would return 401.
 */

import { useQuery } from '@tanstack/react-query';
import { authService, type AuthStatus } from '../services/authService';

export const authKeys = {
  all: ['auth'] as const,
  status: () => [...authKeys.all, 'status'] as const,
};

export function useAuthStatus() {
  return useQuery<AuthStatus, Error>({
    queryKey: authKeys.status(),
    queryFn: () => authService.getStatus(),
    staleTime: 5 * 60 * 1000, // 5 minutes
    retry: 1,
  });
}
