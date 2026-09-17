import { renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { transcriptionEstimateKeys, useTranscriptionEstimate } from './useTranscriptionEstimate';
import { transcriptionService, type TranscriptionEstimate } from '../services/transcriptionService';

// Story dsr-6a AC #5 — the price query behind the paid 生成字幕 / 重試 buttons.

vi.mock('../services/transcriptionService', () => ({
  transcriptionService: { getTranscriptionEstimate: vi.fn() },
}));

const mockedGet = vi.mocked(transcriptionService.getTranscriptionEstimate);

const ESTIMATE: TranscriptionEstimate = {
  mediaId: 'ep-1',
  mediaType: 'episode',
  plan: 'full',
  asrAvailable: true,
  selfHostedAsr: false,
  translationConfigured: true,
  modelId: 'claude-sonnet-5',
  runtimeMinutes: 48,
  runtimeKnown: true,
  runtimeSource: 'ffprobe',
  estimatedUsd: 0.67,
};

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
  mockedGet.mockResolvedValue(ESTIMATE);
});

describe('useTranscriptionEstimate', () => {
  it('asks for THIS media, passing the abort signal, under a per-item key', async () => {
    const { queryClient, Wrapper } = wrapper();
    const { result } = renderHook(
      () => useTranscriptionEstimate('episode', 'ep-1', { enabled: true }),
      { wrapper: Wrapper }
    );

    await waitFor(() => expect(result.current.data).toEqual(ESTIMATE));
    expect(mockedGet).toHaveBeenCalledWith('episode', 'ep-1', expect.any(AbortSignal));
    expect(queryClient.getQueryData(transcriptionEstimateKeys.item('episode', 'ep-1'))).toEqual(
      ESTIMATE
    );
  });

  it('does not ask while disabled (closed dialog)', () => {
    const { Wrapper } = wrapper();
    renderHook(() => useTranscriptionEstimate('movie', 'm-1', { enabled: false }), {
      wrapper: Wrapper,
    });
    expect(mockedGet).not.toHaveBeenCalled();
  });

  it('never asks for a series — it has no generate route to price', () => {
    const { Wrapper } = wrapper();
    renderHook(() => useTranscriptionEstimate(null, 'series-1', { enabled: true }), {
      wrapper: Wrapper,
    });
    expect(mockedGet).not.toHaveBeenCalled();
  });
});
