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

// Media-id fixture convention (9R-18 AC 7): media ids are UUID STRINGS —
// mirror the prod creation path (uuid.New().String()); do NOT invent numeric ids.
const MOVIE_UUID = '4f8c2d1a-5b6e-4c7d-8e9f-0a1b2c3d4e5f';

/**
 * Build the FULL SSE Event struct exactly as the backend writes the `data:`
 * line (sse/handler.go `sendSSEEvent(w, type, event)`): the payload is
 * DOUBLE-nested under `.data`, snake_case, all 11 contract keys
 * (9R-16 [@contract-v2] AC #9 — `current_media_id` is a UUID STRING).
 */
const wireEvent = (payload: Record<string, unknown>) => ({
  id: 'uuid-1',
  type: 'generation_batch_progress',
  data: {
    batch_id: 'gb-1',
    total_items: 38,
    current_index: 12,
    current_media_id: MOVIE_UUID,
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
      currentMediaId: MOVIE_UUID,
      currentItem: '怪奇物語',
      successCount: 11,
      failCount: 0,
      pausedCount: 0,
      status: 'running',
      spentUsd: 0.42,
      budgetUsd: 5,
      // dsr-6d-a: the queue rides the snapshot endpoints; a running SSE event
      // sends items:null and a single changed_item instead.
      items: null,
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

// ---------------------------------------------------------------------------
// dsr-6d-b: the queue lives in the hook now (items + changed_item + attach)
// ---------------------------------------------------------------------------

const ITEM_A = '11111111-1111-4111-8111-111111111111';
const ITEM_B = '22222222-2222-4222-8222-222222222222';

const wireItem = (mediaId: string, over: Record<string, unknown> = {}) => ({
  media_id: mediaId,
  title: mediaId === ITEM_A ? '沙丘：第二部' : '奧本海默',
  media_type: 'movie',
  series_title: '',
  status: 'queued',
  reason: '',
  ...over,
});

const seededQueue = () => [
  {
    mediaId: ITEM_A,
    title: '沙丘：第二部',
    mediaType: 'movie' as const,
    seriesTitle: '',
    status: 'done' as const,
    reason: '' as const,
  },
  {
    mediaId: ITEM_B,
    title: '奧本海默',
    mediaType: 'movie' as const,
    seriesTitle: '',
    status: 'running' as const,
    reason: '' as const,
  },
];

describe('useGenerationBatchProgress — queue merge (dsr-6d-b AC #5)', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    MockEventSource.instances = [];
    (global as Record<string, unknown>).EventSource = MockEventSource;
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('[P0] changed_item REPLACES that one entry and leaves the rest alone', () => {
    const { result } = renderHook(() => useGenerationBatchProgress());
    act(() => result.current.startTracking({ batchId: 'gb-1', items: seededQueue() }));
    const es = MockEventSource.instances[0];

    act(() =>
      es.emit(
        'generation_batch_progress',
        wireEvent({
          items: null,
          changed_item: wireItem(ITEM_B, { status: 'failed', reason: 'busy_elsewhere' }),
        })
      )
    );

    expect(result.current.progress.items).toEqual([
      expect.objectContaining({ mediaId: ITEM_A, status: 'done' }),
      expect.objectContaining({ mediaId: ITEM_B, status: 'failed', reason: 'busy_elsewhere' }),
    ]);
  });

  it('[P0] a changed_item for an id that is not in the queue is DROPPED, never appended', () => {
    const { result } = renderHook(() => useGenerationBatchProgress());
    act(() => result.current.startTracking({ items: seededQueue() }));
    const es = MockEventSource.instances[0];

    act(() =>
      es.emit(
        'generation_batch_progress',
        wireEvent({
          items: null,
          changed_item: wireItem('99999999-9999-4999-8999-999999999999', { status: 'done' }),
        })
      )
    );

    expect(result.current.progress.items).toHaveLength(2);
    expect(result.current.progress.items?.map((i) => i.mediaId)).toEqual([ITEM_A, ITEM_B]);
  });

  it('[P0] with NO queue yet (items null), a changed_item does NOT invent a one-row queue', () => {
    const { result } = renderHook(() => useGenerationBatchProgress());
    act(() => result.current.startTracking());
    const es = MockEventSource.instances[0];

    act(() =>
      es.emit(
        'generation_batch_progress',
        wireEvent({ items: null, changed_item: wireItem(ITEM_A, { status: 'running' }) })
      )
    );

    expect(result.current.progress.items).toBeNull();
  });

  it('[P0] the terminal broadcast carries the WHOLE queue and replaces what we had', () => {
    const { result } = renderHook(() => useGenerationBatchProgress());
    act(() => result.current.startTracking({ items: seededQueue() }));
    const es = MockEventSource.instances[0];

    act(() =>
      es.emit(
        'generation_batch_progress',
        wireEvent({
          status: 'complete',
          changed_item: null,
          items: [
            wireItem(ITEM_A, { status: 'done' }),
            wireItem(ITEM_B, { status: 'failed', reason: 'skipped' }),
          ],
        })
      )
    );

    expect(result.current.status).toBe('complete');
    expect(result.current.progress.items).toEqual([
      expect.objectContaining({ mediaId: ITEM_A, status: 'done', reason: '' }),
      expect.objectContaining({ mediaId: ITEM_B, status: 'failed', reason: 'skipped' }),
    ]);
  });

  it('[P0] an SSE event with neither items nor changed_item leaves the queue untouched', () => {
    const { result } = renderHook(() => useGenerationBatchProgress());
    act(() => result.current.startTracking({ items: seededQueue() }));
    const es = MockEventSource.instances[0];

    act(() => es.emit('generation_batch_progress', wireEvent({})));

    expect(result.current.progress.items).toEqual(seededQueue());
  });
});

describe('useGenerationBatchProgress — attachSnapshot + connectionEpoch (dsr-6d-b AC #5)', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    MockEventSource.instances = [];
    (global as Record<string, unknown>).EventSource = MockEventSource;
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  const lastSnapshot = {
    batchId: 'gb-done',
    totalItems: 2,
    currentIndex: 2,
    currentMediaId: ITEM_B,
    currentItem: '奧本海默',
    successCount: 1,
    failCount: 1,
    pausedCount: 0,
    status: 'complete' as const,
    spentUsd: 1.5,
    budgetUsd: 5,
    items: [
      {
        mediaId: ITEM_A,
        title: '沙丘：第二部',
        mediaType: 'movie' as const,
        seriesTitle: '',
        status: 'done' as const,
        reason: '' as const,
      },
      {
        mediaId: ITEM_B,
        title: '奧本海默',
        mediaType: 'movie' as const,
        seriesTitle: '',
        status: 'failed' as const,
        reason: 'error' as const,
      },
    ],
  };

  it('[P0] attachSnapshot does NOT open a stream and keeps the TERMINAL status (not running)', () => {
    const { result } = renderHook(() => useGenerationBatchProgress());

    act(() => result.current.attachSnapshot(lastSnapshot));

    expect(MockEventSource.instances).toHaveLength(0);
    expect(result.current.status).toBe('complete');
    expect(result.current.progress.items).toHaveLength(2);
    expect(result.current.progress.successCount).toBe(1);
    expect(result.current.progress.failCount).toBe(1);
    expect(result.current.progress.budgetUsd).toBe(5);
  });

  it('[P0] attachSnapshot is reference-stable across renders (it feeds an effect dep array)', () => {
    const { result, rerender } = renderHook(() => useGenerationBatchProgress());
    const first = result.current.attachSnapshot;
    act(() => result.current.attachSnapshot(lastSnapshot));
    rerender();
    expect(result.current.attachSnapshot).toBe(first);
  });

  it('[P0] connectionEpoch does NOT tick for startTracking’s first connect', () => {
    const { result } = renderHook(() => useGenerationBatchProgress());
    expect(result.current.connectionEpoch).toBe(0);

    act(() => result.current.startTracking({ batchId: 'gb-1' }));

    expect(MockEventSource.instances).toHaveLength(1);
    expect(result.current.connectionEpoch).toBe(0);
  });

  it('[P0] connectionEpoch ticks once per backoff RECONNECT (the dropped-terminal escape hatch)', async () => {
    const { result } = renderHook(() => useGenerationBatchProgress());
    act(() => result.current.startTracking());

    act(() => MockEventSource.instances[0].triggerError());
    expect(result.current.connectionEpoch).toBe(0); // not yet — still in backoff

    await act(async () => {
      await vi.advanceTimersByTimeAsync(10000);
    });
    expect(MockEventSource.instances).toHaveLength(2);
    expect(result.current.connectionEpoch).toBe(1);

    act(() => MockEventSource.instances[1].triggerError());
    await act(async () => {
      await vi.advanceTimersByTimeAsync(10000);
    });
    expect(result.current.connectionEpoch).toBe(2);
  });

  it('[P1] reset after attachSnapshot returns to idle with an empty queue', () => {
    const { result } = renderHook(() => useGenerationBatchProgress());
    act(() => result.current.attachSnapshot(lastSnapshot));

    act(() => result.current.reset());

    expect(result.current.status).toBe('idle');
    expect(result.current.progress.items).toBeNull();
  });
});
