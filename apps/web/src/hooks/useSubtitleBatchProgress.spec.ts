import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useSubtitleBatchProgress } from './useSubtitleBatchProgress';

// Mock EventSource (mirrors useScanProgress.spec pattern)
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

const PROGRESS = (overrides: Record<string, unknown> = {}) => ({
  data: {
    batch_id: 'b-1',
    total_items: 10,
    current_index: 3,
    current_item: '電影 A',
    success_count: 2,
    fail_count: 1,
    status: 'running',
    ...overrides,
  },
});

describe('useSubtitleBatchProgress (lazy SSE)', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    MockEventSource.instances = [];
    (global as Record<string, unknown>).EventSource = MockEventSource;
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('[P0] does NOT open EventSource on mount (lazy SSE)', async () => {
    renderHook(() => useSubtitleBatchProgress());
    await vi.advanceTimersByTimeAsync(0);
    expect(MockEventSource.instances).toHaveLength(0);
  });

  it('[P0] starts idle', async () => {
    const { result } = renderHook(() => useSubtitleBatchProgress());
    expect(result.current.status).toBe('idle');
  });

  it('[P0] opens EventSource only after startTracking, seeding totals', async () => {
    const { result } = renderHook(() => useSubtitleBatchProgress());
    expect(MockEventSource.instances).toHaveLength(0);

    act(() => result.current.startTracking({ batchId: 'b-1', totalItems: 10 }));

    expect(MockEventSource.instances).toHaveLength(1);
    expect(MockEventSource.instances[0].url).toBe('/api/v1/events');
    expect(result.current.status).toBe('running');
    expect(result.current.progress.totalItems).toBe(10);
  });

  it('[P0] reduces a subtitle_batch_progress event', async () => {
    const { result } = renderHook(() => useSubtitleBatchProgress());
    act(() => result.current.startTracking());

    const es = MockEventSource.instances[0];
    act(() => es.emit('subtitle_batch_progress', PROGRESS()));

    expect(result.current.progress.currentIndex).toBe(3);
    expect(result.current.progress.currentItem).toBe('電影 A');
    expect(result.current.progress.successCount).toBe(2);
    expect(result.current.progress.failCount).toBe(1);
    expect(result.current.status).toBe('running');
  });

  it('[P0] closes the stream on the terminal complete event', async () => {
    const { result } = renderHook(() => useSubtitleBatchProgress());
    act(() => result.current.startTracking());

    const es = MockEventSource.instances[0];
    act(() => es.emit('subtitle_batch_progress', PROGRESS()));
    act(() =>
      es.emit('subtitle_batch_progress', {
        data: {
          batch_id: 'b-1',
          total_items: 10,
          current_index: 10,
          success_count: 8,
          fail_count: 2,
          status: 'complete',
        },
      })
    );

    expect(result.current.status).toBe('complete');
    expect(result.current.progress.successCount).toBe(8);
    expect(es.readyState).toBe(2); // CLOSED
  });

  it('[P1] handles a cancelled terminal event', async () => {
    const { result } = renderHook(() => useSubtitleBatchProgress());
    act(() => result.current.startTracking());

    const es = MockEventSource.instances[0];
    act(() => es.emit('subtitle_batch_progress', PROGRESS({ status: 'cancelled' })));

    expect(result.current.status).toBe('cancelled');
    expect(es.readyState).toBe(2);
  });

  it('[P0] does NOT dispatch after unmount', async () => {
    const { result, unmount } = renderHook(() => useSubtitleBatchProgress());
    act(() => result.current.startTracking());
    const es = MockEventSource.instances[0];

    const before = result.current.progress.currentIndex;
    unmount();
    // Emitting after unmount must not throw or mutate observable state.
    act(() => es.emit('subtitle_batch_progress', PROGRESS({ current_index: 9 })));

    expect(result.current.progress.currentIndex).toBe(before);
    expect(es.readyState).toBe(2); // closed on unmount
  });

  it('[P1] reset returns to idle and closes the stream', async () => {
    const { result } = renderHook(() => useSubtitleBatchProgress());
    act(() => result.current.startTracking({ totalItems: 5 }));
    const es = MockEventSource.instances[0];

    act(() => result.current.reset());

    expect(result.current.status).toBe('idle');
    expect(result.current.progress.totalItems).toBe(0);
    expect(es.readyState).toBe(2);
  });

  it('[P1] schedules a reconnect on SSE error while running', async () => {
    const { result } = renderHook(() => useSubtitleBatchProgress());
    act(() => result.current.startTracking());
    const es = MockEventSource.instances[0];

    act(() => es.triggerError());
    // Backoff reconnect after 10s opens a fresh EventSource.
    await act(async () => {
      await vi.advanceTimersByTimeAsync(10000);
    });

    expect(MockEventSource.instances.length).toBeGreaterThanOrEqual(2);
  });
});
