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

// While the server checks the English subtitle beside the file in the
// background (-b2), ask again until it has an answer — the quote then drops to
// the translation price on its own. A cold 4K remux takes ~40 s; after about
// three minutes the dialog stops asking and keeps the speech-recognition
// quote it already shows (the server answers once the check is done, the next
// time the dialog opens).
export const SIDECAR_CHECK_POLL_MS = 3000;
export const SIDECAR_CHECK_MAX_POLLS = 60;

export function estimateRefetchInterval(
  data: TranscriptionEstimate | undefined,
  fetchesSoFar = 0
): number | false {
  if (data?.sidecarCheck?.status !== 'checking') return false;
  return fetchesSoFar < SIDECAR_CHECK_MAX_POLLS ? SIDECAR_CHECK_POLL_MS : false;
}

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
    refetchInterval: (query) =>
      estimateRefetchInterval(query.state.data, query.state.dataUpdateCount),
  });
}
