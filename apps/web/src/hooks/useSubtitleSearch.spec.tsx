import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

vi.mock('../services/subtitleService', () => ({
  subtitleService: {
    searchSubtitles: vi.fn(),
    downloadSubtitle: vi.fn(() => new Promise(() => {})), // stays in flight
    previewSubtitle: vi.fn(),
  },
}));

import { useSubtitleSearch } from './useSubtitleSearch';

// disc-2026-11-online-sub-download-progress: the ring follows `percent` from
// subtitle_progress events — for THIS download's media only, never backwards.
class FakeEventSource {
  static last: FakeEventSource | null = null;
  listeners: Record<string, (e: MessageEvent) => void> = {};
  onerror: (() => void) | null = null;
  constructor() {
    FakeEventSource.last = this;
  }
  addEventListener(type: string, fn: (e: MessageEvent) => void) {
    this.listeners[type] = fn;
  }
  close() {}
  emit(data: Record<string, unknown>) {
    this.listeners['subtitle_progress']?.({ data: JSON.stringify({ data }) } as MessageEvent);
  }
}

describe('useSubtitleSearch — download progress', () => {
  beforeEach(() => {
    vi.stubGlobal('EventSource', FakeEventSource);
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('tracks percent for the downloading media only, and never goes backwards', async () => {
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    );
    const { result } = renderHook(() => useSubtitleSearch(), { wrapper });
    expect(result.current.downloadProgress).toBeNull();

    act(() => {
      result.current.download({
        mediaId: 'ep-2',
        mediaType: 'episode',
        mediaFilePath: '/x.mkv',
        subtitleId: 's1',
        provider: 'assrt',
      });
    });
    await waitFor(() => expect(FakeEventSource.last).not.toBeNull());
    const es = FakeEventSource.last!;

    act(() => es.emit({ media_id: 'ep-2', stage: 'downloading', percent: 25 }));
    expect(result.current.downloadProgress).toBe(25);

    act(() => es.emit({ media_id: 'other', stage: 'downloading', percent: 90 }));
    expect(result.current.downloadProgress).toBe(25);

    act(() => es.emit({ media_id: 'ep-2', stage: 'downloading', percent: 10 }));
    expect(result.current.downloadProgress).toBe(25);

    act(() => es.emit({ media_id: 'ep-2', stage: 'placing', percent: 95 }));
    expect(result.current.downloadProgress).toBe(95);
  });
});
