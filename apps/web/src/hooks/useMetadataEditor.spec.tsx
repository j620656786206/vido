/**
 * useMetadataEditor — invalidation keys (poster-upload-a AC #6).
 * Real QueryClient: the point is WHICH cached queries go stale.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { useUpdateMetadata, useUploadPoster } from './useMetadataEditor';
import { detailKeys } from './useMediaDetails';
import { libraryKeys } from './useLibrary';
import { libraryKeys as folderKeys } from './useMediaLibrary';
import { metadataService } from '../services/metadata';

vi.mock('../services/metadata', () => ({
  metadataService: { updateMetadata: vi.fn(), uploadPoster: vi.fn() },
}));

function setup() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  for (const key of [
    detailKeys.localMovie('m1'),
    detailKeys.localSeries('s1'),
    libraryKeys.list({} as never),
    folderKeys.all,
  ]) {
    qc.setQueryData(key, { cached: true });
  }
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  );
  const stale = (key: readonly unknown[]) => qc.getQueryState(key)?.isInvalidated;
  return { wrapper, stale };
}

describe('useMetadataEditor invalidation', () => {
  beforeEach(() => {
    vi.mocked(metadataService.updateMetadata).mockResolvedValue({} as never);
    vi.mocked(metadataService.uploadPoster).mockResolvedValue({} as never);
  });

  it('an edit to a movie refreshes that movie’s detail page and the library lists', async () => {
    const { wrapper, stale } = setup();
    const { result } = renderHook(() => useUpdateMetadata(), { wrapper });
    await act(() =>
      result.current.mutateAsync({ id: 'm1', mediaType: 'movie', title: 'x' } as never)
    );
    expect(stale(detailKeys.localMovie('m1'))).toBe(true);
    expect(stale(libraryKeys.list({} as never))).toBe(true);
    expect(stale(detailKeys.localSeries('s1'))).toBe(false);
    expect(stale(folderKeys.all)).toBe(false);
  });

  it('an edit to a series refreshes the series detail page', async () => {
    const { wrapper, stale } = setup();
    const { result } = renderHook(() => useUpdateMetadata(), { wrapper });
    await act(() =>
      result.current.mutateAsync({ id: 's1', mediaType: 'series', title: 'x' } as never)
    );
    expect(stale(detailKeys.localSeries('s1'))).toBe(true);
    expect(stale(detailKeys.localMovie('m1'))).toBe(false);
  });

  it('a poster upload refreshes the same queries', async () => {
    const { wrapper, stale } = setup();
    const { result } = renderHook(() => useUploadPoster(), { wrapper });
    await act(() =>
      result.current.mutateAsync({ mediaId: 'm1', mediaType: 'movie', file: new File([], 'p.jpg') })
    );
    expect(stale(detailKeys.localMovie('m1'))).toBe(true);
    expect(stale(libraryKeys.list({} as never))).toBe(true);
  });
});
