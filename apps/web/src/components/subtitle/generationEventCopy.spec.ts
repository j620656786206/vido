import { describe, it, expect } from 'vitest';
import {
  FAILURE_COPY_BY_REASON,
  failureCopyForReason,
  feedRowView,
  type RunFailureReason,
} from './generationEventCopy';
import { recentSubtitleDetail } from '../activity/recentEventCopy';
import { queueRowBadge, queueRowLabel } from './generationQueueRow';

// dsr-6d-c-2 AC #2 — the ten row kinds, as the view draws them.
describe('feedRowView', () => {
  const film = { mediaId: 'm1', title: '奧本海默', seriesTitle: '' };

  it('① a live stage: spinner + accent, and a percentage only while translating', () => {
    const v = feedRowView({
      seq: 1,
      kind: 'stage',
      ...film,
      stage: 'translating',
      state: 'live',
      pipeline: false,
      percentage: 45,
    });
    expect(v).toMatchObject({
      glyph: 'loader',
      spin: true,
      stage: '翻譯中',
      parts: ['奧本海默'],
      trail: '45%',
      announce: null,
    });
    expect(v.stageClass).toContain('--accent-text');
    expect(
      feedRowView({
        seq: 1,
        kind: 'stage',
        ...film,
        stage: 'transcribing',
        state: 'live',
        pipeline: false,
        percentage: null,
      }).trail
    ).toBeNull();
    expect(
      feedRowView({
        seq: 1,
        kind: 'stage',
        ...film,
        stage: 'track',
        state: 'live',
        pipeline: false,
        percentage: null,
      }).stage
    ).toBe('抽取字幕');
  });

  it('② a passed stage: still, neutral, no percentage, not announced', () => {
    const v = feedRowView({
      seq: 1,
      kind: 'stage',
      ...film,
      stage: 'extracting',
      state: 'passed',
      pipeline: false,
      percentage: null,
    });
    expect(v).toMatchObject({
      glyph: 'check',
      spin: false,
      stage: '提取音訊',
      trail: null,
      announce: null,
    });
    expect(v.glyphClass).toContain('--text-muted');
    expect(v.stageClass).toContain('--text-secondary');
  });

  it('③ done names the film and is announced', () => {
    const v = feedRowView({ seq: 1, kind: 'done', ...film });
    expect(v).toMatchObject({
      glyph: 'check',
      stage: '完成',
      parts: ['奧本海默'],
      announce: '完成：奧本海默',
    });
    expect(v.stageClass).toContain('--success-text');
  });

  // disc-2026-10-generation-queue-failure-reason AC #7/#11: the reason is the
  // row's SECOND line (`detail`), no longer a `·` part next to the film name.
  it('④ a batch failure uses the backend reason; `error` is refined by its reason code', () => {
    // disc-2026-10-no-text-source-skip-b AC #7: `skipped` is no longer ④ — it
    // is the grey 已略過 row (⑧'s tones), the reason still on line two.
    expect(
      feedRowView({ seq: 1, kind: 'failed', ...film, reason: 'skipped', failureReason: null })
    ).toMatchObject({
      glyph: 'circle-dashed',
      glyphClass: 'text-[var(--text-muted)]',
      stage: '已略過',
      stageClass: 'text-[var(--text-secondary)]',
      parts: ['奧本海默'],
      detail: '沒有可用的字幕來源',
      announce: '已略過：奧本海默，沒有可用的字幕來源',
    });
    expect(
      feedRowView({
        seq: 1,
        kind: 'failed',
        ...film,
        reason: 'skipped',
        failureReason: 'no_text_source',
      })
    ).toMatchObject({ stage: '已略過', detail: '沒有可用的字幕來源，也沒有設定語音辨識' });
    expect(
      feedRowView({ seq: 1, kind: 'failed', ...film, reason: 'error', failureReason: 'timeout' })
    ).toMatchObject({ parts: ['奧本海默'], detail: '處理逾時' });
    expect(
      feedRowView({ seq: 1, kind: 'failed', ...film, reason: 'error', failureReason: null })
    ).toMatchObject({ parts: ['奧本海默'], detail: '生成失敗' });
    // disc-2026-10-live-activity-failure-reasons-b: the batch's translate-then-
    // write failure (正式機 和平使者 S01E01) used to say 生成失敗 here.
    const notWritable = feedRowView({
      seq: 1,
      kind: 'failed',
      ...film,
      reason: 'error',
      failureReason: 'target_not_writable',
    });
    expect(notWritable).toMatchObject({
      stage: '失敗',
      parts: ['奧本海默'],
      detail: '影片所在的資料夾寫不進字幕',
      announce: '失敗：奧本海默，影片所在的資料夾寫不進字幕',
    });
    expect(notWritable.stageClass).toContain('--error-text');
    // A code refines `error` and `skipped`, never busy_elsewhere: that one
    // keeps its own words and stays a red failure, whatever code came with it.
    expect(
      feedRowView({
        seq: 1,
        kind: 'failed',
        ...film,
        reason: 'busy_elsewhere',
        failureReason: 'target_not_writable',
      })
    ).toMatchObject({ parts: ['奧本海默'], detail: '這部正在別處處理' });
    expect(
      feedRowView({
        seq: 1,
        kind: 'failed',
        ...film,
        reason: 'busy_elsewhere',
        failureReason: 'no_text_source',
      })
    ).toMatchObject({ stage: '失敗', glyph: 'triangle-alert', detail: '這部正在別處處理' });
  });

  it('a single job: the code decides the words; no code, an unknown or a prototype name says 生成失敗', () => {
    const single = (failureReason: string | null) =>
      feedRowView({ seq: 1, kind: 'failed', ...film, reason: null, failureReason });
    expect(single('target_not_writable')).toMatchObject({
      glyph: 'triangle-alert',
      stage: '失敗',
      parts: ['奧本海默'],
      detail: '影片所在的資料夾寫不進字幕',
      announce: '失敗：奧本海默，影片所在的資料夾寫不進字幕',
    });
    expect(single('cancelled')).toMatchObject({
      stage: '失敗',
      parts: ['奧本海默'],
      detail: '中途停止',
      announce: '失敗：奧本海默，中途停止',
    });
    expect(single('queue_timeout')).toMatchObject({ parts: ['奧本海默'], detail: '處理逾時' });
    for (const code of [null, '', 'nope', 'constructor', '__proto__']) {
      expect(single(code)).toMatchObject({
        stage: '失敗',
        parts: ['奧本海默'],
        detail: '生成失敗',
      });
    }
  });

  // Contract v2 (#855) + disc-2026-10-no-text-source-skip-b AC #7: a single
  // job with nothing to generate from is a grey 已略過, not 失敗 · 生成失敗.
  it('a single job with no_text_source is the grey 已略過 row', () => {
    const v = feedRowView({
      seq: 1,
      kind: 'failed',
      ...film,
      reason: null,
      failureReason: 'no_text_source',
    });
    expect(v).toEqual({
      glyph: 'circle-dashed',
      glyphClass: 'text-[var(--text-muted)]',
      spin: false,
      stage: '已略過',
      stageClass: 'text-[var(--text-secondary)]',
      parts: ['奧本海默'],
      detail: '沒有可用的字幕來源，也沒有設定語音辨識',
      trail: null,
      trailClass: 'text-[var(--text-secondary)]',
      announce: '已略過：奧本海默，沒有可用的字幕來源，也沒有設定語音辨識',
      srState: null,
    });
    // asr_unconfigured on a single job is still a failure (the old no-key path
    // the user has to fix); only no_text_source is flagged skipped.
    expect(
      feedRowView({
        seq: 1,
        kind: 'failed',
        ...film,
        reason: null,
        failureReason: 'asr_unconfigured',
      })
    ).toMatchObject({ stage: '失敗', detail: '語音辨識未設定' });
  });

  it('⑤ a single job stopped by the budget is 已停止 in ochre', () => {
    const v = feedRowView({
      seq: 1,
      kind: 'failed',
      ...film,
      reason: null,
      failureReason: 'budget_exceeded',
    });
    expect(v).toMatchObject({
      glyph: 'circle-alert',
      stage: '已停止',
      parts: ['奧本海默'],
      detail: '已達預算上限',
      announce: '已停止：奧本海默，已達預算上限',
    });
    expect(v.stageClass).toContain('--warning-text');
  });

  it('⑤b a single job stopped by an empty AI account is 已停止 in ochre, not 失敗', () => {
    const v = feedRowView({
      seq: 1,
      kind: 'failed',
      ...film,
      reason: null,
      failureReason: 'credit_exhausted',
    });
    expect(v).toMatchObject({
      glyph: 'circle-alert',
      stage: '已停止',
      parts: ['奧本海默'],
      detail: 'AI 帳號沒有額度',
      announce: '已停止：奧本海默，AI 帳號沒有額度',
    });
    expect(v.stageClass).toContain('--warning-text');
  });

  it('an episode is prefixed with its show; no title at all says 處理中的項目', () => {
    expect(
      feedRowView({
        seq: 1,
        kind: 'done',
        mediaId: 'e',
        title: 'S04E07 第七章',
        seriesTitle: '怪奇物語',
      }).parts
    ).toEqual(['怪奇物語 S04E07 第七章']);
    expect(
      feedRowView({ seq: 1, kind: 'done', mediaId: 'e', title: '', seriesTitle: '怪奇物語' }).parts
    ).toEqual(['處理中的項目']);
  });

  const batch = (
    status: 'complete' | 'cancelled' | 'error' | 'budget_ceiling' | 'credit_exhausted',
    failCount = 0,
    creditProvider = ''
  ) =>
    feedRowView({
      seq: 1,
      kind: 'batch',
      batchId: 'b',
      status,
      successCount: 3,
      failCount,
      budgetUsd: 5,
      creditProvider,
    });

  it('⑥–⑩ batch rows are about 本批次', () => {
    expect(batch('complete')).toMatchObject({ stage: '批次完成', parts: ['本批次'], trail: null });
    expect(batch('complete').stageClass).toContain('--success-text');
    // F11-SPEC-LOG ⑦ stays one line: the counts are about the batch, not a
    // film's reason (Sally 裁定 3) — no second line.
    expect(batch('complete', 2)).toMatchObject({
      stage: '批次完成',
      parts: ['本批次', '完成 3 部、失敗 2 部'],
      detail: null,
    });
    expect(batch('complete', 2).stageClass).not.toContain('--success');
    expect(batch('cancelled')).toMatchObject({
      glyph: 'circle-pause',
      stage: '批次已取消',
      parts: ['本批次', '完成 3 部'],
    });
    expect(batch('error')).toMatchObject({
      glyph: 'circle-alert',
      stage: '批次發生錯誤',
      parts: ['本批次'],
    });
    expect(batch('error').stageClass).toContain('--error-text');
    const budget = batch('budget_ceiling');
    expect(budget).toMatchObject({ stage: '已達預算上限', parts: ['本批次'], trail: '$5.00' });
    expect(budget.stageClass).toContain('--warning-text');
    expect(budget.trailClass).toContain('--text-primary');
  });

  it('⑪ a batch stopped by an empty AI account names whose account, neutral', () => {
    const v = batch('credit_exhausted', 0, 'openai');
    expect(v).toMatchObject({
      glyph: 'circle-alert',
      stage: 'AI 帳號沒有額度',
      parts: ['本批次'],
      trail: 'OpenAI',
    });
    expect(v.stageClass).toContain('--warning-text');
    expect(v.trailClass).toContain('--text-primary');
    expect(batch('credit_exhausted', 0, 'anthropic').trail).toBe('Anthropic');
    // An older server sends no provider — say AI, never an empty trail.
    expect(batch('credit_exhausted').trail).toBe('AI');
  });

  // disc-2026-10-generation-queue-failure-reason AC #7/#9/#11: only "one film +
  // why" (④ 失敗, ⑤ / ⑤b 已停止) has a second line; every other row is one line.
  it('only ④ ⑤ ⑤b carry a second line; stage, done and batch rows have none', () => {
    const stage = (state: 'live' | 'passed' | 'stopped') =>
      feedRowView({
        seq: 1,
        kind: 'stage',
        ...film,
        stage: 'transcribing',
        state,
        percentage: null,
        pipeline: false,
      });
    const oneLine = [
      stage('live'),
      stage('passed'),
      stage('stopped'),
      feedRowView({ seq: 1, kind: 'done', ...film }),
      batch('complete'),
      batch('complete', 2),
      batch('cancelled'),
      batch('error'),
      batch('budget_ceiling'),
      batch('credit_exhausted', 0, 'openai'),
    ];
    for (const v of oneLine) expect(v.detail).toBeNull();
    // The batch rows keep their parts exactly as before (Sally 裁定 3).
    expect(batch('cancelled').parts).toEqual(['本批次', '完成 3 部']);
    expect(batch('budget_ceiling')).toMatchObject({ parts: ['本批次'], trail: '$5.00' });
    expect(stage('live').parts).toEqual(['奧本海默']);
  });

  it('no row ever uses a base semantic colour as text', () => {
    const all = [
      feedRowView({
        seq: 1,
        kind: 'stage',
        ...film,
        stage: 'translating',
        state: 'live',
        pipeline: false,
        percentage: 1,
      }),
      feedRowView({ seq: 1, kind: 'done', ...film }),
      feedRowView({ seq: 1, kind: 'failed', ...film, reason: null, failureReason: null }),
      feedRowView({
        seq: 1,
        kind: 'failed',
        ...film,
        reason: null,
        failureReason: 'budget_exceeded',
      }),
      feedRowView({
        seq: 1,
        kind: 'failed',
        ...film,
        reason: null,
        failureReason: 'no_text_source',
      }),
      batch('complete'),
      batch('error'),
      batch('budget_ceiling'),
      batch('credit_exhausted', 0, 'openai'),
    ];
    for (const v of all) {
      for (const cls of [v.glyphClass, v.stageClass, v.trailClass]) {
        expect(cls).not.toMatch(/--(success|warning|error|info|accent-primary)\)/);
      }
    }
  });

  it('CR H2: a step that did not finish is never a tick', () => {
    const v = feedRowView({
      seq: 1,
      kind: 'stage',
      ...film,
      stage: 'transcribing',
      state: 'stopped',
      percentage: null,
      pipeline: false,
    });
    expect(v).toMatchObject({ glyph: 'circle-dashed', spin: false, trail: null, announce: null });
    expect(v.glyphClass).toContain('--text-muted');
  });

  it('CR L10: a stage row says its state in words for screen readers', () => {
    const stage = (state: 'live' | 'passed' | 'stopped') =>
      feedRowView({
        seq: 1,
        kind: 'stage',
        ...film,
        stage: 'transcribing',
        state,
        percentage: null,
        pipeline: false,
      }).srState;
    expect(stage('live')).toBe('（進行中）');
    expect(stage('passed')).toBe('（這一步已完成）');
    expect(stage('stopped')).toBe('（已中斷）');
    expect(feedRowView({ seq: 1, kind: 'done', ...film }).srState).toBeNull();
  });
});

// disc-2026-10-live-activity-failure-reasons-b AC #5 + disc-2026-10-generation-
// queue-failure-reason AC #10 — the live log, the activity record AND the queue
// row word every reason code the same way. Iterates the table's own keys, so a
// code added later is covered without touching this test.
describe('live log ⇄ activity record ⇄ queue: one sentence per reason code', () => {
  const codes = Object.keys(FAILURE_COPY_BY_REASON) as RunFailureReason[];

  it.each(codes)('%s', (code) => {
    const { text, stopped, skipped } = FAILURE_COPY_BY_REASON[code];
    const single = feedRowView({
      seq: 1,
      kind: 'failed',
      mediaId: 'm1',
      title: '奧本海默',
      seriesTitle: '',
      reason: null,
      failureReason: code,
    });
    // The live log's reason line is the table's sentence…
    expect(single.detail).toBe(text);
    expect(single.parts).toEqual(['奧本海默']);
    expect(single.stage).toBe(skipped ? '已略過' : stopped ? '已停止' : '失敗');
    // …and the activity record's description ends with the very same words.
    const activity = recentSubtitleDetail({
      kind: 'subtitle',
      result: 'failed',
      at: '2026-10-19T00:00:00Z',
      mediaType: 'movie',
      mediaTitle: '奧本海默',
      failureReason: code,
    });
    expect(activity).toBe(`奧本海默 · ${single.detail}`);
    // A batch `error` row with the same code says the same words…
    const batchRow = feedRowView({
      seq: 1,
      kind: 'failed',
      mediaId: 'm1',
      title: '奧本海默',
      seriesTitle: '',
      reason: 'error',
      failureReason: code,
    });
    expect(batchRow.detail).toBe(text);
    expect(batchRow.stage).toBe(skipped ? '已略過' : '失敗');
    // …and so does the queue row on the left of the same screen — a red
    // failure whatever the code's `stopped` says (a real money stop pauses),
    // a grey 已略過 when the code is flagged `skipped`.
    const queueView = { status: 'failed', reason: 'error', failureReason: code } as const;
    const queue = queueRowLabel(queueView);
    expect(queue.text).toBe(text);
    expect(queue.text).toBe(single.detail);
    expect(queue.text).toBe(batchRow.detail);
    expect(queueRowBadge(queueView, '')).toBe(skipped ? '已略過' : '失敗');
    expect(queue.className).toContain(skipped ? '--text-muted' : '--error-text');
    expect(queue.icon).toBe(skipped ? null : 'alert');
  });

  // disc-2026-10-no-text-source-skip-b AC #9: the skip the backend really
  // sends — a single job's code, and a batch `skipped` item carrying it — is
  // one sentence on all four surfaces, and all three call it a skip.
  it('no_text_source: one sentence, and 已略過, everywhere', () => {
    const want = FAILURE_COPY_BY_REASON.no_text_source.text;
    // Pinned verbatim: the backend's sentence (solo_runner.go) minus 「這部影片」.
    expect(want).toBe('沒有可用的字幕來源，也沒有設定語音辨識');
    const base = {
      seq: 1,
      kind: 'failed',
      mediaId: 'm1',
      title: '奧本海默',
      seriesTitle: '',
    } as const;
    const single = feedRowView({ ...base, reason: null, failureReason: 'no_text_source' });
    const batchRow = feedRowView({ ...base, reason: 'skipped', failureReason: 'no_text_source' });
    const queueView = {
      status: 'failed',
      reason: 'skipped',
      failureReason: 'no_text_source',
    } as const;
    const activity = recentSubtitleDetail({
      kind: 'subtitle',
      result: 'failed',
      at: '2026-10-19T00:00:00Z',
      mediaType: 'movie',
      mediaTitle: '奧本海默',
      failureReason: 'no_text_source',
    });
    expect(single.detail).toBe(want);
    expect(batchRow.detail).toBe(want);
    expect(queueRowLabel(queueView).text).toBe(want);
    expect(activity.split(' · ')[1]).toBe(want);
    expect(single.stage).toBe('已略過');
    expect(batchRow.stage).toBe('已略過');
    expect(queueRowBadge(queueView, '')).toBe('已略過');
  });
});

// disc-2026-10-activity-subtitle-run-time-b AC #6 — one copy of the failure words.
describe('FAILURE_COPY_BY_REASON', () => {
  it.each(Object.keys(FAILURE_COPY_BY_REASON) as RunFailureReason[])(
    'failureCopyForReason(%s) is the entry itself',
    (reason) => {
      expect(failureCopyForReason(reason)).toBe(FAILURE_COPY_BY_REASON[reason]);
    }
  );

  it('has exactly the twelve contract codes', () => {
    expect(Object.keys(FAILURE_COPY_BY_REASON).sort()).toEqual(
      [
        'asr_unconfigured',
        'budget_exceeded',
        'cancelled',
        'credit_exhausted',
        'ffmpeg_missing',
        'no_audio_track',
        'no_text_source',
        'other',
        'queue_timeout',
        'target_not_writable',
        'timeout',
        'unauthorized',
      ].sort()
    );
  });

  // disc-2026-10-no-text-source-skip-b AC #1/#9: no_text_source is the one
  // "not done, nothing broken" code, and no entry is both stopped and skipped.
  it('only no_text_source is skipped; nothing is both stopped and skipped', () => {
    const entries = Object.entries(FAILURE_COPY_BY_REASON);
    expect(entries.filter(([, c]) => c.skipped).map(([k]) => k)).toEqual(['no_text_source']);
    for (const [, c] of entries) expect(c.stopped && c.skipped).toBe(false);
  });

  it('an unknown, missing or prototype-named code falls back to 生成失敗', () => {
    expect(failureCopyForReason('nope').text).toBe('生成失敗');
    expect(failureCopyForReason(undefined).text).toBe('生成失敗');
    expect(failureCopyForReason('constructor').text).toBe('生成失敗');
    expect(failureCopyForReason('target_not_writable').text).toBe('影片所在的資料夾寫不進字幕');
    expect(failureCopyForReason('cancelled').text).toBe('中途停止');
    expect(failureCopyForReason('queue_timeout').text).toBe('處理逾時');
  });
});
