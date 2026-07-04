import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';

vi.mock('../services/requestService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../services/requestService')>();
  return {
    ...actual,
    requestService: { ...actual.requestService, listRequests: vi.fn() },
  };
});

import { requestService, type MediaRequest } from '../services/requestService';
import { useRequestedMedia } from './useRequestedMedia';

const row = (over: Partial<MediaRequest> = {}): MediaRequest => ({
  id: 'r1',
  tmdbId: 550,
  mediaType: 'movie',
  title: '鬥陣俱樂部',
  status: 'pending',
  fulfilmentSource: null,
  externalId: null,
  seasons: null,
  episodes: null,
  errorMessage: null,
  requestedAt: '2026-07-04T12:00:00Z',
  updatedAt: '2026-07-04T12:00:00Z',
  ...over,
});

function setup() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  );
  return { wrapper };
}

describe('useRequestedMedia', () => {
  beforeEach(() => {
    vi.mocked(requestService.listRequests).mockReset();
  });

  it('isRequested is true only for ACTIVE statuses (pending/searching/downloading)', async () => {
    vi.mocked(requestService.listRequests).mockResolvedValue([
      row({ id: 'a', tmdbId: 1, status: 'pending' }),
      row({ id: 'b', tmdbId: 2, status: 'searching' }),
      row({ id: 'c', tmdbId: 3, status: 'downloading' }),
      row({ id: 'd', tmdbId: 4, status: 'completed' }),
      row({ id: 'e', tmdbId: 5, status: 'failed' }),
    ]);

    const { result } = renderHook(() => useRequestedMedia(), { wrapper: setup().wrapper });
    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.isRequested(1, 'movie')).toBe(true);
    expect(result.current.isRequested(2, 'movie')).toBe(true);
    expect(result.current.isRequested(3, 'movie')).toBe(true);
    expect(result.current.isRequested(4, 'movie')).toBe(false);
    expect(result.current.isRequested(5, 'movie')).toBe(false);
  });

  it('with mediaType the check is exact; without it any type matches (useOwnedMedia semantic)', async () => {
    vi.mocked(requestService.listRequests).mockResolvedValue([
      row({ tmdbId: 550, mediaType: 'tv' }),
    ]);

    const { result } = renderHook(() => useRequestedMedia(), { wrapper: setup().wrapper });
    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.isRequested(550, 'tv')).toBe(true);
    expect(result.current.isRequested(550, 'movie')).toBe(false);
    expect(result.current.isRequested(550)).toBe(true);
  });

  it('rejects invalid ids and never fetches when disabled', async () => {
    vi.mocked(requestService.listRequests).mockResolvedValue([]);

    const { result } = renderHook(() => useRequestedMedia(false), { wrapper: setup().wrapper });

    expect(result.current.isRequested(0)).toBe(false);
    expect(result.current.isRequested(null)).toBe(false);
    expect(requestService.listRequests).not.toHaveBeenCalled();
  });
});
