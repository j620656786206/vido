import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useGenerationJobsFeed, FEED_CAP, type FeedRow } from './useGenerationJobsFeed';

// Records every value handed to snakeToCamel so a test can prove the terminal
// batch event's full `items` queue is never converted (dsr-6d-c-2 AC #3).
const camel = vi.hoisted(() => ({ inputs: [] as unknown[] }));
vi.mock('../utils/caseTransform', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../utils/caseTransform')>();
  return {
    ...actual,
    snakeToCamel: <T>(value: unknown): T => {
      camel.inputs.push(value);
      return actual.snakeToCamel<T>(value);
    },
  };
});

// Minimal EventSource stub — records instances + emits a named event with a
// double-nested Event envelope payload (the wire shape).
class MockEventSource {
  static instances: MockEventSource[] = [];
  url: string;
  readyState = 0;
  onerror: (() => void) | null = null;
  onopen: (() => void) | null = null;
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
  open() {
    this.readyState = 1;
    this.onopen?.();
  }
  fail() {
    this.onerror?.();
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

function connected() {
  const result = render();
  act(() => result.current.startTracking());
  const es = MockEventSource.instances[MockEventSource.instances.length - 1];
  act(() => es.open());
  return { result, es };
}

const item = (mediaId: string, title: string, status: string, reason = '', seriesTitle = '') => ({
  media_id: mediaId,
  title,
  media_type: seriesTitle ? 'episode' : 'movie',
  series_title: seriesTitle,
  status,
  reason,
});

/** A running-batch frame carrying one changed item (dsr-6d-a AC #2). */
const running = (changed: ReturnType<typeof item>, batchId = 'b1') => ({
  batch_id: batchId,
  status: 'running',
  total_items: 3,
  current_index: 1,
  current_media_id: changed.media_id,
  current_item: changed.title,
  success_count: 0,
  fail_count: 0,
  paused_count: 0,
  spent_usd: 0.1,
  budget_usd: 5,
  items: null,
  changed_item: changed,
});

/** A terminal frame: full items, no changed_item. */
const terminal = (status: string, over: Record<string, unknown> = {}, batchId = 'b1') => ({
  batch_id: batchId,
  status,
  total_items: 3,
  current_index: 3,
  current_media_id: '',
  current_item: '',
  success_count: 1,
  fail_count: 0,
  paused_count: 0,
  spent_usd: 5,
  budget_usd: 5,
  items: [item('m1', 'A', 'done'), item('m2', 'B', 'paused'), item('m3', 'C', 'paused')],
  changed_item: null,
  ...over,
});

const rowsFor = (feed: FeedRow[], mediaId: string) =>
  feed.filter((r) => r.kind !== 'batch' && r.mediaId === mediaId);

describe('useGenerationJobsFeed (ux3-ai-2 AC 4/5, reworked by dsr-6d-c-2)', () => {
  beforeEach(() => {
    MockEventSource.instances = [];
    camel.inputs = [];
    vi.stubGlobal('EventSource', MockEventSource);
  });

  // --- kept: lazy SSE, stop, cap -------------------------------------------

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
    const { result, es } = connected();
    act(() => result.current.stop());
    expect(es.readyState).toBe(2);
  });

  it('caps the feed at FEED_CAP (drop-oldest) and keeps seq keys unique+monotonic', () => {
    const { result, es } = connected();
    act(() => {
      for (let i = 0; i < FEED_CAP + 20; i++) {
        es.emit('transcription_progress', { media_id: `m${i}`, phase: 'transcribing', title: '' });
      }
    });
    expect(result.current.feed).toHaveLength(FEED_CAP);
    const seqs = result.current.feed.map((r) => r.seq);
    expect(new Set(seqs).size).toBe(FEED_CAP);
    expect(seqs[seqs.length - 1]).toBeGreaterThan(seqs[0]);
  });

  // --- 🔴 #7: one row per item per stage ---------------------------------

  it('150 translation frames for one film are ONE row whose percentage updates in place', () => {
    const { result, es } = connected();
    act(() => {
      for (let i = 0; i <= 150; i++) {
        es.emit('translation_progress', {
          media_id: 'm1',
          phase: 'translating',
          percentage: (i / 150) * 100,
          title: '沙丘：第二部',
        });
      }
    });
    expect(result.current.feed).toHaveLength(1);
    expect(result.current.feed[0]).toMatchObject({
      kind: 'stage',
      stage: 'translating',
      state: 'live',
      percentage: 100,
      title: '沙丘：第二部',
    });
  });

  it('a new stage demotes the previous row of the same film and drops its percentage', () => {
    const { result, es } = connected();
    act(() => {
      es.emit('transcription_extracting', { media_id: 'm1', phase: 'extracting', title: 'A' });
      es.emit('transcription_progress', { media_id: 'm1', phase: 'transcribing', title: 'A' });
      es.emit('translation_progress', {
        media_id: 'm1',
        phase: 'translating',
        percentage: 40,
        title: 'A',
      });
    });
    const [extract, transcribe, translate] = result.current.feed;
    expect(extract).toMatchObject({ kind: 'stage', stage: 'extracting', state: 'passed' });
    expect(transcribe).toMatchObject({
      kind: 'stage',
      stage: 'transcribing',
      state: 'passed',
      percentage: null,
    });
    expect(translate).toMatchObject({
      kind: 'stage',
      stage: 'translating',
      state: 'live',
      percentage: 40,
    });
  });

  // --- single jobs ------------------------------------------------------

  it('a single job: stage rows + singleJobs entry, retired with ONE done row', () => {
    const { result, es } = connected();
    act(() =>
      es.emit('transcription_progress', {
        media_id: 'm1',
        phase: 'transcribing',
        title: '奧本海默',
      })
    );
    expect(result.current.singleJobs['m1']).toMatchObject({
      phase: 'transcribing',
      title: '奧本海默',
    });
    act(() =>
      es.emit('transcription_complete', { media_id: 'm1', phase: 'complete', title: '奧本海默' })
    );
    expect(result.current.singleJobs['m1']).toBeUndefined();
    expect(result.current.feed.map((r) => r.kind)).toEqual(['stage', 'done']);
    expect(result.current.feed[0]).toMatchObject({ state: 'passed' });
    expect(result.current.feed[1]).toMatchObject({
      kind: 'done',
      mediaId: 'm1',
      title: '奧本海默',
    });
  });

  it('a single failure keeps the raw error for the view to translate (reason null = not a batch item)', () => {
    const { result, es } = connected();
    act(() => {
      es.emit('transcription_extracting', { media_id: 'm2', phase: 'extracting', title: '' });
      es.emit('transcription_failed', {
        media_id: 'm2',
        phase: 'failed',
        title: '',
        error: 'select audio track: no audio track found in media file',
      });
    });
    expect(result.current.singleJobs['m2']).toBeUndefined();
    expect(result.current.feed.at(-1)).toMatchObject({
      kind: 'failed',
      reason: null,
      error: 'select audio track: no audio track found in media file',
    });
  });

  it('never keeps a media id as a title (a UUID is not a title)', () => {
    const { result, es } = connected();
    act(() =>
      es.emit('transcription_progress', {
        media_id: 'uuid-1',
        phase: 'transcribing',
        title: 'uuid-1',
      })
    );
    expect(result.current.feed[0]).toMatchObject({ title: '' });
    expect(result.current.singleJobs['uuid-1']).toMatchObject({ title: '' });
  });

  // --- batch members (🔴 #3–#6, #10, #11, #16) ---------------------------

  it('titles come from changed_item, and a batch item never becomes a single job', () => {
    const { result, es } = connected();
    act(() => {
      es.emit(
        'generation_batch_progress',
        running(item('e7', 'S04E07 第七章', 'running', '', '怪奇物語'))
      );
      es.emit('transcription_progress', { media_id: 'e7', phase: 'transcribing', title: '' });
    });
    expect(result.current.singleJobs).toEqual({});
    expect(result.current.feed[0]).toMatchObject({
      kind: 'stage',
      title: 'S04E07 第七章',
      seriesTitle: '怪奇物語',
    });
  });

  it('seedBatch() makes a mid-batch attach know the members and their titles', () => {
    const { result, es } = connected();
    act(() =>
      result.current.seedBatch('b1', [
        {
          mediaId: 'm1',
          title: '奧本海默',
          mediaType: 'movie',
          seriesTitle: '',
          status: 'running',
          reason: '',
        },
      ])
    );
    act(() =>
      es.emit('transcription_progress', { media_id: 'm1', phase: 'transcribing', title: '' })
    );
    expect(result.current.singleJobs).toEqual({});
    expect(result.current.feed[0]).toMatchObject({ title: '奧本海默' });
  });

  it('seedBatch() retires a batch item that arrived before the seed as a single job, and back-fills its title', () => {
    const { result, es } = connected();
    act(() =>
      es.emit('transcription_progress', { media_id: 'm1', phase: 'transcribing', title: '' })
    );
    expect(result.current.singleJobs['m1']).toBeDefined();
    act(() =>
      result.current.seedBatch('b1', [
        {
          mediaId: 'm1',
          title: '奧本海默',
          mediaType: 'movie',
          seriesTitle: '',
          status: 'running',
          reason: '',
        },
      ])
    );
    expect(result.current.singleJobs).toEqual({});
    expect(result.current.feed[0]).toMatchObject({ title: '奧本海默' });
  });

  it('a batch item terminal comes ONLY from changed_item (done)', () => {
    const { result, es } = connected();
    act(() => {
      es.emit('generation_batch_progress', running(item('m1', 'A', 'running')));
      es.emit('transcription_progress', { media_id: 'm1', phase: 'transcribing', title: '' });
      es.emit('transcription_complete', { media_id: 'm1', phase: 'complete', title: '' });
    });
    expect(rowsFor(result.current.feed, 'm1').map((r) => r.kind)).toEqual(['stage']);
    expect(rowsFor(result.current.feed, 'm1')[0]).toMatchObject({ state: 'passed' });
    act(() => es.emit('generation_batch_progress', running(item('m1', 'A', 'done'))));
    expect(rowsFor(result.current.feed, 'm1').map((r) => r.kind)).toEqual(['stage', 'done']);
  });

  it('🔴 #3: a budget stop is NOT a failure — the item is paused and the batch row says so', () => {
    const { result, es } = connected();
    act(() => {
      es.emit('generation_batch_progress', running(item('m2', 'B', 'running')));
      es.emit('translation_progress', {
        media_id: 'm2',
        phase: 'translating',
        percentage: 60,
        title: '',
      });
      es.emit('transcription_failed', {
        media_id: 'm2',
        phase: 'failed',
        title: '',
        error: 'translate: translate: translation stopped at block 9: AI_BUDGET_EXCEEDED: ceiling',
      });
      es.emit('generation_batch_progress', terminal('budget_ceiling'));
    });
    expect(result.current.feed.some((r) => r.kind === 'failed')).toBe(false);
    expect(rowsFor(result.current.feed, 'm2')[0]).toMatchObject({
      state: 'stopped',
      percentage: null,
    });
    expect(result.current.feed.at(-1)).toMatchObject({
      kind: 'batch',
      status: 'budget_ceiling',
      budgetUsd: 5,
    });
  });

  it('🔴 #4: cancelling the batch does not record the in-flight item as failed', () => {
    const { result, es } = connected();
    act(() => {
      es.emit('generation_batch_progress', running(item('m2', 'B', 'running')));
      es.emit('transcription_extracting', { media_id: 'm2', phase: 'extracting', title: '' });
      es.emit('transcription_failed', {
        media_id: 'm2',
        phase: 'failed',
        title: '',
        error: 'extract audio: context canceled',
      });
      es.emit('generation_batch_progress', terminal('cancelled'));
    });
    expect(result.current.feed.some((r) => r.kind === 'failed')).toBe(false);
    expect(result.current.feed.at(-1)).toMatchObject({ kind: 'batch', status: 'cancelled' });
  });

  it('🔴 #5: a pipeline ASR-fallback item (both event families) ends with exactly ONE done row', () => {
    const { result, es } = connected();
    act(() => {
      es.emit('generation_batch_progress', running(item('m1', 'A', 'running')));
      es.emit('subtitle_progress', {
        media_id: 'm1',
        media_type: 'movie',
        stage: 'probing',
        message: '',
      });
      es.emit('subtitle_progress', {
        media_id: 'm1',
        media_type: 'movie',
        stage: 'extracting',
        message: '',
      });
      es.emit('transcription_extracting', { media_id: 'm1', phase: 'extracting', title: '' });
      es.emit('transcription_complete', { media_id: 'm1', phase: 'complete', title: '' });
      es.emit('subtitle_progress', {
        media_id: 'm1',
        media_type: 'movie',
        stage: 'complete',
        message: '',
      });
      es.emit('generation_batch_progress', running(item('m1', 'A', 'done')));
    });
    const kinds = rowsFor(result.current.feed, 'm1').map((r) => r.kind);
    expect(kinds.filter((k) => k === 'done')).toHaveLength(1);
    expect(kinds).toEqual(['stage', 'stage', 'done']);
    expect(rowsFor(result.current.feed, 'm1')[0]).toMatchObject({ stage: 'track' });
  });

  it('🔴 #6: an item refused before any transcription event still gets its failed row (busy_elsewhere)', () => {
    const { result, es } = connected();
    act(() => {
      es.emit('generation_batch_progress', running(item('m3', 'C', 'running')));
      es.emit('generation_batch_progress', running(item('m3', 'C', 'failed', 'busy_elsewhere')));
    });
    expect(result.current.feed.at(-1)).toMatchObject({
      kind: 'failed',
      mediaId: 'm3',
      title: 'C',
      reason: 'busy_elsewhere',
      error: null,
    });
  });

  it('reason "error" keeps the member transcription_failed text so the view can be specific', () => {
    const { result, es } = connected();
    act(() => {
      es.emit('generation_batch_progress', running(item('m3', 'C', 'running')));
      es.emit('transcription_failed', {
        media_id: 'm3',
        phase: 'failed',
        title: '',
        error: 'select audio track: no audio track found in media file',
      });
      es.emit('generation_batch_progress', running(item('m3', 'C', 'failed', 'error')));
    });
    expect(result.current.feed.at(-1)).toMatchObject({
      kind: 'failed',
      reason: 'error',
      error: 'select audio track: no audio track found in media file',
    });
  });

  it('🔴 #11: batch rows are about the batch — written once per (batch_id, status), not per tick', () => {
    const { result, es } = connected();
    act(() => es.emit('generation_batch_progress', running(item('m1', 'A', 'running'))));
    expect(result.current.feed.filter((r) => r.kind === 'batch')).toHaveLength(0);
    act(() => {
      es.emit(
        'generation_batch_progress',
        terminal('complete', { fail_count: 2, success_count: 1 })
      );
      es.emit(
        'generation_batch_progress',
        terminal('complete', { fail_count: 2, success_count: 1 })
      );
    });
    const batchRows = result.current.feed.filter((r) => r.kind === 'batch');
    expect(batchRows).toHaveLength(1);
    expect(batchRows[0]).toMatchObject({ status: 'complete', successCount: 1, failCount: 2 });
    expect(batchRows[0]).not.toHaveProperty('mediaId');
  });

  it('a batch terminal ends membership: the same film run later from its detail page is a single job', () => {
    const { result, es } = connected();
    act(() => {
      es.emit('generation_batch_progress', running(item('m1', 'A', 'running')));
      es.emit('generation_batch_progress', terminal('complete'));
      // title '' — so it is the ended membership, not a solo title, that decides.
      es.emit('transcription_progress', { media_id: 'm1', phase: 'transcribing', title: '' });
    });
    expect(result.current.singleJobs['m1']).toMatchObject({ mediaId: 'm1', phase: 'transcribing' });
  });

  // --- 🔴 #1, #2: subtitle_progress --------------------------------------

  it('🔴 #1: an embedded-track batch item shows up (subtitle_progress for a member)', () => {
    const { result, es } = connected();
    act(() => {
      es.emit('generation_batch_progress', running(item('m1', 'A', 'running')));
      es.emit('subtitle_progress', {
        media_id: 'm1',
        media_type: 'movie',
        stage: 'extracting',
        message: '抽取內嵌字幕中…',
      });
      es.emit('subtitle_progress', {
        media_id: 'm1',
        media_type: 'movie',
        stage: 'translating',
        message: '翻譯中（第 1/4 段）',
      });
      es.emit('subtitle_progress', {
        media_id: 'm1',
        media_type: 'movie',
        stage: 'translating',
        message: '翻譯中（第 2/4 段）',
      });
    });
    expect(rowsFor(result.current.feed, 'm1')).toMatchObject([
      { kind: 'stage', stage: 'track', state: 'passed' },
      { kind: 'stage', stage: 'translating', state: 'live', percentage: null },
    ]);
  });

  it('🔴 #2: subtitle_progress for anything that is not a batch member is dropped (search, manual download)', () => {
    const { result, es } = connected();
    act(() => {
      es.emit('subtitle_progress', {
        media_id: 'x1',
        media_type: 'movie',
        stage: 'searching',
        message: 'Searching subtitle providers...',
      });
      es.emit('subtitle_progress', {
        media_id: 'x1',
        media_type: 'movie',
        stage: 'extracting',
        message: '',
      });
      es.emit('subtitle_progress', {
        media_id: 'x1',
        media_type: 'movie',
        stage: 'complete',
        message: 'Subtitle found and placed!',
      });
    });
    expect(result.current.feed).toEqual([]);
    expect(result.current.singleJobs).toEqual({});
  });

  // --- 🔴 #14, #15: connection honesty -----------------------------------

  it('connected follows onopen / onerror / stop()', () => {
    const result = render();
    expect(result.current.connected).toBe(false);
    act(() => result.current.startTracking());
    const es = MockEventSource.instances[0];
    expect(result.current.connected).toBe(false); // not until the socket says so
    act(() => es.open());
    expect(result.current.connected).toBe(true);
    act(() => es.fail());
    expect(result.current.connected).toBe(false);
    act(() => es.open());
    act(() => result.current.stop());
    expect(result.current.connected).toBe(false);
  });

  it('a (re)opened stream demotes every row that was still live — the gap may have swallowed its end', () => {
    vi.useFakeTimers();
    try {
      const { result, es } = connected();
      act(() =>
        es.emit('translation_progress', {
          media_id: 'm1',
          phase: 'translating',
          percentage: 30,
          title: 'A',
        })
      );
      // The REAL path: error → 10s backoff → a NEW EventSource, which must be
      // wired again (listeners + onopen) for the next frames to land.
      act(() => es.fail());
      expect(result.current.connected).toBe(false);
      act(() => vi.advanceTimersByTime(10000));
      expect(MockEventSource.instances).toHaveLength(2);
      const reopened = MockEventSource.instances[1];
      act(() => reopened.open());
      expect(result.current.connected).toBe(true);
      expect(result.current.feed[0]).toMatchObject({ state: 'stopped', percentage: null });
      act(() =>
        reopened.emit('translation_progress', {
          media_id: 'm1',
          phase: 'translating',
          percentage: 50,
          title: 'A',
        })
      );
      expect(result.current.feed).toHaveLength(2);
      expect(result.current.feed[1]).toMatchObject({ state: 'live', percentage: 50 });
    } finally {
      vi.useRealTimers();
    }
  });

  it('a manual stop/start (tab hidden then shown) also leaves nothing claiming to run', () => {
    const { result, es } = connected();
    act(() =>
      es.emit('translation_progress', {
        media_id: 'm1',
        phase: 'translating',
        percentage: 30,
        title: 'A',
      })
    );
    act(() => es.fail());
    act(() => es.open());
    expect(result.current.feed[0]).toMatchObject({ state: 'stopped', percentage: null });
    // The next frame for the same stage opens a NEW live row: the gap stays visible.
    act(() =>
      es.emit('translation_progress', {
        media_id: 'm1',
        phase: 'translating',
        percentage: 50,
        title: 'A',
      })
    );
    expect(result.current.feed).toHaveLength(2);
    expect(result.current.feed[1]).toMatchObject({ state: 'live', percentage: 50 });
  });

  // --- 🔴 #17: the terminal queue is never converted ----------------------

  it('the terminal event full items[] is stripped BEFORE snakeToCamel', () => {
    const { es } = connected();
    act(() => es.emit('generation_batch_progress', terminal('complete')));
    expect(camel.inputs.length).toBeGreaterThan(0);
    for (const input of camel.inputs) {
      expect(input).not.toHaveProperty('items');
    }
  });

  // --- /ship CR fixes -----------------------------------------------------

  it('CR H2: a step that ends by FAILURE is stopped, not passed (no tick claiming it finished)', () => {
    const { result, es } = connected();
    act(() => {
      es.emit('transcription_progress', { media_id: 'm1', phase: 'transcribing', title: 'A' });
      es.emit('transcription_failed', {
        media_id: 'm1',
        phase: 'failed',
        title: 'A',
        error: 'transcribe: whisper: request timed out',
      });
    });
    expect(result.current.feed[0]).toMatchObject({ kind: 'stage', state: 'stopped' });
  });

  it('CR H2: the batch ending on the budget leaves the running step stopped; the steps before it passed', () => {
    const { result, es } = connected();
    act(() => {
      es.emit('generation_batch_progress', running(item('m2', 'B', 'running')));
      es.emit('transcription_extracting', { media_id: 'm2', phase: 'extracting', title: '' });
      es.emit('transcription_progress', { media_id: 'm2', phase: 'transcribing', title: '' });
      es.emit('generation_batch_progress', terminal('budget_ceiling'));
    });
    expect(rowsFor(result.current.feed, 'm2')).toMatchObject([
      { stage: 'extracting', state: 'passed' },
      { stage: 'transcribing', state: 'stopped' },
    ]);
  });

  it('CR H1: endBatch() ends a membership whose terminal this stream never saw', () => {
    const { result, es } = connected();
    act(() => {
      es.emit('generation_batch_progress', running(item('m1', 'A', 'running')));
      es.emit('transcription_progress', { media_id: 'm1', phase: 'transcribing', title: '' });
    });
    act(() => result.current.endBatch());
    expect(rowsFor(result.current.feed, 'm1')[0]).toMatchObject({ state: 'stopped' });
    // The same film run later from its detail page is a single job again.
    act(() =>
      es.emit('transcription_extracting', { media_id: 'm1', phase: 'extracting', title: '' })
    );
    expect(result.current.singleJobs['m1']).toMatchObject({ phase: 'extracting' });
    act(() => es.emit('transcription_complete', { media_id: 'm1', phase: 'complete', title: '' }));
    expect(result.current.feed.at(-1)).toMatchObject({ kind: 'done', mediaId: 'm1' });
  });

  it('CR H1: seedBatch() REPLACES membership — the previous batch film is no longer a member', () => {
    const { result, es } = connected();
    const seed = (id: string, status: 'running' | 'queued') => ({
      mediaId: id,
      title: id,
      mediaType: 'movie',
      seriesTitle: '',
      status,
      reason: '' as const,
    });
    act(() => result.current.seedBatch('b1', [seed('x', 'running')]));
    act(() => result.current.seedBatch('b2', [seed('y', 'running')]));
    act(() =>
      es.emit('transcription_progress', { media_id: 'x', phase: 'transcribing', title: '' })
    );
    expect(result.current.singleJobs['x']).toMatchObject({ mediaId: 'x' });
  });

  it('CR M4: only the RUNNING film is a member — a queued film run from its detail page is a single job', () => {
    const { result, es } = connected();
    act(() =>
      result.current.seedBatch('b1', [
        {
          mediaId: 'q1',
          title: '排隊中',
          mediaType: 'movie',
          seriesTitle: '',
          status: 'queued',
          reason: '',
        },
      ])
    );
    act(() => {
      es.emit('transcription_extracting', { media_id: 'q1', phase: 'extracting', title: '' });
      es.emit('transcription_failed', {
        media_id: 'q1',
        phase: 'failed',
        title: '',
        error: 'select audio track: no audio track found in media file',
      });
    });
    expect(result.current.feed.at(-1)).toMatchObject({
      kind: 'failed',
      mediaId: 'q1',
      reason: null,
    });
  });

  it('CR M4: a solo run already on the film keeps its own result when the batch reaches it (busy_elsewhere)', () => {
    const { result, es } = connected();
    act(() => {
      es.emit('transcription_progress', { media_id: 'm3', phase: 'transcribing', title: '芭比' });
      es.emit('generation_batch_progress', running(item('m3', '芭比', 'running')));
      es.emit('generation_batch_progress', running(item('m3', '芭比', 'failed', 'busy_elsewhere')));
    });
    // The batch's refusal is logged…
    expect(result.current.feed.at(-1)).toMatchObject({ kind: 'failed', reason: 'busy_elsewhere' });
    // …but the solo run is untouched: still in flight, still live.
    expect(result.current.singleJobs['m3']).toMatchObject({ title: '芭比' });
    expect(rowsFor(result.current.feed, 'm3')[0]).toMatchObject({ kind: 'stage', state: 'live' });
    act(() =>
      es.emit('transcription_complete', { media_id: 'm3', phase: 'complete', title: '芭比' })
    );
    expect(result.current.feed.at(-1)).toMatchObject({ kind: 'done', mediaId: 'm3' });
    expect(result.current.singleJobs['m3']).toBeUndefined();
  });

  it('CR M4: an event carrying a backend title is a solo run even for the batch member id', () => {
    const { result, es } = connected();
    act(() => {
      es.emit('generation_batch_progress', running(item('m1', 'A', 'running')));
      es.emit('transcription_complete', { media_id: 'm1', phase: 'complete', title: '沙丘' });
    });
    expect(result.current.feed.at(-1)).toMatchObject({ kind: 'done', title: '沙丘' });
  });

  it('CR M8: a search-engine subtitle_progress terminal does not end a transcription step', () => {
    const { result, es } = connected();
    act(() => {
      es.emit('generation_batch_progress', running(item('m1', 'A', 'running')));
      es.emit('transcription_progress', { media_id: 'm1', phase: 'transcribing', title: '' });
      es.emit('subtitle_progress', {
        media_id: 'm1',
        media_type: 'movie',
        stage: 'complete',
        message: 'Subtitle found and placed!',
      });
    });
    expect(rowsFor(result.current.feed, 'm1')[0]).toMatchObject({ state: 'live' });
  });

  it('a pipeline failure ends the pipeline step as stopped', () => {
    const { result, es } = connected();
    act(() => {
      es.emit('generation_batch_progress', running(item('m1', 'A', 'running')));
      es.emit('subtitle_progress', {
        media_id: 'm1',
        media_type: 'movie',
        stage: 'extracting',
        message: '',
      });
      es.emit('subtitle_progress', {
        media_id: 'm1',
        media_type: 'movie',
        stage: 'failed',
        message: '',
      });
    });
    expect(rowsFor(result.current.feed, 'm1')[0]).toMatchObject({
      stage: 'track',
      state: 'stopped',
    });
  });

  it('the same result is not logged twice when the run first looked like a single job', () => {
    const { result, es } = connected();
    act(() => {
      // changed_item: running was lost — the run looks like an untitled single job
      es.emit('transcription_progress', { media_id: 'm1', phase: 'transcribing', title: '' });
      es.emit('transcription_complete', { media_id: 'm1', phase: 'complete', title: '' });
      es.emit('generation_batch_progress', running(item('m1', 'A', 'done')));
    });
    expect(rowsFor(result.current.feed, 'm1').filter((r) => r.kind === 'done')).toHaveLength(1);
    // …and the late batch word still names the film.
    expect(rowsFor(result.current.feed, 'm1').at(-1)).toMatchObject({ title: 'A' });
  });

  it('a result that arrives after endBatch() still settles the film step', () => {
    const { result, es } = connected();
    act(() => {
      es.emit('generation_batch_progress', running(item('m1', 'A', 'running')));
      es.emit('transcription_progress', { media_id: 'm1', phase: 'transcribing', title: '' });
    });
    act(() => result.current.endBatch());
    act(() =>
      es.emit('transcription_progress', { media_id: 'm1', phase: 'translating', title: '' })
    );
    act(() => es.emit('generation_batch_progress', running(item('m1', 'A', 'done'))));
    expect(result.current.feed.some((r) => r.kind === 'stage' && r.state === 'live')).toBe(false);
  });

  it('CR M4: a titled progress event is the solo run, not the batch member with the same id', () => {
    const { result, es } = connected();
    act(() => {
      es.emit('generation_batch_progress', running(item('m1', 'A', 'running')));
      es.emit('transcription_progress', { media_id: 'm1', phase: 'transcribing', title: '沙丘' });
    });
    expect(result.current.singleJobs['m1']).toMatchObject({ title: '沙丘', phase: 'transcribing' });
  });

  it('CR M4: the batch reaching a film an UNTITLED solo run already holds does not adopt it', () => {
    const { result, es } = connected();
    act(() => {
      es.emit('transcription_progress', { media_id: 'm3', phase: 'transcribing', title: '' });
      es.emit('generation_batch_progress', running(item('m3', '芭比', 'running')));
      // The solo run moves on before the batch reports its refusal.
      es.emit('translation_progress', {
        media_id: 'm3',
        phase: 'translating',
        percentage: 10,
        title: '',
      });
    });
    expect(result.current.singleJobs['m3']).toMatchObject({ phase: 'translating', percentage: 10 });
  });
});
