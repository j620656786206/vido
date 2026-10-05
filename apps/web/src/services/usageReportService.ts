/**
 * Opt-in anonymous usage report client (infra-optin-usage-report-b1).
 *
 * Mirrors GET/PUT /api/v1/settings/usage-report (a2): whether this build has
 * a receiver at all, whether the user turned the report on, and the last
 * report that was actually sent — `lastPayload` is the exact body, verbatim.
 * snakeToCamel only renames keys, so the payload string is never touched.
 */

import type { ApiResponse } from '../types/tmdb';
import { snakeToCamel } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

/** The page that lists what the report sends and what it never sends. */
export const USAGE_REPORT_DOCS_URL =
  'https://github.com/j620656786206/vido/blob/main/docs/usage-report.zh-TW.md';

export interface UsageReportStatus {
  /** false = this build has no receiver configured; the switch cannot turn on. */
  available: boolean;
  enabled: boolean;
  /** ISO time of the last SUCCESSFUL send; null = never sent. */
  lastSentAt: string | null;
  /** The exact body of that send, verbatim; null = never sent. */
  lastPayload: string | null;
}

export class UsageReportApiError extends Error {
  readonly code: string;

  constructor(message: string, code: string) {
    super(message);
    this.name = 'UsageReportApiError';
    this.code = code;
  }
}

async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, options);
  const data = (await response.json().catch(() => null)) as ApiResponse<T> | null;

  if (!response.ok || !data?.success) {
    throw new UsageReportApiError(
      data?.error?.message || `API request failed: ${response.status}`,
      data?.error?.code || 'INTERNAL_ERROR'
    );
  }

  return snakeToCamel<T>(data.data as T);
}

export const usageReportService = {
  get(): Promise<UsageReportStatus> {
    return fetchApi<UsageReportStatus>('/settings/usage-report');
  },

  setEnabled(enabled: boolean): Promise<UsageReportStatus> {
    return fetchApi<UsageReportStatus>('/settings/usage-report', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enabled }),
    });
  },
};
