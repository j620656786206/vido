/**
 * qBittorrent hooks using TanStack Query (Story 4.1)
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  qbittorrentService,
  type QBConfigResponse,
  type QBVersionInfo,
  type SaveQBConfigParams,
} from '../services/qbittorrent';

export const qbittorrentKeys = {
  all: ['qbittorrent'] as const,
  config: () => [...qbittorrentKeys.all, 'config'] as const,
};

/**
 * Hook for fetching qBittorrent configuration (AC1)
 */
export function useQBittorrentConfig() {
  return useQuery<QBConfigResponse, Error>({
    queryKey: qbittorrentKeys.config(),
    queryFn: () => qbittorrentService.getConfig(),
    staleTime: 5 * 60 * 1000,
  });
}

/**
 * Hook for saving qBittorrent configuration (AC2, AC5)
 */
export function useSaveQBConfig() {
  const queryClient = useQueryClient();

  return useMutation<void, Error, SaveQBConfigParams>({
    mutationFn: (config) => qbittorrentService.saveConfig(config),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qbittorrentKeys.config() });
    },
  });
}

/**
 * Hook for testing qBittorrent connection (AC3)
 * Accepts config directly so connection can be tested without saving first.
 */
export function useTestQBConnection() {
  return useMutation<QBVersionInfo, Error, SaveQBConfigParams>({
    mutationFn: (config) => qbittorrentService.testConnection(config),
  });
}
