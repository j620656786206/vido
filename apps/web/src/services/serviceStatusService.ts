/**
 * Service status API client for settings dashboard (Story 6.4)
 */

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export type ServiceConnectionStatus =
  | 'connected'
  | 'rate_limited'
  | 'error'
  | 'disconnected'
  | 'unconfigured';

export interface ServiceStatus {
  name: string;
  displayName: string;
  status: ServiceConnectionStatus;
  message: string;
  lastSuccessAt: string | null;
  lastCheckAt: string;
  responseTimeMs: number;
  errorMessage?: string;
}

export interface ServiceStatusResponse {
  services: ServiceStatus[];
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

export const serviceStatusService = {
  async getAllStatuses(): Promise<ServiceStatusResponse> {
    return fetchApi<ServiceStatusResponse>('/settings/services');
  },

  async testService(serviceName: string): Promise<ServiceStatus> {
    return fetchApi<ServiceStatus>(`/settings/services/${encodeURIComponent(serviceName)}/test`, {
      method: 'POST',
    });
  },
};

export default serviceStatusService;
