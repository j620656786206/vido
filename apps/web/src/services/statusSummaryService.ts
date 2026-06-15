/**
 * Status-summary API client (UX Redesign ux3-0-4 / D4-2) for the sidebar-footer
 * status strip. Mirrors serviceStatusService's fetch + snakeToCamel pattern (Rule 18):
 * the backend returns snake_case (`disk_headroom`, `used_bytes`, …); this boundary
 * camelCases it. Each section is fail-soft (`status: 'ok' | 'unavailable'`).
 */
import { snakeToCamel } from '../utils/caseTransform';
import type { ServiceStatus } from './serviceStatusService';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export type SectionStatus = 'ok' | 'unavailable';

export interface DiskHeadroom {
  status: SectionStatus;
  usedBytes: number;
  totalBytes: number;
  volumes: number;
  error?: string;
}
export interface ActiveScan {
  status: SectionStatus;
  active: boolean;
  percentDone: number;
  currentFile?: string;
  error?: string;
}
export interface DownloadQueue {
  status: SectionStatus;
  downloading: number;
  total: number;
  error?: string;
}
export interface ServiceHealth {
  status: SectionStatus;
  services: ServiceStatus[];
  error?: string;
}
export interface StatusSummary {
  diskHeadroom: DiskHeadroom;
  activeScan: ActiveScan;
  downloadQueue: DownloadQueue;
  serviceHealth: ServiceHealth;
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

export const statusSummaryService = {
  async getSummary(): Promise<StatusSummary> {
    return fetchApi<StatusSummary>('/status/summary');
  },
};

export default statusSummaryService;
