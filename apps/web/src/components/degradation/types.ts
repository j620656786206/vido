// Degradation level enum matching backend
export type DegradationLevel = 'normal' | 'partial' | 'minimal' | 'offline';

// Service status enum matching backend
export type ServiceStatus = 'healthy' | 'degraded' | 'down';

// Service health status
export interface ServiceHealth {
  name: string;
  displayName: string;
  status: ServiceStatus;
  lastCheck: string;
  lastSuccess: string;
  errorCount: number;
  message?: string;
}

// All services health
export interface ServicesHealth {
  tmdb: ServiceHealth;
  douban: ServiceHealth;
  wikipedia: ServiceHealth;
  ai: ServiceHealth;
}

// Health status API response
export interface HealthStatusResponse {
  degradationLevel: DegradationLevel;
  services: ServicesHealth;
  message?: string;
}

// Degraded result from API
export interface DegradedResult<T> {
  data: T;
  degradationLevel: DegradationLevel;
  missingFields?: string[];
  fallbackUsed?: string[];
  message?: string;
}

// Field availability info
export interface FieldAvailability {
  field: string;
  available: boolean;
  source?: string;
}
