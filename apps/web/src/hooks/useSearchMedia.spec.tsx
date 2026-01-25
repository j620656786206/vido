import { renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useSearchMovies, useSearchTVShows, tmdbKeys } from './useSearchMedia';
import * as tmdbModule from '../services/tmdb';

vi.mock('../services/tmdb', () => ({
  tmdbService: {
    searchMovies: vi.fn(),
    searchTVShows: vi.fn(),
  },
}));

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

describe('useSearchMovies', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should not fetch when query is less than 2 characters', () => {
    const { result } = renderHook(() => useSearchMovies('鬼'), {
      wrapper: createWrapper(),
    });

    expect(result.current.isFetching).toBe(false);
    expect(result.current.data).toBeUndefined();
    expect(tmdbModule.tmdbService.searchMovies).not.toHaveBeenCalled();
  });

  it('should fetch movies when query is 2+ characters', async () => {
    const mockData = {
      page: 1,
      results: [
        {
          id: 1,
          title: '鬼滅之刃',
          original_title: 'Demon Slayer',
          overview: '',
          release_date: '2019-04-06',
          poster_path: null,
          backdrop_path: null,
          vote_average: 8.5,
          vote_count: 1000,
          genre_ids: [],
        },
      ],
      total_pages: 1,
      total_results: 1,
    };

    vi.mocked(tmdbModule.tmdbService.searchMovies).mockResolvedValueOnce(mockData);

    const { result } = renderHook(() => useSearchMovies('鬼滅'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(result.current.data).toEqual(mockData);
    expect(tmdbModule.tmdbService.searchMovies).toHaveBeenCalledWith('鬼滅', 1);
  });

  it('should fetch with custom page number', async () => {
    const mockData = {
      page: 2,
      results: [],
      total_pages: 5,
      total_results: 100,
    };

    vi.mocked(tmdbModule.tmdbService.searchMovies).mockResolvedValueOnce(mockData);

    const { result } = renderHook(() => useSearchMovies('test', 2), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(tmdbModule.tmdbService.searchMovies).toHaveBeenCalledWith('test', 2);
  });

  it('should handle errors', async () => {
    vi.mocked(tmdbModule.tmdbService.searchMovies).mockRejectedValueOnce(new Error('API error'));

    const { result } = renderHook(() => useSearchMovies('test query'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });

    expect(result.current.error?.message).toBe('API error');
  });
});

describe('useSearchTVShows', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should not fetch when query is less than 2 characters', () => {
    const { result } = renderHook(() => useSearchTVShows('進'), {
      wrapper: createWrapper(),
    });

    expect(result.current.isFetching).toBe(false);
    expect(tmdbModule.tmdbService.searchTVShows).not.toHaveBeenCalled();
  });

  it('should fetch TV shows when query is 2+ characters', async () => {
    const mockData = {
      page: 1,
      results: [
        {
          id: 2,
          name: '進擊的巨人',
          original_name: 'Attack on Titan',
          overview: '',
          first_air_date: '2013-04-07',
          poster_path: null,
          backdrop_path: null,
          vote_average: 9.0,
          vote_count: 2000,
          genre_ids: [],
        },
      ],
      total_pages: 1,
      total_results: 1,
    };

    vi.mocked(tmdbModule.tmdbService.searchTVShows).mockResolvedValueOnce(mockData);

    const { result } = renderHook(() => useSearchTVShows('進擊'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(result.current.data).toEqual(mockData);
    expect(tmdbModule.tmdbService.searchTVShows).toHaveBeenCalledWith('進擊', 1);
  });
});

describe('tmdbKeys', () => {
  it('should generate correct query keys', () => {
    expect(tmdbKeys.all).toEqual(['tmdb']);
    expect(tmdbKeys.searches()).toEqual(['tmdb', 'search']);
    expect(tmdbKeys.searchMovies('test', 1)).toEqual(['tmdb', 'search', 'movies', 'test', 1]);
    expect(tmdbKeys.searchTV('test', 2)).toEqual(['tmdb', 'search', 'tv', 'test', 2]);
  });
});
