/**
 * The price of a click on 生成字幕 (story dsr-6a AC #5).
 *
 * Rule 5: the estimate is SERVER state. Two deliberate choices:
 * - the global default staleTime, NOT 0 — the dialog removes this entry when it
 *   closes (so every open re-estimates), and a seeded visual fixture must not be
 *   refetched against a CI that has no backend;
 * - `enabled` gates on the dialog being open for something that CAN generate —
 *   a series has no generate route, so it never asks.
 */
import { useQuery } from '@tanstack/react-query';
import { transcriptionService, type TranscriptionEstimate } from '../services/transcriptionService';

export const transcriptionEstimateKeys = {
  all: ['subtitles', 'transcription-estimate'] as const,
  item: (mediaType: 'movie' | 'episode', mediaId: string) =>
    [...transcriptionEstimateKeys.all, mediaType, mediaId] as const,
};

export function useTranscriptionEstimate(
  mediaType: 'movie' | 'episode' | null,
  mediaId: string,
  options: { enabled: boolean }
) {
  return useQuery<TranscriptionEstimate, Error>({
    queryKey: transcriptionEstimateKeys.item(mediaType ?? 'movie', mediaId),
    queryFn: ({ signal }) =>
      transcriptionService.getTranscriptionEstimate(mediaType ?? 'movie', mediaId, signal),
    enabled: options.enabled && mediaType !== null,
  });
}
