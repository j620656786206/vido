/**
 * Home v3 readout-band API client (ux3-1-7), over ux3-1-6's
 * GET /api/v1/home-summary `[@contract-v1]`. Mirrors activityService's
 * fetch + snakeToCamel pattern (Rule 18). Every cell is fail-soft
 * (`status: 'ok' | 'unavailable'`) — a degraded cell arrives as data, never a
 * thrown error (B1/F3); the band hides that cell's number and renders the rest.
 */
import { snakeToCamel } from '../utils/caseTransform';
import type { SectionStatus } from './activityService';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

/** 繁中字幕 N/M by 部 (movies + series). Fileless items count in total only. */
export interface CoverageCell {
  status: SectionStatus;
  covered: number;
  total: number;
  error?: string;
}
/** Distinct media completed since server-local start-of-day (parse ∪ runs). */
export interface ProcessedTodayCell {
  status: SectionStatus;
  count: number;
  error?: string;
}
/**
 * The exception readout. The spend trio is ABSENT (not zero) when nothing has
 * been recorded yet — absent ≠ $0. `spendSource` picks the client copy:
 * live_batch = 執行中, last_run = 最近一次執行.
 */
export interface AttentionCell {
  status: SectionStatus;
  failedCount: number;
  spentUsd?: number;
  budgetUsd?: number;
  spendSource?: 'live_batch' | 'last_run' | string;
  error?: string;
}
/** Same number the nav badge shows (one counting path with /activity). */
export interface InFlightCell {
  status: SectionStatus;
  count: number;
  error?: string;
}
export interface HomeSummary {
  coverage: CoverageCell;
  processedToday: ProcessedTodayCell;
  attention: AttentionCell;
  inFlight: InFlightCell;
}

interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: { code: string; message: string };
}

async function fetchApi<T>(endpoint: string): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    headers: { 'Content-Type': 'application/json' },
  });

  const data: ApiResponse<T> = await response.json();

  if (!response.ok || !data.success) {
    throw new Error(data.error?.message || `API request failed: ${response.status}`);
  }
  if (data.data === undefined) {
    throw new Error('API response missing data field');
  }

  return snakeToCamel(data.data);
}

export const homeSummaryService = {
  async getHomeSummary(): Promise<HomeSummary> {
    return fetchApi<HomeSummary>('/home-summary');
  },
};

export default homeSummaryService;
