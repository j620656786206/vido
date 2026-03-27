/**
 * Cache management API service (Story 6.2)
 */

import { snakeToCamel } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export interface CacheTypeInfo {
  type: string;
  label: string;
  sizeBytes: number;
  entryCount: number;
}

export interface CacheStats {
  cacheTypes: CacheTypeInfo[];
  totalSizeBytes: number;
}

export interface CleanupResult {
  type: string;
  entriesRemoved: number;
  bytesReclaimed: number;
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

export const cacheService = {
  async getStats(): Promise<CacheStats> {
    return fetchApi<CacheStats>('/settings/cache');
  },

  async clearByType(cacheType: string): Promise<CleanupResult> {
    return fetchApi<CleanupResult>(`/settings/cache/${cacheType}`, {
      method: 'DELETE',
    });
  },

  async clearByAge(days: number): Promise<CleanupResult> {
    return fetchApi<CleanupResult>(`/settings/cache?older_than_days=${days}`, {
      method: 'DELETE',
    });
  },

  async clearAll(): Promise<CleanupResult> {
    return fetchApi<CleanupResult>('/settings/cache', {
      method: 'DELETE',
    });
  },
};

export default cacheService;
