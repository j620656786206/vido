import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import {
  ENRICHMENT_CONNECT_TIMEOUT_MS,
  ENRICHMENT_POLL_INTERVAL_MS,
  useEnrichmentRefresh,
} from './useEnrichmentRefresh';
import { libraryKeys } from './useLibrary';
import { scannerService, type EnrichStatus } from '../services/scannerService';

vi.mock('../services/scannerService', () => ({
  scannerService: { getSSEUrl: () => '/api/v1/events', getEnrichStatus: vi.fn() },
}));

const status = (over: Partial<EnrichStatus>): EnrichStatus => ({
  total: 0,
  processed: 0,
  succeeded: 0,
  failed: 0,
  skipped: 0,
  currentTitle: '',
  isActive: true,
  ...over,
});

class MockEventSource {
  static instances: MockEventSource[] = [];
  listeners: Record<string, ((e: Event) => void)[]> = {};
  closed = false;
  onopen: (() => void) | null = null;
  onerror: (() => void) | null = null;
  constructor(public url: string) {
    MockEventSource.instances.push(this);
  }
  addEventListener(type: string, cb: (e: Event) => void) {
    (this.listeners[type] ??= []).push(cb);
  }
  removeEventListener(type: string, cb: (e: Event) => void) {
    this.listeners[type] = (this.listeners[type] ?? []).filter((l) => l !== cb);
  }
  close() {
    this.closed = true;
  }
  emit(type: string, data: unknown) {
    this.listeners[type]?.forEach((cb) =>
      cb(new MessageEvent(type, { data: JSON.stringify({ type, data }) }))
    );
  }
}

describe('useEnrichmentRefresh (batch 重新解析 → background matching)', () => {
  let queryClient: QueryClient;
  const wrapper = ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);

  beforeEach(() => {
    MockEventSource.instances = [];
    (globalThis as Record<string, unknown>).EventSource = MockEventSource;
    queryClient = new QueryClient();
  });
  afterEach(() => {
    delete (globalThis as Record<string, unknown>).EventSource;
    vi.useRealTimers();
  });

  it('opens no SSE connection and reports nothing until a re-parse asks for one', () => {
    const { result } = renderHook(() => useEnrichmentRefresh(), { wrapper });
    expect(MockEventSource.instances).toHaveLength(0);
    expect(result.current.matching).toBeNull();
  });

  // disc-2026-10-reparse-dialog-misses-instant-pass AC #1: the page awaits this
  // BEFORE posting, so it must not resolve until the stream is really open.
  it('[P0] watch() resolves only once the stream is open', async () => {
    const { result } = renderHook(() => useEnrichmentRefresh(), { wrapper });
    let connected = false;
    act(() => {
      void result.current.watch().then(() => (connected = true));
    });
    expect(MockEventSource.instances).toHaveLength(1);
    expect(MockEventSource.instances[0].url).toBe('/api/v1/events');
    expect(result.current.matching?.phase).toBe('queued');
    await Promise.resolve();
    expect(connected).toBe(false);

    await act(async () => {
      MockEventSource.instances[0].onopen?.();
    });
    expect(connected).toBe(true);
  });

  it('watch() still resolves when the stream never opens (timeout) or errors', async () => {
    vi.useFakeTimers();
    const { result } = renderHook(() => useEnrichmentRefresh(), { wrapper });
    let timedOut = false;
    act(() => {
      void result.current.watch().then(() => (timedOut = true));
    });
    await act(async () => {
      vi.advanceTimersByTime(ENRICHMENT_CONNECT_TIMEOUT_MS);
    });
    expect(timedOut).toBe(true);

    let errored = false;
    act(() => {
      void result.current.watch().then(() => (errored = true));
    });
    await act(async () => {
      MockEventSource.instances[1].onerror?.();
    });
    expect(errored).toBe(true);
  });

  // AC #2: a pass that ends the instant the POST lands is still heard, because the
  // stream was open before the POST.
  it('[P0] a 完成 that arrives right after connecting reaches the dialog', async () => {
    const invalidate = vi.spyOn(queryClient, 'invalidateQueries');
    const { result } = renderHook(() => useEnrichmentRefresh(), { wrapper });
    let p!: Promise<void>;
    act(() => {
      p = result.current.watch();
    });
    const es = MockEventSource.instances[0];
    await act(async () => {
      es.onopen?.();
      await p;
    });
    act(() => {
      es.emit('enrich_complete', { total: 2, succeeded: 0, failed: 2, skipped: 0 });
    });
    expect(result.current.matching).toMatchObject({ phase: 'done', total: 2, failed: 2 });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: libraryKeys.all });
  });

  it('queued → running (enrich_progress) → done (enrich_complete, refetch) → running again', () => {
    const invalidate = vi.spyOn(queryClient, 'invalidateQueries');
    const { result, unmount } = renderHook(() => useEnrichmentRefresh(), { wrapper });
    act(() => {
      void result.current.watch();
    });
    const es = MockEventSource.instances[0];
    // Nothing heard yet: the dialog says 等待開始.
    expect(result.current.matching?.phase).toBe('queued');

    act(() => {
      es.emit('enrich_progress', {
        total: 20,
        processed: 3,
        succeeded: 2,
        failed: 1,
        skipped: 0,
        current_title: '你的名字',
        is_active: true,
      });
    });
    expect(result.current.matching).toEqual({
      phase: 'running',
      total: 20,
      processed: 3,
      succeeded: 2,
      failed: 1,
      skipped: 0,
      currentTitle: '你的名字',
    });
    expect(invalidate).not.toHaveBeenCalled();

    act(() => {
      es.emit('enrich_complete', {
        total: 20,
        succeeded: 18,
        failed: 2,
        skipped: 0,
        duration: '1m3s',
      });
    });
    expect(result.current.matching).toMatchObject({
      phase: 'done',
      total: 20,
      succeeded: 18,
      failed: 2,
    });
    expect(result.current.matching?.currentTitle).toBe('');
    expect(invalidate).toHaveBeenCalledWith({ queryKey: libraryKeys.all });

    // The rows were queued behind that pass: a second pass starts — back to 比對中.
    act(() => {
      es.emit('enrich_progress', {
        total: 5,
        processed: 5,
        succeeded: 5,
        failed: 0,
        skipped: 0,
        current_title: '瀑布',
      });
    });
    expect(result.current.matching?.phase).toBe('running');
    act(() => {
      es.emit('enrich_complete', { total: 5, succeeded: 5, failed: 0, skipped: 0 });
    });
    expect(invalidate).toHaveBeenCalledTimes(2);

    unmount();
    expect(es.closed).toBe(true);
  });

  it('a second watch() starts over from queued on a fresh connection', () => {
    const { result } = renderHook(() => useEnrichmentRefresh(), { wrapper });
    act(() => {
      void result.current.watch();
    });
    const first = MockEventSource.instances[0];
    act(() => {
      first.emit('enrich_complete', { total: 5, succeeded: 5, failed: 0, skipped: 0 });
    });
    expect(result.current.matching?.phase).toBe('done');

    act(() => {
      void result.current.watch();
    });
    expect(first.closed).toBe(true);
    expect(MockEventSource.instances).toHaveLength(2);
    expect(result.current.matching?.phase).toBe('queued');
  });

  // AC #3: nothing was queued → no stream left open, no dialog state.
  it('stop() closes the stream and clears the state', () => {
    const { result } = renderHook(() => useEnrichmentRefresh(), { wrapper });
    act(() => {
      void result.current.watch();
    });
    act(() => result.current.stop());
    expect(MockEventSource.instances[0].closed).toBe(true);
    expect(result.current.matching).toBeNull();
  });

  it('a malformed progress payload is ignored, not a crash', () => {
    const { result } = renderHook(() => useEnrichmentRefresh(), { wrapper });
    act(() => {
      void result.current.watch();
    });
    const es = MockEventSource.instances[0];
    act(() => {
      es.listeners['enrich_progress']?.forEach((cb) =>
        cb(new MessageEvent('enrich_progress', { data: 'not json' }))
      );
    });
    expect(result.current.matching?.phase).toBe('queued');
  });

  // ─── disc-2026-10-reparse-dialog-stuck-over-tunnel ───────────────────────
  // Through the Cloudflare tunnel the stream never opened, and while it hung the
  // re-parse POST's answer never came back: 處理中 0 / 1 after a 2-second pass.

  describe('when the stream does not deliver', () => {
    beforeEach(() => {
      vi.useFakeTimers();
      vi.mocked(scannerService.getEnrichStatus).mockReset();
    });

    // Let the awaited poll settle after advancing the clock.
    const tick = async (ms: number) => {
      await act(async () => {
        await vi.advanceTimersByTimeAsync(ms);
      });
    };

    it('[P0] a stream that has not opened in time is closed, not left hanging', async () => {
      const { result } = renderHook(() => useEnrichmentRefresh(), { wrapper });
      act(() => {
        void result.current.watch();
      });
      await tick(ENRICHMENT_CONNECT_TIMEOUT_MS);
      expect(MockEventSource.instances[0].closed).toBe(true);
    });

    it('an error before opening closes it; after opening the browser reconnects', async () => {
      const { result } = renderHook(() => useEnrichmentRefresh(), { wrapper });
      act(() => {
        void result.current.watch();
      });
      await act(async () => MockEventSource.instances[0].onerror?.());
      expect(MockEventSource.instances[0].closed).toBe(true);

      act(() => {
        void result.current.watch();
      });
      const es = MockEventSource.instances[1];
      await act(async () => es.onopen?.());
      await act(async () => es.onerror?.());
      expect(es.closed).toBe(false);
    });

    it('[P0] track() polls the pass status: running, then 完成 with the result, then stops', async () => {
      const invalidate = vi.spyOn(queryClient, 'invalidateQueries');
      vi.mocked(scannerService.getEnrichStatus)
        .mockResolvedValueOnce(status({ total: 1, processed: 0, currentTitle: '將夜' }))
        .mockResolvedValueOnce(status({ total: 1, processed: 1, succeeded: 1, isActive: false }));
      const { result } = renderHook(() => useEnrichmentRefresh(), { wrapper });
      act(() => {
        void result.current.watch();
      });
      await tick(ENRICHMENT_CONNECT_TIMEOUT_MS); // never opened
      act(() => result.current.track());

      await tick(ENRICHMENT_POLL_INTERVAL_MS);
      expect(result.current.matching).toMatchObject({
        phase: 'running',
        total: 1,
        currentTitle: '將夜',
      });

      await tick(ENRICHMENT_POLL_INTERVAL_MS);
      expect(result.current.matching).toMatchObject({ phase: 'done', total: 1, succeeded: 1 });
      expect(invalidate).toHaveBeenCalledWith({ queryKey: libraryKeys.all });

      await tick(ENRICHMENT_POLL_INTERVAL_MS * 3);
      expect(scannerService.getEnrichStatus).toHaveBeenCalledTimes(2);
    });

    it('a failed poll is asked again', async () => {
      vi.mocked(scannerService.getEnrichStatus)
        .mockRejectedValueOnce(new Error('network'))
        .mockResolvedValueOnce(status({ total: 2, succeeded: 2, isActive: false }));
      const { result } = renderHook(() => useEnrichmentRefresh(), { wrapper });
      act(() => {
        void result.current.watch();
      });
      act(() => result.current.track());
      await tick(ENRICHMENT_POLL_INTERVAL_MS * 2);
      expect(result.current.matching).toMatchObject({ phase: 'done', total: 2, succeeded: 2 });
    });

    it('an empty follow-up pass (zeros) keeps the numbers this pass showed', async () => {
      vi.mocked(scannerService.getEnrichStatus)
        .mockResolvedValueOnce(status({ total: 3, processed: 1, succeeded: 1 }))
        .mockResolvedValueOnce(status({ isActive: false }));
      const { result } = renderHook(() => useEnrichmentRefresh(), { wrapper });
      act(() => {
        void result.current.watch();
      });
      act(() => result.current.track());
      await tick(ENRICHMENT_POLL_INTERVAL_MS * 2);
      expect(result.current.matching).toMatchObject({ phase: 'done', total: 3, succeeded: 1 });
    });

    it('a 完成 from the stream stops the polling', async () => {
      vi.mocked(scannerService.getEnrichStatus).mockResolvedValue(status({ total: 1 }));
      const { result } = renderHook(() => useEnrichmentRefresh(), { wrapper });
      act(() => {
        void result.current.watch();
      });
      const es = MockEventSource.instances[0];
      await act(async () => es.onopen?.());
      act(() => result.current.track());
      act(() => es.emit('enrich_complete', { total: 1, succeeded: 1, failed: 0, skipped: 0 }));
      await tick(ENRICHMENT_POLL_INTERVAL_MS * 3);
      expect(scannerService.getEnrichStatus).not.toHaveBeenCalled();
      expect(result.current.matching).toMatchObject({ phase: 'done', succeeded: 1 });
    });

    it('stop() and unmount end the polling', async () => {
      vi.mocked(scannerService.getEnrichStatus).mockResolvedValue(status({ total: 1 }));
      const { result, unmount } = renderHook(() => useEnrichmentRefresh(), { wrapper });
      act(() => {
        void result.current.watch();
      });
      act(() => result.current.track());
      act(() => result.current.stop());
      await tick(ENRICHMENT_POLL_INTERVAL_MS * 2);
      expect(scannerService.getEnrichStatus).not.toHaveBeenCalled();

      act(() => {
        void result.current.watch();
      });
      act(() => result.current.track());
      unmount();
      await tick(ENRICHMENT_POLL_INTERVAL_MS * 2);
      expect(scannerService.getEnrichStatus).not.toHaveBeenCalled();
    });
  });
});
