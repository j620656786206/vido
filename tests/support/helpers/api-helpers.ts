/**
 * API Helpers for Vido E2E Tests
 *
 * Pure functions for interacting with the Vido backend API.
 * Use these for fast test setup/teardown instead of UI actions.
 *
 * Pattern: Pure Function → Fixture
 * @see tests/support/fixtures/index.ts for fixture wrapper
 */

import type { APIRequestContext } from '@playwright/test';

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

export interface Movie {
  id: string;
  tmdb_id: number;
  title: string;
  original_title: string;
  overview: string;
  release_date: string;
  poster_path: string;
  backdrop_path: string;
}

export interface SearchResult {
  results: Movie[];
  total_results: number;
  page: number;
  total_pages: number;
}

// =============================================================================
// API Helper Functions
// =============================================================================

export interface ApiHelpers {
  /**
   * Search for movies by query
   */
  searchMovies: (query: string) => Promise<ApiResponse<SearchResult>>;

  /**
   * Get movie details by ID
   */
  getMovie: (id: string) => Promise<ApiResponse<Movie>>;

  /**
   * Health check endpoint
   */
  healthCheck: () => Promise<{ status: string }>;

  /**
   * Generic GET request
   */
  get: <T>(endpoint: string) => Promise<ApiResponse<T>>;

  /**
   * Generic POST request
   */
  post: <T>(endpoint: string, data?: unknown) => Promise<ApiResponse<T>>;
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

  return {
    searchMovies: async (query: string) => {
      return get<SearchResult>(`/movies/search?q=${encodeURIComponent(query)}`);
    },

    getMovie: async (id: string) => {
      return get<Movie>(`/movies/${id}`);
    },

    healthCheck: async () => {
      const response = await request.get(`${API_BASE_URL.replace('/api/v1', '')}/health`);
      return response.json();
    },

    get,
    post,
  };
}
