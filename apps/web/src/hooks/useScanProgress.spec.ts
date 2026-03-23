import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useScanProgress } from './useScanProgress';

// Mock scannerService
const mockGetScanStatus = vi.fn();
const mockGetSSEUrl = vi.fn(() => 'http://localhost:8080/api/v1/events');

vi.mock('../services/scannerService', () => ({
  scannerService: {
    getScanStatus: (...args: unknown[]) => mockGetScanStatus(...args),
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

const idleStatus = {
  is_scanning: false,
  files_found: 0,
  files_processed: 0,
  current_file: '',
  percent_done: 0,
  error_count: 0,
  estimated_time: '',
  last_scan_at: '',
  last_scan_files: 0,
  last_scan_duration: '',
};

const scanningStatus = {
  is_scanning: true,
  files_found: 200,
  files_processed: 50,
  current_file: 'init.mkv',
  percent_done: 25,
  error_count: 0,
  estimated_time: '3 分',
  last_scan_at: '',
  last_scan_files: 0,
  last_scan_duration: '',
};

describe('useScanProgress', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    MockEventSource.instances = [];
    (global as Record<string, unknown>).EventSource = MockEventSource;
    mockGetScanStatus.mockResolvedValue(idleStatus);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('does NOT create EventSource when idle (lazy SSE)', async () => {
    renderHook(() => useScanProgress());
    await vi.advanceTimersByTimeAsync(0);
    expect(MockEventSource.instances).toHaveLength(0);
  });

  it('creates EventSource when scan is active on mount', async () => {
    mockGetScanStatus.mockResolvedValue(scanningStatus);
    renderHook(() => useScanProgress());
    await vi.advanceTimersByTimeAsync(0);
    expect(MockEventSource.instances).toHaveLength(1);
    expect(MockEventSource.instances[0].url).toBe('http://localhost:8080/api/v1/events');
  });

  it('fetches initial status on mount', async () => {
    renderHook(() => useScanProgress());
    await vi.advanceTimersByTimeAsync(0);
    expect(mockGetScanStatus).toHaveBeenCalledTimes(1);
  });

  it('starts as not visible when idle', async () => {
    const { result } = renderHook(() => useScanProgress());
    await vi.advanceTimersByTimeAsync(0);
    expect(result.current.isVisible).toBe(false);
    expect(result.current.isScanning).toBe(false);
  });

  it('updates state on SSE scan_progress event', async () => {
    mockGetScanStatus.mockResolvedValue(scanningStatus);
    const { result } = renderHook(() => useScanProgress());
    await vi.advanceTimersByTimeAsync(0);

    const es = MockEventSource.instances[0];
    act(() => {
      es.emit('scan_progress', {
        data: {
          files_found: 500,
          current_file: 'test.mkv',
          percent_done: 42,
          error_count: 2,
          estimated_time: '1 分 30 秒',
        },
      });
    });

    expect(result.current.isScanning).toBe(true);
    expect(result.current.isVisible).toBe(true);
    expect(result.current.percentDone).toBe(42);
    expect(result.current.filesFound).toBe(500);
    expect(result.current.currentFile).toBe('test.mkv');
    expect(result.current.errorCount).toBe(2);
    expect(result.current.estimatedTime).toBe('1 分 30 秒');
  });

  it('handles scan_complete SSE event', async () => {
    mockGetScanStatus.mockResolvedValue(scanningStatus);
    const { result } = renderHook(() => useScanProgress());
    await vi.advanceTimersByTimeAsync(0);

    const es = MockEventSource.instances[0];

    act(() => {
      es.emit('scan_progress', {
        data: {
          files_found: 100,
          current_file: 'a.mkv',
          percent_done: 50,
          error_count: 0,
          estimated_time: '30 秒',
        },
      });
    });
    expect(result.current.isScanning).toBe(true);

    act(() => {
      es.emit('scan_complete', {
        data: { files_found: 200, error_count: 1 },
      });
    });

    expect(result.current.isScanning).toBe(false);
    expect(result.current.isComplete).toBe(true);
    expect(result.current.isVisible).toBe(true);
    expect(result.current.percentDone).toBe(100);
  });

  it('handles scan_cancelled SSE event', async () => {
    mockGetScanStatus.mockResolvedValue(scanningStatus);
    const { result } = renderHook(() => useScanProgress());
    await vi.advanceTimersByTimeAsync(0);

    const es = MockEventSource.instances[0];

    act(() => {
      es.emit('scan_progress', {
        data: {
          files_found: 50,
          current_file: 'x.mkv',
          percent_done: 25,
          error_count: 0,
          estimated_time: '1 分',
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

  it('toggles minimize state', async () => {
    const { result } = renderHook(() => useScanProgress());
    await vi.advanceTimersByTimeAsync(0);

    expect(result.current.isMinimized).toBe(false);
    act(() => result.current.toggleMinimize());
    expect(result.current.isMinimized).toBe(true);
    act(() => result.current.toggleMinimize());
    expect(result.current.isMinimized).toBe(false);
  });

  it('dismiss hides the card', async () => {
    mockGetScanStatus.mockResolvedValue(scanningStatus);
    const { result } = renderHook(() => useScanProgress());
    await vi.advanceTimersByTimeAsync(0);

    const es = MockEventSource.instances[0];
    act(() => {
      es.emit('scan_complete', { data: { files_found: 10, error_count: 0 } });
    });
    expect(result.current.isVisible).toBe(true);

    act(() => result.current.dismiss());
    expect(result.current.isVisible).toBe(false);
  });

  it('falls back to polling on SSE error', async () => {
    vi.useRealTimers();
    mockGetScanStatus.mockResolvedValue(scanningStatus);
    const { result } = renderHook(() => useScanProgress());
    await new Promise((r) => setTimeout(r, 50));

    const es = MockEventSource.instances[0];
    mockGetScanStatus.mockResolvedValue({
      ...scanningStatus,
      files_found: 300,
      current_file: 'poll.mkv',
      percent_done: 33,
    });

    act(() => {
      es.triggerError();
    });

    await waitFor(() => {
      expect(result.current.connectionStatus).toBe('polling');
    });
  });

  it('shows scanning state from initial status fetch', async () => {
    vi.useRealTimers();
    mockGetScanStatus.mockResolvedValue(scanningStatus);

    const { result } = renderHook(() => useScanProgress());

    await waitFor(() => {
      expect(result.current.isScanning).toBe(true);
      expect(result.current.filesFound).toBe(200);
    });
  });

  it('closes EventSource on unmount', async () => {
    mockGetScanStatus.mockResolvedValue(scanningStatus);
    const { unmount } = renderHook(() => useScanProgress());
    await vi.advanceTimersByTimeAsync(0);

    const es = MockEventSource.instances[0];
    unmount();

    expect(es.readyState).toBe(2); // CLOSED
  });

  it('connects SSE via idle poll when scan starts after mount', async () => {
    vi.useRealTimers();
    // Start idle
    mockGetScanStatus.mockResolvedValue(idleStatus);
    const { result } = renderHook(() => useScanProgress());
    await new Promise((r) => setTimeout(r, 50));
    expect(MockEventSource.instances).toHaveLength(0);

    // Scan starts — use startTracking to trigger immediate SSE connection
    mockGetScanStatus.mockResolvedValue(scanningStatus);
    act(() => result.current.startTracking());

    expect(MockEventSource.instances).toHaveLength(1);
  });

  it('exposes startTracking to manually connect SSE', async () => {
    const { result } = renderHook(() => useScanProgress());
    await vi.advanceTimersByTimeAsync(0);
    expect(MockEventSource.instances).toHaveLength(0);

    act(() => result.current.startTracking());
    expect(MockEventSource.instances).toHaveLength(1);
  });
});
