import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useGenerationJobsFeed, FEED_CAP } from './useGenerationJobsFeed';

// Minimal EventSource stub — records instances + emits a named event with a
// double-nested Event envelope payload (the wire shape).
class MockEventSource {
  static instances: MockEventSource[] = [];
  url: string;
  readyState = 0;
  onerror: (() => void) | null = null;
  private listeners: Record<string, ((e: MessageEvent) => void)[]> = {};
  constructor(url: string) {
    this.url = url;
    MockEventSource.instances.push(this);
  }
  addEventListener(type: string, cb: (e: MessageEvent) => void) {
    (this.listeners[type] ||= []).push(cb);
  }
  close() {
    this.readyState = 2;
  }
  /** Wire wraps the payload as the whole Event {id,type,data}; snake_case data. */
  emit(type: string, data: unknown) {
    const frame = { data: JSON.stringify({ id: 'x', type, data }) } as MessageEvent;
    (this.listeners[type] || []).forEach((cb) => cb(frame));
  }
}

function render() {
  const { result } = renderHook(() => useGenerationJobsFeed());
  return result;
}

describe('useGenerationJobsFeed (ux3-ai-2 AC 4/5 — unfiltered live feed)', () => {
  beforeEach(() => {
    MockEventSource.instances = [];
    vi.stubGlobal('EventSource', MockEventSource);
  });

  it('does NOT open an EventSource on mount (§8 lazy-SSE)', () => {
    render();
    expect(MockEventSource.instances).toHaveLength(0);
  });

  it('opens the SSE only after startTracking() and points at the events endpoint', () => {
    const result = render();
    act(() => result.current.startTracking());
    expect(MockEventSource.instances).toHaveLength(1);
    expect(MockEventSource.instances[0].url).toContain('/events');
  });

  it('stop() closes the connection', () => {
    const result = render();
    act(() => result.current.startTracking());
    const es = MockEventSource.instances[0];
    act(() => result.current.stop());
    expect(es.readyState).toBe(2);
  });

  it('a transcription phase event appends a feed row AND tracks the single job (envelope + snake→camel)', () => {
    const result = render();
    act(() => result.current.startTracking());
    act(() =>
      MockEventSource.instances[0].emit('translation_progress', {
        media_id: 'm1',
        phase: 'translating',
        percentage: 45.4,
        message: '翻譯中',
      })
    );
    expect(result.current.feed).toHaveLength(1);
    expect(result.current.feed[0]).toMatchObject({
      tone: 'active',
      stage: '翻譯中',
      mediaId: 'm1',
      trail: '45%', // percentage rounded, Mono-ready
    });
    expect(result.current.singleJobs['m1']).toMatchObject({
      mediaId: 'm1',
      phase: 'translating',
      percentage: 45,
    });
  });

  it('transcription_complete retires the single job and appends a done row', () => {
    const result = render();
    act(() => result.current.startTracking());
    act(() =>
      MockEventSource.instances[0].emit('transcription_progress', {
        media_id: 'm1',
        phase: 'transcribing',
      })
    );
    expect(result.current.singleJobs['m1']).toBeDefined();
    act(() =>
      MockEventSource.instances[0].emit('transcription_complete', {
        media_id: 'm1',
        phase: 'complete',
      })
    );
    expect(result.current.singleJobs['m1']).toBeUndefined(); // retired
    expect(result.current.feed.at(-1)).toMatchObject({
      tone: 'done',
      stage: '完成',
      mediaId: 'm1',
    });
  });

  it('transcription_failed retires the job and surfaces error text', () => {
    const result = render();
    act(() => result.current.startTracking());
    act(() =>
      MockEventSource.instances[0].emit('transcription_extracting', {
        media_id: 'm2',
        phase: 'extracting',
      })
    );
    act(() =>
      MockEventSource.instances[0].emit('transcription_failed', {
        media_id: 'm2',
        phase: 'failed',
        error: '轉錄失敗',
      })
    );
    expect(result.current.singleJobs['m2']).toBeUndefined();
    expect(result.current.feed.at(-1)).toMatchObject({
      tone: 'failed',
      stage: '失敗',
      message: '轉錄失敗',
    });
  });

  it('batch status CHANGE appends a lifecycle row; running adds none; same status does not duplicate', () => {
    const result = render();
    act(() => result.current.startTracking());
    const es = MockEventSource.instances[0];
    act(() => es.emit('generation_batch_progress', { status: 'running', current_item: 'A' }));
    expect(result.current.feed).toHaveLength(0); // running → no lifecycle row
    act(() =>
      es.emit('generation_batch_progress', { status: 'budget_ceiling', current_item: 'A' })
    );
    act(() =>
      es.emit('generation_batch_progress', { status: 'budget_ceiling', current_item: 'A' })
    );
    const lifecycle = result.current.feed.filter((r) => r.stage === '已達本次預算上限');
    expect(lifecycle).toHaveLength(1); // appended once, not per-tick
    expect(lifecycle[0].tone).toBe('info'); // budget_ceiling = non-error tone
  });

  it('caps the feed at FEED_CAP (drop-oldest) and keeps seq keys unique+monotonic', () => {
    const result = render();
    act(() => result.current.startTracking());
    const es = MockEventSource.instances[0];
    act(() => {
      for (let i = 0; i < FEED_CAP + 20; i++) {
        es.emit('transcription_progress', { media_id: `m${i}`, phase: 'transcribing' });
      }
    });
    expect(result.current.feed).toHaveLength(FEED_CAP);
    const seqs = result.current.feed.map((r) => r.seq);
    expect(new Set(seqs).size).toBe(FEED_CAP); // unique React keys
    expect(seqs[seqs.length - 1]).toBeGreaterThan(seqs[0]); // monotonic, oldest dropped
  });
});
