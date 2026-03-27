/**
 * Media library service (Story 4.3)
 */

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export interface RecentMedia {
  id: string;
  title: string;
  year?: number;
  posterUrl?: string;
  mediaType: 'movie' | 'tv';
  justAdded: boolean;
  addedAt: string;
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

async function fetchApi<T>(endpoint: string): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    headers: {
      'Content-Type': 'application/json',
    },
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

export const mediaService = {
  async getRecentMedia(limit: number = 10): Promise<RecentMedia[]> {
    return fetchApi<RecentMedia[]>(`/media/recent?limit=${limit}`);
  },
};

export default mediaService;
