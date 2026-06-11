import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { ReactNode } from 'react';
import { useDoubanRating } from './useDoubanRating';
import { libraryService } from '../services/libraryService';

vi.mock('../services/libraryService', () => ({
  libraryService: {
    getMovieDoubanRating: vi.fn(),
    getSeriesDoubanRating: vi.fn(),
  },
}));

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

describe('useDoubanRating', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('fetches the movie Douban rating when enabled', async () => {
    vi.mocked(libraryService.getMovieDoubanRating).mockResolvedValue({
      doubanId: '1292052',
      doubanRating: 9.7,
      doubanVoteCount: 2130000,
    });

    const { result } = renderHook(() => useDoubanRating('m1', 'movie', true), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(libraryService.getMovieDoubanRating).toHaveBeenCalledWith('m1');
    expect(result.current.data?.doubanRating).toBe(9.7);
  });

  it('uses the series endpoint for series type', async () => {
    vi.mocked(libraryService.getSeriesDoubanRating).mockResolvedValue({
      doubanId: '26794435',
      doubanRating: 9.4,
      doubanVoteCount: 880000,
    });

    const { result } = renderHook(() => useDoubanRating('s1', 'series', true), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(libraryService.getSeriesDoubanRating).toHaveBeenCalledWith('s1');
    expect(result.current.data?.doubanRating).toBe(9.4);
  });

  it('does not fetch when disabled (e.g. tmdbId === 0)', () => {
    renderHook(() => useDoubanRating('m1', 'movie', false), {
      wrapper: createWrapper(),
    });
    expect(libraryService.getMovieDoubanRating).not.toHaveBeenCalled();
  });

  it('handles a null response (graceful degradation)', async () => {
    vi.mocked(libraryService.getMovieDoubanRating).mockResolvedValue(null);

    const { result } = renderHook(() => useDoubanRating('m1', 'movie', true), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toBeNull();
  });
});
