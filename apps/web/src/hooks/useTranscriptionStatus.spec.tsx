import { renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { transcriptionStatusKeys, useTranscriptionStatus } from './useTranscriptionStatus';
import { transcriptionService } from '../services/transcriptionService';

// bugfix-dialog-reopen-shows-idle-during-run — "is this one generating right now?"

vi.mock('../services/transcriptionService', () => ({
  transcriptionService: { getTranscriptionStatus: vi.fn() },
}));

const mockedGet = vi.mocked(transcriptionService.getTranscriptionStatus);

function wrapper() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return {
    queryClient,
    Wrapper: ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    ),
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  mockedGet.mockResolvedValue({ inProgress: true });
});

describe('useTranscriptionStatus', () => {
  it('asks about THIS media under a per-item key', async () => {
    const { queryClient, Wrapper } = wrapper();
    const { result } = renderHook(
      () => useTranscriptionStatus('episode', 'ep-1', { enabled: true }),
      { wrapper: Wrapper }
    );

    await waitFor(() => expect(result.current.data).toEqual({ inProgress: true }));
    expect(mockedGet).toHaveBeenCalledWith('episode', 'ep-1', expect.any(AbortSignal));
    expect(queryClient.getQueryData(transcriptionStatusKeys.item('episode', 'ep-1'))).toEqual({
      inProgress: true,
    });
  });

  it('does not ask while disabled (dialog closed) or for a series (no generate route)', async () => {
    const { Wrapper } = wrapper();
    renderHook(() => useTranscriptionStatus('movie', 'm-1', { enabled: false }), {
      wrapper: Wrapper,
    });
    renderHook(() => useTranscriptionStatus(null, 's-1', { enabled: true }), {
      wrapper: Wrapper,
    });

    await new Promise((r) => setTimeout(r, 0));
    expect(mockedGet).not.toHaveBeenCalled();
  });
});
