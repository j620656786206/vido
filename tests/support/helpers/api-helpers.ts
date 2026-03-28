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
    wal_enabled: boolean;
    wal_mode: string;
    sync_mode: string;
    open_connections: number;
    idle_connections: number;
    error?: string;
  };
}

// =============================================================================
// Degradation Types (Story 3-12)
// =============================================================================

export type DegradationLevel = 'normal' | 'partial' | 'minimal' | 'offline';

export interface ServiceHealth {
  name: string;
  display_name: string;
  status: 'healthy' | 'degraded' | 'down';
  last_check: string;
  last_success: string;
  error_count: number;
  message?: string;
}

export interface ServicesHealth {
  tmdb: ServiceHealth;
  douban: ServiceHealth;
  wikipedia: ServiceHealth;
  ai: ServiceHealth;
}

export interface HealthStatusResponse {
  degradation_level: DegradationLevel;
  services: ServicesHealth;
  message: string;
}

// =============================================================================
// Download Types (Story 4-2)
// =============================================================================

export interface DownloadItem {
  hash: string;
  name: string;
  size: number;
  progress: number;
  downloadSpeed: number;
  uploadSpeed: number;
  eta: number;
  status: string;
  addedOn: string;
  completedOn?: string;
  seeds: number;
  peers: number;
  downloaded: number;
  uploaded: number;
  ratio: number;
  savePath: string;
}

export interface DownloadDetailsItem extends DownloadItem {
  pieceSize: number;
  comment?: string;
  createdBy?: string;
  creationDate: string;
  totalWasted: number;
  timeElapsed: number;
  seedingTime: number;
  avgDownSpeed: number;
  avgUpSpeed: number;
}

// =============================================================================
// qBittorrent Types (Story 4-1)
// =============================================================================

export interface QBConfigResponse {
  host: string;
  username: string;
  basePath: string;
  configured: boolean;
}

export interface QBVersionInfo {
  appVersion: string;
  apiVersion: string;
}

export interface SaveQBConfigRequest {
  host: string;
  username: string;
  password: string;
  basePath?: string;
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
  media_type?: 'movie' | 'tv';
  year?: number;
  source?: 'all' | 'tmdb' | 'douban' | 'wikipedia';
}

export interface ManualSearchResultItem {
  id: string;
  source: 'tmdb' | 'douban' | 'wikipedia';
  title: string;
  title_zh_tw?: string;
  year: number;
  media_type: 'movie' | 'tv';
  overview?: string;
  poster_url?: string;
  rating?: number;
}

export interface ManualSearchResponse {
  results: ManualSearchResultItem[];
  total_count: number;
  searched_sources: string[];
}

export interface ApplyMetadataRequest {
  media_id: string;
  media_type: 'movie' | 'series';
  selected_item: {
    id: string;
    source: string;
  };
  learn_pattern?: boolean;
}

export interface ApplyMetadataResponse {
  success: boolean;
  media_id: string;
  media_type: string;
  title: string;
  source: string;
}

// =============================================================================
// Metadata Editor Types (Story 3-8)
// =============================================================================

export interface UpdateMetadataRequest {
  media_type?: 'movie' | 'series';
  title: string;
  title_english?: string;
  year: number;
  genres?: string[];
  director?: string;
  cast?: string[];
  overview?: string;
  poster_url?: string;
}

export interface UpdateMetadataResponse {
  id: string;
  title: string;
  metadata_source: string;
  updated_at: string;
}

export interface UploadPosterResponse {
  poster_url: string;
  thumbnail_url: string;
}

// =============================================================================
// Learning Types (Story 3-9)
// =============================================================================

export interface CreatePatternRequest {
  filename: string;
  metadata_id: string;
  metadata_type: 'movie' | 'series';
  tmdb_id?: number;
}

export interface LearnedPattern {
  id: string;
  pattern: string;
  pattern_type: string;
  pattern_regex?: string;
  fansub_group?: string;
  title_pattern?: string;
  metadata_type: string;
  metadata_id: string;
  tmdb_id?: number;
  confidence: number;
  use_count: number;
  created_at: string;
  last_used_at?: string;
}

export interface PatternStats {
  total_patterns: number;
  total_applied: number;
  most_used_pattern?: string;
  most_used_count?: number;
}

export interface PatternListResponse {
  patterns: LearnedPattern[];
  total_count: number;
  stats?: PatternStats;
}

// =============================================================================
// API Helper Functions
// =============================================================================

export interface ApiHelpers {
  // Movies
  listMovies: (params?: {
    page?: number;
    pageSize?: number;
  }) => Promise<ApiResponse<PaginatedResponse<Movie>>>;
  searchMovies: (
    query: string,
    params?: { page?: number; pageSize?: number }
  ) => Promise<ApiResponse<PaginatedResponse<Movie>>>;
  getMovie: (id: string) => Promise<ApiResponse<Movie>>;
  createMovie: (data: Partial<Movie>) => Promise<ApiResponse<Movie>>;
  updateMovie: (id: string, data: Partial<Movie>) => Promise<ApiResponse<Movie>>;
  deleteMovie: (id: string) => Promise<APIResponse>;

  // Series
  listSeries: (params?: {
    page?: number;
    pageSize?: number;
  }) => Promise<ApiResponse<PaginatedResponse<Series>>>;
  searchSeries: (
    query: string,
    params?: { page?: number; pageSize?: number }
  ) => Promise<ApiResponse<PaginatedResponse<Series>>>;
  getSeries: (id: string) => Promise<ApiResponse<Series>>;
  createSeries: (data: Partial<Series>) => Promise<ApiResponse<Series>>;
  updateSeries: (id: string, data: Partial<Series>) => Promise<ApiResponse<Series>>;
  deleteSeries: (id: string) => Promise<APIResponse>;

  // Settings
  listSettings: () => Promise<ApiResponse<Setting[]>>;
  getSetting: (key: string) => Promise<ApiResponse<Setting>>;
  setSetting: (
    key: string,
    value: string | number | boolean,
    type: 'string' | 'int' | 'bool'
  ) => Promise<ApiResponse<Setting>>;
  deleteSetting: (key: string) => Promise<APIResponse>;

  // Metadata (Story 3-7)
  manualSearch: (request: ManualSearchRequest) => Promise<ApiResponse<ManualSearchResponse>>;
  applyMetadata: (request: ApplyMetadataRequest) => Promise<ApiResponse<ApplyMetadataResponse>>;

  // Metadata Editor (Story 3-8)
  updateMetadata: (
    id: string,
    request: UpdateMetadataRequest
  ) => Promise<ApiResponse<UpdateMetadataResponse>>;
  uploadPoster: (
    id: string,
    file: Buffer,
    filename: string,
    mediaType?: string
  ) => Promise<ApiResponse<UploadPosterResponse>>;

  // Learning (Story 3-9)
  createPattern: (request: CreatePatternRequest) => Promise<ApiResponse<LearnedPattern>>;
  listPatterns: () => Promise<ApiResponse<PatternListResponse>>;
  deletePattern: (id: string) => Promise<APIResponse>;
  getPatternStats: () => Promise<ApiResponse<PatternStats>>;

  // Downloads (Story 4-2)
  listDownloads: (params?: {
    sort?: string;
    order?: string;
  }) => Promise<ApiResponse<DownloadItem[]>>;
  getDownloadDetails: (hash: string) => Promise<ApiResponse<DownloadDetailsItem>>;

  // qBittorrent (Story 4-1)
  getQBConfig: () => Promise<ApiResponse<QBConfigResponse>>;
  saveQBConfig: (config: SaveQBConfigRequest) => Promise<ApiResponse<{ message: string }>>;
  testQBConnection: () => Promise<ApiResponse<QBVersionInfo>>;

  // Health
  healthCheck: () => Promise<HealthResponse>;
  servicesHealth: () => Promise<ApiResponse<HealthStatusResponse>>;

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

    // Learning (Story 3-9)
    createPattern: async (patternRequest) =>
      post<LearnedPattern>('/learning/patterns', patternRequest),

    listPatterns: async () => get<PatternListResponse>('/learning/patterns'),

    deletePattern: async (id) => del(`/learning/patterns/${id}`),

    getPatternStats: async () => get<PatternStats>('/learning/stats'),

    // Downloads (Story 4-2)
    listDownloads: async (params) => {
      const qs = buildQueryString({ sort: params?.sort, order: params?.order });
      return get<DownloadItem[]>(`/downloads${qs}`);
    },

    getDownloadDetails: async (hash) => get<DownloadDetailsItem>(`/downloads/${hash}`),

    // qBittorrent (Story 4-1)
    getQBConfig: async () => get<QBConfigResponse>('/settings/qbittorrent'),

    saveQBConfig: async (config) => put<{ message: string }>('/settings/qbittorrent', config),

    testQBConnection: async () => post<QBVersionInfo>('/settings/qbittorrent/test'),

    // Health
    healthCheck: async () => {
      const response = await request.get(`${API_BASE_URL.replace('/api/v1', '')}/health`);
      return response.json();
    },

    servicesHealth: async () => get<HealthStatusResponse>('/health/services'),

    // Generic helpers
    get,
    post,
    put,
    delete: del,
  };
}
