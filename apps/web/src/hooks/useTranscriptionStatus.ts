/**
 * "Is this one generating right now?" — asked by the 管理字幕 dialog on open
 * (bugfix-dialog-reopen-shows-idle-during-run), so a reopen during a run shows
 * the progress instead of a paid 生成字幕 button.
 *
 * Rule 5: server state. Same shape as useTranscriptionEstimate: the dialog
 * removes this entry when it closes, so every open asks again; `enabled` gates
 * on the dialog being open for something that CAN generate (a series cannot).
 * No refetch on focus: the answer matters at the moment of opening — after
 * that, the progress stream (or the trigger's 409) owns the truth.
 */
import { useQuery } from '@tanstack/react-query';
import { transcriptionService, type TranscriptionStatus } from '../services/transcriptionService';

export const transcriptionStatusKeys = {
  all: ['subtitles', 'transcription-status'] as const,
  item: (mediaType: 'movie' | 'episode', mediaId: string) =>
    [...transcriptionStatusKeys.all, mediaType, mediaId] as const,
};

export function useTranscriptionStatus(
  mediaType: 'movie' | 'episode' | null,
  mediaId: string,
  options: { enabled: boolean }
) {
  return useQuery<TranscriptionStatus, Error>({
    queryKey: transcriptionStatusKeys.item(mediaType ?? 'movie', mediaId),
    queryFn: ({ signal }) =>
      transcriptionService.getTranscriptionStatus(mediaType ?? 'movie', mediaId, signal),
    enabled: options.enabled && mediaType !== null,
    refetchOnWindowFocus: false,
  });
}
