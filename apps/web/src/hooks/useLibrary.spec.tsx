import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  libraryKeys,
  useLibraryList,
  useDeleteLibraryItem,
  useReparseItem,
  useExportItem,
} from './useLibrary';
import { libraryService } from '../services/libraryService';
import type { LibraryListResponse } from '../types/library';

vi.mock('../services/libraryService', () => ({
  libraryService: {
    listLibrary: vi.fn(),
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
