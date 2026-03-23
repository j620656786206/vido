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

// --- Types ---

export interface SubtitleSearchParams {
  mediaId: string;
  mediaType: 'movie' | 'series';
  providers?: string[];
  query?: string;
}

export interface SubtitleScoreBreakdown {
  language: number;
  resolution: number;
  sourceTrust: number;
  group: number;
  downloads: number;
}

export interface SubtitleSearchResult {
  ID: string;
  Source: string;
  Filename: string;
  Language: string;
  DownloadURL: string;
  Downloads: number;
  Group: string;
  Resolution: string;
  Format: string;
  score: number;
  scoreBreakdown: SubtitleScoreBreakdown;
}

export interface SubtitleDownloadParams {
  mediaId: string;
  mediaType: 'movie' | 'series';
  mediaFilePath: string;
  subtitleId: string;
  provider: string;
  resolution?: string;
}

export interface SubtitleDownloadResult {
  subtitlePath: string;
  language: string;
}

export interface SubtitlePreviewParams {
  subtitleId: string;
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
