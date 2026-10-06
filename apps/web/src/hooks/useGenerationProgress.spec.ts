import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useGenerationProgress, TERMINAL_LOST_MESSAGE } from './useGenerationProgress';

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

// Media-id fixture convention (9R-18 AC 7): media ids are UUID STRINGS —
// mirror the prod creation path (uuid.New().String()); do NOT invent numeric ids.
const MOVIE_UUID = '4f8c2d1a-5b6e-4c7d-8e9f-0a1b2c3d4e5f';
const OTHER_UUID = '9ab0afe8-9acd-4f9e-9fed-a7c8d9e0f107';

/**
 * Build the FULL SSE Event struct exactly as the backend writes the `data:` line
 * (sse/handler.go `sendSSEEvent(w, type, event)`): the payload is DOUBLE-nested
 * under `.data`, snake_case, with `media_id` as the UUID-string movie id (9R-18).
 */
const wireEvent = (type: string, payload: Record<string, unknown>) => ({
  id: 'uuid-1',
  type,
  data: {
    job_id: 'job-9',
    media_id: MOVIE_UUID,
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

  // /ship CR L7 (dsr-6b) — 稍後再試 can unmount the dialog while a retry POST is
  // still in flight; its onSuccess then calls startTracking on a dead hook.
  it('does not open a stream after the hook has unmounted', () => {
    const { result, unmount } = renderHook(() => useGenerationProgress());
    const { startTracking } = result.current;
    unmount();
    act(() => startTracking(MOVIE_UUID));
    expect(MockEventSource.instances).toHaveLength(0);
  });

  it('[P0] opens EventSource only after startTracking and enters extracting', () => {
    const { result } = renderHook(() => useGenerationProgress());

    act(() => result.current.startTracking(MOVIE_UUID));

    expect(MockEventSource.instances).toHaveLength(1);
    expect(MockEventSource.instances[0].url).toBe('/api/v1/events');
    expect(result.current.progress.phase).toBe('extracting');
  });

  it('[P0] unwraps the DOUBLE-nested Event envelope — payload is parsed.data, snakeToCamel applied', () => {
    const { result } = renderHook(() => useGenerationProgress());
    act(() => result.current.startTracking(MOVIE_UUID));
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
    act(() => result.current.startTracking(MOVIE_UUID));
    const es = MockEventSource.instances[0];

    act(() =>
      es.emit(
        'transcription_progress',
        wireEvent('transcription_progress', { media_id: OTHER_UUID, phase: 'transcribing' })
      )
    );

    expect(result.current.progress.phase).toBe('extracting'); // unchanged
  });

  it('[P1] maps each wire phase event to its stage', () => {
    const { result } = renderHook(() => useGenerationProgress());
    act(() => result.current.startTracking(MOVIE_UUID));
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
    act(() => result.current.startTracking(MOVIE_UUID));
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

  // bugfix-j CR H2: the additive partial keys reach state; absence = full
  // success (partial=false, englishKeptBlocks=null — never 0-by-default).
  it('transcription_complete carries partial + english_kept_blocks into state; absent keys mean full success', () => {
    const { result } = renderHook(() => useGenerationProgress());
    act(() => result.current.startTracking(MOVIE_UUID));
    const es = MockEventSource.instances[0];

    act(() =>
      es.emit(
        'transcription_complete',
        wireEvent('transcription_complete', {
          phase: 'complete',
          srt_path: '/media/a.en.srt',
          zh_srt_path: '/media/a.zh-Hant.srt',
          message: '轉錄完成（部分翻譯失敗，5 句保留英文）',
          partial: true,
          english_kept_blocks: 5,
        })
      )
    );
    expect(result.current.progress.partial).toBe(true);
    expect(result.current.progress.englishKeptBlocks).toBe(5);

    // Fresh hook, full-success payload (keys absent).
    const second = renderHook(() => useGenerationProgress());
    act(() => second.result.current.startTracking(MOVIE_UUID));
    const es2 = MockEventSource.instances[1];
    act(() =>
      es2.emit(
        'transcription_complete',
        wireEvent('transcription_complete', {
          phase: 'complete',
          srt_path: '/media/b.en.srt',
          zh_srt_path: '/media/b.zh-Hant.srt',
          message: '轉錄完成',
        })
      )
    );
    expect(second.result.current.progress.partial).toBe(false);
    expect(second.result.current.progress.englishKeptBlocks).toBeNull();
  });

  it('[P0] transcription_failed records the failed-at stage from the last live phase', () => {
    const { result } = renderHook(() => useGenerationProgress());
    act(() => result.current.startTracking(MOVIE_UUID));
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
    act(() => result.current.startTracking(MOVIE_UUID));
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
    act(() => result.current.startTracking(MOVIE_UUID));
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
    act(() => result.current.startTracking(MOVIE_UUID));
    const es = MockEventSource.instances[0];

    act(() => result.current.reset());

    expect(result.current.progress.phase).toBe('idle');
    expect(es.readyState).toBe(2);
  });

  it('[P1] schedules a 10s backoff reconnect on SSE error (no polling fallback)', async () => {
    const { result } = renderHook(() => useGenerationProgress());
    act(() => result.current.startTracking(MOVIE_UUID));
    const es = MockEventSource.instances[0];

    act(() => es.triggerError());
    await act(async () => {
      await vi.advanceTimersByTimeAsync(10000);
    });

    expect(MockEventSource.instances.length).toBeGreaterThanOrEqual(2);
  });

  it('[P1] startTracking after an SSE error cancels the stale backoff timer (no healthy-stream bounce)', async () => {
    const { result } = renderHook(() => useGenerationProgress());
    act(() => result.current.startTracking(MOVIE_UUID));
    const es = MockEventSource.instances[0];

    act(() => es.triggerError()); // schedules the 10s backoff
    act(() => result.current.startTracking(MOVIE_UUID)); // user re-triggers immediately

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
    act(() => result.current.startTracking(MOVIE_UUID));
    const es = MockEventSource.instances[0];

    act(() => {
      const bad = new MessageEvent('transcription_progress', { data: '{not json' });
      es.listeners['transcription_progress']?.forEach((cb) => cb(bad));
    });

    expect(result.current.progress.phase).toBe('extracting');
  });
});

// ─── sub-4-3 AC #8: D6 subtitle_progress dual-family join ───────────────────

describe('useGenerationProgress D6 subtitle_progress family (sub-4-3 AC #8)', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    MockEventSource.instances = [];
    (global as Record<string, unknown>).EventSource = MockEventSource;
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  function tracked() {
    const rendered = renderHook(() => useGenerationProgress());
    act(() => rendered.result.current.startTracking(MOVIE_UUID));
    return { rendered, es: MockEventSource.instances[0] };
  }

  it.each([
    ['probing', 'extracting'],
    ['extracting', 'extracting'],
    ['translating', 'translating'],
  ] as const)('[P0] D6 stage %s maps to phase %s', (stage, phase) => {
    const { rendered, es } = tracked();
    act(() =>
      es.emit('subtitle_progress', wireEvent('subtitle_progress', { stage, message: 'm' }))
    );
    expect(rendered.result.current.progress.phase).toBe(phase);
  });

  it('[P0] D6 complete is terminal and fires onComplete — after an observed pipeline stage', () => {
    const onComplete = vi.fn();
    const rendered = renderHook(() => useGenerationProgress({ onComplete }));
    act(() => rendered.result.current.startTracking(MOVIE_UUID));
    const es = MockEventSource.instances[0];

    act(() =>
      es.emit('subtitle_progress', wireEvent('subtitle_progress', { stage: 'extracting' }))
    );
    act(() => es.emit('subtitle_progress', wireEvent('subtitle_progress', { stage: 'complete' })));

    expect(rendered.result.current.progress.phase).toBe('complete');
    expect(onComplete).toHaveBeenCalledOnce();
    expect(es.readyState).toBe(2);
  });

  it('[P0 CR M7] a SEARCH-flow D6 complete (no pipeline stage observed) must NOT terminalize generation tracking', () => {
    const onComplete = vi.fn();
    const rendered = renderHook(() => useGenerationProgress({ onComplete }));
    act(() => rendered.result.current.startTracking(MOVIE_UUID));
    const es = MockEventSource.instances[0];

    // The search engine emits searching → … → complete on the SAME event type
    // and media_id. Without a prior pipeline stage the terminal is ignored.
    act(() => es.emit('subtitle_progress', wireEvent('subtitle_progress', { stage: 'searching' })));
    act(() => es.emit('subtitle_progress', wireEvent('subtitle_progress', { stage: 'complete' })));

    expect(rendered.result.current.progress.phase).toBe('extracting'); // START state, untouched
    expect(onComplete).not.toHaveBeenCalled();
    expect(es.readyState).not.toBe(2);

    // The generation pipeline then really starts — its terminal IS honored.
    act(() =>
      es.emit('subtitle_progress', wireEvent('subtitle_progress', { stage: 'translating' }))
    );
    act(() => es.emit('subtitle_progress', wireEvent('subtitle_progress', { stage: 'complete' })));
    expect(rendered.result.current.progress.phase).toBe('complete');
    expect(onComplete).toHaveBeenCalledOnce();
  });

  it.each(['failed', 'skipped'] as const)('[P0] D6 %s is terminal failed', (stage) => {
    const { rendered, es } = tracked();
    act(() => es.emit('subtitle_progress', wireEvent('subtitle_progress', { stage: 'probing' })));
    act(() =>
      es.emit(
        'subtitle_progress',
        wireEvent('subtitle_progress', { stage, message: '無文字字幕軌' })
      )
    );
    expect(rendered.result.current.progress.phase).toBe('failed');
    expect(es.readyState).toBe(2);
  });

  it('drops D6 events for a foreign media_id — including episode UUIDs it does not track', () => {
    const { rendered, es } = tracked();
    act(() =>
      es.emit(
        'subtitle_progress',
        wireEvent('subtitle_progress', { media_id: OTHER_UUID, stage: 'extracting' })
      )
    );
    expect(rendered.result.current.progress.phase).toBe('extracting'); // START state, not from the event
    act(() =>
      es.emit(
        'subtitle_progress',
        wireEvent('subtitle_progress', { media_id: OTHER_UUID, stage: 'complete' })
      )
    );
    expect(rendered.result.current.progress.phase).toBe('extracting');
    expect(es.readyState).not.toBe(2);
  });

  it('ignores unmapped search-path stages (searching etc.) without state change', () => {
    const { rendered, es } = tracked();
    act(() => es.emit('subtitle_progress', wireEvent('subtitle_progress', { stage: 'searching' })));
    expect(rendered.result.current.progress.phase).toBe('extracting'); // unchanged START state
  });

  it('tracks an EPISODE media id — sub-3-2 v2 semantics (media row id, movie or episode)', () => {
    const EPISODE_UUID = '8fa9fed7-8fbc-4e8d-8edc-f6b7c8d9e006';
    const rendered = renderHook(() => useGenerationProgress());
    act(() => rendered.result.current.startTracking(EPISODE_UUID));
    const es = MockEventSource.instances[0];

    act(() =>
      es.emit(
        'subtitle_progress',
        wireEvent('subtitle_progress', { media_id: EPISODE_UUID, stage: 'translating' })
      )
    );
    expect(rendered.result.current.progress.phase).toBe('translating');
  });

  describe('9R-17 solo-run cost', () => {
    it('picks up spent_usd / budget_usd, keeps the last figure when an event omits them', () => {
      const { result } = renderHook(() => useGenerationProgress());
      act(() => result.current.startTracking(MOVIE_UUID));
      const es = MockEventSource.instances[0];
      expect(result.current.progress.spentUsd).toBeNull();

      act(() =>
        es.emit(
          'translation_progress',
          wireEvent('translation_progress', {
            phase: 'translating',
            percentage: 40,
            spent_usd: 0.31,
            budget_usd: 2.5,
          })
        )
      );
      expect(result.current.progress.spentUsd).toBe(0.31);
      expect(result.current.progress.budgetUsd).toBe(2.5);

      act(() =>
        es.emit(
          'translation_progress',
          wireEvent('translation_progress', { phase: 'translating', percentage: 60 })
        )
      );
      expect(result.current.progress.spentUsd).toBe(0.31);

      act(() =>
        es.emit(
          'transcription_complete',
          wireEvent('transcription_complete', {
            phase: 'complete',
            srt_path: '/x.srt',
            spent_usd: 0.42,
            budget_usd: 2.5,
          })
        )
      );
      expect(result.current.progress.spentUsd).toBe(0.42);
    });

    it('a batch item (no cost keys) stays null — never 0', () => {
      const { result } = renderHook(() => useGenerationProgress());
      act(() => result.current.startTracking(MOVIE_UUID));
      act(() =>
        MockEventSource.instances[0].emit(
          'transcription_progress',
          wireEvent('transcription_progress', { phase: 'transcribing' })
        )
      );
      expect(result.current.progress.spentUsd).toBeNull();
      expect(result.current.progress.budgetUsd).toBeNull();
    });
  });

  // disc-2026-10-single-generate-ignores-embedded-english-b — the solo job id
  // decides which terminal is ours (-a §1b: the ASR leg's own complete and the
  // pipeline's D6 complete fire BEFORE the job's terminal).
  describe('solo job id (pipeline mode)', () => {
    function tracked(jobId?: string) {
      const onComplete = vi.fn();
      const { result } = renderHook(() => useGenerationProgress({ onComplete }));
      act(() => result.current.startTracking(MOVIE_UUID, jobId));
      const es = MockEventSource.instances[0];
      return { result, es, onComplete };
    }

    it('[P0] with a job id, only the complete carrying THAT id ends tracking; earlier completes are notes', () => {
      const { result, es, onComplete } = tracked('solo-1');

      act(() =>
        es.emit(
          'transcription_extracting',
          wireEvent('transcription_extracting', {
            job_id: 'solo-1',
            phase: 'extracting',
            message: '正在檢查片內字幕…',
            predicted_route: 'asr',
          })
        )
      );
      expect(result.current.progress.route).toBe('asr');

      // ① the ASR leg's own complete — a different job id, no cost, no route
      act(() =>
        es.emit(
          'transcription_complete',
          wireEvent('transcription_complete', {
            job_id: 'asr-leg-9',
            phase: 'complete',
            zh_srt_path: '/m/x.zh-Hant.srt',
            message: '轉錄完成',
          })
        )
      );
      expect(result.current.progress.phase).not.toBe('complete');
      expect(onComplete).not.toHaveBeenCalled();
      expect(es.readyState).not.toBe(2);

      // ② the pipeline's D6 terminal — item-level news
      act(() =>
        es.emit(
          'subtitle_progress',
          wireEvent('subtitle_progress', { stage: 'extracting', media_type: 'movie', message: '' })
        )
      );
      act(() =>
        es.emit(
          'subtitle_progress',
          wireEvent('subtitle_progress', { stage: 'complete', media_type: 'movie', message: '' })
        )
      );
      expect(result.current.progress.phase).not.toBe('complete');
      expect(onComplete).not.toHaveBeenCalled();

      // ③ ours
      act(() =>
        es.emit(
          'transcription_complete',
          wireEvent('transcription_complete', {
            job_id: 'solo-1',
            phase: 'complete',
            route: 'asr',
            zh_srt_path: '/m/x.zh-Hant.srt',
            message: '轉錄完成',
            spent_usd: 1.36,
            budget_usd: 5,
          })
        )
      );
      expect(result.current.progress.phase).toBe('complete');
      expect(result.current.progress.route).toBe('asr');
      expect(result.current.progress.zhSrtPath).toBe('/m/x.zh-Hant.srt');
      expect(result.current.progress.spentUsd).toBe(1.36);
      expect(result.current.progress.budgetUsd).toBe(5);
      expect(onComplete).toHaveBeenCalledTimes(1);
      expect(es.readyState).toBe(2);
    });

    it('a failed carrying another job id is a note; ours ends tracking as failed', () => {
      const { result, es } = tracked('solo-2');
      act(() =>
        es.emit(
          'transcription_failed',
          wireEvent('transcription_failed', { job_id: 'other', phase: 'failed', error: 'x' })
        )
      );
      expect(result.current.progress.phase).toBe('extracting');
      act(() =>
        es.emit(
          'transcription_failed',
          wireEvent('transcription_failed', {
            job_id: 'solo-2',
            phase: 'failed',
            message: '這部影片沒有可用的字幕來源，也沒有設定語音辨識',
            error: 'skipped',
          })
        )
      );
      expect(result.current.progress.phase).toBe('failed');
      expect(result.current.progress.message).toBe(
        '這部影片沒有可用的字幕來源，也沒有設定語音辨識'
      );
      expect(es.readyState).toBe(2);
    });

    it("the terminal's route overrides the start event's prediction", () => {
      const { result, es } = tracked('solo-3');
      act(() =>
        es.emit(
          'transcription_extracting',
          wireEvent('transcription_extracting', {
            job_id: 'solo-3',
            phase: 'extracting',
            predicted_route: 'extract',
          })
        )
      );
      expect(result.current.progress.route).toBe('extract');
      act(() =>
        es.emit(
          'subtitle_progress',
          wireEvent('subtitle_progress', {
            stage: 'translating',
            media_type: 'movie',
            message: '翻譯中（第 2/7 段）',
          })
        )
      );
      expect(result.current.progress.phase).toBe('translating');
      act(() =>
        es.emit(
          'transcription_complete',
          wireEvent('transcription_complete', {
            job_id: 'solo-3',
            phase: 'complete',
            route: 'deliver_direct',
            zh_srt_path: '/m/y.zh-Hant.srt',
            message: '字幕已生成（直接使用片內中文字幕，沒有花錢）',
            spent_usd: 0,
            budget_usd: 2,
          })
        )
      );
      expect(result.current.progress.route).toBe('deliver_direct');
      expect(result.current.progress.spentUsd).toBe(0);
    });

    it('WITHOUT a job id (legacy / attach) the first complete is terminal, as before', () => {
      const { result, es, onComplete } = tracked();
      act(() =>
        es.emit(
          'transcription_complete',
          wireEvent('transcription_complete', { job_id: 'whatever', phase: 'complete' })
        )
      );
      expect(result.current.progress.phase).toBe('complete');
      expect(onComplete).toHaveBeenCalledTimes(1);
      expect(result.current.progress.route).toBeNull();
    });

    it('a NOTE never overwrites the stage message (D6 terminals are English log lines)', () => {
      const { result, es } = tracked('solo-5');
      act(() =>
        es.emit(
          'subtitle_progress',
          wireEvent('subtitle_progress', {
            stage: 'translating',
            media_type: 'movie',
            message: '翻譯中（第 3/7 段）',
          })
        )
      );
      act(() =>
        es.emit(
          'subtitle_progress',
          wireEvent('subtitle_progress', {
            stage: 'complete',
            media_type: 'movie',
            message: 'subtitle generated',
          })
        )
      );
      act(() =>
        es.emit(
          'transcription_complete',
          wireEvent('transcription_complete', {
            job_id: 'asr-leg',
            phase: 'complete',
            message: '轉錄完成',
          })
        )
      );
      expect(result.current.progress.phase).toBe('translating');
      expect(result.current.progress.message).toBe('翻譯中（第 3/7 段）');
    });

    describe('lost solo terminal (CR M3)', () => {
      function trackedWithProbe(inProgress: boolean | Error) {
        const probe = vi.fn(() =>
          inProgress instanceof Error ? Promise.reject(inProgress) : Promise.resolve(inProgress)
        );
        const onComplete = vi.fn();
        const { result } = renderHook(() =>
          useGenerationProgress({ onComplete, probeInProgress: probe })
        );
        act(() => result.current.startTracking(MOVIE_UUID, 'solo-6'));
        const es = MockEventSource.instances[MockEventSource.instances.length - 1];
        act(() =>
          es.emit(
            'subtitle_progress',
            wireEvent('subtitle_progress', {
              stage: 'extracting',
              media_type: 'movie',
              message: '',
            })
          )
        );
        return { result, es, probe, onComplete };
      }

      it('D6 complete + no solo terminal → after the wait the server says idle → ends as complete with the lost-result sentence', async () => {
        const { result, es, probe, onComplete } = trackedWithProbe(false);
        act(() =>
          es.emit(
            'subtitle_progress',
            wireEvent('subtitle_progress', {
              stage: 'complete',
              media_type: 'movie',
              message: 'subtitle generated',
            })
          )
        );
        expect(result.current.progress.phase).toBe('extracting');
        expect(probe).not.toHaveBeenCalled();

        await act(async () => {
          await vi.advanceTimersByTimeAsync(5000);
        });

        expect(probe).toHaveBeenCalledTimes(1);
        expect(result.current.progress.phase).toBe('complete');
        expect(result.current.progress.message).toBe(TERMINAL_LOST_MESSAGE);
        expect(onComplete).toHaveBeenCalledTimes(1);
        expect(es.readyState).toBe(2);
      });

      it('the real solo terminal arriving first cancels the probe', async () => {
        const { result, es, probe } = trackedWithProbe(false);
        act(() =>
          es.emit(
            'subtitle_progress',
            wireEvent('subtitle_progress', { stage: 'complete', media_type: 'movie', message: '' })
          )
        );
        act(() =>
          es.emit(
            'transcription_complete',
            wireEvent('transcription_complete', {
              job_id: 'solo-6',
              phase: 'complete',
              route: 'translate',
              message: '翻譯完成（翻譯片內英文字幕）',
            })
          )
        );
        await act(async () => {
          await vi.advanceTimersByTimeAsync(6000);
        });
        expect(probe).not.toHaveBeenCalled();
        expect(result.current.progress.message).toBe('翻譯完成（翻譯片內英文字幕）');
      });

      it('the server still says running → keep waiting, nothing ends', async () => {
        const { result, es, probe } = trackedWithProbe(true);
        act(() =>
          es.emit(
            'subtitle_progress',
            wireEvent('subtitle_progress', { stage: 'complete', media_type: 'movie', message: '' })
          )
        );
        await act(async () => {
          await vi.advanceTimersByTimeAsync(5000);
        });
        expect(probe).toHaveBeenCalledTimes(1);
        expect(result.current.progress.phase).toBe('extracting');
        expect(es.readyState).not.toBe(2);
      });

      it('D6 failed + idle server → ends as failed with the lost-result sentence', async () => {
        const { result, es } = trackedWithProbe(false);
        act(() =>
          es.emit(
            'subtitle_progress',
            wireEvent('subtitle_progress', {
              stage: 'failed',
              media_type: 'movie',
              message: 'boom',
            })
          )
        );
        await act(async () => {
          await vi.advanceTimersByTimeAsync(5000);
        });
        expect(result.current.progress.phase).toBe('failed');
        expect(result.current.progress.error).toBe(TERMINAL_LOST_MESSAGE);
      });
    });

    it('reset forgets the job id', () => {
      const { result, es } = tracked('solo-4');
      act(() => result.current.reset());
      act(() => result.current.startTracking(MOVIE_UUID));
      const es2 = MockEventSource.instances[MockEventSource.instances.length - 1] ?? es;
      act(() =>
        es2.emit(
          'transcription_complete',
          wireEvent('transcription_complete', { job_id: 'anything', phase: 'complete' })
        )
      );
      expect(result.current.progress.phase).toBe('complete');
    });
  });
});
