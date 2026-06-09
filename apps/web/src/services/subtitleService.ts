import type { ApiResponse } from '../types/tmdb';
import { snakeToCamel, camelToSnake } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, options);

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error?.message || `API request failed: ${response.status}`);
  }

  const data: ApiResponse<T> = await response.json();

  if (!data.success) {
    throw new Error(data.error?.message || 'API request failed');
  }

  return snakeToCamel<T>(data.data);
}

// --- Types (camelCase frontend convention, transformed at API boundary) ---

export interface SubtitleSearchParams {
  mediaId: string;
  mediaType: 'movie' | 'series';
  providers?: string[];
  query?: string;
}

export interface SubtitleScoreBreakdown {
  language: number;
  resolution: number;
  sourceTrust: number;
  group: number;
  downloads: number;
}

export interface SubtitleSearchResult {
  id: string;
  source: string;
  filename: string;
  language: string;
  downloadUrl: string;
  downloads: number;
  group: string;
  resolution: string;
  format: string;
  score: number;
  scoreBreakdown: SubtitleScoreBreakdown;
}

export interface SubtitleDownloadParams {
  mediaId: string;
  mediaType: 'movie' | 'series';
  mediaFilePath: string;
  subtitleId: string;
  provider: string;
  resolution?: string;
  convertToTraditional?: boolean;
  score?: number;
}

export interface SubtitleDownloadResult {
  subtitlePath: string;
  language: string;
  score: number;
}

export interface SubtitlePreviewParams {
  subtitleId: string;
  provider: string;
}

export interface SubtitlePreviewResult {
  lines: string[];
  language: string;
}

// --- Batch types (Story 8-11) ---
// NOTE: contract reconciled against the ACTUAL Story 8-9 backend (Rule 20 ack):
//  - `season_id` is a STRING on the wire (subtitle_handler.go BatchStartRequest), not a number.
//  - GET /batch/status returns { running, progress? } — NOT a bare progress object.

export type BatchScope = 'library' | 'season';

export interface BatchStartParams {
  scope: BatchScope;
  /** Required when scope === 'season'. String id per backend contract. */
  seasonId?: string;
}

export interface BatchStartResult {
  batchId: string;
  totalItems: number;
}

/** Live progress shape mirrored from the `subtitle_batch_progress` SSE payload. */
export interface BatchProgress {
  batchId: string;
  totalItems: number;
  currentIndex: number;
  currentItem: string;
  successCount: number;
  failCount: number;
  status: 'running' | 'complete' | 'cancelled' | 'error';
}

/** GET /subtitles/batch/status response (camelCase). */
export interface BatchStatusResponse {
  running: boolean;
  progress?: BatchProgress;
}

/**
 * Outcome of startBatch: either the batch started (202) or one was already
 * running (409), in which case the in-progress snapshot is surfaced instead of
 * throwing (AC #7).
 */
export type StartBatchOutcome =
  | { conflict: false; result: BatchStartResult }
  | { conflict: true; progress: BatchProgress };

export interface BatchCancelResult {
  cancelled: boolean;
}

// --- Service ---

export const subtitleService = {
  async searchSubtitles(params: SubtitleSearchParams): Promise<SubtitleSearchResult[]> {
    return fetchApi<SubtitleSearchResult[]>('/subtitles/search', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(camelToSnake(params)),
    });
  },

  async downloadSubtitle(params: SubtitleDownloadParams): Promise<SubtitleDownloadResult> {
    return fetchApi<SubtitleDownloadResult>('/subtitles/download', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(camelToSnake(params)),
    });
  },

  async previewSubtitle(params: SubtitlePreviewParams): Promise<SubtitlePreviewResult> {
    return fetchApi<SubtitlePreviewResult>('/subtitles/preview', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(camelToSnake(params)),
    });
  },

  // --- Batch (Story 8-11, consumes Story 8-9 backend) ---

  /**
   * POST /subtitles/batch. Returns the started batch on 202, or the in-progress
   * snapshot on 409 (AC #7) — never throws on a conflict. Other non-2xx throw.
   */
  async startBatch(params: BatchStartParams): Promise<StartBatchOutcome> {
    const response = await fetch(`${API_BASE_URL}/subtitles/batch`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(camelToSnake(params)),
    });

    const json = await response.json().catch(() => ({}) as Record<string, unknown>);

    if (response.status === 409) {
      return {
        conflict: true,
        progress: snakeToCamel<BatchProgress>((json as ApiResponse<unknown>).data),
      };
    }

    if (!response.ok || !(json as ApiResponse<unknown>).success) {
      const err = (json as ApiResponse<unknown>).error;
      throw new Error(err?.message || `API request failed: ${response.status}`);
    }

    return {
      conflict: false,
      result: snakeToCamel<BatchStartResult>((json as ApiResponse<unknown>).data),
    };
  },

  /** GET /subtitles/batch/status — current batch status (AC #7 recovery path). */
  async getBatchStatus(): Promise<BatchStatusResponse> {
    return fetchApi<BatchStatusResponse>('/subtitles/batch/status');
  },

  /** POST /subtitles/batch/cancel — stops the active batch (AC #5). Idempotent. */
  async cancelBatch(): Promise<BatchCancelResult> {
    return fetchApi<BatchCancelResult>('/subtitles/batch/cancel', {
      method: 'POST',
    });
  },
};
