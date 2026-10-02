import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';

vi.mock('../services/requestService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../services/requestService')>();
  return {
    ...actual,
    requestService: {
      ...actual.requestService,
      createRequest: vi.fn(),
      cancelRequest: vi.fn(),
      retryRequest: vi.fn(),
    },
  };
});

import { requestService, RequestApiError, type MediaRequest } from '../services/requestService';
import { useRequestActions } from './useRequestActions';
import { requestKeys } from './useRequestedMedia';

const serverRow: MediaRequest = {
  id: 'server-id',
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
};

function setup(seed: MediaRequest[] = []) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  qc.setQueryData<MediaRequest[]>(requestKeys.list(), seed);
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  );
  return { qc, wrapper };
}

describe('useRequestActions.create', () => {
  beforeEach(() => {
    vi.mocked(requestService.createRequest).mockReset();
  });

  it('optimistically prepends a pending row, then settles (AC #4)', async () => {
    let resolveCreate!: (r: MediaRequest) => void;
    vi.mocked(requestService.createRequest).mockReturnValue(
      new Promise((res) => {
        resolveCreate = res;
      })
    );
    const { qc, wrapper } = setup([]);
    const { result } = renderHook(() => useRequestActions(), { wrapper });

    act(() => {
      result.current.create.mutate({ tmdbId: 550, mediaType: 'movie', title: '鬥陣俱樂部' });
    });

    await waitFor(() => {
      const rows = qc.getQueryData<MediaRequest[]>(requestKeys.list());
      expect(rows).toHaveLength(1);
      expect(rows![0].status).toBe('pending');
      expect(rows![0].id).toContain('optimistic');
    });

    resolveCreate(serverRow);
    await waitFor(() => expect(result.current.create.isSuccess).toBe(true));
  });

  it('rolls back the optimistic row on a non-duplicate error', async () => {
    vi.mocked(requestService.createRequest).mockRejectedValue(
      new RequestApiError('此片已在媒體庫中', 'REQUEST_ALREADY_IN_LIBRARY')
    );
    const { qc, wrapper } = setup([]);
    const { result } = renderHook(() => useRequestActions(), { wrapper });

    act(() => {
      result.current.create.mutate({ tmdbId: 550, mediaType: 'movie', title: 'x' });
    });

    await waitFor(() => expect(result.current.create.isError).toBe(true));
    expect(qc.getQueryData<MediaRequest[]>(requestKeys.list())).toEqual([]);
  });

  it('REQUEST_DUPLICATE keeps the optimistic row — the requested state is true (AC #4)', async () => {
    vi.mocked(requestService.createRequest).mockRejectedValue(
      new RequestApiError('已有進行中的請求', 'REQUEST_DUPLICATE')
    );
    const { qc, wrapper } = setup([]);
    const { result } = renderHook(() => useRequestActions(), { wrapper });

    act(() => {
      result.current.create.mutate({ tmdbId: 550, mediaType: 'movie', title: 'x' });
    });

    await waitFor(() => expect(result.current.create.isError).toBe(true));
    const rows = qc.getQueryData<MediaRequest[]>(requestKeys.list());
    expect(rows).toHaveLength(1);
    expect(rows![0].id).toContain('optimistic');
  });
});

// --- Story 13-7b ---

const failedRow: MediaRequest = {
  ...serverRow,
  id: 'failed-id',
  status: 'failed',
  errorMessage: '下載發生錯誤，請重試或檢查下載器',
};

describe('useRequestActions.cancel', () => {
  beforeEach(() => {
    vi.mocked(requestService.cancelRequest).mockReset();
  });

  it('optimistically removes the row, then invalidates', async () => {
    let resolve!: () => void;
    vi.mocked(requestService.cancelRequest).mockReturnValue(
      new Promise<void>((res) => {
        resolve = res;
      })
    );
    const { qc, wrapper } = setup([serverRow, failedRow]);
    const invalidate = vi.spyOn(qc, 'invalidateQueries');
    const { result } = renderHook(() => useRequestActions(), { wrapper });

    act(() => result.current.cancel.mutate('server-id'));
    await waitFor(() =>
      expect(qc.getQueryData<MediaRequest[]>(requestKeys.list())?.map((r) => r.id)).toEqual([
        'failed-id',
      ])
    );

    resolve();
    await waitFor(() => expect(result.current.cancel.isSuccess).toBe(true));
    expect(invalidate).toHaveBeenCalledWith({ queryKey: requestKeys.all });
  });

  it('rolls back on a 409 (the row moved under us)', async () => {
    vi.mocked(requestService.cancelRequest).mockRejectedValue(
      new RequestApiError('此請求已在處理中，無法取消', 'REQUEST_NOT_CANCELLABLE')
    );
    const { qc, wrapper } = setup([serverRow]);
    vi.spyOn(qc, 'invalidateQueries').mockResolvedValue();
    const { result } = renderHook(() => useRequestActions(), { wrapper });

    act(() => result.current.cancel.mutate('server-id'));
    await waitFor(() => expect(result.current.cancel.isError).toBe(true));
    expect(qc.getQueryData<MediaRequest[]>(requestKeys.list())).toEqual([serverRow]);
  });
});

describe('useRequestActions.retry', () => {
  beforeEach(() => {
    vi.mocked(requestService.retryRequest).mockReset();
  });

  it('is NOT optimistic (an early active status would be dropped by the SSE merge), then takes the returned row', async () => {
    let resolve!: (r: MediaRequest) => void;
    vi.mocked(requestService.retryRequest).mockReturnValue(
      new Promise((res) => {
        resolve = res;
      })
    );
    const { qc, wrapper } = setup([failedRow]);
    vi.spyOn(qc, 'invalidateQueries').mockResolvedValue();
    const { result } = renderHook(() => useRequestActions(), { wrapper });

    act(() => result.current.retry.mutate('failed-id'));
    await waitFor(() => expect(result.current.retry.isPending).toBe(true));
    expect(qc.getQueryData<MediaRequest[]>(requestKeys.list())?.[0].status).toBe('failed');

    resolve({ ...failedRow, status: 'pending', errorMessage: null });
    await waitFor(() => expect(result.current.retry.isSuccess).toBe(true));
    expect(qc.getQueryData<MediaRequest[]>(requestKeys.list())?.[0].status).toBe('pending');
  });
});

// CR: overlapping actions — a failed cancel restores only its own row and
// leaves another row's in-flight change alone.
describe('useRequestActions overlap', () => {
  beforeEach(() => {
    vi.mocked(requestService.cancelRequest).mockReset();
  });

  it('a failed cancel puts back only its own row, at its old position', async () => {
    let rejectA!: (e: Error) => void;
    vi.mocked(requestService.cancelRequest).mockImplementation(
      (id: string) =>
        id === 'a'
          ? new Promise<void>((_res, rej) => {
              rejectA = rej;
            })
          : new Promise<void>(() => {}) // b stays in flight
    );
    const a = { ...serverRow, id: 'a' };
    const b = { ...serverRow, id: 'b', tmdbId: 2 };
    const c = { ...serverRow, id: 'c', tmdbId: 3 };
    const { qc, wrapper } = setup([a, b, c]);
    vi.spyOn(qc, 'invalidateQueries').mockResolvedValue();
    const { result } = renderHook(() => useRequestActions(), { wrapper });

    act(() => result.current.cancel.mutate('a'));
    act(() => result.current.cancel.mutate('b'));
    await waitFor(() =>
      expect(qc.getQueryData<MediaRequest[]>(requestKeys.list())?.map((r) => r.id)).toEqual(['c'])
    );

    act(() =>
      rejectA(new RequestApiError('此請求已在處理中，無法取消', 'REQUEST_NOT_CANCELLABLE'))
    );
    await waitFor(() =>
      expect(qc.getQueryData<MediaRequest[]>(requestKeys.list())?.map((r) => r.id)).toEqual([
        'a',
        'c',
      ])
    );
  });
});
