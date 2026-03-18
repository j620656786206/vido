/**
 * TanStack Query hooks for system logs (Story 6.3)
 */

import { useQuery, useMutation, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import { logService } from '../services/logService';
import type { LogsResponse, LogClearResult, LogFilter } from '../services/logService';

export const logKeys = {
  all: ['logs'] as const,
  list: (filter: LogFilter) => [...logKeys.all, 'list', filter] as const,
};

export function useLogs(filter: LogFilter = {}) {
  return useQuery<LogsResponse, Error>({
    queryKey: logKeys.list(filter),
    queryFn: () => logService.getLogs(filter),
    staleTime: 10 * 1000, // 10 seconds
    placeholderData: keepPreviousData,
  });
}

export function useClearLogs() {
  const queryClient = useQueryClient();

  return useMutation<LogClearResult, Error, number>({
    mutationFn: (days) => logService.clearLogs(days),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: logKeys.all });
    },
  });
}
