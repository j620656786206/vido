import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';

vi.mock('../services/downloadService', () => ({
  downloadService: {
    pauseDownload: vi.fn().mockResolvedValue(undefined),
    resumeDownload: vi.fn().mockResolvedValue(undefined),
    removeDownload: vi.fn().mockResolvedValue(undefined),
  },
}));

import {
  downloadService,
  type Download,
  type PaginatedDownloads,
} from '../services/downloadService';
import { useDownloadActions } from './useDownloadActions';
import { downloadKeys } from './useDownloads';

const item = (over: Partial<Download> = {}): Download => ({
  hash: 'a',
  name: 'A.mkv',
  size: 100,
  progress: 0.5,
  downloadSpeed: 10,
  uploadSpeed: 0,
  eta: 60,
  status: 'downloading',
  addedOn: '2026-07-01T00:00:00Z',
  seeds: 1,
  peers: 1,
  downloaded: 50,
  uploaded: 0,
  ratio: 0,
  savePath: '/dl',
  ...over,
});

const LIST_KEY = downloadKeys.list('all', 'added_on', 'desc', 1, 100);

function setup(items: Download[]) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  qc.setQueryData<PaginatedDownloads>(LIST_KEY, {
    items,
    page: 1,
    pageSize: 100,
    totalItems: items.length,
    totalPages: 1,
  });
  const wrapper = ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: qc }, children);
  const { result } = renderHook(() => useDownloadActions(), { wrapper });
  const read = () => qc.getQueryData<PaginatedDownloads>(LIST_KEY)!;
  return { qc, result, read };
}

describe('useDownloadActions (ux3-4-3b AC3/AC5)', () => {
  beforeEach(() => vi.clearAllMocks());

  it('pause flips the item to paused and calls the single-hash endpoint', async () => {
    const { result, read } = setup([item({ hash: 'a', status: 'downloading' })]);
    act(() => result.current.pause.mutate(['a']));

    await waitFor(() => expect(downloadService.pauseDownload).toHaveBeenCalledWith('a'));
    await waitFor(() => expect(read().items.find((i) => i.hash === 'a')?.status).toBe('paused'));
  });

  it('resume flips a paused item back to downloading', async () => {
    const { result, read } = setup([item({ hash: 'a', status: 'paused' })]);
    act(() => result.current.resume.mutate(['a']));

    await waitFor(() => expect(downloadService.resumeDownload).toHaveBeenCalledWith('a'));
    await waitFor(() =>
      expect(read().items.find((i) => i.hash === 'a')?.status).toBe('downloading')
    );
  });

  it('remove drops the item + decrements totalItems and passes deleteFiles', async () => {
    const { result, read } = setup([
      item({ hash: 'a' }),
      item({ hash: 'b', parseStatus: { status: 'completed' } }),
    ]);
    act(() => result.current.remove.mutate({ hashes: ['a'], deleteFiles: true }));

    await waitFor(() => expect(downloadService.removeDownload).toHaveBeenCalledWith('a', true));
    await waitFor(() => expect(read().items.find((i) => i.hash === 'a')).toBeUndefined());
    expect(read().totalItems).toBe(1);
  });

  it('a batch pause fires one single-hash request per hash (no batch HTTP endpoint exists)', async () => {
    const { result } = setup([item({ hash: 'a' }), item({ hash: 'b' })]);
    act(() => result.current.pause.mutate(['a', 'b']));

    await waitFor(() => expect(downloadService.pauseDownload).toHaveBeenCalledTimes(2));
    expect(downloadService.pauseDownload).toHaveBeenCalledWith('a');
    expect(downloadService.pauseDownload).toHaveBeenCalledWith('b');
  });

  it('rolls the optimistic patch back when the request fails', async () => {
    (downloadService.pauseDownload as ReturnType<typeof vi.fn>).mockRejectedValueOnce(
      new Error('boom')
    );
    const { result, read } = setup([
      item({ hash: 'b', status: 'downloading', parseStatus: { status: 'completed' } }),
    ]);

    act(() => result.current.pause.mutate(['b']));
    await waitFor(() => expect(result.current.pause.isError).toBe(true));

    // rolled back to the pre-mutation snapshot (status restored, parseStatus intact)
    const b = read().items.find((i) => i.hash === 'b');
    expect(b?.status).toBe('downloading');
    expect(b?.parseStatus?.status).toBe('completed');
  });
});
