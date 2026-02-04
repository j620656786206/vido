/**
 * Learning service for filename pattern learning operations (Story 3.9)
 */

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

// Types for learned patterns
export interface LearnPatternParams {
  filename: string;
  metadataId: string;
  metadataType: 'movie' | 'series';
  tmdbId?: number;
}

export interface LearnedPattern {
  id: string;
  pattern: string;
  patternType: string;
  fansubGroup?: string;
  titlePattern?: string;
  metadataType: string;
  metadataId: string;
  tmdbId?: number;
  confidence: number;
  useCount: number;
  createdAt: string;
  lastUsedAt?: string;
}

export interface PatternStats {
  totalPatterns: number;
  totalApplied: number;
  mostUsedPattern?: string;
  mostUsedCount?: number;
}

export interface PatternListResponse {
  patterns: LearnedPattern[];
  totalCount: number;
  stats?: PatternStats;
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

export const learningService = {
  /**
   * Learn a new pattern from a filename correction (AC1)
   */
  async learnPattern(params: LearnPatternParams): Promise<LearnedPattern> {
    return fetchApi<LearnedPattern>('/learning/patterns', {
      method: 'POST',
      body: JSON.stringify(params),
    });
  },

  /**
   * List all learned patterns with stats (AC3)
   */
  async listPatterns(): Promise<PatternListResponse> {
    return fetchApi<PatternListResponse>('/learning/patterns');
  },

  /**
   * Delete a learned pattern (AC3)
   */
  async deletePattern(id: string): Promise<void> {
    return fetchApi<void>(`/learning/patterns/${id}`, {
      method: 'DELETE',
    });
  },

  /**
   * Get pattern statistics
   */
  async getStats(): Promise<PatternStats> {
    return fetchApi<PatternStats>('/learning/stats');
  },
};

export default learningService;
