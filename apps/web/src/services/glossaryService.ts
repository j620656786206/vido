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

/**
 * Where a term came from. `official_subtitle` (aligned out of an official
 * zh-Hant subtitle of the same show, sub-7-5) and `community` (imported from
 * another install, sub-8-1) arrived with sub-7-1, which dropped the SQLite
 * CHECK so the set is now closed by the Go model, not the schema.
 */
export type GlossarySource = 'subtitle' | 'metadata' | 'manual' | 'official_subtitle' | 'community';

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

// --- sub-8-1: export / import (`[@contract-v1]`) ---

/** A term both sides have with different renderings; `id` is MY row. */
export interface GlossaryImportConflict {
  id: string;
  termSrc: string;
  mine: string;
  theirs: string;
  mineSource: string;
  mineConfirmed: boolean;
}

/** POST /media/{mediaId}/glossary/import result. */
export interface GlossaryImportResult {
  imported: number;
  skipped: number;
  conflicts: GlossaryImportConflict[];
  title: string;
}

/**
 * A refused export/import. `code` is the backend's GLOSSARY_* code; the
 * message already carries the backend's 「怎麼辦」 suggestion when it sent one,
 * so the panel can show it as-is (F6c-D note ①–③).
 */
export class GlossaryExchangeError extends Error {
  readonly code: string;
  constructor(message: string, code: string) {
    super(message);
    this.name = 'GlossaryExchangeError';
    this.code = code;
  }
}

async function exchangeError(response: Response): Promise<GlossaryExchangeError> {
  const body = await response.json().catch(() => ({}));
  const message: string = body.error?.message || `API request failed: ${response.status}`;
  const suggestion: string = body.error?.suggestion || '';
  return new GlossaryExchangeError(
    suggestion ? `${message}${suggestion}` : message,
    body.error?.code || 'UNKNOWN'
  );
}

/** Pull the filename out of `attachment; filename="…"`. */
function filenameFrom(disposition: string | null, fallback: string): string {
  const m = disposition?.match(/filename="?([^";]+)"?/i);
  return m?.[1] ?? fallback;
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

  /**
   * PUT /media/{mediaId}/glossary/{termId} → 204. Body is `{term_zh, confirmed}`
   * only; the server confirms every edit regardless (⚖️ 2026-10-02).
   */
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

  /**
   * GET /media/{mediaId}/glossary/export — the vido-glossary FILE (not an
   * envelope). Returns the bytes and the server's filename for a download.
   */
  async exportFile(mediaId: string): Promise<{ blob: Blob; filename: string }> {
    const response = await fetch(
      `${API_BASE_URL}/media/${encodeURIComponent(mediaId)}/glossary/export`
    );
    if (!response.ok) throw await exchangeError(response);
    const blob = await response.blob();
    return {
      blob,
      filename: filenameFrom(response.headers.get('Content-Disposition'), 'vido-glossary.json'),
    };
  },

  /** POST /media/{mediaId}/glossary/import (multipart `file`). */
  async importFile(mediaId: string, file: File): Promise<GlossaryImportResult> {
    const form = new FormData();
    form.append('file', file);
    const response = await fetch(
      `${API_BASE_URL}/media/${encodeURIComponent(mediaId)}/glossary/import`,
      { method: 'POST', body: form }
    );
    if (!response.ok) throw await exchangeError(response);
    const data: ApiResponse<GlossaryImportResult> = await response.json();
    if (!data.success)
      throw new GlossaryExchangeError(
        data.error?.message || '匯入失敗',
        data.error?.code || 'UNKNOWN'
      );
    const result = snakeToCamel<GlossaryImportResult>(data.data);
    return { ...result, conflicts: result.conflicts ?? [] };
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
