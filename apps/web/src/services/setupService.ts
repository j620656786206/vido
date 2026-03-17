/**
 * Setup wizard API service (Story 6.1)
 */

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

export interface SetupStatus {
  needsSetup: boolean;
}

export interface SetupConfig {
  language: string;
  qbtUrl?: string;
  qbtUsername?: string;
  qbtPassword?: string;
  mediaFolderPath: string;
  tmdbApiKey?: string;
  aiProvider?: string;
  aiApiKey?: string;
}

export interface ValidateStepRequest {
  step: string;
  data: Record<string, unknown>;
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

  return data.data as T;
}

export const setupService = {
  async getStatus(): Promise<SetupStatus> {
    return fetchApi<SetupStatus>('/setup/status');
  },

  async completeSetup(config: SetupConfig): Promise<{ message: string }> {
    return fetchApi<{ message: string }>('/setup/complete', {
      method: 'POST',
      body: JSON.stringify(config),
    });
  },

  async validateStep(step: string, data: Record<string, unknown>): Promise<{ valid: boolean }> {
    return fetchApi<{ valid: boolean }>('/setup/validate-step', {
      method: 'POST',
      body: JSON.stringify({ step, data }),
    });
  },
};

export default setupService;
