import type { ApiResponse } from '../types/tmdb';
import { camelToSnake, snakeToCamel } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export type RequestStatus = 'pending' | 'searching' | 'downloading' | 'completed' | 'failed';
export type RequestMediaType = 'movie' | 'tv';

/**
 * Media request resource — confirmed against `[@contract-v1]` (Story 13-1a
 * AC #2 create shape, AC #3 list shape). camelCase after snakeToCamel;
 * mediaType speaks TMDB vocabulary ('movie'|'tv', matching the $type route
 * param — never 'series').
 */
export interface MediaRequest {
  id: string;
  tmdbId: number;
  mediaType: RequestMediaType;
  title: string;
  status: RequestStatus;
  fulfilmentSource: 'arr' | 'builtin' | null;
  externalId: string | null;
  seasons: string | null;
  episodes: string | null;
  errorMessage: string | null;
  requestedAt: string;
  updatedAt: string;
}

/** Statuses that count as "an open request exists" for the 想要 button. */
export const ACTIVE_REQUEST_STATUSES: readonly RequestStatus[] = [
  'pending',
  'searching',
  'downloading',
];

/**
 * Error carrying the backend's Rule-7 code (e.g. REQUEST_DUPLICATE) so callers
 * can branch on it — the plain-Error house fetchApi loses the code, and the
 * 想要 button needs REQUEST_DUPLICATE to settle into the requested state
 * rather than surfacing an error (13-1b AC #4).
 */
export class RequestApiError extends Error {
  readonly code: string;

  constructor(message: string, code: string) {
    super(message);
    this.name = 'RequestApiError';
    this.code = code;
  }
}

async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, options);
  const data = (await response.json().catch(() => null)) as ApiResponse<T> | null;

  if (!response.ok || !data?.success) {
    throw new RequestApiError(
      data?.error?.message || `API request failed: ${response.status}`,
      data?.error?.code || 'INTERNAL_ERROR'
    );
  }

  return snakeToCamel<T>(data.data as T);
}

/**
 * Request-system API client (Story 13-1b, Epic 13). Envelope + case transforms
 * per Rule 18 (snakeToCamel on responses, camelToSnake on POST bodies).
 */
export const requestService = {
  /** GET /api/v1/requests — newest first; [] when none. */
  async listRequests(): Promise<MediaRequest[]> {
    const res = await fetchApi<{ requests: MediaRequest[] }>('/requests');
    return res.requests ?? [];
  },

  /** POST /api/v1/requests — records a pending request (one-click 想要). */
  async createRequest(input: {
    tmdbId: number;
    mediaType: RequestMediaType;
  }): Promise<MediaRequest> {
    return fetchApi<MediaRequest>('/requests', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(camelToSnake(input)),
    });
  },

  /** Shared SSE endpoint — 13-3b's useRequestProgress consumes it; unused here. */
  getSSEUrl(): string {
    return `${API_BASE_URL}/events`;
  },
};

export default requestService;
