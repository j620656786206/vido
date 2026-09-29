import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useScanProgress, requestScanTracking, subscribeScanTracking } from './useScanProgress';

// Mock scannerService
const mockGetSSEUrl = vi.fn(() => '/api/v1/events');

vi.mock('../services/scannerService', () => ({
  scannerService: {
    getSSEUrl: () => mockGetSSEUrl(),
  },
}));

// Mock EventSource
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
    // noop for tests
  }

  close() {
    this.readyState = 2;
  }

  // Test helpers
  emit(type: string, data?: unknown) {
    const event =
      data !== undefined ? new MessageEvent(type, { data: JSON.stringify(data) }) : new Event(type);
    this.listeners[type]?.forEach((cb) => cb(event));
  }

  triggerError() {
    if (this.onerror) this.onerror(new Event('error'));
  }
}

describe('useScanProgress (SSE-only, no polling)', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    MockEventSource.instances = [];
    (global as Record<string, unknown>).EventSource = MockEventSource;
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('[P0] does NOT create EventSource on mount (lazy SSE)', async () => {
    renderHook(() => useScanProgress());
    await vi.advanceTimersByTimeAsync(0);
    expect(MockEventSource.instances).toHaveLength(0);
  });

  it('[P0] does NOT poll scanner status — SSE only', async () => {
    renderHook(() => useScanProgress());
    await vi.advanceTimersByTimeAsync(30000);
    // No scannerService.getScanStatus calls — it's no longer imported for polling
    expect(MockEventSource.instances).toHaveLength(0);
  });

  it('[P0] starts as not visible when idle', async () => {
    const { result } = renderHook(() => useScanProgress());
    await vi.advanceTimersByTimeAsync(0);
    expect(result.current.isVisible).toBe(false);
    expect(result.current.isScanning).toBe(false);
  });

  it('[P0] connects SSE via startTracking', async () => {
    const { result } = renderHook(() => useScanProgress());
    await vi.advanceTimersByTimeAsync(0);
    expect(MockEventSource.instances).toHaveLength(0);

    act(() => {
      void result.current.startTracking();
    });
    expect(MockEventSource.instances).toHaveLength(1);
    expect(MockEventSource.instances[0].url).toBe('/api/v1/events');
  });

  it('[P0] updates state on SSE scan_progress event', async () => {
    const { result } = renderHook(() => useScanProgress());
    act(() => {
      void result.current.startTracking();
    });

    const es = MockEventSource.instances[0];
    act(() => {
      es.emit('scan_progress', {
        data: {
          filesFound: 500,
          currentFile: 'test.mkv',
          percentDone: 42,
          errorCount: 2,
          estimatedTime: '1 分 30 秒',
        },
      });
    });

    expect(result.current.isScanning).toBe(true);
    expect(result.current.isVisible).toBe(true);
    expect(result.current.percentDone).toBe(42);
    expect(result.current.filesFound).toBe(500);
    expect(result.current.currentFile).toBe('test.mkv');
  });

  it('[P0] handles scan_complete SSE event', async () => {
    const { result } = renderHook(() => useScanProgress());
    act(() => {
      void result.current.startTracking();
    });

    const es = MockEventSource.instances[0];
    act(() => {
      es.emit('scan_progress', {
        data: {
          filesFound: 100,
          currentFile: 'a.mkv',
          percentDone: 50,
          errorCount: 0,
          estimatedTime: '30 秒',
        },
      });
    });

    act(() => {
      es.emit('scan_complete', {
        data: {
          filesFound: 200,
          filesCreated: 150,
          filesUpdated: 41,
          filesUnmatched: 8,
          errorCount: 1,
        },
      });
    });

    expect(result.current.isScanning).toBe(false);
    expect(result.current.isComplete).toBe(true);
    expect(result.current.percentDone).toBe(100);
    // dsr-5: the toast prints what the scanner wrote, straight from the payload.
    expect(result.current.filesCreated).toBe(150);
    expect(result.current.filesUpdated).toBe(41);
    expect(result.current.filesUnmatched).toBe(8);
  });

  it('[P1] handles scan_cancelled SSE event', async () => {
    const { result } = renderHook(() => useScanProgress());
    act(() => {
      void result.current.startTracking();
    });

    const es = MockEventSource.instances[0];
    act(() => {
      es.emit('scan_progress', {
        data: {
          filesFound: 50,
          currentFile: 'x.mkv',
          percentDone: 25,
          errorCount: 0,
          estimatedTime: '1 分',
        },
      });
    });

    act(() => {
      es.emit('scan_cancelled', undefined);
    });

    expect(result.current.isScanning).toBe(false);
    expect(result.current.isCancelled).toBe(true);
    expect(result.current.isVisible).toBe(true);
  });

  it('[P1] sets disconnected on SSE error (no polling fallback)', async () => {
    const { result } = renderHook(() => useScanProgress());
    act(() => {
      void result.current.startTracking();
    });

    const es = MockEventSource.instances[0];
    act(() => {
      es.triggerError();
    });

    expect(result.current.connectionStatus).toBe('disconnected');
  });

  it('[P1] toggles minimize state', async () => {
    const { result } = renderHook(() => useScanProgress());
    expect(result.current.isMinimized).toBe(false);
    act(() => result.current.toggleMinimize());
    expect(result.current.isMinimized).toBe(true);
    act(() => result.current.toggleMinimize());
    expect(result.current.isMinimized).toBe(false);
  });

  it('[P1] dismiss hides the card', async () => {
    const { result } = renderHook(() => useScanProgress());
    act(() => {
      void result.current.startTracking();
    });

    const es = MockEventSource.instances[0];
    act(() => {
      es.emit('scan_complete', { data: { filesFound: 10, errorCount: 0 } });
    });
    expect(result.current.isVisible).toBe(true);

    act(() => result.current.dismiss());
    expect(result.current.isVisible).toBe(false);
  });

  it('[P1] closes EventSource on unmount', async () => {
    const { result, unmount } = renderHook(() => useScanProgress());
    act(() => {
      void result.current.startTracking();
    });

    const es = MockEventSource.instances[0];
    unmount();

    expect(es.readyState).toBe(2); // CLOSED
  });

  // bugfix-scan-progress-sse-unwired: the scan trigger (ScannerSettings) and the
  // shell card (ScanProgress) are separate hook instances; the trigger bridges to
  // the card via this module-level signal. Without it the card never connects.
  it('[P0] requestScanTracking opens the SSE on a subscribed instance', async () => {
    const { result } = renderHook(() => useScanProgress());
    const unsubscribe = subscribeScanTracking(result.current.startTracking);
    await vi.advanceTimersByTimeAsync(0);
    expect(MockEventSource.instances).toHaveLength(0); // lazy — nothing yet

    act(() => {
      void requestScanTracking();
    }); // the trigger fires the signal
    expect(MockEventSource.instances).toHaveLength(1); // SSE now open

    unsubscribe();
  });

  // bugfix-scan-instant-completion-no-feedback AC #1: the trigger waits for the
  // stream to be CONNECTED before POSTing — a tiny library finishes in
  // milliseconds, and events broadcast before the client registers are lost.
  describe('requestScanTracking waits until the stream is connected', () => {
    function settledFlag(p: Promise<void>) {
      const flag = { done: false };
      void p.then(() => {
        flag.done = true;
      });
      return flag;
    }

    it('resolves on the server’s `connected` event, not before', async () => {
      const { result } = renderHook(() => useScanProgress());
      const unsubscribe = subscribeScanTracking(result.current.startTracking);
      let flag = { done: false };
      act(() => {
        flag = settledFlag(requestScanTracking());
      });
      await vi.advanceTimersByTimeAsync(100);
      expect(flag.done).toBe(false);

      act(() => MockEventSource.instances[0].emit('connected', { clientId: 'c1' }));
      await vi.advanceTimersByTimeAsync(0);
      expect(flag.done).toBe(true);
      unsubscribe();
    });

    it('also resolves when the connection opens', async () => {
      const { result } = renderHook(() => useScanProgress());
      const unsubscribe = subscribeScanTracking(result.current.startTracking);
      let flag = { done: false };
      act(() => {
        flag = settledFlag(requestScanTracking());
      });
      const es = MockEventSource.instances[0] as unknown as { onopen: ((e: Event) => void) | null };
      act(() => es.onopen?.(new Event('open')));
      await vi.advanceTimersByTimeAsync(0);
      expect(flag.done).toBe(true);
      unsubscribe();
    });

    it('gives up waiting after 3 s so a broken stream never blocks scanning', async () => {
      const { result } = renderHook(() => useScanProgress());
      const unsubscribe = subscribeScanTracking(result.current.startTracking);
      let flag = { done: false };
      act(() => {
        flag = settledFlag(requestScanTracking());
      });
      await vi.advanceTimersByTimeAsync(2900);
      expect(flag.done).toBe(false);
      await vi.advanceTimersByTimeAsync(200);
      expect(flag.done).toBe(true);
      unsubscribe();
    });

    it('an error before connecting releases the wait at once (no 3 s stall)', async () => {
      const { result } = renderHook(() => useScanProgress());
      const unsubscribe = subscribeScanTracking(result.current.startTracking);
      let flag = { done: false };
      act(() => {
        flag = settledFlag(requestScanTracking());
      });
      act(() => MockEventSource.instances[0].triggerError());
      await vi.advanceTimersByTimeAsync(0);
      expect(flag.done).toBe(true);
      unsubscribe();
    });

    it('a click during the reconnect gap keeps its new stream (old timer does not close it)', async () => {
      const { result } = renderHook(() => useScanProgress());
      const unsubscribe = subscribeScanTracking(result.current.startTracking);
      act(() => {
        void requestScanTracking();
      });
      act(() => MockEventSource.instances[0].emit('connected', { clientId: 'c1' }));
      act(() => MockEventSource.instances[0].triggerError()); // stream drops; reconnect in 10 s

      act(() => {
        void requestScanTracking(); // user clicks during the gap
      });
      expect(MockEventSource.instances).toHaveLength(2);
      const fresh = MockEventSource.instances[1];

      await vi.advanceTimersByTimeAsync(15_000);
      expect(fresh.readyState).not.toBe(2); // not closed by the stale timer
      expect(MockEventSource.instances).toHaveLength(2);
      unsubscribe();
    });

    it('resolves at once when nothing is listening', async () => {
      const flag = settledFlag(requestScanTracking());
      await vi.advanceTimersByTimeAsync(0);
      expect(flag.done).toBe(true);
    });

    it('resolves at once when the stream is already connected', async () => {
      const { result } = renderHook(() => useScanProgress());
      const unsubscribe = subscribeScanTracking(result.current.startTracking);
      act(() => {
        void requestScanTracking();
      });
      act(() => MockEventSource.instances[0].emit('connected', { clientId: 'c1' }));
      await vi.advanceTimersByTimeAsync(0);

      let flag = { done: false };
      act(() => {
        flag = settledFlag(requestScanTracking());
      });
      await vi.advanceTimersByTimeAsync(0);
      expect(flag.done).toBe(true);
      expect(MockEventSource.instances).toHaveLength(1); // no second stream
      unsubscribe();
    });
  });

  it('[P1] unsubscribe stops an instance from reacting to the signal', async () => {
    const { result } = renderHook(() => useScanProgress());
    const unsubscribe = subscribeScanTracking(result.current.startTracking);
    unsubscribe();

    act(() => {
      void requestScanTracking();
    });
    expect(MockEventSource.instances).toHaveLength(0);
  });
});
