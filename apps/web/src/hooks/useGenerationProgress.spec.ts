import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useGenerationProgress } from './useGenerationProgress';

// Mock EventSource (mirrors useSubtitleBatchProgress.spec pattern)
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
 * Build the FULL SSE Event struct exactly as the backend writes the `data:` line
 * (sse/handler.go `sendSSEEvent(w, type, event)`): the payload is DOUBLE-nested
 * under `.data`, snake_case, with `media_id` as the int64 movie id.
 */
const wireEvent = (type: string, payload: Record<string, unknown>) => ({
  id: 'uuid-1',
  type,
  data: {
    job_id: 'job-9',
    media_id: 42,
    ...payload,
  },
});

describe('useGenerationProgress (lazy SSE, double-nested envelope)', () => {
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
    renderHook(() => useGenerationProgress());
    await vi.advanceTimersByTimeAsync(0);
    expect(MockEventSource.instances).toHaveLength(0);
    expect(renderHook(() => useGenerationProgress()).result.current.progress.phase).toBe('idle');
  });

  it('[P0] opens EventSource only after startTracking and enters extracting', () => {
    const { result } = renderHook(() => useGenerationProgress());

    act(() => result.current.startTracking(42));

    expect(MockEventSource.instances).toHaveLength(1);
    expect(MockEventSource.instances[0].url).toBe('/api/v1/events');
    expect(result.current.progress.phase).toBe('extracting');
  });

  it('[P0] unwraps the DOUBLE-nested Event envelope — payload is parsed.data, snakeToCamel applied', () => {
    const { result } = renderHook(() => useGenerationProgress());
    act(() => result.current.startTracking(42));
    const es = MockEventSource.instances[0];

    // The data: line is the full Event struct {id,type,data} — NOT the bare payload.
    act(() =>
      es.emit(
        'transcription_progress',
        wireEvent('transcription_progress', { phase: 'transcribing', message: '正在轉錄音訊' })
      )
    );

    expect(result.current.progress.phase).toBe('transcribing');
    expect(result.current.progress.message).toBe('正在轉錄音訊');
    expect(result.current.progress.jobId).toBe('job-9'); // job_id → jobId (snakeToCamel)
  });

  it('[P0] filters events by media_id — other movies do not touch state', () => {
    const { result } = renderHook(() => useGenerationProgress());
    act(() => result.current.startTracking(42));
    const es = MockEventSource.instances[0];

    act(() =>
      es.emit(
        'transcription_progress',
        wireEvent('transcription_progress', { media_id: 777, phase: 'transcribing' })
      )
    );

    expect(result.current.progress.phase).toBe('extracting'); // unchanged
  });

  it('[P1] maps each wire phase event to its stage', () => {
    const { result } = renderHook(() => useGenerationProgress());
    act(() => result.current.startTracking(42));
    const es = MockEventSource.instances[0];

    act(() =>
      es.emit(
        'transcription_extracting',
        wireEvent('transcription_extracting', { phase: 'extracting', message: '提取音訊中' })
      )
    );
    expect(result.current.progress.phase).toBe('extracting');

    act(() =>
      es.emit(
        'transcription_progress',
        wireEvent('transcription_progress', { phase: 'transcribing' })
      )
    );
    expect(result.current.progress.phase).toBe('transcribing');

    act(() =>
      es.emit(
        'translation_progress',
        wireEvent('translation_progress', { phase: 'translating', percentage: 62.5 })
      )
    );
    expect(result.current.progress.phase).toBe('translating');
    expect(result.current.progress.percentage).toBe(62.5);
  });

  it('[P0] transcription_complete is terminal: state, onComplete callback, stream closed', () => {
    const onComplete = vi.fn();
    const { result } = renderHook(() => useGenerationProgress({ onComplete }));
    act(() => result.current.startTracking(42));
    const es = MockEventSource.instances[0];

    act(() =>
      es.emit(
        'transcription_complete',
        wireEvent('transcription_complete', {
          phase: 'complete',
          srt_path: '/media/a.en.srt',
          zh_srt_path: '/media/a.zh-Hant.srt',
          duration: 2710,
          message: '完成',
        })
      )
    );

    expect(result.current.progress.phase).toBe('complete');
    expect(result.current.progress.srtPath).toBe('/media/a.en.srt');
    expect(result.current.progress.zhSrtPath).toBe('/media/a.zh-Hant.srt');
    expect(onComplete).toHaveBeenCalledTimes(1);
    expect(onComplete).toHaveBeenCalledWith({
      srtPath: '/media/a.en.srt',
      zhSrtPath: '/media/a.zh-Hant.srt',
    });
    expect(es.readyState).toBe(2); // terminal close
  });

  it('[P0] transcription_failed records the failed-at stage from the last live phase', () => {
    const { result } = renderHook(() => useGenerationProgress());
    act(() => result.current.startTracking(42));
    const es = MockEventSource.instances[0];

    act(() =>
      es.emit(
        'translation_progress',
        wireEvent('translation_progress', { phase: 'translating', percentage: 30 })
      )
    );
    act(() =>
      es.emit(
        'transcription_failed',
        wireEvent('transcription_failed', { phase: 'failed', error: 'AI 服務逾時' })
      )
    );

    expect(result.current.progress.phase).toBe('failed');
    expect(result.current.progress.failedPhase).toBe('translating');
    expect(result.current.progress.error).toBe('AI 服務逾時');
    expect(es.readyState).toBe(2); // terminal close
  });

  it('[P1] startTracking is the 409-attach path — mid-flight events land without a local trigger', () => {
    const { result } = renderHook(() => useGenerationProgress());

    // No POST happened locally; we attach to a job already running server-side.
    act(() => result.current.startTracking(42));
    const es = MockEventSource.instances[0];

    act(() =>
      es.emit(
        'translation_progress',
        wireEvent('translation_progress', { phase: 'translating', percentage: 80 })
      )
    );

    expect(result.current.progress.phase).toBe('translating');
    expect(result.current.progress.percentage).toBe(80);
  });

  it('[P0] does NOT dispatch after unmount and closes the stream', () => {
    const { result, unmount } = renderHook(() => useGenerationProgress());
    act(() => result.current.startTracking(42));
    const es = MockEventSource.instances[0];

    unmount();
    act(() =>
      es.emit(
        'transcription_progress',
        wireEvent('transcription_progress', { phase: 'transcribing' })
      )
    );

    expect(result.current.progress.phase).toBe('extracting'); // frozen pre-unmount state
    expect(es.readyState).toBe(2);
  });

  it('[P1] reset returns to idle and closes the stream', () => {
    const { result } = renderHook(() => useGenerationProgress());
    act(() => result.current.startTracking(42));
    const es = MockEventSource.instances[0];

    act(() => result.current.reset());

    expect(result.current.progress.phase).toBe('idle');
    expect(es.readyState).toBe(2);
  });

  it('[P1] schedules a 10s backoff reconnect on SSE error (no polling fallback)', async () => {
    const { result } = renderHook(() => useGenerationProgress());
    act(() => result.current.startTracking(42));
    const es = MockEventSource.instances[0];

    act(() => es.triggerError());
    await act(async () => {
      await vi.advanceTimersByTimeAsync(10000);
    });

    expect(MockEventSource.instances.length).toBeGreaterThanOrEqual(2);
  });

  it('[P1] startTracking after an SSE error cancels the stale backoff timer (no healthy-stream bounce)', async () => {
    const { result } = renderHook(() => useGenerationProgress());
    act(() => result.current.startTracking(42));
    const es = MockEventSource.instances[0];

    act(() => es.triggerError()); // schedules the 10s backoff
    act(() => result.current.startTracking(42)); // user re-triggers immediately

    expect(MockEventSource.instances).toHaveLength(2);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(10000);
    });
    // The stale timer must NOT fire a third connect that bounces the healthy stream.
    expect(MockEventSource.instances).toHaveLength(2);
    expect(MockEventSource.instances[1].readyState).not.toBe(2);
  });

  it('[P1] ignores malformed frames without crashing', () => {
    const { result } = renderHook(() => useGenerationProgress());
    act(() => result.current.startTracking(42));
    const es = MockEventSource.instances[0];

    act(() => {
      const bad = new MessageEvent('transcription_progress', { data: '{not json' });
      es.listeners['transcription_progress']?.forEach((cb) => cb(bad));
    });

    expect(result.current.progress.phase).toBe('extracting');
  });
});
