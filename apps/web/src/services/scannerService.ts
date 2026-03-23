/**
 * Scanner API client (Story 7.3)
 */

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

export type ScheduleFrequency = 'hourly' | 'daily' | 'manual';

export interface ScanStatus {
  is_scanning: boolean;
  files_found: number;
  files_processed: number;
  current_file: string;
  percent_done: number;
  error_count: number;
  estimated_time: string;
  last_scan_at: string;
  last_scan_files: number;
  last_scan_duration: string;
}

export interface ScanResult {
  files_found: number;
  files_new: number;
  errors: number;
  duration: string;
}

export interface ScheduleConfig {
  frequency: ScheduleFrequency;
}

export interface ScanProgressEvent {
  files_found: number;
  current_file: string;
  percent_done: number;
  error_count: number;
  estimated_time: string;
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

  return data.data;
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

  async updateSchedule(frequency: ScheduleFrequency): Promise<ScheduleConfig> {
    return fetchApi<ScheduleConfig>('/scanner/schedule', {
      method: 'PUT',
      body: JSON.stringify({ frequency }),
    });
  },

  getSSEUrl(): string {
    return `${API_BASE_URL}/events`;
  },
};

export default scannerService;
