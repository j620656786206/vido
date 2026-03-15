import type { ApiResponse } from '../types/tmdb';
import type { LibraryItem, LibraryListResponse, LibraryListParams } from '../types/library';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

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

  return data.data as T;
}

export const libraryService = {
  async listLibrary(params: LibraryListParams = {}): Promise<LibraryListResponse> {
    const searchParams = new URLSearchParams();
    if (params.page) searchParams.set('page', String(params.page));
    if (params.pageSize) searchParams.set('page_size', String(params.pageSize));
    if (params.type && params.type !== 'all') searchParams.set('type', params.type);
    if (params.sortBy) searchParams.set('sort_by', params.sortBy);
    if (params.sortOrder) searchParams.set('sort_order', params.sortOrder);

    const qs = searchParams.toString();
    return fetchApi<LibraryListResponse>(`/library${qs ? `?${qs}` : ''}`);
  },

  async getRecentlyAdded(limit: number = 20): Promise<LibraryItem[]> {
    const response = await fetchApi<LibraryListResponse>(`/library/recent?limit=${limit}`);
    return response.items;
  },

  async deleteMovie(id: string): Promise<void> {
    const response = await fetch(`${API_BASE_URL}/library/movies/${id}`, {
      method: 'DELETE',
    });
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error?.message || 'Failed to delete movie');
    }
  },

  async deleteSeries(id: string): Promise<void> {
    const response = await fetch(`${API_BASE_URL}/library/series/${id}`, {
      method: 'DELETE',
    });
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error?.message || 'Failed to delete series');
    }
  },

  async reparseMovie(id: string): Promise<{ id: string; status: string }> {
    return fetchApi<{ id: string; status: string }>(`/library/movies/${id}/reparse`, {
      method: 'POST',
    });
  },

  async reparseSeries(id: string): Promise<{ id: string; status: string }> {
    return fetchApi<{ id: string; status: string }>(`/library/series/${id}/reparse`, {
      method: 'POST',
    });
  },

  async exportMovie(id: string): Promise<unknown> {
    return fetchApi(`/library/movies/${id}/export`, { method: 'POST' });
  },

  async exportSeries(id: string): Promise<unknown> {
    return fetchApi(`/library/series/${id}/export`, { method: 'POST' });
  },
};
