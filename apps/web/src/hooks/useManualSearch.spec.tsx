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
});
