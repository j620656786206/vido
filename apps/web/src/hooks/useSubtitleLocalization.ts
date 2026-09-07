/**
 * Subtitle localization-level hooks (sub-7-4). Server state in TanStack
 * Query; the PUT response IS the fresh state and seeds the cache, so the
 * radio reflects the save immediately.
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  subtitleLocalizationService,
  type LocalizationLevel,
  type LocalizationSettings,
} from '../services/subtitleLocalizationService';

export const subtitleLocalizationQueryKeys = {
  all: ['settings', 'subtitle-localization'] as const,
};

export function useSubtitleLocalization() {
  return useQuery<LocalizationSettings, Error>({
    queryKey: subtitleLocalizationQueryKeys.all,
    queryFn: () => subtitleLocalizationService.get(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useSaveSubtitleLocalization() {
  const queryClient = useQueryClient();
  return useMutation<LocalizationSettings, Error, LocalizationLevel>({
    mutationFn: (level) => subtitleLocalizationService.set(level),
    onSuccess: (fresh) => {
      queryClient.setQueryData(subtitleLocalizationQueryKeys.all, fresh);
    },
  });
}
