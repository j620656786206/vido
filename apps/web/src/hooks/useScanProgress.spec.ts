import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useScanProgress } from './useScanProgress';

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

    act(() => result.current.startTracking());
    expect(MockEventSource.instances).toHaveLength(1);
    expect(MockEventSource.instances[0].url).toBe('/api/v1/events');
  });

  it('[P0] updates state on SSE scan_progress event', async () => {
    const { result } = renderHook(() => useScanProgress());
    act(() => result.current.startTracking());

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
    act(() => result.current.startTracking());

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
        data: { filesFound: 200, errorCount: 1 },
      });
    });

    expect(result.current.isScanning).toBe(false);
    expect(result.current.isComplete).toBe(true);
    expect(result.current.percentDone).toBe(100);
  });

  it('[P1] handles scan_cancelled SSE event', async () => {
    const { result } = renderHook(() => useScanProgress());
    act(() => result.current.startTracking());

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
    act(() => result.current.startTracking());

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
    act(() => result.current.startTracking());

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
    act(() => result.current.startTracking());

    const es = MockEventSource.instances[0];
    unmount();

    expect(es.readyState).toBe(2); // CLOSED
  });
});
