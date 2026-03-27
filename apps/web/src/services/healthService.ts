/**
 * Health service for connection health monitoring (Story 4.6)
 */

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export interface ServiceHealth {
  name: string;
  displayName: string;
  status: 'healthy' | 'degraded' | 'down';
  lastCheck: string;
  lastSuccess: string;
  errorCount: number;
  message?: string;
}

export interface ServicesHealth {
  tmdb: ServiceHealth;
  douban: ServiceHealth;
  wikipedia: ServiceHealth;
  ai: ServiceHealth;
  qbittorrent: ServiceHealth;
}

export interface HealthStatusResponse {
  degradationLevel: 'normal' | 'partial' | 'minimal' | 'offline';
  services: ServicesHealth;
  message: string;
}

export type ConnectionEventType = 'connected' | 'disconnected' | 'error' | 'recovered';

export interface ConnectionEvent {
  id: string;
  service: string;
  eventType: ConnectionEventType;
  status: string;
  message?: string;
  createdAt: string;
}

interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
  };
}

async function fetchApi<T>(endpoint: string): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`);

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

export const healthService = {
  async getServicesHealth(): Promise<HealthStatusResponse> {
    return fetchApi<HealthStatusResponse>('/health/services');
  },

  async getConnectionHistory(service: string, limit = 20): Promise<ConnectionEvent[]> {
    return fetchApi<ConnectionEvent[]>(
      `/health/services/${encodeURIComponent(service)}/history?limit=${limit}`
    );
  },
};

export default healthService;
