/**
 * System logs API service (Story 6.3)
 */

import { snakeToCamel } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export interface SystemLog {
  id: number;
  level: 'ERROR' | 'WARN' | 'INFO' | 'DEBUG';
  message: string;
  source?: string;
  context?: Record<string, unknown>;
  hint?: string;
  createdAt: string;
}

export interface LogsResponse {
  logs: SystemLog[];
  total: number;
  page: number;
  perPage: number;
}

export interface LogClearResult {
  entriesRemoved: number;
  days: number;
}

export interface LogFilter {
  level?: string;
  keyword?: string;
  page?: number;
  perPage?: number;
}

interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
  };
}

async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });

  const data: ApiResponse<T> = await response.json();

  if (!response.ok || !data.success) {
    throw new Error(data.error?.message || `API request failed: ${response.status}`);
  }

  return snakeToCamel<T>(data.data);
}

export const logService = {
  async getLogs(filter: LogFilter = {}): Promise<LogsResponse> {
    const params = new URLSearchParams();
    if (filter.level) params.set('level', filter.level);
    if (filter.keyword) params.set('keyword', filter.keyword);
    if (filter.page) params.set('page', String(filter.page));
    if (filter.perPage) params.set('per_page', String(filter.perPage));

    const qs = params.toString();
    return fetchApi<LogsResponse>(`/settings/logs${qs ? `?${qs}` : ''}`);
  },

  async clearLogs(days: number): Promise<LogClearResult> {
    return fetchApi<LogClearResult>(`/settings/logs?older_than_days=${days}`, {
      method: 'DELETE',
    });
  },
};

export default logService;
