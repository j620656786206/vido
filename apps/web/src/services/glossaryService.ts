/**
 * Glossary API client (ux3-subtitle-v2 Task 1, consumes the 9R-15 REST surface).
 *
 * Six routes under `/api/v1/media/{mediaId}/glossary` — `mediaId` is always the
 * STRINGIFIED local media id (⚠️ the transcribe trigger uses the int64 movie id;
 * the glossary group keys the SAME movie by string — callers convert).
 *
 * Rule 18: snakeToCamel on every response payload, camelToSnake on every request
 * body. 204 routes (edit / confirm / delete) resolve void — no body to parse.
 */
import type { ApiResponse } from '../types/tmdb';
import { snakeToCamel, camelToSnake } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

// --- Types (camelCase frontend convention, transformed at API boundary) ---

export type GlossarySource = 'subtitle' | 'metadata' | 'manual';

export interface GlossaryTerm {
  id: string;
  mediaId: string;
  termSrc: string;
  termZh: string;
  /** BCP-47-ish target language, backend default "zh-Hant". */
  language: string;
  source: GlossarySource;
  confirmed: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface GlossaryAddParams {
  termSrc: string;
  termZh: string;
  language?: string;
  /** Backend maps "" → manual; UI-added terms are manual. */
  source?: GlossarySource;
  confirmed?: boolean;
}

export interface GlossaryEditParams {
  termZh: string;
  confirmed: boolean;
}

export interface GlossaryConfirmAllResult {
  /** Number of terms flipped to confirmed. */
  confirmed: number;
}

// --- Fetch helpers ---

async function parseError(response: Response): Promise<Error> {
  const errorData = await response.json().catch(() => ({}));
  return new Error(errorData.error?.message || `API request failed: ${response.status}`);
}

/** JSON-returning routes: unwrap the `{success, data}` envelope + snakeToCamel. */
async function fetchJson<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, options);
  if (!response.ok) throw await parseError(response);

  const data: ApiResponse<T> = await response.json();
  if (!data.success) throw new Error(data.error?.message || 'API request failed');

  return snakeToCamel<T>(data.data);
}

/** 204 routes: success has no body — resolve void, still surface envelope errors. */
async function fetchNoContent(endpoint: string, options?: RequestInit): Promise<void> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, options);
  if (!response.ok) throw await parseError(response);
}

const jsonHeaders = { 'Content-Type': 'application/json' };

// --- Service (6 routes, shapes per glossary_handler.go — do NOT invent fields) ---

export const glossaryService = {
  /** GET /media/{mediaId}/glossary → `{terms: [...]}` (never null). */
  async listTerms(mediaId: string): Promise<GlossaryTerm[]> {
    const data = await fetchJson<{ terms: GlossaryTerm[] }>(
      `/media/${encodeURIComponent(mediaId)}/glossary`
    );
    return data.terms ?? [];
  },

  /** POST /media/{mediaId}/glossary → 201 with the created term. */
  async addTerm(mediaId: string, params: GlossaryAddParams): Promise<GlossaryTerm> {
    return fetchJson<GlossaryTerm>(`/media/${encodeURIComponent(mediaId)}/glossary`, {
      method: 'POST',
      headers: jsonHeaders,
      body: JSON.stringify(camelToSnake(params)),
    });
  },

  /** POST /media/{mediaId}/glossary/confirm-all → `{confirmed: <count>}`. */
  async confirmAll(mediaId: string): Promise<GlossaryConfirmAllResult> {
    return fetchJson<GlossaryConfirmAllResult>(
      `/media/${encodeURIComponent(mediaId)}/glossary/confirm-all`,
      { method: 'POST' }
    );
  },

  /** PUT /media/{mediaId}/glossary/{termId} → 204. Body is `{term_zh, confirmed}` only. */
  async editTerm(mediaId: string, termId: string, params: GlossaryEditParams): Promise<void> {
    return fetchNoContent(
      `/media/${encodeURIComponent(mediaId)}/glossary/${encodeURIComponent(termId)}`,
      {
        method: 'PUT',
        headers: jsonHeaders,
        body: JSON.stringify(camelToSnake(params)),
      }
    );
  },

  /** POST /media/{mediaId}/glossary/{termId}/confirm → 204. */
  async confirmTerm(mediaId: string, termId: string): Promise<void> {
    return fetchNoContent(
      `/media/${encodeURIComponent(mediaId)}/glossary/${encodeURIComponent(termId)}/confirm`,
      { method: 'POST' }
    );
  },

  /** DELETE /media/{mediaId}/glossary/{termId} → 204. */
  async deleteTerm(mediaId: string, termId: string): Promise<void> {
    return fetchNoContent(
      `/media/${encodeURIComponent(mediaId)}/glossary/${encodeURIComponent(termId)}`,
      { method: 'DELETE' }
    );
  },
};
