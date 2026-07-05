import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useGenerationBatchProgress } from './useGenerationBatchProgress';

// Mock EventSource (mirrors useGenerationProgress.spec pattern)
class MockEventSource {
  static instances: MockEventSource[] = [];
  url: string;
  listeners: Record<string, ((e: MessageEvent | Event) => void)[]> = {};
  onerror: ((e: Event) => void) | null = null;
  readyState = 0;

  constructor(url: string) {
    this.url = url;
    MockEventSource.instances.push(this);
  }

  addEventListener(type: string, cb: (e: MessageEvent | Event) => void) {
    if (!this.listeners[type]) this.listeners[type] = [];
    this.listeners[type].push(cb);
  }

  removeEventListener() {
    // noop
  }

  close() {
    this.readyState = 2;
  }

  emit(type: string, data?: unknown) {
    const event =
      data !== undefined ? new MessageEvent(type, { data: JSON.stringify(data) }) : new Event(type);
    this.listeners[type]?.forEach((cb) => cb(event));
  }

  triggerError() {
    if (this.onerror) this.onerror(new Event('error'));
  }
}

/**
 * Build the FULL SSE Event struct exactly as the backend writes the `data:`
 * line (sse/handler.go `sendSSEEvent(w, type, event)`): the payload is
 * DOUBLE-nested under `.data`, snake_case, all 11 contract keys
 * (9R-16 [@contract-v1] AC #9).
 */
const wireEvent = (payload: Record<string, unknown>) => ({
  id: 'uuid-1',
  type: 'generation_batch_progress',
  data: {
    batch_id: 'gb-1',
    total_items: 38,
    current_index: 12,
    current_media_id: 42,
    current_item: '怪奇物語',
    success_count: 11,
    fail_count: 0,
    paused_count: 0,
    status: 'running',
    spent_usd: 0.42,
    budget_usd: 5,
    ...payload,
  },
});

describe('useGenerationBatchProgress (lazy SSE, double-nested envelope)', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    MockEventSource.instances = [];
    (global as Record<string, unknown>).EventSource = MockEventSource;
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('[P0] does NOT open EventSource on mount (lazy SSE §8)', async () => {
    const { result } = renderHook(() => useGenerationBatchProgress());
    await vi.advanceTimersByTimeAsync(0);
    expect(MockEventSource.instances).toHaveLength(0);
    expect(result.current.status).toBe('idle');
  });

  it('[P0] opens EventSource only after startTracking and enters running (with seed)', () => {
    const { result } = renderHook(() => useGenerationBatchProgress());

    act(() => result.current.startTracking({ batchId: 'gb-1', totalItems: 38 }));

    expect(MockEventSource.instances).toHaveLength(1);
    expect(MockEventSource.instances[0].url).toBe('/api/v1/events');
    expect(result.current.status).toBe('running');
    expect(result.current.progress.batchId).toBe('gb-1');
    expect(result.current.progress.totalItems).toBe(38);
  });

  it('[P0] unwraps the DOUBLE-nested Event envelope — payload is parsed.data, snakeToCamel applied', () => {
    const { result } = renderHook(() => useGenerationBatchProgress());
    act(() => result.current.startTracking());
    const es = MockEventSource.instances[0];

    // The data: line is the full Event struct {id,type,data} — NOT the bare payload.
    act(() => es.emit('generation_batch_progress', wireEvent({})));

    expect(result.current.progress).toEqual({
      batchId: 'gb-1',
      totalItems: 38,
      currentIndex: 12,
      currentMediaId: 42,
      currentItem: '怪奇物語',
      successCount: 11,
      failCount: 0,
      pausedCount: 0,
      status: 'running',
      spentUsd: 0.42,
      budgetUsd: 5,
    });
  });

  it.each(['complete', 'cancelled', 'error', 'budget_ceiling'] as const)(
    '[P0] terminal status %s closes the stream',
    (status) => {
      const { result } = renderHook(() => useGenerationBatchProgress());
      act(() => result.current.startTracking());
      const es = MockEventSource.instances[0];

      act(() =>
        es.emit(
          'generation_batch_progress',
          wireEvent({ status, paused_count: status === 'budget_ceiling' ? 26 : 0 })
        )
      );

      expect(result.current.status).toBe(status);
      expect(es.readyState).toBe(2); // closed
      if (status === 'budget_ceiling') {
        expect(result.current.progress.pausedCount).toBe(26);
      }
    }
  );

  it('[P1] budget_ceiling carries spent/budget for the F9 banner (no wall-clock)', () => {
    const { result } = renderHook(() => useGenerationBatchProgress());
    act(() => result.current.startTracking());
    const es = MockEventSource.instances[0];

    act(() =>
      es.emit(
        'generation_batch_progress',
        wireEvent({
          status: 'budget_ceiling',
          success_count: 12,
          paused_count: 26,
          spent_usd: 5,
          budget_usd: 5,
        })
      )
    );

    expect(result.current.progress.spentUsd).toBe(5);
    expect(result.current.progress.budgetUsd).toBe(5);
    expect(result.current.progress.successCount).toBe(12);
  });

  it('[P1] ignores malformed frames', () => {
    const { result } = renderHook(() => useGenerationBatchProgress());
    act(() => result.current.startTracking());
    const es = MockEventSource.instances[0];

    act(() => {
      const bad = new MessageEvent('generation_batch_progress', { data: '{not json' });
      es.listeners['generation_batch_progress']?.forEach((cb) => cb(bad));
    });

    expect(result.current.status).toBe('running'); // unchanged
  });

  it('[P1] reconnects with backoff on error and cancels the timer on a fresh connect', async () => {
    const { result } = renderHook(() => useGenerationBatchProgress());
    act(() => result.current.startTracking());
    const first = MockEventSource.instances[0];

    act(() => first.triggerError());
    expect(MockEventSource.instances).toHaveLength(1);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(10000);
    });
    expect(MockEventSource.instances).toHaveLength(2);
  });

  it('[P1] reset closes the stream and returns to idle', () => {
    const { result } = renderHook(() => useGenerationBatchProgress());
    act(() => result.current.startTracking());
    const es = MockEventSource.instances[0];

    act(() => result.current.reset());

    expect(es.readyState).toBe(2);
    expect(result.current.status).toBe('idle');
    expect(result.current.progress.totalItems).toBe(0);
  });

  it('[P1] closes the stream on unmount', () => {
    const { result, unmount } = renderHook(() => useGenerationBatchProgress());
    act(() => result.current.startTracking());
    const es = MockEventSource.instances[0];

    unmount();

    expect(es.readyState).toBe(2);
  });
});
