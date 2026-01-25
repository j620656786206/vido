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

async function fetchApi<T>(
  endpoint: string,
  options?: RequestInit
): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    headers: {
      'Content-Type': 'application/json',
    },
    ...options,
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(
      errorData.error?.message || `API request failed: ${response.status}`
    );
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
};

export default metadataService;
