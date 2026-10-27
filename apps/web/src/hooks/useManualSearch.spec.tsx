import React from 'react';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useApplyMetadata } from './useManualSearch';
import { metadataService } from '../services/metadata';
import { detailKeys } from './useMediaDetails';
import { libraryKeys } from './useLibrary';

vi.mock('../services/metadata', () => ({
  metadataService: { manualSearch: vi.fn(), applyMetadata: vi.fn() },
}));

function setup() {
  const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  const invalidate = vi.spyOn(queryClient, 'invalidateQueries');
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  return { invalidate, wrapper };
}

// dsr-2b-b AC #4: after a match is applied the detail page must show it. The
// hook used to invalidate ['media', id] — a key nothing reads — so the page kept
// showing the unmatched item until a manual reload.
describe('useApplyMetadata', () => {
  beforeEach(() => vi.clearAllMocks());

  it.each([
    ['movie', detailKeys.localMovie('m-1')],
    ['series', detailKeys.localSeries('m-1')],
  ] as const)('refreshes the %s detail query and the library lists', async (mediaType, key) => {
    vi.mocked(metadataService.applyMetadata).mockResolvedValue({
      success: true,
      mediaId: 'm-1',
      mediaType,
      title: '鬥陣俱樂部',
      source: 'tmdb',
      tmdbId: 550,
      parseStatus: 'success',
    });
    const { invalidate, wrapper } = setup();

    const { result } = renderHook(() => useApplyMetadata(), { wrapper });
    result.current.mutate({
      mediaId: 'm-1',
      mediaType,
      selectedItem: {
        id: 'tmdb-550',
        source: 'tmdb',
        mediaType: mediaType === 'movie' ? 'movie' : 'tv',
      },
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(invalidate).toHaveBeenCalledWith({ queryKey: key });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: libraryKeys.all });
  });

  // disc-2026-10-series-wrong-version-b: re-matching a matched series changes its
  // seasons (將夜: 60 episodes → 20); the season list is cached for an hour.
  it.each([
    ['series', true],
    ['movie', false],
  ] as const)('a %s re-match refreshes the season list: %s', async (mediaType, refreshes) => {
    vi.mocked(metadataService.applyMetadata).mockResolvedValue({
      success: true,
      mediaId: 'm-1',
      mediaType,
      title: '將夜',
      source: 'tmdb',
      tmdbId: 282136,
      parseStatus: 'success',
    });
    const { invalidate, wrapper } = setup();

    const { result } = renderHook(() => useApplyMetadata(), { wrapper });
    result.current.mutate({
      mediaId: 'm-1',
      mediaType,
      selectedItem: {
        id: 'tmdb-282136',
        source: 'tmdb',
        mediaType: mediaType === 'movie' ? 'movie' : 'tv',
      },
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    const seasons = { queryKey: detailKeys.seasons('m-1') };
    const episodes = { queryKey: [...detailKeys.all, 'season-episodes', 'm-1'] };
    if (refreshes) {
      expect(invalidate).toHaveBeenCalledWith(seasons);
      expect(invalidate).toHaveBeenCalledWith(episodes);
    } else {
      expect(invalidate).not.toHaveBeenCalledWith(seasons);
      expect(invalidate).not.toHaveBeenCalledWith(episodes);
    }
  });
});
