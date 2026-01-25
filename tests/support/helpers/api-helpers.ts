/**
 * API Helpers for Vido E2E Tests
 *
 * Pure functions for interacting with the Vido backend API.
 * Use these for fast test setup/teardown instead of UI actions.
 *
 * Pattern: Pure Function → Fixture
 * @see tests/support/fixtures/index.ts for fixture wrapper
 */

import type { APIRequestContext, APIResponse } from '@playwright/test';

// =============================================================================
// Configuration
// =============================================================================

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080/api/v1';

// =============================================================================
// Types
// =============================================================================

export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
    suggestion?: string;
  };
}

export interface PaginatedResponse<T> {
  items: T[];
  page: number;
  pageSize: number;
  totalItems: number;
  totalPages: number;
}

export interface Movie {
  id: string;
  tmdbId?: number;
  title: string;
  originalTitle?: string;
  overview?: string;
  releaseDate: string;
  posterPath?: string;
  backdropPath?: string;
  genres?: string[];
  rating?: number;
  runtime?: number;
  status?: string;
  createdAt: string;
  updatedAt: string;
}

export interface Series {
  id: string;
  tmdbId?: number;
  title: string;
  originalTitle?: string;
  overview?: string;
  firstAirDate: string;
  lastAirDate?: string;
  posterPath?: string;
  backdropPath?: string;
  genres?: string[];
  rating?: number;
  numberOfSeasons?: number;
  numberOfEpisodes?: number;
  status?: string;
  inProduction?: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface Setting {
  key: string;
  value: string;
  type: 'string' | 'int' | 'bool';
  createdAt: string;
  updatedAt: string;
}

export interface HealthResponse {
  status: 'healthy' | 'degraded' | 'unhealthy';
  service: string;
  database?: {
    status: string;
    latency: number;
    walEnabled: boolean;
    walMode: string;
    syncMode: string;
    openConnections: number;
    idleConnections: number;
    error?: string;
  };
}

// Legacy types for backwards compatibility
export interface SearchResult {
  results: Movie[];
  total_results: number;
  page: number;
  total_pages: number;
}

// =============================================================================
// Metadata Types (Story 3-7)
// =============================================================================

export interface ManualSearchRequest {
  query: string;
  mediaType?: 'movie' | 'tv';
  year?: number;
  source?: 'all' | 'tmdb' | 'douban' | 'wikipedia';
}

export interface ManualSearchResultItem {
  id: string;
  source: 'tmdb' | 'douban' | 'wikipedia';
  title: string;
  titleZhTW?: string;
  year: number;
  mediaType: 'movie' | 'tv';
  overview?: string;
  posterUrl?: string;
  rating?: number;
}

export interface ManualSearchResponse {
  results: ManualSearchResultItem[];
  totalCount: number;
  searchedSources: string[];
}

export interface ApplyMetadataRequest {
  mediaId: string;
  mediaType: 'movie' | 'series';
  selectedItem: {
    id: string;
    source: string;
  };
  learnPattern?: boolean;
}

export interface ApplyMetadataResponse {
  success: boolean;
  mediaId: string;
  mediaType: string;
  title: string;
  source: string;
}

// =============================================================================
// Metadata Editor Types (Story 3-8)
// =============================================================================

export interface UpdateMetadataRequest {
  mediaType?: 'movie' | 'series';
  title: string;
  titleEnglish?: string;
  year: number;
  genres?: string[];
  director?: string;
  cast?: string[];
  overview?: string;
  posterUrl?: string;
}

export interface UpdateMetadataResponse {
  id: string;
  title: string;
  metadataSource: string;
  updatedAt: string;
}

export interface UploadPosterResponse {
  posterUrl: string;
  thumbnailUrl: string;
}

// =============================================================================
// API Helper Functions
// =============================================================================

export interface ApiHelpers {
  // Movies
  listMovies: (params?: { page?: number; pageSize?: number }) => Promise<ApiResponse<PaginatedResponse<Movie>>>;
  searchMovies: (query: string, params?: { page?: number; pageSize?: number }) => Promise<ApiResponse<PaginatedResponse<Movie>>>;
  getMovie: (id: string) => Promise<ApiResponse<Movie>>;
  createMovie: (data: Partial<Movie>) => Promise<ApiResponse<Movie>>;
  updateMovie: (id: string, data: Partial<Movie>) => Promise<ApiResponse<Movie>>;
  deleteMovie: (id: string) => Promise<APIResponse>;

  // Series
  listSeries: (params?: { page?: number; pageSize?: number }) => Promise<ApiResponse<PaginatedResponse<Series>>>;
  searchSeries: (query: string, params?: { page?: number; pageSize?: number }) => Promise<ApiResponse<PaginatedResponse<Series>>>;
  getSeries: (id: string) => Promise<ApiResponse<Series>>;
  createSeries: (data: Partial<Series>) => Promise<ApiResponse<Series>>;
  updateSeries: (id: string, data: Partial<Series>) => Promise<ApiResponse<Series>>;
  deleteSeries: (id: string) => Promise<APIResponse>;

  // Settings
  listSettings: () => Promise<ApiResponse<Setting[]>>;
  getSetting: (key: string) => Promise<ApiResponse<Setting>>;
  setSetting: (key: string, value: string | number | boolean, type: 'string' | 'int' | 'bool') => Promise<ApiResponse<Setting>>;
  deleteSetting: (key: string) => Promise<APIResponse>;

  // Metadata (Story 3-7)
  manualSearch: (request: ManualSearchRequest) => Promise<ApiResponse<ManualSearchResponse>>;
  applyMetadata: (request: ApplyMetadataRequest) => Promise<ApiResponse<ApplyMetadataResponse>>;

  // Metadata Editor (Story 3-8)
  updateMetadata: (id: string, request: UpdateMetadataRequest) => Promise<ApiResponse<UpdateMetadataResponse>>;
  uploadPoster: (id: string, file: Buffer, filename: string, mediaType?: string) => Promise<ApiResponse<UploadPosterResponse>>;

  // Health
  healthCheck: () => Promise<HealthResponse>;

  // Generic helpers
  get: <T>(endpoint: string) => Promise<ApiResponse<T>>;
  post: <T>(endpoint: string, data?: unknown) => Promise<ApiResponse<T>>;
  put: <T>(endpoint: string, data?: unknown) => Promise<ApiResponse<T>>;
  delete: (endpoint: string) => Promise<APIResponse>;
}

/**
 * Create API helpers bound to a Playwright request context
 */
export function apiHelpers(request: APIRequestContext): ApiHelpers {
  const get = async <T>(endpoint: string): Promise<ApiResponse<T>> => {
    const response = await request.get(`${API_BASE_URL}${endpoint}`);
    return response.json();
  };

  const post = async <T>(endpoint: string, data?: unknown): Promise<ApiResponse<T>> => {
    const response = await request.post(`${API_BASE_URL}${endpoint}`, {
      data,
      headers: {
        'Content-Type': 'application/json',
      },
    });
    return response.json();
  };

  const put = async <T>(endpoint: string, data?: unknown): Promise<ApiResponse<T>> => {
    const response = await request.put(`${API_BASE_URL}${endpoint}`, {
      data,
      headers: {
        'Content-Type': 'application/json',
      },
    });
    return response.json();
  };

  const del = async (endpoint: string): Promise<APIResponse> => {
    return request.delete(`${API_BASE_URL}${endpoint}`);
  };

  const buildQueryString = (params?: Record<string, unknown>): string => {
    if (!params) return '';
    const searchParams = new URLSearchParams();
    for (const [key, value] of Object.entries(params)) {
      if (value !== undefined && value !== null) {
        searchParams.append(key, String(value));
      }
    }
    const qs = searchParams.toString();
    return qs ? `?${qs}` : '';
  };

  return {
    // Movies
    listMovies: async (params) => {
      const qs = buildQueryString({ page: params?.page, page_size: params?.pageSize });
      return get<PaginatedResponse<Movie>>(`/movies${qs}`);
    },

    searchMovies: async (query, params) => {
      const qs = buildQueryString({ q: query, page: params?.page, page_size: params?.pageSize });
      return get<PaginatedResponse<Movie>>(`/movies/search${qs}`);
    },

    getMovie: async (id) => get<Movie>(`/movies/${id}`),

    createMovie: async (data) => post<Movie>('/movies', data),

    updateMovie: async (id, data) => put<Movie>(`/movies/${id}`, data),

    deleteMovie: async (id) => del(`/movies/${id}`),

    // Series
    listSeries: async (params) => {
      const qs = buildQueryString({ page: params?.page, page_size: params?.pageSize });
      return get<PaginatedResponse<Series>>(`/series${qs}`);
    },

    searchSeries: async (query, params) => {
      const qs = buildQueryString({ q: query, page: params?.page, page_size: params?.pageSize });
      return get<PaginatedResponse<Series>>(`/series/search${qs}`);
    },

    getSeries: async (id) => get<Series>(`/series/${id}`),

    createSeries: async (data) => post<Series>('/series', data),

    updateSeries: async (id, data) => put<Series>(`/series/${id}`, data),

    deleteSeries: async (id) => del(`/series/${id}`),

    // Settings
    listSettings: async () => get<Setting[]>('/settings'),

    getSetting: async (key) => get<Setting>(`/settings/${key}`),

    setSetting: async (key, value, type) => post<Setting>('/settings', { key, value, type }),

    deleteSetting: async (key) => del(`/settings/${key}`),

    // Metadata (Story 3-7)
    manualSearch: async (searchRequest) =>
      post<ManualSearchResponse>('/metadata/manual-search', searchRequest),

    applyMetadata: async (applyRequest) =>
      post<ApplyMetadataResponse>('/metadata/apply', applyRequest),

    // Metadata Editor (Story 3-8)
    updateMetadata: async (id, updateRequest) =>
      put<UpdateMetadataResponse>(`/media/${id}/metadata`, updateRequest),

    uploadPoster: async (id, file, filename, mediaType = 'movie') => {
      const mimeType = filename.endsWith('.png')
        ? 'image/png'
        : filename.endsWith('.webp')
          ? 'image/webp'
          : 'image/jpeg';
      const response = await request.post(
        `${API_BASE_URL}/media/${id}/poster?mediaType=${mediaType}`,
        {
          multipart: {
            file: {
              name: filename,
              mimeType,
              buffer: file,
            },
          },
        }
      );
      return response.json();
    },

    // Health
    healthCheck: async () => {
      const response = await request.get(`${API_BASE_URL.replace('/api/v1', '')}/health`);
      return response.json();
    },

    // Generic helpers
    get,
    post,
    put,
    delete: del,
  };
}
