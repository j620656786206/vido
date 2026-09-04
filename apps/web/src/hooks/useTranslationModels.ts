/**
 * Translation-model catalog hook (story sub-6-8b AC #2, consumes sub-6-8a
 * AC #2 `[@contract-v1]`).
 *
 * Rule 5: the model list is SERVER state — it is a property of the deployment
 * (which API keys resolve, what `CLAUDE_MODEL` is set to), not of the dialog
 * that renders it — so it lives in TanStack Query and never in a store.
 */

import { useQuery } from '@tanstack/react-query';
import { subtitleService, type TranslationModelList } from '../services/subtitleService';

export const translationModelQueryKeys = {
  all: ['settings', 'models'] as const,
};

/**
 * GET the selectable translation models + the deployment's default.
 *
 * `enabled` gates the fetch on actual need — the consent flow passes its
 * `open` flag so a closed dialog costs no request (useKeySettings precedent).
 * The catalog only moves when an operator saves a key or restarts with a
 * different `CLAUDE_MODEL`, so a long staleTime is honest here; an empty list
 * (no AI key configured) is a legitimate 200, not an error.
 */
export function useTranslationModels(options?: { enabled?: boolean }) {
  return useQuery<TranslationModelList, Error>({
    queryKey: translationModelQueryKeys.all,
    queryFn: () => subtitleService.getModels(),
    enabled: options?.enabled ?? true,
    staleTime: 5 * 60 * 1000,
  });
}
