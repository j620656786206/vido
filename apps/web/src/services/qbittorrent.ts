/**
 * qBittorrent settings service (Story 4.1)
 */

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

export interface QBConfigResponse {
  host: string;
  username: string;
  basePath: string;
  configured: boolean;
}

export interface SaveQBConfigParams {
  host: string;
  username: string;
  password: string;
  basePath?: string;
}

export interface QBVersionInfo {
  appVersion: string;
  apiVersion: string;
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

export const qbittorrentService = {
  async getConfig(): Promise<QBConfigResponse> {
    return fetchApi<QBConfigResponse>('/settings/qbittorrent');
  },

  async saveConfig(config: SaveQBConfigParams): Promise<void> {
    await fetchApi<{ message: string }>('/settings/qbittorrent', {
      method: 'PUT',
      body: JSON.stringify(config),
    });
  },

  async testConnection(): Promise<QBVersionInfo> {
    return fetchApi<QBVersionInfo>('/settings/qbittorrent/test', {
      method: 'POST',
    });
  },
};

export default qbittorrentService;
