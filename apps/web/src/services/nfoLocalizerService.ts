/**
 * .nfo localization API client (Story 9R-13b, consumes the 9R-13 + 9R-13a surface).
 *
 * Three routes, two write semantics:
 *   - MOVIE  `POST /movies/:id/localize-nfo`   — ADDITIVE. Two recognised nfo
 *     filenames exist, so a free slot is written and the user's original file is
 *     never touched. Takes no body.
 *   - SERIES `POST /series/:id/localize-nfo`   — REPLACE. TV nfo names are
 *     single-slot (spike 9R-S1), so localizing overwrites `tvshow.nfo` after
 *     backing it up to `tvshow.nfo.orig`.
 *   - EPISODE `POST /episodes/:id/localize-nfo` — REPLACE, same as series.
 *
 * 🔴 The two REPLACE routes require `{"confirm_replace": true}`. That flag is a
 * USER decision, not a constant: it is a parameter here precisely so no caller
 * can hard-code it and quietly overwrite someone's curated metadata. The backend
 * answers 409 NFO_REPLACE_NOT_CONFIRMED without it.
 *
 * Rule 18: snakeToCamel on every response (so the wire's `backup_path` reaches
 * the UI as `backupPath`), camelToSnake is NOT used — the two request bodies are
 * a single already-snake_case flag.
 */
import type { ApiResponse } from '../types/tmdb';
import { snakeToCamel } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

/**
 * Carries the envelope's `error.code` through to the UI.
 *
 * The generic `parseError` helper other services use throws a bare Error and
 * DROPS the code, which would make the four documented failure modes
 * indistinguishable — 503 needs a "go configure a key" affordance, 400 needs
 * "scan first". Mirrors KeySettingsApiError (keySettingsService.ts).
 */
export class NfoLocalizeApiError extends Error {
  readonly code: string;

  constructor(message: string, code: string) {
    super(message);
    this.name = 'NfoLocalizeApiError';
    this.code = code;
  }
}

/** Error codes this surface can answer with (backend: nfo_localizer_handler.go). */
export const NFO_ERROR_CODES = {
  /** 409 — the confirm flag was missing. Reaching this from the UI is a BUG. */
  notConfirmed: 'NFO_REPLACE_NOT_CONFIRMED',
  /** 503 — no translation provider key configured. */
  disabled: 'NFO_LOCALIZE_DISABLED',
  /** 500 — the localization itself failed. */
  failed: 'NFO_LOCALIZE_FAILED',
  /** 400 — the row has no file path yet (never scanned). */
  missingPath: 'VALIDATION_REQUIRED_FIELD',
} as const;

/** One written .nfo. `replaced` is true only when an original was backed up. */
export interface NfoLocalizeResult {
  path: string;
  /** Non-empty only when `replaced` is true. */
  backupPath: string;
  replaced: boolean;
}

/**
 * A whole-show run. `skipped` counts episodes the database knows about but whose
 * video file is not on disk — there is nowhere to put their .nfo. That is not a
 * failure, which is why it is a separate number from `failed`.
 */
export interface NfoSeriesLocalizeResult {
  show: NfoLocalizeResult;
  episodes: NfoLocalizeResult[];
  succeeded: number;
  failed: number;
  skipped: number;
}

const jsonHeaders = { 'Content-Type': 'application/json' };

async function postJson<T>(endpoint: string, body?: unknown): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    method: 'POST',
    headers: jsonHeaders,
    ...(body === undefined ? {} : { body: JSON.stringify(body) }),
  });
  const data = (await response.json().catch(() => null)) as ApiResponse<T> | null;

  if (!response.ok || !data?.success) {
    throw new NfoLocalizeApiError(
      data?.error?.message || `API request failed: ${response.status}`,
      data?.error?.code || 'INTERNAL_ERROR'
    );
  }

  return snakeToCamel<T>(data.data as T);
}

export const nfoLocalizerService = {
  /** Movies are additive — no confirmation flag exists for this route. */
  localizeMovieNfo(id: string): Promise<NfoLocalizeResult> {
    return postJson<NfoLocalizeResult>(`/movies/${id}/localize-nfo`);
  },

  /**
   * `confirmReplace` is required and unforgiving on purpose: passing `false`
   * sends `false`, and the backend refuses. The caller must have asked a human.
   */
  localizeSeriesNfo(
    id: string,
    {
      confirmReplace,
      includeEpisodes = false,
    }: { confirmReplace: boolean; includeEpisodes?: boolean }
  ): Promise<NfoLocalizeResult | NfoSeriesLocalizeResult> {
    const query = includeEpisodes ? '?include_episodes=true' : '';
    return postJson<NfoLocalizeResult | NfoSeriesLocalizeResult>(
      `/series/${id}/localize-nfo${query}`,
      { confirm_replace: confirmReplace }
    );
  },

  localizeEpisodeNfo(
    id: string,
    { confirmReplace }: { confirmReplace: boolean }
  ): Promise<NfoLocalizeResult> {
    return postJson<NfoLocalizeResult>(`/episodes/${id}/localize-nfo`, {
      confirm_replace: confirmReplace,
    });
  },
};

/** Narrows a whole-show result from the single-file one. */
export function isSeriesResult(
  result: NfoLocalizeResult | NfoSeriesLocalizeResult
): result is NfoSeriesLocalizeResult {
  return 'show' in result;
}
