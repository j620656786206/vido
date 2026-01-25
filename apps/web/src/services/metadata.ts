/**
 * Metadata service for manual search and apply operations (Story 3.7)
 */

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

// Types for manual search
export interface ManualSearchParams {
  query: string;
  mediaType: 'movie' | 'tv';
  year?: number;
  source: 'tmdb' | 'douban' | 'wikipedia' | 'all';
}

export interface ManualSearchResultItem {
  id: string;
  source: 'tmdb' | 'douban' | 'wikipedia';
  title: string;
  titleZhTW?: string;
  year?: number;
  mediaType: 'movie' | 'tv';
  overview?: string;
  posterUrl?: string;
  rating?: number;
  confidence?: number;
}

export interface ManualSearchResponse {
  results: ManualSearchResultItem[];
  totalCount: number;
  searchedSources: string[];
}

// Types for apply metadata
export interface ApplyMetadataParams {
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

// Types for update metadata (Story 3.8 - AC2)
export interface UpdateMetadataParams {
  id: string;
  mediaType: 'movie' | 'series';
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

// Types for poster upload (Story 3.8 - AC3)
export interface UploadPosterResponse {
  posterUrl: string;
  thumbnailUrl: string;
}

// API response wrapper
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
    headers: {
      'Content-Type': 'application/json',
    },
    ...options,
  });

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

export const metadataService = {
  /**
   * Perform manual search across selected sources (AC1, AC4)
   */
  async manualSearch(params: ManualSearchParams): Promise<ManualSearchResponse> {
    return fetchApi<ManualSearchResponse>('/metadata/manual-search', {
      method: 'POST',
      body: JSON.stringify(params),
    });
  },

  /**
   * Apply selected metadata to a media item (AC3)
   */
  async applyMetadata(params: ApplyMetadataParams): Promise<ApplyMetadataResponse> {
    return fetchApi<ApplyMetadataResponse>('/metadata/apply', {
      method: 'POST',
      body: JSON.stringify(params),
    });
  },

  /**
   * Update metadata for a media item (Story 3.8 - AC2)
   */
  async updateMetadata(params: UpdateMetadataParams): Promise<UpdateMetadataResponse> {
    return fetchApi<UpdateMetadataResponse>(`/media/${params.id}/metadata`, {
      method: 'PUT',
      body: JSON.stringify(params),
    });
  },

  /**
   * Upload a poster image for a media item (Story 3.8 - AC3)
   */
  async uploadPoster(
    mediaId: string,
    mediaType: 'movie' | 'series',
    file: File
  ): Promise<UploadPosterResponse> {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('mediaType', mediaType);

    const response = await fetch(`${API_BASE_URL}/media/${mediaId}/poster`, {
      method: 'POST',
      body: formData,
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error?.message || `Upload failed: ${response.status}`);
    }

    const data = await response.json();
    if (!data.success) {
      throw new Error(data.error?.message || 'Upload failed');
    }

    return data.data as UploadPosterResponse;
  },
};

export default metadataService;
