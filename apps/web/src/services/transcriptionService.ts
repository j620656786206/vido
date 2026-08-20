/**
 * Route C generation trigger (ux3-subtitle-v2 AC 2, consumes 9R-9/9R-10 backend).
 *
 * `POST /api/v1/movies/{id}/transcribe?translate=true` — `:id` is the movie row
 * id, a UUID STRING (9R-18 — same key the glossary routes use; no conversion),
 * no body, 202 → `{job_id, message}`. `translate=true` is ALWAYS sent: it runs the full
 * Route C pipeline (glossary-aware translate → OpenCC s2twp → atomic place);
 * omitting it would produce an EN-only SRT.
 *
 * Outcome discrimination (never throws for the two designed states):
 *   - 503 TRANSCRIPTION_DISABLED → `{status:'disabled'}` → 尚未設定 state (AC 5)
 *   - 409 TRANSCRIPTION_IN_PROGRESS → `{status:'inProgress'}` → attach to the
 *     running job's SSE stream instead of erroring (AC 2)
 *   - other non-2xx (404/400/500) → throws → fail-soft error state + 重試
 *
 * Per-EPISODE sibling (9R-10c, consuming 9R-10a AC #2 [@contract-v1]):
 * `POST /api/v1/episodes/{id}/transcribe` — same 202/503/409 shape, but takes
 * NO `translate` param: the backend forces translation on that route, since an
 * English-only SRT has no consumer.
 *
 * A SERIES has no generate route of its own — series generation runs per
 * episode (or through the generation batch), so callers render the series-level
 * CTA disabled and point at the episode list (capability honor).
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

/**
 * Shared outcome discrimination for BOTH transcribe routes. Deliberately ONE
 * implementation: the subtle part is that the two designed states are gated on
 * the WIRE ERROR CODE, not the bare HTTP status — a reverse-proxy 503 (backend
 * down, HTML body → empty envelope) must fail-soft with 重試 and must NOT
 * render the 尚未設定 settings CTA. A second copy of this would drift.
 */
async function parseTranscribeResponse(response: Response): Promise<TranscribeOutcome> {
  const json = await response.json().catch(() => ({}) as Record<string, unknown>);
  const envelope = json as ApiResponse<unknown>;

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
}

export const transcriptionService = {
  async startTranscription(movieId: string): Promise<TranscribeOutcome> {
    const response = await fetch(`${API_BASE_URL}/movies/${movieId}/transcribe?translate=true`, {
      method: 'POST',
    });
    return parseTranscribeResponse(response);
  },

  /**
   * Per-episode trigger (9R-10c AC #2). `episodeId` is the EPISODE ROW id from
   * MergedEpisode.episodeId — NOT the series id, and NOT the glossary key: the
   * glossary is per-show and keyed on the series id (see ManageSubtitleDialogV2's
   * glossaryMediaId prop). No `translate` param — the route forces it.
   */
  async startEpisodeTranscription(episodeId: string): Promise<TranscribeOutcome> {
    const response = await fetch(`${API_BASE_URL}/episodes/${episodeId}/transcribe`, {
      method: 'POST',
    });
    return parseTranscribeResponse(response);
  },
};
