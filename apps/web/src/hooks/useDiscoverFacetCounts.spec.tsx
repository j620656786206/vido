import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { Query } from '@tanstack/react-query';
import { useDiscoverFacetCounts, repollInterval, facetCountKeys } from './useDiscoverFacetCounts';
import * as tmdbModule from '../services/tmdb';
import type { FacetCounts } from '../services/tmdb';
import type { DiscoverFilters } from '../lib/discoverFilters';

vi.mock('../services/tmdb', () => ({
  tmdbService: { discoverFacetCounts: vi.fn() },
}));

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

const filtersA: DiscoverFilters = { genre: [28], platform: [], sortBy: 'popularity' };
const filtersB: DiscoverFilters = { genre: [28, 16], platform: [], sortBy: 'popularity' };
const filtersC: DiscoverFilters = { genre: [18], platform: [], sortBy: 'popularity' };

const okResponse: FacetCounts = { counts: { genre: { '28': 100 } }, partial: false };

const mockFn = vi.mocked(tmdbModule.tmdbService.discoverFacetCounts);

describe('useDiscoverFacetCounts', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockFn.mockResolvedValue(okResponse);
  });

  it('fetches and returns the keyed counts + partial flag on success (AC1/AC3)', async () => {
    const { result } = renderHook(() => useDiscoverFacetCounts(filtersA, { enabled: true }), {
      wrapper: createWrapper(),
    });
    await waitFor(() => expect(result.current.counts).toEqual({ genre: { '28': 100 } }));
    expect(result.current.partial).toBe(false);
    expect(mockFn).toHaveBeenCalledTimes(1);
  });

  it('sends the base filter + the four *_values candidate CSVs (Task 3 contract)', async () => {
    renderHook(() => useDiscoverFacetCounts(filtersA, { enabled: true }), {
      wrapper: createWrapper(),
    });
    await waitFor(() => expect(mockFn).toHaveBeenCalled());
    const params = mockFn.mock.calls[0][0];
    expect(params.get('genre')).toBe('28'); // base filter
    expect(params.get('genre_values')).toBeTruthy(); // candidates
    expect(params.get('region_values')).toBeTruthy();
    expect(params.get('rating_values')).toBeTruthy();
    expect(params.get('platform_values')).toBeTruthy();
    // sort/page are stripped so the key is stable across sort/page changes (AR-F2).
    expect(params.get('sort')).toBeNull();
    expect(params.get('page')).toBeNull();
  });

  it('does not fetch unless the caller opts in (desktop-only, AC7)', async () => {
    renderHook(() => useDiscoverFacetCounts(filtersA), { wrapper: createWrapper() });
    await Promise.resolve();
    expect(mockFn).not.toHaveBeenCalled();
  });

  it('keeps counts undefined when the endpoint errors — rail falls back to single total (AC6)', async () => {
    mockFn.mockRejectedValue(new Error('TMDB unavailable'));
    const { result } = renderHook(() => useDiscoverFacetCounts(filtersA, { enabled: true }), {
      wrapper: createWrapper(),
    });
    await waitFor(() => expect(result.current.isFetching).toBe(false));
    expect(result.current.counts).toBeUndefined();
    expect(result.current.partial).toBe(false);
  });

  it('debounces rapid filter changes into a single counts query (~350ms, AC4)', async () => {
    vi.useFakeTimers();
    try {
      const { rerender } = renderHook(({ f }) => useDiscoverFacetCounts(f, { enabled: true }), {
        wrapper: createWrapper(),
        initialProps: { f: filtersA },
      });
      // Initial query fires immediately for A (debouncedParams seeded to A).
      await act(async () => {
        await vi.advanceTimersByTimeAsync(1);
      });
      expect(mockFn).toHaveBeenCalledTimes(1);

      // Two rapid changes within the debounce window → no extra query yet.
      rerender({ f: filtersB });
      await act(async () => {
        await vi.advanceTimersByTimeAsync(100);
      });
      rerender({ f: filtersC });
      await act(async () => {
        await vi.advanceTimersByTimeAsync(100);
      });
      expect(mockFn).toHaveBeenCalledTimes(1);

      // Settle past 350ms → exactly ONE more query, for the latest filters (C).
      await act(async () => {
        await vi.advanceTimersByTimeAsync(350);
      });
      expect(mockFn).toHaveBeenCalledTimes(2);
      const lastParams = mockFn.mock.calls[1][0];
      expect(lastParams.get('genre')).toBe('18'); // filtersC
    } finally {
      vi.useRealTimers();
    }
  });

  // AR-F7 backoff cadence — pure-function unit test (deterministic, no timer flake).
  describe('repollInterval (AR-F7 backoff)', () => {
    const q = (data: FacetCounts | undefined, dataUpdateCount: number) =>
      ({ state: { data, dataUpdateCount } }) as unknown as Query<FacetCounts, Error>;

    it('stops polling when there is no data', () => {
      expect(repollInterval(q(undefined, 0))).toBe(false);
    });

    it('stops polling once partial clears', () => {
      expect(repollInterval(q({ counts: {}, partial: false }, 3))).toBe(false);
    });

    it('backs off exponentially while partial, capped at 30s, then stops after max attempts', () => {
      const partial = (n: number) => repollInterval(q({ counts: {}, partial: true }, n));
      expect(partial(0)).toBe(2000);
      expect(partial(1)).toBe(4000);
      expect(partial(2)).toBe(8000);
      expect(partial(5)).toBe(30000); // 2000*2^5 = 64000 → 30s ceiling
      expect(partial(6)).toBe(false); // max attempts reached → give up, keep last partial counts
    });
  });

  it('builds a stable query key from the param string', () => {
    expect(facetCountKeys.for('genre=28')).toEqual([
      'tmdb',
      'discover',
      'facet-counts',
      'genre=28',
    ]);
  });
});

afterEach(() => {
  vi.useRealTimers();
});
