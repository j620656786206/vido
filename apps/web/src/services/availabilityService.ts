import type { ApiResponse } from '../types/tmdb';
import { camelToSnake, snakeToCamel } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

/**
 * Response shape returned by POST /api/v1/media/check-owned.
 * Transformed from snake_case by snakeToCamel on the way in.
 */
export interface CheckOwnedResponse {
  ownedIds: number[];
}

async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, options);

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error?.message || `API request failed: ${response.status}`);
  }

  const data: ApiResponse<T> = await response.json();
  if (!data.success) {
    throw new Error(data.error?.message || 'API request failed');
  }

  return snakeToCamel<T>(data.data);
}

/**
 * Availability lookups used by homepage poster cards (Story 10-4, P2-006).
 * The service is standalone rather than bundled into libraryService so the
 * homepage feature can evolve independently (e.g. batching strategy changes).
 */
export const availabilityService = {
  /**
   * Batch-check which TMDb IDs from the current view are already in the local
   * library. Returns an empty array when the input is empty — avoids an
   * unnecessary network hit for empty homepage sections.
   *
   * Rule 18: POST body is camelToSnake-transformed before stringify.
   */
  async checkOwned(tmdbIds: number[]): Promise<number[]> {
    if (!tmdbIds || tmdbIds.length === 0) return [];

    const body = camelToSnake({ tmdbIds });
    const res = await fetchApi<CheckOwnedResponse>('/media/check-owned', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    return res.ownedIds ?? [];
  },
};

export default availabilityService;
