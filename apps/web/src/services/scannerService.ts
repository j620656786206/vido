/**
 * Scanner API client (Story 7.3)
 */

import { snakeToCamel } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export type ScheduleInterval = 'hourly' | 'daily' | 'manual';

/**
 * The most recent COMPLETED scan (GET /scanner/status `last_scan`,
 * bugfix-last-scan-never-shown AC #1 [@contract-v1]).
 */
export interface LastScan {
  completedAt: string;
  filesFound: number;
  durationMs: number;
}

/**
 * GET /scanner/status — the backend's ScanProgress fields (flat) plus
 * `last_scan`. Until bugfix-last-scan-never-shown this type named fields the
 * backend never sent (isScanning, lastScanAt…), so the page always read
 * 「尚未執行過掃描」 and never saw a scan running.
 */
export interface ScanStatus {
  isActive: boolean;
  filesFound: number;
  filesCreated: number;
  filesUpdated: number;
  filesSkipped: number;
  filesRemoved: number;
  /** 掃描器真實回報的未比對數(無法判斷集數等);舊後端沒有這個欄位 */
  filesUnmatched?: number;
  errorCount: number;
  currentFile: string;
  percentDone: number;
  startedAt?: string;
  /** null until a scan has completed (older servers: absent). */
  lastScan?: LastScan | null;
}

export interface ScanResult {
  filesFound: number;
  filesNew: number;
  errors: number;
  duration: string;
}

// The backend's field is `interval` (scanner_handler.go scheduleRequest /
// GetSchedule). It was `frequency` here from Story 7-3 on, so every save was a
// 400 and the select always read 僅手動 (bugfix-scan-schedule-field-mismatch).
export interface ScheduleConfig {
  interval: ScheduleInterval;
}

export interface ScanProgressEvent {
  filesFound: number;
  currentFile: string;
  percentDone: number;
  errorCount: number;
  estimatedTime: string;
  /** 掃描器真實回報的未比對數;SSE payload 的 files_unmatched */
  filesUnmatched?: number;
  /** Only on scan_complete: rows the scanner actually inserted / updated. */
  filesCreated?: number;
  filesUpdated?: number;
}

interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
  };
}

export class ScannerApiError extends Error {
  code: string;
  constructor(code: string, message: string) {
    super(message);
    this.code = code;
    this.name = 'ScannerApiError';
  }
}

async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });

  const data: ApiResponse<T> = await response.json();

  if (!response.ok || !data.success) {
    throw new ScannerApiError(
      data.error?.code || `HTTP_${response.status}`,
      data.error?.message || `API request failed: ${response.status}`
    );
  }

  if (data.data === undefined) {
    throw new Error('API response missing data field');
  }

  return snakeToCamel<T>(data.data);
}

export const scannerService = {
  async triggerScan(): Promise<ScanResult> {
    return fetchApi<ScanResult>('/scanner/scan', { method: 'POST' });
  },

  async getScanStatus(): Promise<ScanStatus> {
    return fetchApi<ScanStatus>('/scanner/status');
  },

  async cancelScan(): Promise<void> {
    await fetchApi('/scanner/cancel', { method: 'POST' });
  },

  async getSchedule(): Promise<ScheduleConfig> {
    return fetchApi<ScheduleConfig>('/scanner/schedule');
  },

  async updateSchedule(interval: ScheduleInterval): Promise<ScheduleConfig> {
    return fetchApi<ScheduleConfig>('/scanner/schedule', {
      method: 'PUT',
      body: JSON.stringify({ interval }),
    });
  },

  getSSEUrl(): string {
    return `${API_BASE_URL}/events`;
  },
};

export default scannerService;
