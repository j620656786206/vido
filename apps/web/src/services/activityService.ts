/**
 * Activity-hub API client (UX Redesign ux3-2-3 / D4-1) for the /activity route.
 * Mirrors statusSummaryService's fetch + snakeToCamel pattern (Rule 18): the backend
 * returns snake_case (`active_jobs`, `percent_done`, `parse_count`, …); this boundary
 * camelCases it. Each section is fail-soft (`status: 'ok' | 'unavailable'`), so a
 * degraded section arrives as data, never a thrown error (B1/F3).
 *
 * `kind` / `result` are stable enums; all human copy + icons live on the client (i18n) —
 * see ux3-2-2-activity-api.md for the contract.
 */
import { snakeToCamel } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export type SectionStatus = 'ok' | 'unavailable';

/** An in-flight background job. `kind` drives the row's icon + title on the client. */
export interface ActiveJob {
  kind: 'scan' | 'subtitle_batch' | string;
  percentDone: number;
  detail?: string;
  current?: number;
  total?: number;
}
export interface ActiveJobsSection {
  status: SectionStatus;
  jobs: ActiveJob[];
  error?: string;
}
export interface PendingSection {
  status: SectionStatus;
  parseCount: number;
  error?: string;
}
export interface DownloadsSection {
  status: SectionStatus;
  downloading: number;
  queued: number;
  total: number;
  error?: string;
}
/** A recently-finished job. v1 sources parse events only (see contract doc). */
export interface RecentEvent {
  kind: 'parse' | string;
  result: 'completed' | 'failed' | string;
  detail?: string;
  at: string;
}
export interface RecentSection {
  status: SectionStatus;
  events: RecentEvent[];
  error?: string;
}
export interface ActivitySummary {
  activeJobs: ActiveJobsSection;
  pending: PendingSection;
  downloads: DownloadsSection;
  recent: RecentSection;
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

export const activityService = {
  async getActivity(): Promise<ActivitySummary> {
    return fetchApi<ActivitySummary>('/activity');
  },
};

export default activityService;
