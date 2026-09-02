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

/**
 * A session can die under a tab that is sitting open — the 30-day cookie
 * expires, or the session secret is rotated. There is no global 401 interceptor,
 * so nothing else in the app would notice: the shell stays fully rendered while
 * every request behind it fails, with no explanation. Re-checking the (public,
 * cheap) status endpoint on focus and on a slow interval is what turns that
 * silent dead end into a redirect to the login screen.
 */
const RECHECK_MS = 60 * 1000;

export function useAuthStatus() {
  return useQuery<AuthStatus, Error>({
    queryKey: authKeys.status(),
    queryFn: () => authService.getStatus(),
    staleTime: RECHECK_MS,
    refetchInterval: RECHECK_MS,
    refetchOnWindowFocus: true,
    retry: 1,
  });
}
