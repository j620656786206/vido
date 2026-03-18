/**
 * TanStack Query hooks for service connection status dashboard (Story 6.4)
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useSyncExternalStore } from 'react';
import { serviceStatusService } from '../services/serviceStatusService';
import type { ServiceStatusResponse, ServiceStatus } from '../services/serviceStatusService';

export const serviceStatusKeys = {
  all: ['serviceStatus'] as const,
  list: () => [...serviceStatusKeys.all, 'list'] as const,
};

const subscribeVisibility = (callback: () => void) => {
  document.addEventListener('visibilitychange', callback);
  return () => document.removeEventListener('visibilitychange', callback);
};
const getVisibilitySnapshot = () => document.visibilityState === 'visible';
const getServerSnapshot = () => true;

function usePageVisibility() {
  return useSyncExternalStore(subscribeVisibility, getVisibilitySnapshot, getServerSnapshot);
}

export function useServiceStatuses() {
  const isVisible = usePageVisibility();

  return useQuery<ServiceStatusResponse, Error>({
    queryKey: serviceStatusKeys.list(),
    queryFn: () => serviceStatusService.getAllStatuses(),
    refetchInterval: isVisible ? 30000 : false,
    staleTime: 25000,
    refetchOnWindowFocus: true,
  });
}

export function useTestServiceConnection() {
  const queryClient = useQueryClient();

  return useMutation<ServiceStatus, Error, string>({
    mutationFn: (serviceName) => serviceStatusService.testService(serviceName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: serviceStatusKeys.list() });
    },
  });
}
