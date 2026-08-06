/**
 * Provider API-key settings hooks (Story sub-2-1b — FR25).
 *
 * Rule 5: the key list is server state and lives in TanStack Query; the
 * un-saved form (typed values, per-visit HTTPS acknowledgement) is local
 * component state and deliberately does NOT go through a store.
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  keySettingsService,
  type KeySettings,
  type KeyUpdates,
} from '../services/keySettingsService';

export const keySettingsQueryKeys = {
  all: ['settings', 'keys'] as const,
  list: () => [...keySettingsQueryKeys.all, 'list'] as const,
};

/**
 * GET the current key state (source / configured / masked hint + writability).
 * `enabled` (default true) lets always-mounted consumers gate the fetch on
 * actual need — e.g. ManageSubtitleDialogV2 passes its `open` flag (sub-2-2d
 * AC #2) so a closed dialog costs no request, mirroring useGlossaryTerms.
 */
export function useKeySettings(options?: { enabled?: boolean }) {
  return useQuery<KeySettings, Error>({
    queryKey: keySettingsQueryKeys.list(),
    queryFn: () => keySettingsService.getKeys(),
    enabled: options?.enabled ?? true,
    // Secrets change only when this page writes them, and the PUT response
    // seeds the cache directly — so a short poll would buy nothing.
    staleTime: 5 * 60 * 1000,
  });
}

/**
 * PUT a partial update. The response IS the fresh state, so it seeds the cache
 * directly: the row flips 尚未設定 → 已設定 (or 環境變數 → 已設定) immediately,
 * which is the user-visible proof the save took effect.
 */
export function useSaveKeys() {
  const queryClient = useQueryClient();

  return useMutation<KeySettings, Error, KeyUpdates>({
    mutationFn: (updates) => keySettingsService.saveKeys(updates),
    onSuccess: (fresh) => {
      queryClient.setQueryData(keySettingsQueryKeys.list(), fresh);
    },
  });
}

/** Probe a Claude key against the live provider. `undefined` tests the resolved one. */
export function useTestClaudeKey() {
  return useMutation<void, Error, string | undefined>({
    mutationFn: (candidate) => keySettingsService.testClaudeKey(candidate),
  });
}
