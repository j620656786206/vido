/**
 * Opt-in anonymous usage report hooks (infra-optin-usage-report-b1). The PUT
 * response IS the fresh state and seeds the cache, so the switch and the
 * last-sent readout reflect the save immediately.
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { usageReportService, type UsageReportStatus } from '../services/usageReportService';

export const usageReportQueryKeys = {
  all: ['settings', 'usage-report'] as const,
};

export function useUsageReport() {
  return useQuery<UsageReportStatus, Error>({
    queryKey: usageReportQueryKeys.all,
    queryFn: () => usageReportService.get(),
    staleTime: 60 * 1000,
  });
}

export function useSetUsageReport() {
  const queryClient = useQueryClient();
  return useMutation<UsageReportStatus, Error, boolean>({
    mutationFn: (enabled) => usageReportService.setEnabled(enabled),
    // A GET still in flight would land after the save and put the old switch
    // position back on screen.
    onMutate: () => queryClient.cancelQueries({ queryKey: usageReportQueryKeys.all }),
    onSuccess: (fresh) => {
      queryClient.setQueryData(usageReportQueryKeys.all, fresh);
    },
  });
}
