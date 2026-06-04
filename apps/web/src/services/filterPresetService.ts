/**
 * Saved filter preset CRUD API service — Story 11.4 (P2-015).
 *
 * `filters` is an opaque JSON string in the URL search-param shape
 * (e.g. {"genre":"28","year_gte":"2024","region":"KR"}). It is intentionally a
 * string, not a nested object: the API boundary case-transforms object keys
 * (Rule 18), which would mangle the snake_case URL-param keys inside a nested
 * object. The frontend owns serialization via serializeFilters() + JSON.stringify.
 */

import { snakeToCamel, camelToSnake } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export interface FilterPreset {
  id: string;
  name: string;
  /** JSON string in URL search-param shape — parse with JSON.parse before use. */
  filters: string;
  sortOrder: number;
  createdAt: string;
}

export interface CreateFilterPresetRequest {
  name: string;
  filters: string;
}

interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
    suggestion?: string;
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

export const filterPresetService = {
  async getAll(): Promise<{ presets: FilterPreset[] }> {
    return fetchApi<{ presets: FilterPreset[] }>('/filter-presets');
  },

  async create(req: CreateFilterPresetRequest): Promise<FilterPreset> {
    return fetchApi<FilterPreset>('/filter-presets', {
      method: 'POST',
      body: JSON.stringify(camelToSnake(req)),
    });
  },

  async remove(id: string): Promise<void> {
    await fetchApi<{ deleted: boolean }>(`/filter-presets/${id}`, { method: 'DELETE' });
  },
};
