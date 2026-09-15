/**
 * Sonarr / Radarr settings hooks (story 13-6). One key space per plugin, so saving Sonarr
 * never refetches Radarr.
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  dvrSettingsService,
  type DvrConfig,
  type DvrPlugin,
  type DvrQualityProfile,
  type DvrRootFolder,
  type SaveDvrConfigParams,
  type TestDvrConnectionParams,
} from '../services/dvrSettings';

export const dvrSettingsKeys = {
  all: ['dvr-settings'] as const,
  plugin: (plugin: DvrPlugin) => [...dvrSettingsKeys.all, plugin] as const,
  config: (plugin: DvrPlugin) => [...dvrSettingsKeys.plugin(plugin), 'config'] as const,
  qualityProfiles: (plugin: DvrPlugin) =>
    [...dvrSettingsKeys.plugin(plugin), 'quality-profiles'] as const,
  rootFolders: (plugin: DvrPlugin) => [...dvrSettingsKeys.plugin(plugin), 'root-folders'] as const,
};

export function useDvrConfig(plugin: DvrPlugin) {
  return useQuery<DvrConfig, Error>({
    queryKey: dvrSettingsKeys.config(plugin),
    queryFn: () => dvrSettingsService.getConfig(plugin),
    staleTime: 30 * 1000,
    // The form is seeded from this query; a focus refetch would overwrite what the user is typing.
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });
}

export function useSaveDvrConfig(plugin: DvrPlugin) {
  const queryClient = useQueryClient();
  return useMutation<void, Error, SaveDvrConfigParams>({
    mutationFn: (params) => dvrSettingsService.saveConfig(plugin, params),
    onSuccess: () => {
      // A new URL/key can mean a different server: its profiles and folders go stale too.
      queryClient.invalidateQueries({ queryKey: dvrSettingsKeys.plugin(plugin) });
    },
  });
}

export function useTestDvrConnection(plugin: DvrPlugin) {
  return useMutation<void, Error, TestDvrConnectionParams>({
    mutationFn: (params) => dvrSettingsService.testConnection(plugin, params),
  });
}

/** The passthrough lists read the SAVED, ENABLED config — callers gate `enabled` on that. */
export function useDvrQualityProfiles(plugin: DvrPlugin, enabled: boolean) {
  return useQuery<DvrQualityProfile[], Error>({
    queryKey: dvrSettingsKeys.qualityProfiles(plugin),
    queryFn: () => dvrSettingsService.getQualityProfiles(plugin),
    enabled,
    retry: false,
    refetchOnWindowFocus: false,
  });
}

export function useDvrRootFolders(plugin: DvrPlugin, enabled: boolean) {
  return useQuery<DvrRootFolder[], Error>({
    queryKey: dvrSettingsKeys.rootFolders(plugin),
    queryFn: () => dvrSettingsService.getRootFolders(plugin),
    enabled,
    retry: false,
    refetchOnWindowFocus: false,
  });
}
