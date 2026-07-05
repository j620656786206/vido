/**
 * Route C generation trigger (ux3-subtitle-v2 AC 2, consumes 9R-9/9R-10 backend).
 *
 * `POST /api/v1/movies/{id}/transcribe?translate=true` — `:id` is the INT64 movie
 * id (⚠️ glossary routes key the same movie by STRING — callers convert), no body,
 * 202 → `{job_id, message}`. `translate=true` is ALWAYS sent: it runs the full
 * Route C pipeline (glossary-aware translate → OpenCC s2twp → atomic place);
 * omitting it would produce an EN-only SRT.
 *
 * Outcome discrimination (never throws for the two designed states):
 *   - 503 TRANSCRIPTION_DISABLED → `{status:'disabled'}` → 尚未設定 state (AC 5)
 *   - 409 TRANSCRIPTION_IN_PROGRESS → `{status:'inProgress'}` → attach to the
 *     running job's SSE stream instead of erroring (AC 2)
 *   - other non-2xx (404/400/500) → throws → fail-soft error state + 重試
 *
 * Movies-only today — the series route is 9R-10a; callers render the series CTA
 * disabled (capability honor) and never call this for series.
 */
import type { ApiResponse } from '../types/tmdb';
import { snakeToCamel } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export interface TranscribeStarted {
  jobId: string;
  message: string;
}

export type TranscribeOutcome =
  | { status: 'started'; result: TranscribeStarted }
  | { status: 'disabled' }
  | { status: 'inProgress' };

export const transcriptionService = {
  async startTranscription(movieId: number): Promise<TranscribeOutcome> {
    const response = await fetch(`${API_BASE_URL}/movies/${movieId}/transcribe?translate=true`, {
      method: 'POST',
    });

    const json = await response.json().catch(() => ({}) as Record<string, unknown>);
    const envelope = json as ApiResponse<unknown>;

    // Gate the two designed states on the WIRE ERROR CODE, not the bare HTTP
    // status: a reverse-proxy 503 (backend down, HTML body → empty envelope)
    // must fail-soft with 重試, NOT render the 尚未設定 settings CTA.
    if (response.status === 503 && envelope.error?.code === 'TRANSCRIPTION_DISABLED') {
      return { status: 'disabled' };
    }
    if (response.status === 409 && envelope.error?.code === 'TRANSCRIPTION_IN_PROGRESS') {
      return { status: 'inProgress' };
    }

    if (!response.ok || !envelope.success) {
      throw new Error(envelope.error?.message || `API request failed: ${response.status}`);
    }

    return { status: 'started', result: snakeToCamel<TranscribeStarted>(envelope.data) };
  },
};
