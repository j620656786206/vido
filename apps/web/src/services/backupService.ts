/**
 * Backup management API client (Story 6.5)
 */

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

export type BackupStatus = 'pending' | 'running' | 'completed' | 'failed';

export interface Backup {
  id: string;
  filename: string;
  sizeBytes: number;
  schemaVersion: number;
  checksum: string;
  status: BackupStatus;
  errorMessage?: string;
  createdAt: string;
}

export interface BackupListResponse {
  backups: Backup[];
  totalSizeBytes: number;
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

  if (data.data === undefined) {
    throw new Error('API response missing data field');
  }

  return data.data;
}

export const backupService = {
  async listBackups(): Promise<BackupListResponse> {
    return fetchApi<BackupListResponse>('/settings/backups');
  },

  async createBackup(): Promise<Backup> {
    return fetchApi<Backup>('/settings/backups', { method: 'POST' });
  },

  async getBackup(id: string): Promise<Backup> {
    return fetchApi<Backup>(`/settings/backups/${encodeURIComponent(id)}`);
  },

  async deleteBackup(id: string): Promise<void> {
    await fetchApi(`/settings/backups/${encodeURIComponent(id)}`, { method: 'DELETE' });
  },

  getDownloadUrl(id: string): string {
    return `${API_BASE_URL}/settings/backups/${encodeURIComponent(id)}/download`;
  },
};

export default backupService;
