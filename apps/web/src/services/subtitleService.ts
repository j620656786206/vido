import type { ApiResponse } from '../types/tmdb';

const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

async function fetchApi<T>(
  endpoint: string,
  options?: RequestInit,
): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, options);

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(
      errorData.error?.message || `API request failed: ${response.status}`,
    );
  }

  const data: ApiResponse<T> = await response.json();

  if (!data.success) {
    throw new Error(data.error?.message || 'API request failed');
  }

  return data.data as T;
}

// --- Types (snake_case JSON matching backend DTOs) ---

export interface SubtitleSearchParams {
  media_id: string;
  media_type: 'movie' | 'series';
  providers?: string[];
  query?: string;
}

export interface SubtitleScoreBreakdown {
  language: number;
  resolution: number;
  source_trust: number;
  group: number;
  downloads: number;
}

export interface SubtitleSearchResult {
  id: string;
  source: string;
  filename: string;
  language: string;
  download_url: string;
  downloads: number;
  group: string;
  resolution: string;
  format: string;
  score: number;
  score_breakdown: SubtitleScoreBreakdown;
}

export interface SubtitleDownloadParams {
  media_id: string;
  media_type: 'movie' | 'series';
  media_file_path: string;
  subtitle_id: string;
  provider: string;
  resolution?: string;
  convert_to_traditional?: boolean;
}

export interface SubtitleDownloadResult {
  subtitle_path: string;
  language: string;
  score: number;
}

export interface SubtitlePreviewParams {
  subtitle_id: string;
  provider: string;
}

export interface SubtitlePreviewResult {
  lines: string[];
  language: string;
}

// --- Service ---

export const subtitleService = {
  async searchSubtitles(
    params: SubtitleSearchParams,
  ): Promise<SubtitleSearchResult[]> {
    return fetchApi<SubtitleSearchResult[]>('/subtitles/search', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
    });
  },

  async downloadSubtitle(
    params: SubtitleDownloadParams,
  ): Promise<SubtitleDownloadResult> {
    return fetchApi<SubtitleDownloadResult>('/subtitles/download', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
    });
  },

  async previewSubtitle(
    params: SubtitlePreviewParams,
  ): Promise<SubtitlePreviewResult> {
    return fetchApi<SubtitlePreviewResult>('/subtitles/preview', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
    });
  },
};
