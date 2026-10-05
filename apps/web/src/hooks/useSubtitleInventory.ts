/**
 * TanStack Query hook for one title's real subtitle list
 * (bugfix-subtitle-dialog-real-inventory). Enabled only while the 管理字幕
 * dialog is open for a movie or an episode — a season expand never calls it,
 * because the embedded half runs ffprobe on the file (red line 3).
 */
import { useQuery } from '@tanstack/react-query';
import { subtitleService, type SubtitleInventory } from '../services/subtitleService';

export const subtitleInventoryKeys = {
  all: ['subtitle-inventory'] as const,
  item: (mediaType: 'movie' | 'episode', id: string) =>
    ['subtitle-inventory', mediaType, id] as const,
};

export function useSubtitleInventory(mediaType: 'movie' | 'episode', id: string, enabled: boolean) {
  return useQuery<SubtitleInventory, Error>({
    queryKey: subtitleInventoryKeys.item(mediaType, id),
    queryFn: () => subtitleService.getInventory(mediaType, id),
    enabled: enabled && id !== '',
    // Files change only when someone adds one or a run places one; the dialog
    // refetches after a run, so a short staleTime is enough.
    staleTime: 30_000,
    retry: false,
  });
}
