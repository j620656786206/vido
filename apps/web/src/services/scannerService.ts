/**
 * Scanner API client (Story 7.3)
 */

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export type ScheduleFrequency = 'hourly' | 'daily' | 'manual';

export interface ScanStatus {
  isScanning: boolean;
  filesFound: number;
  filesProcessed: number;
  currentFile: string;
  percentDone: number;
  errorCount: number;
  estimatedTime: string;
  lastScanAt: string;
  lastScanFiles: number;
  lastScanDuration: string;
}

export interface ScanResult {
  filesFound: number;
  filesNew: number;
  errors: number;
  duration: string;
}

export interface ScheduleConfig {
  frequency: ScheduleFrequency;
}

export interface ScanProgressEvent {
  filesFound: number;
  currentFile: string;
  percentDone: number;
  errorCount: number;
  estimatedTime: string;
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
