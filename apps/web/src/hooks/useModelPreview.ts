/**
 * 「試跑 20 句」 (story sub-7-8c AC #4).
 *
 * One mutation per model row, with the row's own state machine kept here so
 * the picker stays presentational: idle → running → done | failed. A finished
 * preview also invalidates the catalog query, so the `local_grade` the server
 * stored shows up on the next open without a reload — but the row shows the
 * result it just got back immediately, without waiting for that refetch.
 */
import { useCallback, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import {
  ModelPreviewError,
  subtitleService,
  type ModelLocalGrade,
} from '../services/subtitleService';
import { translationModelQueryKeys } from './useTranslationModels';

export type ModelPreviewRowState =
  | { status: 'idle' }
  | { status: 'running' }
  | { status: 'done'; result: ModelLocalGrade }
  | { status: 'failed'; message: string; unpaid: boolean; code: string };

export interface ModelPreviewController {
  /** Per-model row state; a model never previewed in this session is absent. */
  states: Record<string, ModelPreviewRowState>;
  /** Any model previewing right now — the dialog locks the quote meanwhile. */
  busy: boolean;
  preview: (modelId: string) => void;
}

/** User-facing line for a refused/failed preview, by backend code. */
export function previewFailureMessage(err: unknown): {
  message: string;
  unpaid: boolean;
  code: string;
} {
  if (err instanceof ModelPreviewError) {
    switch (err.code) {
      case 'AI_PREVIEW_TOO_SOON':
        return { message: '剛剛試跑過，稍後再試', unpaid: true, code: err.code };
      case 'AI_NOT_CONFIGURED':
        return { message: '先到 設定 → API 金鑰 存一組 Claude 金鑰', unpaid: true, code: err.code };
      case 'AI_UNAUTHORIZED':
        return { message: '金鑰無效，請到 設定 → API 金鑰 檢查', unpaid: true, code: err.code };
      default:
        return {
          message: err.unpaid ? '試跑失敗，沒有扣款' : '試跑失敗',
          unpaid: err.unpaid,
          code: err.code,
        };
    }
  }
  return { message: '試跑失敗', unpaid: false, code: 'UNKNOWN' };
}

export function useModelPreview(): ModelPreviewController {
  const queryClient = useQueryClient();
  const [states, setStates] = useState<Record<string, ModelPreviewRowState>>({});

  const preview = useCallback(
    (modelId: string) => {
      setStates((prev) => {
        if (prev[modelId]?.status === 'running') return prev;
        return { ...prev, [modelId]: { status: 'running' } };
      });
      subtitleService
        .previewModel(modelId)
        .then((result) => {
          setStates((prev) => ({ ...prev, [modelId]: { status: 'done', result } }));
          void queryClient.invalidateQueries({ queryKey: translationModelQueryKeys.all });
        })
        .catch((err: unknown) => {
          setStates((prev) => ({
            ...prev,
            [modelId]: { status: 'failed', ...previewFailureMessage(err) },
          }));
        });
    },
    [queryClient]
  );

  const busy = Object.values(states).some((s) => s.status === 'running');
  return { states, busy, preview };
}
