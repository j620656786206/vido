import { renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useDiscoverResults, discoverKeys } from './useDiscoverResults';
import * as tmdbModule from '../services/tmdb';
import type { DiscoverFilters } from '../lib/discoverFilters';

vi.mock('../services/tmdb', () => ({
  tmdbService: {
    discoverMovies: vi.fn(),
    discoverTVShows: vi.fn(),
  },
}));

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

const filters: DiscoverFilters = { genre: [28], platform: [], sortBy: 'popularity' };

const emptyResponse = { page: 1, results: [], totalPages: 1, totalResults: 0 };

describe('useDiscoverResults', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(tmdbModule.tmdbService.discoverMovies).mockResolvedValue({
      ...emptyResponse,
      totalResults: 3,
    });
    vi.mocked(tmdbModule.tmdbService.discoverTVShows).mockResolvedValue({
      ...emptyResponse,
      totalResults: 5,
    });
  });

  it('only queries movies when media type is movie', async () => {
    const { result } = renderHook(() => useDiscoverResults(filters, 'movie'), {
      wrapper: createWrapper(),
    });
    await waitFor(() => expect(result.current.moviesQuery.isSuccess).toBe(true));

    expect(tmdbModule.tmdbService.discoverMovies).toHaveBeenCalledTimes(1);
    expect(tmdbModule.tmdbService.discoverTVShows).not.toHaveBeenCalled();
    expect(result.current.totalResults).toBe(3);
  });

  it('queries both endpoints and sums totals for media type all', async () => {
    const { result } = renderHook(() => useDiscoverResults(filters, 'all'), {
      wrapper: createWrapper(),
    });
    await waitFor(() => expect(result.current.totalResults).toBe(8));

    expect(tmdbModule.tmdbService.discoverMovies).toHaveBeenCalledTimes(1);
    expect(tmdbModule.tmdbService.discoverTVShows).toHaveBeenCalledTimes(1);
  });

  it('passes the mapped backend params (vote_gte) to the service', async () => {
    const ratingFilters: DiscoverFilters = {
      genre: [],
      platform: [],
      sortBy: 'popularity',
      ratingGte: 7,
    };
    renderHook(() => useDiscoverResults(ratingFilters, 'movie'), { wrapper: createWrapper() });
    await waitFor(() => expect(tmdbModule.tmdbService.discoverMovies).toHaveBeenCalled());

    const params = vi.mocked(tmdbModule.tmdbService.discoverMovies).mock.calls[0][0];
    expect(params.get('vote_gte')).toBe('7');
  });

  it('builds distinct query keys per media type', () => {
    expect(discoverKeys.movies(filters, 1)).not.toEqual(discoverKeys.tv(filters, 1));
  });

  it('coalesces the movie+tv pair into one loading state, then settles (AC #6)', async () => {
    const { result } = renderHook(() => useDiscoverResults(filters, 'all'), {
      wrapper: createWrapper(),
    });
    // One logical refresh: loading until BOTH enabled queries resolve.
    expect(result.current.isLoading).toBe(true);
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.isFetching).toBe(false);
    expect(result.current.totalResults).toBe(8);
  });

  it('does not fetch when the hook is disabled (gated draft-count off)', () => {
    renderHook(() => useDiscoverResults(filters, 'all', 1, { enabled: false }), {
      wrapper: createWrapper(),
    });
    expect(tmdbModule.tmdbService.discoverMovies).not.toHaveBeenCalled();
    expect(tmdbModule.tmdbService.discoverTVShows).not.toHaveBeenCalled();
  });
});
