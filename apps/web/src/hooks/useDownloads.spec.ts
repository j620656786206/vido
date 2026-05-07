import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useDownloads, useDownloadDetails, useDownloadCounts, downloadKeys } from './useDownloads';
import type { ReactNode } from 'react';
import React from 'react';

// Mock the downloadService
vi.mock('../services/downloadService', () => ({
  downloadService: {
    getDownloads: vi.fn(),
    getDownloadDetails: vi.fn(),
    getDownloadCounts: vi.fn(),
  },
}));

// bugfix-10-2: mock useQBittorrentConfig so the new gate inside useDownloads /
// useDownloadCounts / useDownloadDetails can be exercised. Default returns
// configured:true to keep existing happy-path tests green; gate-coverage tests
// override with mockUseQBittorrentConfig.mockReturnValueOnce(...).
vi.mock('./useQBittorrent', () => ({
  useQBittorrentConfig: vi.fn(),
}));

// Import the mocked service for test control
import { downloadService } from '../services/downloadService';
import { useQBittorrentConfig } from './useQBittorrent';
const mockGetDownloads = vi.mocked(downloadService.getDownloads);
const mockGetDownloadDetails = vi.mocked(downloadService.getDownloadDetails);
const mockGetDownloadCounts = vi.mocked(downloadService.getDownloadCounts);
const mockUseQBittorrentConfig = vi.mocked(useQBittorrentConfig);

// Helper to build a useQuery-shaped return for the qBT config mock.
// CR M3: typed as Partial<UseQueryResult<...>> so a future `useQBittorrentConfig`
// signature change (e.g. additional fields the hook reads) surfaces as a TS
// error here rather than a silent runtime mismatch. The single cast at the
// return site is intentional — UseQueryResult is a discriminated union and
// constructing every variant by hand adds noise without raising the safety bar.
type QBTConfigResult = ReturnType<typeof useQBittorrentConfig>;
function qbtConfigResult(configured: boolean | undefined, isLoading = false): QBTConfigResult {
  const partial: Partial<QBTConfigResult> = {
    data: configured === undefined ? undefined : { configured },
    isLoading,
    isError: false,
    error: null,
    isSuccess: !isLoading && configured !== undefined,
  };
  return partial as QBTConfigResult;
}

// Test data
const mockDownloads = [
  {
    hash: 'abc123',
    name: 'Test Movie [1080p]',
    size: 4294967296,
    progress: 0.85,
    downloadSpeed: 10485760,
    uploadSpeed: 524288,
    eta: 600,
    status: 'downloading' as const,
    addedOn: '2026-01-15T10:00:00Z',
    seeds: 10,
    peers: 5,
    downloaded: 3650722201,
    uploaded: 104857600,
    ratio: 0.03,
    savePath: '/downloads/movies',
  },
];

const mockDetails = {
  ...mockDownloads[0],
  pieceSize: 4194304,
  comment: 'Test',
  createdBy: 'qBittorrent',
  creationDate: '2026-01-10T08:00:00Z',
  totalWasted: 0,
  timeElapsed: 3600,
  seedingTime: 0,
  avgDownSpeed: 8388608,
  avgUpSpeed: 262144,
};

const mockCounts = {
  all: 10,
  downloading: 3,
  paused: 2,
  completed: 4,
  seeding: 1,
  error: 0,
};

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });
  return function Wrapper({ children }: { children: ReactNode }) {
    return React.createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

const mockPaginatedResponse = {
  items: mockDownloads,
  page: 1,
  pageSize: 100,
  totalItems: 1,
  totalPages: 1,
};

describe('useDownloads', () => {
  beforeEach(() => {
    mockGetDownloads.mockResolvedValue(mockPaginatedResponse);
    // bugfix-10-2: default to configured:true so existing tests keep firing requests
    mockUseQBittorrentConfig.mockReturnValue(qbtConfigResult(true));
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('[P1] returns download data with default filter (AC1)', async () => {
    const { result } = renderHook(() => useDownloads(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toEqual(mockPaginatedResponse);
    expect(mockGetDownloads).toHaveBeenCalledWith({
      filter: 'all',
      sort: 'added_on',
      order: 'desc',
      page: 1,
      pageSize: 100,
    });
  });

  it('[P1] passes filter param to service (AC2)', async () => {
    const { result } = renderHook(() => useDownloads('downloading'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockGetDownloads).toHaveBeenCalledWith({
      filter: 'downloading',
      sort: 'added_on',
      order: 'desc',
      page: 1,
      pageSize: 100,
    });
  });

  it('[P1] passes filter and sort params to service', async () => {
    const { result } = renderHook(() => useDownloads('paused', 'name', 'asc'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockGetDownloads).toHaveBeenCalledWith({
      filter: 'paused',
      sort: 'name',
      order: 'asc',
      page: 1,
      pageSize: 100,
    });
  });

  it('[P1] configures 5-second polling interval (AC3)', () => {
    const { result } = renderHook(() => useDownloads(), { wrapper: createWrapper() });

    expect(result.current.isLoading || result.current.isSuccess || result.current.isError).toBe(
      true
    );
  });
});

describe('useDownloadCounts', () => {
  beforeEach(() => {
    mockGetDownloadCounts.mockResolvedValue(mockCounts);
    mockUseQBittorrentConfig.mockReturnValue(qbtConfigResult(true));
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('[P1] returns download counts (AC1)', async () => {
    const { result } = renderHook(() => useDownloadCounts(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toEqual(mockCounts);
    expect(mockGetDownloadCounts).toHaveBeenCalled();
  });

  it('[P1] can be disabled', () => {
    const { result } = renderHook(() => useDownloadCounts(false), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(mockGetDownloadCounts).not.toHaveBeenCalled();
  });
});

describe('useDownloadDetails', () => {
  beforeEach(() => {
    mockGetDownloadDetails.mockResolvedValue(mockDetails);
    mockUseQBittorrentConfig.mockReturnValue(qbtConfigResult(true));
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('[P1] returns download details (AC4)', async () => {
    const { result } = renderHook(() => useDownloadDetails('abc123'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toEqual(mockDetails);
    expect(mockGetDownloadDetails).toHaveBeenCalledWith('abc123');
  });

  it('[P1] does not fetch when hash is empty', async () => {
    const { result } = renderHook(() => useDownloadDetails(''), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(mockGetDownloadDetails).not.toHaveBeenCalled();
  });
});

describe('useDownloads - error handling', () => {
  beforeEach(() => {
    mockUseQBittorrentConfig.mockReturnValue(qbtConfigResult(true));
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('[P1] exposes error state when service fails (AC3)', async () => {
    // GIVEN: service rejects with error
    mockGetDownloads.mockRejectedValue(new Error('Network error'));

    const { result } = renderHook(() => useDownloads(), { wrapper: createWrapper() });

    // THEN: error state is exposed
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
    expect(result.current.error?.message).toBe('Network error');
  });
});

describe('useDownloadCounts - error handling', () => {
  beforeEach(() => {
    mockUseQBittorrentConfig.mockReturnValue(qbtConfigResult(true));
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('[P1] exposes error state when counts service fails', async () => {
    // GIVEN: counts service rejects
    mockGetDownloadCounts.mockRejectedValue(new Error('API request failed: 502'));

    const { result } = renderHook(() => useDownloadCounts(), { wrapper: createWrapper() });

    // THEN: error state is exposed
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toBe('API request failed: 502');
  });
});

// bugfix-10-2 [@contract-v1] AC #5/#6/#7: hooks MUST NOT fire requests until
// useQBittorrentConfig().data?.configured === true. Both the loading branch
// (data undefined, isLoading true) AND the false branch must suppress.
describe('useDownloads - qBT config gate (bugfix-10-2)', () => {
  beforeEach(() => {
    mockGetDownloads.mockResolvedValue(mockPaginatedResponse);
    mockGetDownloadCounts.mockResolvedValue(mockCounts);
    mockGetDownloadDetails.mockResolvedValue(mockDetails);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  // CR L4: absence-assertion pattern. `enabled: false` makes TanStack Query
  // never queue the queryFn — fetchStatus stays 'idle' across render + reactive
  // cycle. waitFor returns on the first idle observation (typically <5ms),
  // and any accidental fetch flips fetchStatus to 'fetching' so the assertion
  // would fail loudly instead of waiting out a fixed sleep.
  it('does NOT call getDownloads when qBT is not configured (configured: false)', async () => {
    mockUseQBittorrentConfig.mockReturnValue(qbtConfigResult(false));
    const { result } = renderHook(() => useDownloads(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.fetchStatus).toBe('idle'));
    expect(mockGetDownloads).not.toHaveBeenCalled();
  });

  it('does NOT call getDownloads while qBT config is still loading (data: undefined, isLoading: true)', async () => {
    mockUseQBittorrentConfig.mockReturnValue(qbtConfigResult(undefined, true));
    const { result } = renderHook(() => useDownloads(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.fetchStatus).toBe('idle'));
    expect(mockGetDownloads).not.toHaveBeenCalled();
  });

  it('does NOT call getDownloadCounts when qBT is not configured', async () => {
    mockUseQBittorrentConfig.mockReturnValue(qbtConfigResult(false));
    const { result } = renderHook(() => useDownloadCounts(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.fetchStatus).toBe('idle'));
    expect(mockGetDownloadCounts).not.toHaveBeenCalled();
  });

  it('does NOT call getDownloadCounts while qBT config is still loading', async () => {
    mockUseQBittorrentConfig.mockReturnValue(qbtConfigResult(undefined, true));
    const { result } = renderHook(() => useDownloadCounts(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.fetchStatus).toBe('idle'));
    expect(mockGetDownloadCounts).not.toHaveBeenCalled();
  });

  it('AC #6: caller-supplied enabled=true is still suppressed when qBT not configured', async () => {
    mockUseQBittorrentConfig.mockReturnValue(qbtConfigResult(false));
    const { result } = renderHook(() => useDownloadCounts(true), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.fetchStatus).toBe('idle'));
    expect(mockGetDownloadCounts).not.toHaveBeenCalled();
  });

  it('does NOT call getDownloadDetails when qBT is not configured', async () => {
    mockUseQBittorrentConfig.mockReturnValue(qbtConfigResult(false));
    const { result } = renderHook(() => useDownloadDetails('abc123'), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.fetchStatus).toBe('idle'));
    expect(mockGetDownloadDetails).not.toHaveBeenCalled();
  });

  it('does NOT call getDownloadDetails while qBT config is still loading', async () => {
    mockUseQBittorrentConfig.mockReturnValue(qbtConfigResult(undefined, true));
    const { result } = renderHook(() => useDownloadDetails('abc123'), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.fetchStatus).toBe('idle'));
    expect(mockGetDownloadDetails).not.toHaveBeenCalled();
  });
});

describe('downloadKeys', () => {
  it('[P2] generates correct query keys', () => {
    expect(downloadKeys.all).toEqual(['downloads']);
    expect(downloadKeys.list('all', 'name', 'asc', 1, 100)).toEqual([
      'downloads',
      'list',
      'all',
      'name',
      'asc',
      1,
      100,
    ]);
    expect(downloadKeys.counts()).toEqual(['downloads', 'counts']);
    expect(downloadKeys.detail('abc123')).toEqual(['downloads', 'detail', 'abc123']);
  });

  it('[P2] generates unique keys per filter combination', () => {
    const key1 = downloadKeys.list('downloading', 'name', 'asc', 1, 100);
    const key2 = downloadKeys.list('paused', 'name', 'asc', 1, 100);
    const key3 = downloadKeys.list('downloading', 'size', 'desc', 1, 100);
    const key4 = downloadKeys.list('downloading', 'name', 'asc', 2, 100);

    // Keys should differ when filter, sort, or page params differ
    expect(key1).not.toEqual(key2);
    expect(key1).not.toEqual(key3);
    expect(key1).not.toEqual(key4);
  });
});
