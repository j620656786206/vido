/**
 * qBittorrent settings service (Story 4.1)
 */

import { snakeToCamel, camelToSnake } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

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

/**
 * The one GetConfig failure the user can act on: a password is stored but
 * ENCRYPTION_KEY can no longer decrypt it. Mirrors
 * `qbittorrent.ErrCodeConfigDecryptFailed` (Rule 7).
 */
export const QB_CONFIG_DECRYPT_FAILED = 'QBITTORRENT_CONFIG_DECRYPT_FAILED';

/**
 * Carries the backend's Rule-7 code and suggestion alongside the message.
 *
 * Follows the KeySettingsApiError / ScannerApiError precedent. It extends Error,
 * so every existing `error.message` consumer keeps working untouched — the code
 * is additive, for callers that need to tell one failure from another.
 */
export class QBittorrentApiError extends Error {
  readonly code: string;
  readonly suggestion?: string;

  constructor(message: string, code: string, suggestion?: string) {
    super(message);
    this.name = 'QBittorrentApiError';
    this.code = code;
    this.suggestion = suggestion;
  }
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
    throw new QBittorrentApiError(
      errorData.error?.message || `API request failed: ${response.status}`,
      errorData.error?.code || 'INTERNAL_ERROR',
      errorData.error?.suggestion
    );
  }

  const data: ApiResponse<T> = await response.json();

  if (!data.success) {
    throw new QBittorrentApiError(
      data.error?.message || 'API request failed',
      data.error?.code || 'INTERNAL_ERROR',
      data.error?.suggestion
    );
  }

  return snakeToCamel<T>(data.data);
}

export const qbittorrentService = {
  async getConfig(): Promise<QBConfigResponse> {
    return fetchApi<QBConfigResponse>('/settings/qbittorrent');
  },

  async saveConfig(config: SaveQBConfigParams): Promise<void> {
    await fetchApi<{ message: string }>('/settings/qbittorrent', {
      method: 'PUT',
      body: JSON.stringify(camelToSnake(config)),
    });
  },

  async testConnection(config?: SaveQBConfigParams): Promise<QBVersionInfo> {
    return fetchApi<QBVersionInfo>('/settings/qbittorrent/test', {
      method: 'POST',
      body: config ? JSON.stringify(camelToSnake(config)) : undefined,
    });
  },
};

export default qbittorrentService;
