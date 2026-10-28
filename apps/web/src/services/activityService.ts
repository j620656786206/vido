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
/**
 * Where the generating item is right now (disc-2026-10-activity-generation-progress-detail).
 * `current`/`total`: transcribing = chunks, translating = cues; absent = no count yet.
 */
export interface ActiveJobStep {
  phase: 'extracting' | 'transcribing' | 'translating' | string;
  current?: number;
  total?: number;
}

export interface ActiveJob {
  kind: 'scan' | 'subtitle_batch' | 'generation_batch' | 'transcription' | string;
  percentDone: number;
  detail?: string;
  current?: number;
  total?: number;
  /** transcription / generation_batch only; absent when unknown, and on a queued row. */
  step?: ActiveJobStep;
  /**
   * transcription only — one row per solo 生成字幕 job
   * (disc-2026-10-activity-solo-jobs-collapsed AC #7 `[@contract-v1]`). The
   * item the job generates; the row's identity.
   */
  mediaId?: string;
  /** transcription only — `true` while the job waits for the extraction slot; absent = running. */
  queued?: boolean;
  /**
   * transcription only — what the job has spent so far, raw (round with
   * decimal.js at display). `0` = nothing paid yet; absent = no meter yet,
   * never 0 (disc-2026-10-solo-live-spend AC #5 `[@contract-v1]`). Queued rows
   * carry it too. No screen reads it yet — the row design is pending.
   */
  spentUsd?: number;
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
  /**
   * `not_configured`: qBittorrent has not been set up — nothing failed
   * (disc-activity-downloads-unconfigured-copy [@contract-v1], downloads only).
   */
  status: SectionStatus | 'not_configured';
  downloading: number;
  queued: number;
  /** Count of errored torrents (bugfix-e). Distinct from `error`, the section failure MESSAGE. */
  errored: number;
  paused: number;
  total: number;
  error?: string;
}
/**
 * A recently-finished job. Two sources, merged newest first by the backend:
 * parse jobs (`kind: "parse"` — `detail` only) and subtitle-generation runs
 * (`kind: "subtitle"` — disc-2026-10-activity-subtitle-run-time-a AC #1
 * `[@contract-v2]`). A subtitle event never carries `detail` (that would be the
 * raw Go error with a server path); its copy is built on the client from the
 * fields below. Every subtitle-only field is optional: an absent `spentUsd`
 * means "not recorded", NOT $0 — the backend sends `0` for a free run.
 */
export interface RecentEvent {
  kind: 'parse' | 'subtitle' | string;
  result: 'completed' | 'failed' | string;
  detail?: string;
  at: string;
  /** `movie` | `episode` | `series`. */
  mediaType?: string;
  /** Film / series title; `""` when the item has left the library. */
  mediaTitle?: string;
  /** Only for `mediaType === 'episode'` when the episode still exists. */
  seasonNumber?: number;
  episodeNumber?: number;
  /** `models.SubtitleRunRoute*` vocabulary; absent on pre-migration-041 runs. */
  route?: string;
  /** Completed runs with at least one cue. */
  cueCount?: number;
  /** completed_at − started_at, rounded to the second. */
  durationSeconds?: number;
  /** Raw spend (not pre-rounded); absent = not recorded. */
  spentUsd?: number;
  /** One of the eleven `RunFailureReason` codes; failed runs only. */
  failureReason?: string;
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
