/**
 * Media Library CRUD API service (Story 7b-4)
 */

import { snakeToCamel, camelToSnake } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export interface MediaLibrary {
  id: string;
  name: string;
  contentType: 'movie' | 'series';
  autoDetect: boolean;
  sortOrder: number;
  createdAt: string;
  updatedAt: string;
}

export interface MediaLibraryPath {
  id: string;
  libraryId: string;
  path: string;
  status: 'unknown' | 'accessible' | 'not_found' | 'not_readable' | 'not_directory';
  lastCheckedAt: string | null;
  createdAt: string;
}

export interface MediaLibraryWithPaths extends MediaLibrary {
  paths: MediaLibraryPath[];
  mediaCount: number;
}

export interface CreateLibraryRequest {
  name: string;
  contentType: 'movie' | 'series';
  paths?: string[];
}

export interface UpdateLibraryRequest {
  name?: string;
  contentType?: 'movie' | 'series';
  sortOrder?: number;
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

export const mediaLibraryService = {
  async getAll(): Promise<{ libraries: MediaLibraryWithPaths[] }> {
    return fetchApi<{ libraries: MediaLibraryWithPaths[] }>('/libraries');
  },

  async getById(id: string): Promise<MediaLibraryWithPaths> {
    return fetchApi<MediaLibraryWithPaths>(`/libraries/${id}`);
  },

  async create(req: CreateLibraryRequest): Promise<MediaLibrary> {
    return fetchApi<MediaLibrary>('/libraries', {
      method: 'POST',
      body: JSON.stringify(camelToSnake(req)),
    });
  },

  async update(id: string, req: UpdateLibraryRequest): Promise<MediaLibrary> {
    return fetchApi<MediaLibrary>(`/libraries/${id}`, {
      method: 'PUT',
      body: JSON.stringify(camelToSnake(req)),
    });
  },

  async delete(id: string, removeMedia = false): Promise<void> {
    await fetchApi<{ deleted: boolean }>(`/libraries/${id}?remove_media=${removeMedia}`, {
      method: 'DELETE',
    });
  },

  async addPath(libraryId: string, path: string): Promise<MediaLibraryPath> {
    return fetchApi<MediaLibraryPath>(`/libraries/${libraryId}/paths`, {
      method: 'POST',
      body: JSON.stringify({ path }),
    });
  },

  async removePath(libraryId: string, pathId: string): Promise<void> {
    await fetchApi<{ deleted: boolean }>(`/libraries/${libraryId}/paths/${pathId}`, {
      method: 'DELETE',
    });
  },

  async refreshPaths(libraryId: string): Promise<{ paths: MediaLibraryPath[] }> {
    return fetchApi<{ paths: MediaLibraryPath[] }>(`/libraries/${libraryId}/paths/refresh`, {
      method: 'POST',
    });
  },
};
