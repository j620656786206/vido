/**
 * useParseProgress Hook Tests (Story 3.10 - Task 9)
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useParseProgress } from './useParseProgress';
import type { ParseProgress } from './types';

// Mock EventSource
class MockEventSource {
  static instances: MockEventSource[] = [];

  url: string;
  readyState: number = 0;
  onopen: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  private listeners: Map<string, Set<(event: MessageEvent) => void>> = new Map();

  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSED = 2;

  constructor(url: string) {
    this.url = url;
    this.readyState = MockEventSource.CONNECTING;
    MockEventSource.instances.push(this);

    // Auto-open after a tick
    setTimeout(() => {
      this.readyState = MockEventSource.OPEN;
      this.onopen?.(new Event('open'));
    }, 0);
  }

  addEventListener(type: string, listener: (event: MessageEvent) => void) {
    if (!this.listeners.has(type)) {
      this.listeners.set(type, new Set());
    }
    this.listeners.get(type)!.add(listener);
  }

  removeEventListener(type: string, listener: (event: MessageEvent) => void) {
    this.listeners.get(type)?.delete(listener);
  }

  close() {
    this.readyState = MockEventSource.CLOSED;
    const idx = MockEventSource.instances.indexOf(this);
    if (idx > -1) {
      MockEventSource.instances.splice(idx, 1);
    }
  }

  // Test helper: simulate receiving an event
  simulateEvent(type: string, data: unknown) {
    const event = new MessageEvent(type, {
      data: JSON.stringify(data),
    });
    this.listeners.get(type)?.forEach((listener) => listener(event));
    this.onmessage?.(event);
  }

  // Test helper: simulate error
  simulateError() {
    this.onerror?.(new Event('error'));
  }
}

// Install mock
vi.stubGlobal('EventSource', MockEventSource);

describe('useParseProgress', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    MockEventSource.instances = [];
  });

  afterEach(() => {
    vi.useRealTimers();
    MockEventSource.instances.forEach((es) => es.close());
    MockEventSource.instances = [];
  });

  it('returns initial state when no taskId provided', () => {
    const { result } = renderHook(() => useParseProgress(null));

    expect(result.current.progress).toBeNull();
    expect(result.current.status).toBe('pending');
    expect(result.current.error).toBeNull();
    expect(result.current.isConnected).toBe(false);
  });

  it('connects to SSE when taskId is provided', async () => {
    renderHook(() => useParseProgress('task-123'));

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    expect(MockEventSource.instances.length).toBe(1);
    expect(MockEventSource.instances[0].url).toBe('/api/v1/parse/progress/task-123');
  });

  it('initializes progress with default steps', async () => {
    const { result } = renderHook(() => useParseProgress('task-123'));

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    expect(result.current.progress).not.toBeNull();
    expect(result.current.progress?.taskId).toBe('task-123');
    expect(result.current.progress?.steps).toHaveLength(6);
  });

  it('sets isConnected to true when connection opens', async () => {
    const { result } = renderHook(() => useParseProgress('task-123'));

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    expect(result.current.isConnected).toBe(true);
  });

  it('handles parse_started event', async () => {
    const onParseStarted = vi.fn();
    const { result } = renderHook(() =>
      useParseProgress('task-123', { onParseStarted })
    );

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    const es = MockEventSource.instances[0];
    await act(async () => {
      es.simulateEvent('parse_started', {
        data: {
          filename: 'test-movie.mkv',
          totalSteps: 6,
          steps: [
            { name: 'filename_extract', label: '解析檔名', status: 'pending' },
          ],
        },
      });
    });

    expect(onParseStarted).toHaveBeenCalled();
    expect(result.current.progress?.filename).toBe('test-movie.mkv');
  });

  it('handles step_completed event', async () => {
    const onStepCompleted = vi.fn();
    const { result } = renderHook(() =>
      useParseProgress('task-123', { onStepCompleted })
    );

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    const es = MockEventSource.instances[0];
    const mockProgress: ParseProgress = {
      taskId: 'task-123',
      filename: 'test.mkv',
      status: 'pending',
      steps: [
        { name: 'filename_extract', label: '解析檔名', status: 'success' },
      ],
      currentStep: 1,
      percentage: 16,
      startedAt: new Date().toISOString(),
    };

    await act(async () => {
      es.simulateEvent('step_completed', {
        data: {
          stepIndex: 0,
          step: { name: 'filename_extract', label: '解析檔名', status: 'success' },
          progress: mockProgress,
        },
      });
    });

    expect(onStepCompleted).toHaveBeenCalled();
    expect(result.current.progress?.percentage).toBe(16);
  });

  it('handles parse_completed event', async () => {
    const onParseCompleted = vi.fn();
    const { result } = renderHook(() =>
      useParseProgress('task-123', { onParseCompleted })
    );

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    const es = MockEventSource.instances[0];
    const mockProgress: ParseProgress = {
      taskId: 'task-123',
      filename: 'test.mkv',
      status: 'success',
      steps: [],
      currentStep: 6,
      percentage: 100,
      startedAt: new Date().toISOString(),
      result: {
        mediaId: 'movie-456',
        title: 'Test Movie',
        year: 2024,
      },
    };

    await act(async () => {
      es.simulateEvent('parse_completed', {
        data: {
          result: mockProgress.result,
          progress: mockProgress,
        },
      });
    });

    expect(onParseCompleted).toHaveBeenCalled();
    expect(result.current.status).toBe('success');
    expect(result.current.progress?.percentage).toBe(100);
  });

  it('handles parse_failed event', async () => {
    const onParseFailed = vi.fn();
    const { result } = renderHook(() =>
      useParseProgress('task-123', { onParseFailed })
    );

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    const es = MockEventSource.instances[0];
    const mockProgress: ParseProgress = {
      taskId: 'task-123',
      filename: 'test.mkv',
      status: 'failed',
      steps: [],
      currentStep: 0,
      percentage: 0,
      message: 'All sources failed',
      startedAt: new Date().toISOString(),
    };

    await act(async () => {
      es.simulateEvent('parse_failed', {
        data: {
          message: 'All sources failed',
          failedSteps: [],
          progress: mockProgress,
        },
      });
    });

    expect(onParseFailed).toHaveBeenCalled();
    expect(result.current.status).toBe('failed');
  });

  it('sets error on connection failure', async () => {
    const onError = vi.fn();
    const { result } = renderHook(() =>
      useParseProgress('task-123', { onError, autoReconnect: false })
    );

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    const es = MockEventSource.instances[0];
    await act(async () => {
      es.simulateError();
    });

    expect(result.current.error).not.toBeNull();
    expect(result.current.isConnected).toBe(false);
    expect(onError).toHaveBeenCalled();
  });

  it('attempts reconnection on error when autoReconnect is true', async () => {
    const { result } = renderHook(() =>
      useParseProgress('task-123', { autoReconnect: true, reconnectDelay: 1000 })
    );

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    expect(MockEventSource.instances.length).toBe(1);
    const es = MockEventSource.instances[0];

    await act(async () => {
      es.simulateError();
    });

    expect(result.current.isReconnecting).toBe(true);

    // Advance time to trigger reconnection
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });

    // Should have created a new connection
    expect(MockEventSource.instances.length).toBe(1);
  });

  it('disconnects when disconnect is called', async () => {
    const { result } = renderHook(() => useParseProgress('task-123'));

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    expect(result.current.isConnected).toBe(true);

    act(() => {
      result.current.disconnect();
    });

    expect(result.current.isConnected).toBe(false);
    expect(MockEventSource.instances.length).toBe(0);
  });

  it('reconnects when reconnect is called', async () => {
    const { result } = renderHook(() => useParseProgress('task-123'));

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    act(() => {
      result.current.disconnect();
    });

    expect(MockEventSource.instances.length).toBe(0);

    await act(async () => {
      result.current.reconnect();
      await vi.runAllTimersAsync();
    });

    expect(MockEventSource.instances.length).toBe(1);
  });

  it('cleans up on unmount', async () => {
    const { unmount } = renderHook(() => useParseProgress('task-123'));

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    expect(MockEventSource.instances.length).toBe(1);

    unmount();

    expect(MockEventSource.instances.length).toBe(0);
  });

  it('reconnects when taskId changes', async () => {
    const { rerender } = renderHook(({ taskId }) => useParseProgress(taskId), {
      initialProps: { taskId: 'task-123' },
    });

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    expect(MockEventSource.instances[0]?.url).toBe('/api/v1/parse/progress/task-123');

    rerender({ taskId: 'task-456' });

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    expect(MockEventSource.instances[0]?.url).toBe('/api/v1/parse/progress/task-456');
  });

  it('calls onConnected callback when connected', async () => {
    const onConnected = vi.fn();
    renderHook(() => useParseProgress('task-123', { onConnected }));

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    // Simulate the connected event
    const es = MockEventSource.instances[0];
    await act(async () => {
      es.simulateEvent('connected', { taskId: 'task-123', message: 'Connected' });
    });

    expect(onConnected).toHaveBeenCalled();
  });
});
