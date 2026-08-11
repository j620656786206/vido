import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useGenerationCandidatesProgress } from './useGenerationCandidatesProgress';

// Mock EventSource (mirrors useGenerationBatchProgress.spec pattern)
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

/** Full SSE Event struct as the backend writes the data: line (double-nested). */
const wireEvent = (payload: Record<string, unknown>) => ({
  id: 'uuid-1',
  type: 'generation_candidates_progress',
  data: {
    status: 'analyzing',
    analyzed: 234,
    total: 1247,
    ...payload,
  },
});

describe('useGenerationCandidatesProgress (lazy SSE, sub-4-3 AC #1)', () => {
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
    const { result } = renderHook(() => useGenerationCandidatesProgress());
    await vi.advanceTimersByTimeAsync(0);
    expect(MockEventSource.instances).toHaveLength(0);
    expect(result.current.status).toBe('idle');
  });

  it('[P0] opens only after startTracking, seedable from a GET snapshot', () => {
    const { result } = renderHook(() => useGenerationCandidatesProgress());

    act(() => result.current.startTracking({ analyzed: 10, total: 100 }));

    expect(MockEventSource.instances).toHaveLength(1);
    expect(MockEventSource.instances[0].url).toBe('/api/v1/events');
    expect(result.current.status).toBe('analyzing');
    expect(result.current.progress.analyzed).toBe(10);
    expect(result.current.progress.total).toBe(100);
  });

  it('[P0] unwraps the double-nested envelope and updates counts', () => {
    const { result } = renderHook(() => useGenerationCandidatesProgress());
    act(() => result.current.startTracking());
    const es = MockEventSource.instances[0];

    act(() => es.emit('generation_candidates_progress', wireEvent({})));

    expect(result.current.progress).toEqual({
      status: 'analyzing',
      analyzed: 234,
      total: 1247,
      error: null,
    });
  });

  it.each(['ready', 'cancelled', 'error'] as const)(
    '[P0] terminal status %s closes the stream',
    (terminal) => {
      const { result } = renderHook(() => useGenerationCandidatesProgress());
      act(() => result.current.startTracking());
      const es = MockEventSource.instances[0];

      act(() =>
        es.emit(
          'generation_candidates_progress',
          wireEvent({ status: terminal, analyzed: 1247, total: 1247 })
        )
      );

      expect(result.current.status).toBe(terminal);
      expect(es.readyState).toBe(2);
    }
  );

  it('reconnects with 10s backoff on error; a terminal on the fresh stream closes it with no further reconnects', () => {
    const { result } = renderHook(() => useGenerationCandidatesProgress());
    act(() => result.current.startTracking());
    const first = MockEventSource.instances[0];

    act(() => first.triggerError());
    expect(MockEventSource.instances).toHaveLength(1);

    act(() => {
      vi.advanceTimersByTime(10_000);
    });
    expect(MockEventSource.instances).toHaveLength(2);

    const second = MockEventSource.instances[1];
    act(() =>
      second.emit(
        'generation_candidates_progress',
        wireEvent({ status: 'ready', analyzed: 3, total: 3 })
      )
    );
    expect(second.readyState).toBe(2);
    act(() => {
      vi.advanceTimersByTime(30_000);
    });
    expect(MockEventSource.instances).toHaveLength(2); // closed stream stays closed
  });

  it('[P0 CR M3] startTracking after unmount opens NOTHING (no leaked EventSource)', () => {
    const { result, unmount } = renderHook(() => useGenerationCandidatesProgress());
    unmount();
    act(() => result.current.startTracking());
    expect(MockEventSource.instances).toHaveLength(0);
  });

  it('reset returns to idle and closes the stream', () => {
    const { result } = renderHook(() => useGenerationCandidatesProgress());
    act(() => result.current.startTracking());
    const es = MockEventSource.instances[0];

    act(() => result.current.reset());

    expect(result.current.status).toBe('idle');
    expect(es.readyState).toBe(2);
  });

  it('cleans up on unmount', () => {
    const { result, unmount } = renderHook(() => useGenerationCandidatesProgress());
    act(() => result.current.startTracking());
    const es = MockEventSource.instances[0];
    unmount();
    expect(es.readyState).toBe(2);
  });
});
