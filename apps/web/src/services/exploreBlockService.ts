/**
 * Explore block CRUD API service — Story 10.3 (P2-002).
 */

import { snakeToCamel, camelToSnake } from '../utils/caseTransform';
import type { Movie, TVShow } from '../types/tmdb';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export type ExploreBlockContentType = 'movie' | 'tv';

export interface ExploreBlock {
  id: string;
  name: string;
  contentType: ExploreBlockContentType;
  genreIds: string;
  language: string;
  region: string;
  sortBy: string;
  maxItems: number;
  sortOrder: number;
  createdAt: string;
  updatedAt: string;
}

export interface CreateExploreBlockRequest {
  name: string;
  contentType: ExploreBlockContentType;
  genreIds?: string;
  language?: string;
  region?: string;
  sortBy?: string;
  maxItems?: number;
}

export interface UpdateExploreBlockRequest {
  name?: string;
  contentType?: ExploreBlockContentType;
  genreIds?: string;
  language?: string;
  region?: string;
  sortBy?: string;
  maxItems?: number;
}

export interface ExploreBlockContent {
  blockId: string;
  contentType: ExploreBlockContentType;
  movies?: Movie[];
  tvShows?: TVShow[];
  totalItems: number;
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

export const exploreBlockService = {
  async getAll(): Promise<{ blocks: ExploreBlock[] }> {
    return fetchApi<{ blocks: ExploreBlock[] }>('/explore-blocks');
  },

  async getById(id: string): Promise<ExploreBlock> {
    return fetchApi<ExploreBlock>(`/explore-blocks/${id}`);
  },

  async create(req: CreateExploreBlockRequest): Promise<ExploreBlock> {
    return fetchApi<ExploreBlock>('/explore-blocks', {
      method: 'POST',
      body: JSON.stringify(camelToSnake(req)),
    });
  },

  async update(id: string, req: UpdateExploreBlockRequest): Promise<ExploreBlock> {
    return fetchApi<ExploreBlock>(`/explore-blocks/${id}`, {
      method: 'PUT',
      body: JSON.stringify(camelToSnake(req)),
    });
  },

  async remove(id: string): Promise<void> {
    await fetchApi<{ deleted: boolean }>(`/explore-blocks/${id}`, { method: 'DELETE' });
  },

  async reorder(orderedIds: string[]): Promise<{ blocks: ExploreBlock[] }> {
    return fetchApi<{ blocks: ExploreBlock[] }>('/explore-blocks/reorder', {
      method: 'PUT',
      body: JSON.stringify(camelToSnake({ orderedIds })),
    });
  },

  async getContent(id: string): Promise<ExploreBlockContent> {
    return fetchApi<ExploreBlockContent>(`/explore-blocks/${id}/content`);
  },
};
