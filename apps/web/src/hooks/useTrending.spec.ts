import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { useTrendingHero } from './useTrending';
import type { Movie, TVShow } from '../types/tmdb';

vi.mock('../services/tmdb', () => ({
  default: {
    getTrendingMovies: vi.fn(),
    getTrendingTVShows: vi.fn(),
  },
}));

import tmdbService from '../services/tmdb';

const mockGetTrendingMovies = vi.mocked(tmdbService.getTrendingMovies);
const mockGetTrendingTVShows = vi.mocked(tmdbService.getTrendingTVShows);

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

function movie(overrides: Partial<Movie> = {}): Movie {
  return {
    id: 1,
    title: 'Movie',
    originalTitle: 'Movie',
    overview: 'overview',
    releaseDate: '2026-01-01',
    posterPath: '/poster.jpg',
    backdropPath: '/backdrop.jpg',
    voteAverage: 8.0,
    voteCount: 1000,
    genreIds: [],
    ...overrides,
  };
}

function tvShow(overrides: Partial<TVShow> = {}): TVShow {
  return {
    id: 1,
    name: 'Show',
    originalName: 'Show',
    overview: 'overview',
    firstAirDate: '2025-06-01',
    posterPath: '/poster.jpg',
    backdropPath: '/backdrop.jpg',
    voteAverage: 7.5,
    voteCount: 500,
    genreIds: [],
    ...overrides,
  };
}

describe('useTrendingHero', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('[P1] interleaves movies and TV shows up to 5 items (AC #1)', async () => {
    mockGetTrendingMovies.mockResolvedValue({
      page: 1,
      totalPages: 1,
      totalResults: 4,
      results: [
        movie({ id: 101, title: 'M1' }),
        movie({ id: 102, title: 'M2' }),
        movie({ id: 103, title: 'M3' }),
        movie({ id: 104, title: 'M4' }),
      ],
    });
    mockGetTrendingTVShows.mockResolvedValue({
      page: 1,
      totalPages: 1,
      totalResults: 4,
      results: [
        tvShow({ id: 201, name: 'T1' }),
        tvShow({ id: 202, name: 'T2' }),
        tvShow({ id: 203, name: 'T3' }),
        tvShow({ id: 204, name: 'T4' }),
      ],
    });

    const { result } = renderHook(() => useTrendingHero(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toHaveLength(5);
    // Interleave: M1, T1, M2, T2, M3
    expect(result.current.data?.map((i) => i.id)).toEqual([101, 201, 102, 202, 103]);
  });

  it('[P1] drops items missing a backdrop image (banner unrenderable)', async () => {
    mockGetTrendingMovies.mockResolvedValue({
      page: 1,
      totalPages: 1,
      totalResults: 2,
      results: [movie({ id: 1, backdropPath: null }), movie({ id: 2, backdropPath: '/has.jpg' })],
    });
    mockGetTrendingTVShows.mockResolvedValue({
      page: 1,
      totalPages: 1,
      totalResults: 0,
      results: [],
    });

    const { result } = renderHook(() => useTrendingHero(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual([expect.objectContaining({ id: 2, mediaType: 'movie' })]);
  });

  it('[P1] returns empty array when both feeds are empty (AC #5)', async () => {
    mockGetTrendingMovies.mockResolvedValue({
      page: 1,
      totalPages: 1,
      totalResults: 0,
      results: [],
    });
    mockGetTrendingTVShows.mockResolvedValue({
      page: 1,
      totalPages: 1,
      totalResults: 0,
      results: [],
    });

    const { result } = renderHook(() => useTrendingHero(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual([]);
  });

  it('[P1] surfaces error state when API fails (AC #5)', async () => {
    mockGetTrendingMovies.mockRejectedValue(new Error('Network down'));
    mockGetTrendingTVShows.mockResolvedValue({
      page: 1,
      totalPages: 1,
      totalResults: 0,
      results: [],
    });

    const { result } = renderHook(() => useTrendingHero(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toBe('Network down');
  });

  it('[P2] normalizes movie title vs TV show name into a unified shape', async () => {
    mockGetTrendingMovies.mockResolvedValue({
      page: 1,
      totalPages: 1,
      totalResults: 1,
      results: [movie({ id: 7, title: '電影標題' })],
    });
    mockGetTrendingTVShows.mockResolvedValue({
      page: 1,
      totalPages: 1,
      totalResults: 1,
      results: [tvShow({ id: 9, name: '影集名稱', firstAirDate: '2024-12-01' })],
    });

    const { result } = renderHook(() => useTrendingHero(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0]).toMatchObject({
      id: 7,
      title: '電影標題',
      mediaType: 'movie',
    });
    expect(result.current.data?.[1]).toMatchObject({
      id: 9,
      title: '影集名稱',
      mediaType: 'tv',
      releaseDate: '2024-12-01',
    });
  });

  it('[P2] passes timeWindow parameter to API', async () => {
    mockGetTrendingMovies.mockResolvedValue({
      page: 1,
      totalPages: 1,
      totalResults: 0,
      results: [],
    });
    mockGetTrendingTVShows.mockResolvedValue({
      page: 1,
      totalPages: 1,
      totalResults: 0,
      results: [],
    });

    const { result } = renderHook(() => useTrendingHero('day'), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockGetTrendingMovies).toHaveBeenCalledWith('day', 1);
    expect(mockGetTrendingTVShows).toHaveBeenCalledWith('day', 1);
  });
});
