/**
 * Connection health monitoring hooks using TanStack Query (Story 4.6)
 */

import { useQuery } from '@tanstack/react-query';
import { useSyncExternalStore } from 'react';
import { healthService, type ServiceHealth, type ConnectionEvent } from '../services/healthService';

export const healthKeys = {
  all: ['health'] as const,
  services: () => [...healthKeys.all, 'services'] as const,
  qbittorrent: () => [...healthKeys.all, 'qbittorrent'] as const,
  history: (service: string) => [...healthKeys.all, 'history', service] as const,
};

/**
 * Singleton page visibility detection using useSyncExternalStore.
 */
const subscribeVisibility = (callback: () => void) => {
  document.addEventListener('visibilitychange', callback);
  return () => document.removeEventListener('visibilitychange', callback);
};
const getVisibilitySnapshot = () => document.visibilityState === 'visible';
const getServerSnapshot = () => true;

function usePageVisibility() {
  return useSyncExternalStore(subscribeVisibility, getVisibilitySnapshot, getServerSnapshot);
}

/**
 * Hook for qBittorrent connection health status.
 * Polls every 30 seconds per NFR-R6.
 * Exposes: status, lastSuccess, errorCount, message.
 */
export function useQBConnectionHealth() {
  const isVisible = usePageVisibility();

  return useQuery<ServiceHealth, Error>({
    queryKey: healthKeys.qbittorrent(),
    queryFn: async () => {
      const response = await healthService.getServicesHealth();
      return response.services.qbittorrent;
    },
    refetchInterval: isVisible ? 30000 : false, // NFR-R6: 30 second polling
    staleTime: 25000,
    refetchOnWindowFocus: true,
  });
}

/**
 * Hook for connection history of a specific service.
 * Only fetches when enabled (e.g., when modal is open).
 */
export function useConnectionHistory(service: string, enabled = true) {
  return useQuery<ConnectionEvent[], Error>({
    queryKey: healthKeys.history(service),
    queryFn: () => healthService.getConnectionHistory(service),
    enabled,
  });
}

export type { ServiceHealth, ConnectionEvent };
