import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';

vi.mock('../services/requestService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../services/requestService')>();
  return {
    ...actual,
    requestService: { ...actual.requestService, createRequest: vi.fn() },
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
