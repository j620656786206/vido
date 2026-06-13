/**
 * `new_shell_enabled` feature-flag hook (UX Redesign Phase 2 — FOUNDATION / UX2-1).
 *
 * The single source of truth for whether the v2 navigation shell renders. Read in
 * exactly one place — `routes/__root.tsx` — to select the new `AppShell` vs the
 * legacy shell (D1-c F4 single chokepoint). Default OFF.
 *
 * Server state via TanStack Query (Rule 5). The last resolved value is mirrored to
 * localStorage and used as `initialData` so a pilot user who has the flag ON does
 * not see a one-frame legacy→v2 flash on reload; the flag-OFF default path (every
 * other user) pays no cost and renders legacy immediately.
 */
import { useQuery } from '@tanstack/react-query';
import { useEffect } from 'react';
import { settingsService } from '../services/settingsService';

export const NEW_SHELL_FLAG_KEY = 'new_shell_enabled';
const LS_KEY = 'vido:flag:new_shell_enabled';

export const featureFlagKeys = {
  all: ['settings', 'flag'] as const,
  flag: (key: string) => [...featureFlagKeys.all, key] as const,
};

function readCachedFlag(): boolean | undefined {
  try {
    const v = localStorage.getItem(LS_KEY);
    return v === 'true' ? true : v === 'false' ? false : undefined;
  } catch {
    return undefined;
  }
}

/**
 * Returns whether the v2 shell is enabled. Resolves to `false` while the first
 * fetch is in flight unless a cached value exists (then that value seeds it).
 */
export function useNewShellEnabled(): boolean {
  const { data } = useQuery({
    queryKey: featureFlagKeys.flag(NEW_SHELL_FLAG_KEY),
    queryFn: () => settingsService.getBoolFlag(NEW_SHELL_FLAG_KEY),
    initialData: readCachedFlag,
    staleTime: 5 * 60 * 1000,
    retry: 1,
  });

  const enabled = data ?? false;

  useEffect(() => {
    try {
      localStorage.setItem(LS_KEY, String(enabled));
    } catch {
      // ignore — localStorage unavailable (private mode / SSR)
    }
  }, [enabled]);

  return enabled;
}
