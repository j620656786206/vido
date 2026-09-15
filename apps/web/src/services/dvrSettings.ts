/**
 * Sonarr / Radarr connection settings (story 13-6).
 *
 * Consumes the 13-4a [@contract-v1] endpoints, one set per plugin:
 * GET/PUT /settings/{plugin}, POST /settings/{plugin}/test,
 * GET /settings/{plugin}/quality-profiles and /root-folders.
 * The API key is write-only — GET only says whether one is stored (`has_api_key`).
 */

import { snakeToCamel, camelToSnake } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export type DvrPlugin = 'sonarr' | 'radarr';

/** plugins.HealthStatus* — `unconfigured` also covers "saved but switched off". */
export type DvrHealthStatus = 'healthy' | 'unhealthy' | 'unconfigured';

export interface DvrHealth {
  status: DvrHealthStatus;
  lastCheckedAt: string | null;
  message: string;
}

export interface DvrConfig {
  url: string;
  enabled: boolean;
  qualityProfileId: number;
  rootFolderPath: string;
  hasApiKey: boolean;
  health: DvrHealth;
}

export interface SaveDvrConfigParams {
  url: string;
  /** Empty keeps the stored key (the server falls back to it). */
  apiKey: string;
  enabled: boolean;
  qualityProfileId: number;
  rootFolderPath: string;
}

export interface TestDvrConnectionParams {
  url: string;
  /** Empty tests with the stored key. */
  apiKey: string;
}

export interface DvrQualityProfile {
  id: number;
  name: string;
}

export interface DvrRootFolder {
  id: number;
  path: string;
}

/** plugins/errors.go (Rule 7). */
export const DVR_NOT_CONFIGURED = 'DVR_NOT_CONFIGURED';
export const DVR_CONNECTION_FAILED = 'DVR_CONNECTION_FAILED';
export const DVR_AUTH_FAILED = 'DVR_AUTH_FAILED';
export const DVR_TIMEOUT = 'DVR_TIMEOUT';
export const DVR_TEST_FAILED = 'DVR_TEST_FAILED';

/** Carries the backend's Rule-7 code alongside the message (QBittorrentApiError precedent). */
export class DvrSettingsApiError extends Error {
  readonly code: string;
  readonly suggestion?: string;
  /** The innermost reason when `code` wraps another failure (DVR_TEST_FAILED around DVR_AUTH_FAILED). */
  readonly causeCode?: string;

  constructor(message: string, code: string, suggestion?: string, causeCode?: string) {
    super(message);
    this.name = 'DvrSettingsApiError';
    this.code = code;
    this.suggestion = suggestion;
    this.causeCode = causeCode;
  }
}

interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: { code: string; message: string; suggestion?: string; cause_code?: string };
}

async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new DvrSettingsApiError(
      errorData.error?.message || `API request failed: ${response.status}`,
      errorData.error?.code || 'INTERNAL_ERROR',
      errorData.error?.suggestion,
      errorData.error?.cause_code
    );
  }

  const data: ApiResponse<T> = await response.json();
  if (!data.success) {
    throw new DvrSettingsApiError(
      data.error?.message || 'API request failed',
      data.error?.code || 'INTERNAL_ERROR',
      data.error?.suggestion,
      data.error?.cause_code
    );
  }
  return snakeToCamel<T>(data.data);
}

export const dvrSettingsService = {
  getConfig(plugin: DvrPlugin): Promise<DvrConfig> {
    return fetchApi<DvrConfig>(`/settings/${plugin}`);
  },

  async saveConfig(plugin: DvrPlugin, params: SaveDvrConfigParams): Promise<void> {
    await fetchApi<{ message: string }>(`/settings/${plugin}`, {
      method: 'PUT',
      body: JSON.stringify(camelToSnake(params)),
    });
  },

  async testConnection(plugin: DvrPlugin, params: TestDvrConnectionParams): Promise<void> {
    await fetchApi<{ message: string }>(`/settings/${plugin}/test`, {
      method: 'POST',
      body: JSON.stringify(camelToSnake(params)),
    });
  },

  async getQualityProfiles(plugin: DvrPlugin): Promise<DvrQualityProfile[]> {
    const data = await fetchApi<{ qualityProfiles: DvrQualityProfile[] | null }>(
      `/settings/${plugin}/quality-profiles`
    );
    return data.qualityProfiles ?? [];
  },

  async getRootFolders(plugin: DvrPlugin): Promise<DvrRootFolder[]> {
    const data = await fetchApi<{ rootFolders: DvrRootFolder[] | null }>(
      `/settings/${plugin}/root-folders`
    );
    return data.rootFolders ?? [];
  },
};
