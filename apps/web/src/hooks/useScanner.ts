/**
 * TanStack Query hooks for scanner management (Story 7.3)
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { scannerService } from '../services/scannerService';
import type {
  ScanStatus,
  ScanResult,
  ScheduleConfig,
  ScheduleFrequency,
} from '../services/scannerService';

export const scannerKeys = {
  all: ['scanner'] as const,
  status: () => [...scannerKeys.all, 'status'] as const,
  schedule: () => [...scannerKeys.all, 'schedule'] as const,
};

export function useScanStatus() {
  return useQuery<ScanStatus, Error>({
    queryKey: scannerKeys.status(),
    queryFn: () => scannerService.getScanStatus(),
    refetchInterval: (query) => {
      // Poll every 3s while scanning, otherwise every 30s
      return query.state.data?.is_scanning ? 3000 : 30000;
    },
  });
}

export function useTriggerScan() {
  const queryClient = useQueryClient();

  return useMutation<ScanResult, Error>({
    mutationFn: () => scannerService.triggerScan(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: scannerKeys.status() });
    },
  });
}

export function useCancelScan() {
  const queryClient = useQueryClient();

  return useMutation<void, Error>({
    mutationFn: () => scannerService.cancelScan(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: scannerKeys.status() });
    },
  });
}

export function useScanSchedule() {
  return useQuery<ScheduleConfig, Error>({
    queryKey: scannerKeys.schedule(),
    queryFn: () => scannerService.getSchedule(),
  });
}

export function useUpdateScanSchedule() {
  const queryClient = useQueryClient();

  return useMutation<ScheduleConfig, Error, ScheduleFrequency>({
    mutationFn: (frequency) => scannerService.updateSchedule(frequency),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: scannerKeys.schedule() });
    },
  });
}
