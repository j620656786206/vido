/**
 * Download monitoring service (Story 4.2)
 */

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

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

export const downloadService = {
  async getDownloads(params?: GetDownloadsParams): Promise<Download[]> {
    const filter = params?.filter || 'all';
    const sort = params?.sort || 'added_on';
    const order = params?.order || 'desc';
    return fetchApi<Download[]>(`/downloads?filter=${filter}&sort=${sort}&order=${order}`);
  },

  async getDownloadDetails(hash: string): Promise<DownloadDetails> {
    return fetchApi<DownloadDetails>(`/downloads/${hash}`);
  },

  async getDownloadCounts(): Promise<DownloadCounts> {
    return fetchApi<DownloadCounts>(`/downloads/counts`);
  },
};

export default downloadService;
