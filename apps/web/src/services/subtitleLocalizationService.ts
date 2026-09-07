/**
 * Subtitle localization-level client (sub-7-4 AC #3/#4).
 *
 * Mirrors GET/PUT /api/v1/subtitles/localization: the level in force
 * (literal | standard | ott), WHERE it came from (settings beats env beats
 * default — the API-key precedence, surfaced so the page can be honest about
 * it), and the closed list of levels.
 */

import type { ApiResponse } from '../types/tmdb';
import { snakeToCamel } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

/** Closed set — mirrors `prompts.LocalizationLevel`. */
export type LocalizationLevel = 'literal' | 'standard' | 'ott';

/** Where the effective level resolved from. */
export type LocalizationSource = 'settings' | 'env' | 'default';

export interface LocalizationSettings {
  level: LocalizationLevel;
  source: LocalizationSource;
  levels: LocalizationLevel[];
}

export class SubtitleLocalizationApiError extends Error {
  readonly code: string;

  constructor(message: string, code: string) {
    super(message);
    this.name = 'SubtitleLocalizationApiError';
    this.code = code;
  }
}

async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, options);
  const data = (await response.json().catch(() => null)) as ApiResponse<T> | null;

  if (!response.ok || !data?.success) {
    throw new SubtitleLocalizationApiError(
      data?.error?.message || `API request failed: ${response.status}`,
      data?.error?.code || 'INTERNAL_ERROR'
    );
  }

  return snakeToCamel<T>(data.data as T);
}

export const subtitleLocalizationService = {
  get(): Promise<LocalizationSettings> {
    return fetchApi<LocalizationSettings>('/subtitles/localization');
  },

  set(level: LocalizationLevel): Promise<LocalizationSettings> {
    return fetchApi<LocalizationSettings>('/subtitles/localization', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ level }),
    });
  },
};
