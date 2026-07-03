/**
 * Download monitoring service (Story 4.2)
 */

import { snakeToCamel } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export type TorrentStatus =
  | 'downloading'
  | 'paused'
  | 'seeding'
  | 'completed'
  | 'stalled'
  | 'error'
  | 'queued'
  | 'checking';

export type SortField = 'added_on' | 'name' | 'progress' | 'status';
export type SortOrder = 'asc' | 'desc';

export type ParseJobStatus = 'pending' | 'processing' | 'completed' | 'failed' | 'skipped';

export interface DownloadParseStatus {
  status: ParseJobStatus;
  errorMessage?: string;
  mediaId?: string;
}

export interface Download {
  hash: string;
  name: string;
  size: number;
  progress: number;
  downloadSpeed: number;
  uploadSpeed: number;
  eta: number;
  status: TorrentStatus;
  addedOn: string;
  completedOn?: string;
  seeds: number;
  peers: number;
  downloaded: number;
  uploaded: number;
  ratio: number;
  savePath: string;
  parseStatus?: DownloadParseStatus;
}

export interface DownloadDetails extends Download {
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

export type FilterStatus = 'all' | 'downloading' | 'paused' | 'completed' | 'seeding' | 'error';

export interface DownloadCounts {
  all: number;
  downloading: number;
  paused: number;
  completed: number;
  seeding: number;
  error: number;
}

export interface GetDownloadsParams {
  filter?: FilterStatus;
  sort?: SortField;
  order?: SortOrder;
  page?: number;
  pageSize?: number;
}

export interface PaginatedDownloads {
  items: Download[];
  page: number;
  pageSize: number;
  totalItems: number;
  totalPages: number;
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

  return snakeToCamel<T>(data.data);
}

/**
 * Mutation helper for the download action endpoints (ux3-4-2 [@contract-v1]). `fetchApi` above is
 * GET-only; these actions are `POST /downloads/:hash/pause|resume` and `DELETE /downloads/:hash`,
 * each returning `{ success: true }` with a null data body (nothing to unwrap). Non-2xx / `success:false`
 * throws with the backend's zh-TW message (same shape as fetchApi) so `useDownloadActions` can surface it.
 */
async function mutateApi(endpoint: string, method: 'POST' | 'DELETE'): Promise<void> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    method,
    headers: { 'Content-Type': 'application/json' },
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error?.message || `API request failed: ${response.status}`);
  }

  const data: ApiResponse<unknown> = await response.json();
  if (!data.success) {
    throw new Error(data.error?.message || 'API request failed');
  }
}

export const downloadService = {
  async getDownloads(params?: GetDownloadsParams): Promise<PaginatedDownloads> {
    const filter = params?.filter || 'all';
    const sort = params?.sort || 'added_on';
    const order = params?.order || 'desc';
    const page = String(params?.page || 1);
    const pageSize = String(params?.pageSize || 100);
    const searchParams = new URLSearchParams({ filter, sort, order, page, pageSize });
    return fetchApi<PaginatedDownloads>(`/downloads?${searchParams.toString()}`);
  },

  async getDownloadDetails(hash: string): Promise<DownloadDetails> {
    return fetchApi<DownloadDetails>(`/downloads/${encodeURIComponent(hash)}`);
  },

  async getDownloadCounts(): Promise<DownloadCounts> {
    return fetchApi<DownloadCounts>(`/downloads/counts`);
  },

  // Actions (ux3-4-2 [@contract-v1]). Idempotent on qBittorrent's side — a 200 even for an
  // already-paused/unknown hash, so no not-found branch is needed here.
  async pauseDownload(hash: string): Promise<void> {
    return mutateApi(`/downloads/${encodeURIComponent(hash)}/pause`, 'POST');
  },

  async resumeDownload(hash: string): Promise<void> {
    return mutateApi(`/downloads/${encodeURIComponent(hash)}/resume`, 'POST');
  },

  async removeDownload(hash: string, deleteFiles: boolean): Promise<void> {
    return mutateApi(`/downloads/${encodeURIComponent(hash)}?deleteFiles=${deleteFiles}`, 'DELETE');
  },

  // SSE endpoint for the lazy useDownloadProgress hook (ux3-4-2b download_progress fan-out).
  getSSEUrl(): string {
    return `${API_BASE_URL}/events`;
  },
};

export default downloadService;
