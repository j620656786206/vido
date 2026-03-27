/**
 * Backup management API client (Story 6.5)
 */

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export type BackupStatus = 'pending' | 'running' | 'completed' | 'failed' | 'corrupted';

export type VerificationStatus = 'verified' | 'corrupted' | 'missing';

export interface VerificationResult {
  backupId: string;
  status: VerificationStatus;
  storedChecksum: string;
  calculatedChecksum: string;
  match: boolean;
  verifiedAt: string;
}

export type RestoreStatus = 'in_progress' | 'completed' | 'failed';

export interface RestoreResult {
  restoreId: string;
  status: RestoreStatus;
  snapshotId: string;
  message: string;
  error?: string;
}

export interface BackupSchedule {
  enabled: boolean;
  frequency: 'daily' | 'weekly' | 'disabled';
  hour: number;
  dayOfWeek: number;
  nextBackupAt?: string;
  lastBackupAt?: string;
}

export interface ExportResult {
  exportId: string;
  format: 'json' | 'yaml' | 'nfo';
  status: 'in_progress' | 'completed' | 'failed';
  filePath?: string;
  itemCount: number;
  message?: string;
  error?: string;
}

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

  async verifyBackup(id: string): Promise<VerificationResult> {
    return fetchApi<VerificationResult>(`/settings/backups/${encodeURIComponent(id)}/verify`, {
      method: 'POST',
    });
  },

  async restoreBackup(id: string): Promise<RestoreResult> {
    return fetchApi<RestoreResult>(`/settings/backups/${encodeURIComponent(id)}/restore`, {
      method: 'POST',
    });
  },

  async getRestoreStatus(): Promise<RestoreResult> {
    return fetchApi<RestoreResult>('/settings/restore/status');
  },

  async getSchedule(): Promise<BackupSchedule> {
    return fetchApi<BackupSchedule>('/settings/backups/schedule');
  },

  async updateSchedule(schedule: Partial<BackupSchedule>): Promise<BackupSchedule> {
    return fetchApi<BackupSchedule>('/settings/backups/schedule', {
      method: 'PUT',
      body: JSON.stringify(schedule),
    });
  },

  async triggerExport(format: 'json' | 'yaml' | 'nfo'): Promise<ExportResult> {
    return fetchApi<ExportResult>('/settings/export', {
      method: 'POST',
      body: JSON.stringify({ format }),
    });
  },

  async getExportStatus(): Promise<ExportResult> {
    return fetchApi<ExportResult>('/settings/export/status');
  },

  getExportDownloadUrl(id: string): string {
    return `${API_BASE_URL}/settings/export/${encodeURIComponent(id)}/download`;
  },

  getDownloadUrl(id: string): string {
    return `${API_BASE_URL}/settings/backups/${encodeURIComponent(id)}/download`;
  },
};

export default backupService;
