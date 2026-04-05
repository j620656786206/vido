import type { ApiResponse } from '../types/tmdb';
import { snakeToCamel } from '../utils/caseTransform';
import type {
  LibraryItem,
  LibraryListResponse,
  LibraryListParams,
  LibrarySearchResponse,
  LibraryStats,
  MediaStats,
  LibraryMovie,
  LibrarySeries,
  VideosResponse,
  BatchResult,
} from '../types/library';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

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

export const libraryService = {
  async listLibrary(params: LibraryListParams = {}): Promise<LibraryListResponse> {
    const searchParams = new URLSearchParams();
    if (params.page) searchParams.set('page', String(params.page));
    if (params.pageSize) searchParams.set('page_size', String(params.pageSize));
    if (params.type && params.type !== 'all') searchParams.set('type', params.type);
    if (params.sortBy) searchParams.set('sort_by', params.sortBy);
    if (params.sortOrder) searchParams.set('sort_order', params.sortOrder);
    if (params.genres) searchParams.set('genres', params.genres);
    if (params.yearMin) searchParams.set('year_min', String(params.yearMin));
    if (params.yearMax) searchParams.set('year_max', String(params.yearMax));
    if (params.unmatched) searchParams.set('unmatched', 'true');

    const qs = searchParams.toString();
    return fetchApi<LibraryListResponse>(`/library${qs ? `?${qs}` : ''}`);
  },

  async getRecentlyAdded(limit: number = 20): Promise<LibraryItem[]> {
    const response = await fetchApi<LibraryListResponse>(`/library/recent?limit=${limit}`);
    return response.items;
  },

  async searchLibrary(
    query: string,
    params: LibraryListParams = {}
  ): Promise<LibrarySearchResponse> {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    if (params.page) searchParams.set('page', String(params.page));
    if (params.pageSize) searchParams.set('page_size', String(params.pageSize));
    if (params.type && params.type !== 'all') searchParams.set('type', params.type);
    if (params.sortBy) searchParams.set('sort_by', params.sortBy);
    if (params.sortOrder) searchParams.set('sort_order', params.sortOrder);

    return fetchApi<LibrarySearchResponse>(`/library/search?${searchParams.toString()}`);
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

  async getGenres(): Promise<string[]> {
    return fetchApi<string[]>('/library/genres');
  },

  async getStats(): Promise<LibraryStats> {
    return fetchApi<LibraryStats>('/library/stats');
  },

  async getMovieStats(): Promise<MediaStats> {
    return fetchApi<MediaStats>('/movies/stats');
  },

  async getSeriesStats(): Promise<MediaStats> {
    return fetchApi<MediaStats>('/series/stats');
  },

  async getMovieById(id: string): Promise<LibraryMovie> {
    return fetchApi<LibraryMovie>(`/movies/${id}`);
  },

  async getSeriesById(id: string): Promise<LibrarySeries> {
    return fetchApi<LibrarySeries>(`/series/${id}`);
  },

  async getMovieVideos(id: string): Promise<VideosResponse> {
    return fetchApi<VideosResponse>(`/library/movies/${id}/videos`);
  },

  async getSeriesVideos(id: string): Promise<VideosResponse> {
    return fetchApi<VideosResponse>(`/library/series/${id}/videos`);
  },

  async batchDelete(ids: string[], type: 'movie' | 'series'): Promise<BatchResult> {
    return fetchApi<BatchResult>('/library/batch', {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ids, type }),
    });
  },

  async batchReparse(ids: string[], type: 'movie' | 'series'): Promise<BatchResult> {
    return fetchApi<BatchResult>('/library/batch/reparse', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ids, type }),
    });
  },

  async batchExport(ids: string[], type: 'movie' | 'series'): Promise<unknown[]> {
    return fetchApi<unknown[]>(`/library/batch/export?type=${type}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ids, format: 'json' }),
    });
  },
};
