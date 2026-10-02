/**
 * 「從官方字幕學譯名」 client (story sub-7-5b, consumes `[@contract-v1]`
 * GET/POST /api/v1/subtitles/glossary/mine).
 *
 * GET is the miner's status: whether a run is in flight and what the last
 * run learned per show. POST with no body starts a library sweep (202) — the
 * settings button; POST with a series id runs one show synchronously.
 */
import type { ApiResponse } from '../types/tmdb';
import { snakeToCamel } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

/** One episode's outcome inside a show's run. */
export interface MineEpisodeReport {
  episodeId: string;
  file: string;
  zhSource?: string;
  enSource?: string;
  segments?: number;
  skipped?: string;
}

/** One show's run. */
export interface MineResult {
  seriesId: string;
  title: string;
  scope: string;
  episodesTotal: number;
  episodesUsed: number;
  fansubSkipped: number;
  termsFound: number;
  termsInserted: number;
  startedAt: string;
  finishedAt: string;
  error?: string;
  episodes?: MineEpisodeReport[];
}

export interface MineStatus {
  running: boolean;
  runningFor?: string;
  lastRunAt?: string;
  results: MineResult[];
}

export class GlossaryMineApiError extends Error {
  readonly code: string;
  constructor(message: string, code: string) {
    super(message);
    this.name = 'GlossaryMineApiError';
    this.code = code;
  }
}

async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, options);
  const data = (await response.json().catch(() => null)) as ApiResponse<T> | null;
  if (!response.ok || !data?.success) {
    throw new GlossaryMineApiError(
      data?.error?.message || `API request failed: ${response.status}`,
      data?.error?.code || 'INTERNAL_ERROR'
    );
  }
  return snakeToCamel<T>(data.data as T);
}

export const glossaryMineService = {
  status(): Promise<MineStatus> {
    return fetchApi<MineStatus>('/subtitles/glossary/mine');
  },
  /** Start the library sweep. Resolves on 202; GLOSSARY_MINE_RUNNING on 409. */
  startSweep(): Promise<{ started: boolean }> {
    return fetchApi<{ started: boolean }>('/subtitles/glossary/mine', { method: 'POST' });
  },
};
