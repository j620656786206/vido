import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { useRecentMedia } from './useDashboardData';

vi.mock('../services/mediaService', () => ({
  mediaService: {
    getRecentMedia: vi.fn(),
  },
}));

import { mediaService } from '../services/mediaService';

const mockGetRecentMedia = vi.mocked(mediaService.getRecentMedia);

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

const mockMedia = [
  {
    id: 'movie-1',
    title: 'Test Movie',
    year: 2024,
    posterUrl: '/poster1.jpg',
    mediaType: 'movie' as const,
    justAdded: true,
    addedAt: '2026-02-10T10:00:00Z',
  },
  {
    id: 'series-1',
    title: 'Test Series',
    year: 2023,
    posterUrl: '/poster2.jpg',
    mediaType: 'tv' as const,
    justAdded: false,
    addedAt: '2026-02-10T09:00:00Z',
  },
];

describe('useRecentMedia', () => {
  beforeEach(() => {
    mockGetRecentMedia.mockResolvedValue(mockMedia);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('[P1] returns recent media data', async () => {
    const { result } = renderHook(() => useRecentMedia(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(mockMedia);
  });

  it('[P1] passes limit parameter', async () => {
    const { result } = renderHook(() => useRecentMedia(5), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockGetRecentMedia).toHaveBeenCalledWith(5);
  });

  it('[P1] defaults limit to 8', async () => {
    const { result } = renderHook(() => useRecentMedia(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockGetRecentMedia).toHaveBeenCalledWith(8);
  });

  it('[P2] handles error state', async () => {
    mockGetRecentMedia.mockRejectedValue(new Error('Network error'));

    const { result } = renderHook(() => useRecentMedia(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toBe('Network error');
  });
});
