import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  libraryKeys,
  useLibraryList,
  useLibraryGenres,
  useLibraryStats,
  useLibrarySearch,
  useRecentlyAdded,
  useDeleteLibraryItem,
  useReparseItem,
  useExportItem,
} from './useLibrary';
import { libraryService } from '../services/libraryService';
import type { LibraryListResponse, LibraryStats } from '../types/library';

vi.mock('../services/libraryService', () => ({
  libraryService: {
    listLibrary: vi.fn(),
    searchLibrary: vi.fn(),
    getRecentlyAdded: vi.fn(),
    getGenres: vi.fn(),
    getStats: vi.fn(),
    deleteMovie: vi.fn(),
    deleteSeries: vi.fn(),
    reparseMovie: vi.fn(),
    reparseSeries: vi.fn(),
    exportMovie: vi.fn(),
    exportSeries: vi.fn(),
  },
}));

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

const mockListResponse: LibraryListResponse = {
  items: [
    {
      type: 'movie',
      movie: {
        id: 'm1',
        title: 'Test Movie',
        release_date: '2023-01-01',
        genres: ['Action'],
        parse_status: 'success',
        created_at: '2023-01-01',
        updated_at: '2023-01-01',
      },
    },
  ],
  page: 1,
  pageSize: 20,
  totalItems: 1,
  totalPages: 1,
};

describe('libraryKeys', () => {
  it('generates correct key hierarchy', () => {
    expect(libraryKeys.all).toEqual(['library']);
    expect(libraryKeys.lists()).toEqual(['library', 'list']);
    expect(libraryKeys.list({ page: 1 })).toEqual(['library', 'list', { page: 1 }]);
  });

  it('includes all params in list key', () => {
    const params = { page: 2, pageSize: 10, type: 'movie' as const };
    expect(libraryKeys.list(params)).toEqual(['library', 'list', params]);
  });
});

describe('useLibraryList', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('calls libraryService.listLibrary with params', async () => {
    vi.mocked(libraryService.listLibrary).mockResolvedValue(mockListResponse);

    const { result } = renderHook(() => useLibraryList({ page: 1, pageSize: 20 }), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(libraryService.listLibrary).toHaveBeenCalledWith({ page: 1, pageSize: 20 });
    expect(result.current.data).toEqual(mockListResponse);
  });

  it('returns loading state initially', () => {
    vi.mocked(libraryService.listLibrary).mockReturnValue(new Promise(() => {}));

    const { result } = renderHook(() => useLibraryList({ page: 1 }), {
      wrapper: createWrapper(),
    });

    expect(result.current.isLoading).toBe(true);
  });

  it('returns error state on failure', async () => {
    vi.mocked(libraryService.listLibrary).mockRejectedValue(new Error('Fetch failed'));

    const { result } = renderHook(() => useLibraryList({ page: 1 }), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toBe('Fetch failed');
  });
});

describe('useLibraryGenres', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('[P1] calls libraryService.getGenres', async () => {
    const genres = ['科幻', '動作', '劇情'];
    vi.mocked(libraryService.getGenres).mockResolvedValue(genres);

    const { result } = renderHook(() => useLibraryGenres(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(libraryService.getGenres).toHaveBeenCalled();
    expect(result.current.data).toEqual(genres);
  });

  it('[P1] returns error state on failure', async () => {
    vi.mocked(libraryService.getGenres).mockRejectedValue(new Error('Fetch failed'));

    const { result } = renderHook(() => useLibraryGenres(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toBe('Fetch failed');
  });
});

describe('useLibraryStats', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('[P1] calls libraryService.getStats', async () => {
    const stats: LibraryStats = {
      yearMin: 1990,
      yearMax: 2024,
      movieCount: 100,
      tvCount: 50,
      totalCount: 150,
    };
    vi.mocked(libraryService.getStats).mockResolvedValue(stats);

    const { result } = renderHook(() => useLibraryStats(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(libraryService.getStats).toHaveBeenCalled();
    expect(result.current.data).toEqual(stats);
  });

  it('[P1] returns error state on failure', async () => {
    vi.mocked(libraryService.getStats).mockRejectedValue(new Error('Stats failed'));

    const { result } = renderHook(() => useLibraryStats(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toBe('Stats failed');
  });
});

describe('useLibrarySearch', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('[P1] calls libraryService.searchLibrary with query', async () => {
    const searchResponse = {
      items: [],
      totalItems: 0,
      page: 1,
      pageSize: 20,
      totalPages: 0,
    };
    vi.mocked(libraryService.searchLibrary).mockResolvedValue(searchResponse);

    const { result } = renderHook(() => useLibrarySearch('batman'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(libraryService.searchLibrary).toHaveBeenCalledWith('batman', {});
    expect(result.current.data).toEqual(searchResponse);
  });

  it('[P1] does not fetch when query is shorter than 2 chars', () => {
    const { result } = renderHook(() => useLibrarySearch('b'), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(libraryService.searchLibrary).not.toHaveBeenCalled();
  });

  it('[P1] does not fetch when query is empty', () => {
    const { result } = renderHook(() => useLibrarySearch(''), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe('idle');
    expect(libraryService.searchLibrary).not.toHaveBeenCalled();
  });

  it('[P1] passes additional params to search', async () => {
    const searchResponse = {
      items: [],
      totalItems: 0,
      page: 1,
      pageSize: 20,
      totalPages: 0,
    };
    vi.mocked(libraryService.searchLibrary).mockResolvedValue(searchResponse);

    const { result } = renderHook(() => useLibrarySearch('test', { type: 'movie', page: 2 }), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(libraryService.searchLibrary).toHaveBeenCalledWith('test', { type: 'movie', page: 2 });
  });
});

describe('useRecentlyAdded', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('[P2] calls libraryService.getRecentlyAdded with default limit', async () => {
    vi.mocked(libraryService.getRecentlyAdded).mockResolvedValue([]);

    const { result } = renderHook(() => useRecentlyAdded(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(libraryService.getRecentlyAdded).toHaveBeenCalledWith(20);
  });

  it('[P2] calls libraryService.getRecentlyAdded with custom limit', async () => {
    vi.mocked(libraryService.getRecentlyAdded).mockResolvedValue([]);

    const { result } = renderHook(() => useRecentlyAdded(10), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(libraryService.getRecentlyAdded).toHaveBeenCalledWith(10);
  });
});

describe('useDeleteLibraryItem', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('calls deleteMovie for movie type', async () => {
    vi.mocked(libraryService.deleteMovie).mockResolvedValue(undefined);

    const { result } = renderHook(() => useDeleteLibraryItem(), {
      wrapper: createWrapper(),
    });

    result.current.mutate({ type: 'movie', id: 'movie-1' });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(libraryService.deleteMovie).toHaveBeenCalledWith('movie-1');
    expect(libraryService.deleteSeries).not.toHaveBeenCalled();
  });

  it('calls deleteSeries for series type', async () => {
    vi.mocked(libraryService.deleteSeries).mockResolvedValue(undefined);

    const { result } = renderHook(() => useDeleteLibraryItem(), {
      wrapper: createWrapper(),
    });

    result.current.mutate({ type: 'series', id: 'series-1' });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(libraryService.deleteSeries).toHaveBeenCalledWith('series-1');
    expect(libraryService.deleteMovie).not.toHaveBeenCalled();
  });
});

describe('useReparseItem', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('calls reparseMovie for movie type', async () => {
    vi.mocked(libraryService.reparseMovie).mockResolvedValue({
      id: 'movie-1',
      status: 'reparse_queued',
    });

    const { result } = renderHook(() => useReparseItem(), {
      wrapper: createWrapper(),
    });

    result.current.mutate({ type: 'movie', id: 'movie-1' });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(libraryService.reparseMovie).toHaveBeenCalledWith('movie-1');
  });

  it('calls reparseSeries for series type', async () => {
    vi.mocked(libraryService.reparseSeries).mockResolvedValue({
      id: 'series-1',
      status: 'reparse_queued',
    });

    const { result } = renderHook(() => useReparseItem(), {
      wrapper: createWrapper(),
    });

    result.current.mutate({ type: 'series', id: 'series-1' });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(libraryService.reparseSeries).toHaveBeenCalledWith('series-1');
  });
});

describe('useExportItem', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('calls exportMovie for movie type', async () => {
    vi.mocked(libraryService.exportMovie).mockResolvedValue({ metadata: {} });

    const { result } = renderHook(() => useExportItem(), {
      wrapper: createWrapper(),
    });

    result.current.mutate({ type: 'movie', id: 'movie-1' });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(libraryService.exportMovie).toHaveBeenCalledWith('movie-1');
  });

  it('calls exportSeries for series type', async () => {
    vi.mocked(libraryService.exportSeries).mockResolvedValue({ metadata: {} });

    const { result } = renderHook(() => useExportItem(), {
      wrapper: createWrapper(),
    });

    result.current.mutate({ type: 'series', id: 'series-1' });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(libraryService.exportSeries).toHaveBeenCalledWith('series-1');
  });
});
