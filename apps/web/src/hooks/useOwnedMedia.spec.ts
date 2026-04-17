import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { useOwnedMedia } from './useOwnedMedia';

vi.mock('../services/availabilityService', () => ({
  availabilityService: {
    checkOwned: vi.fn(),
  },
}));

import { availabilityService } from '../services/availabilityService';

const mockCheckOwned = vi.mocked(availabilityService.checkOwned);

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe('useOwnedMedia (Story 10-4)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('does not call the API when the input is empty (AC #4)', async () => {
    const { result } = renderHook(() => useOwnedMedia([]), { wrapper: createWrapper() });
    // Give any microtasks a chance to resolve before asserting no-call.
    await new Promise((r) => setTimeout(r, 0));
    expect(mockCheckOwned).not.toHaveBeenCalled();
    expect(result.current.owned.size).toBe(0);
  });

  it('batches visible TMDb IDs into a single request (AC #4)', async () => {
    mockCheckOwned.mockResolvedValue([603, 1396]);

    const { result } = renderHook(() => useOwnedMedia([603, 157336, 1396]), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(mockCheckOwned).toHaveBeenCalledTimes(1));
    // The hook normalises + sorts the input before the call — guaranteed by
    // the query-key factory so ordering swaps hit the cache.
    expect(mockCheckOwned).toHaveBeenCalledWith([603, 1396, 157336]);

    await waitFor(() => expect(result.current.owned.size).toBe(2));
    expect(result.current.isOwned(603)).toBe(true);
    expect(result.current.isOwned(1396)).toBe(true);
    expect(result.current.isOwned(157336)).toBe(false);
  });

  it('deduplicates input so the same ID requested twice collapses to one lookup', async () => {
    mockCheckOwned.mockResolvedValue([603]);

    renderHook(() => useOwnedMedia([603, 603, 603]), { wrapper: createWrapper() });
    await waitFor(() => expect(mockCheckOwned).toHaveBeenCalled());
    expect(mockCheckOwned).toHaveBeenCalledWith([603]);
  });

  it('isRequested is stubbed to false until Phase 3 (AC #5)', async () => {
    mockCheckOwned.mockResolvedValue([603]);
    const { result } = renderHook(() => useOwnedMedia([603]), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isOwned(603)).toBe(true));
    expect(result.current.isRequested(603)).toBe(false);
    expect(result.current.isRequested(9999)).toBe(false);
  });

  it('returns false for null/undefined/zero/negative TMDb IDs', async () => {
    mockCheckOwned.mockResolvedValue([603]);
    const { result } = renderHook(() => useOwnedMedia([603]), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isOwned(603)).toBe(true));
    expect(result.current.isOwned(null)).toBe(false);
    expect(result.current.isOwned(undefined)).toBe(false);
    expect(result.current.isOwned(0)).toBe(false);
    expect(result.current.isOwned(-1)).toBe(false);
  });

  it('exposes loading and error states', async () => {
    mockCheckOwned.mockRejectedValue(new Error('network down'));
    const { result } = renderHook(() => useOwnedMedia([603]), { wrapper: createWrapper() });

    // Initially loading
    expect(result.current.isLoading).toBe(true);

    // The hook sets retry: 1 to absorb one transient failure before surfacing
    // the error; wait up to 3s so the retry-then-fail path completes.
    await waitFor(() => expect(result.current.error).toBeTruthy(), { timeout: 3000 });
    expect(result.current.error?.message).toBe('network down');
    expect(result.current.owned.size).toBe(0);
  });

  it('filters non-positive ids from the request body', async () => {
    mockCheckOwned.mockResolvedValue([]);

    renderHook(() => useOwnedMedia([0, -1, 603, 1396]), { wrapper: createWrapper() });

    await waitFor(() => expect(mockCheckOwned).toHaveBeenCalled());
    expect(mockCheckOwned).toHaveBeenCalledWith([603, 1396]);
  });
});
