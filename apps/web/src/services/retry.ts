/**
 * Retry service for retry queue operations (Story 3.11)
 */

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

// Types for retry queue
export interface RetryItem {
  id: string;
  taskId: string;
  taskType: 'parse' | 'metadata_fetch';
  attemptCount: number;
  maxAttempts: number;
  lastError?: string;
  nextAttemptAt: string;
  timeUntilRetry: string;
}

export interface RetryStats {
  totalPending: number;
  totalSucceeded: number;
  totalFailed: number;
}

export interface PendingRetriesResponse {
  items: RetryItem[];
  stats: RetryStats;
}

export interface TriggerResponse {
  id: string;
  message: string;
}

// API response wrapper
interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
    suggestion?: string;
  };
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
    throw new Error(errorData.error?.message || `API request failed: ${response.status}`);
  }

  // Handle 204 No Content
  if (response.status === 204) {
    return undefined as T;
  }

  const data: ApiResponse<T> = await response.json();

  if (!data.success) {
    throw new Error(data.error?.message || 'API request failed');
  }

  return data.data as T;
}

export const retryService = {
  /**
   * Get all pending retry items with stats (AC4)
   */
  async getPending(): Promise<PendingRetriesResponse> {
    return fetchApi<PendingRetriesResponse>('/retry/pending');
  },

  /**
   * Get a specific retry item by ID
   */
  async getById(id: string): Promise<RetryItem> {
    return fetchApi<RetryItem>(`/retry/${id}`);
  },

  /**
   * Trigger an immediate retry for the specified item (AC4)
   */
  async triggerImmediate(id: string): Promise<TriggerResponse> {
    return fetchApi<TriggerResponse>(`/retry/${id}/trigger`, {
      method: 'POST',
    });
  },

  /**
   * Cancel a pending retry item (AC4)
   */
  async cancel(id: string): Promise<void> {
    return fetchApi<void>(`/retry/${id}`, {
      method: 'DELETE',
    });
  },
};

export default retryService;
