import { describe, it, expect } from 'vitest';
import {
  FAILURE_COPY_BY_REASON,
  failureCopy,
  failureCopyForReason,
  feedRowView,
} from './generationEventCopy';

// dsr-6d-c-2 AC #5 — transcription_failed carries the backend's raw error
// (English, sometimes a server file path). The live log shows one Chinese
// sentence instead; the raw string never reaches the screen.
describe('failureCopy', () => {
  it.each([
    [
      'translate: translate: translation stopped at block 3: AI_BUDGET_EXCEEDED: per-run AI budget ceiling reached',
      '已達預算上限',
      true,
    ],
    ['select audio track: no audio track found in media file', '找不到可用的音軌', false],
    ['extract audio: ffmpeg not available', '伺服器缺少 ffmpeg', false],
    ['list audio tracks: ffmpeg not available', '伺服器缺少 ffmpeg', false],
    ['transcription unavailable and resume not possible', '語音辨識未設定', false],
    ['transcribe: transcribe chunk 2/9: AI_UNAUTHORIZED: invalid key', 'API 金鑰無效', false],
    ['transcribe: AI_TIMEOUT: provider did not answer', '處理逾時', false],
    // The backend's one-line run-deadline sentence (disc-2026-09-transcription-run-5min-hard-timeout).
    [
      'transcribe: transcription run timed out after 45m2s in transcribing (media 157 min, budget 4712 s — raise TRANSCRIPTION_SECONDS_PER_MEDIA_MINUTE): context deadline exceeded',
      '處理逾時',
      false,
    ],
    [
      'extract audio: audio extraction timed out: ffmpeg timed out after 33m24s on Movie.mkv (file 66.8 GB, timeout 2004 s — raise SUBTITLE_EXTRACT_PER_GB_SECONDS for slow disks): context deadline exceeded',
      '處理逾時',
      false,
    ],
    ['transcribe: whisper: API error: status 401 — {"error":"invalid key"}', 'API 金鑰無效', false],
    ['transcribe: whisper: OpenAI API key not configured', '語音辨識未設定', false],
    // disc-2026-09-asr-quota-429-treated-as-transient: the solo runner's sentence, and the legacy raw error.
    // No credit is a stop, not a failure (F11-SPEC-LOG ⑤b).
    [
      'OpenAI 帳號沒有額度了，請到 https://platform.openai.com/settings/organization/billing 儲值後再按一次生成',
      'AI 帳號沒有額度',
      true,
    ],
    [
      'transcribe: transcribe chunk 1/6: whisper: API error: AI_CREDIT_EXHAUSTED: openai account has no credit left (insufficient_quota)',
      'AI 帳號沒有額度',
      true,
    ],
    ['transcribe: whisper: request timed out', '處理逾時', false],
    ['extract audio: context deadline exceeded', '處理逾時', false],
  ])('%s → %s', (error, text, stopped) => {
    expect(failureCopy(error)).toEqual({ text, stopped });
  });

  it('falls back to 生成失敗 for anything it does not recognise, including nothing at all', () => {
    expect(
      failureCopy('save SRT: open /mnt/media/Movies/Dune (2024)/Dune.en.srt: permission denied')
    ).toEqual({
      text: '生成失敗',
      stopped: false,
    });
    expect(failureCopy('')).toEqual({ text: '生成失敗', stopped: false });
    expect(failureCopy(null)).toEqual({ text: '生成失敗', stopped: false });
    expect(failureCopy(undefined)).toEqual({ text: '生成失敗', stopped: false });
  });

  it('the first match wins — a budget stop inside a timeout-shaped message is still a budget stop', () => {
    expect(failureCopy('transcribe: timed out after AI_BUDGET_EXCEEDED: ceiling').stopped).toBe(
      true
    );
  });

  it('never returns any part of the raw backend string', () => {
    const raw = 'save SRT: open /mnt/media/secret/path.srt: no space left on device';
    const { text } = failureCopy(raw);
    expect(text).not.toMatch(/[A-Za-z/]/);
  });
});

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

  it('④ a batch failure uses the backend reason; `error` is refined by the member failure text', () => {
    expect(
      feedRowView({ seq: 1, kind: 'failed', ...film, reason: 'skipped', error: null }).parts
    ).toEqual(['奧本海默', '沒有可用的字幕來源']);
    expect(
      feedRowView({
        seq: 1,
        kind: 'failed',
        ...film,
        reason: 'error',
        error: 'extract audio: context deadline exceeded',
      }).parts
    ).toEqual(['奧本海默', '處理逾時']);
    expect(
      feedRowView({ seq: 1, kind: 'failed', ...film, reason: 'error', error: null }).parts
    ).toEqual(['奧本海默', '生成失敗']);
  });

  it('⑤ a single job stopped by the budget is 已停止 in ochre', () => {
    const v = feedRowView({
      seq: 1,
      kind: 'failed',
      ...film,
      reason: null,
      error: 'AI_BUDGET_EXCEEDED: x',
    });
    expect(v).toMatchObject({
      glyph: 'circle-alert',
      stage: '已停止',
      parts: ['奧本海默', '已達預算上限'],
    });
    expect(v.stageClass).toContain('--warning-text');
  });

  it('⑤b a single job stopped by an empty AI account is 已停止 in ochre, not 失敗', () => {
    const v = feedRowView({
      seq: 1,
      kind: 'failed',
      ...film,
      reason: null,
      error:
        'OpenAI 帳號沒有額度了，請到 https://platform.openai.com/settings/organization/billing 儲值後再按一次生成',
    });
    expect(v).toMatchObject({
      glyph: 'circle-alert',
      stage: '已停止',
      parts: ['奧本海默', 'AI 帳號沒有額度'],
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
    expect(batch('complete', 2)).toMatchObject({
      stage: '批次完成',
      parts: ['本批次', '完成 3 部、失敗 2 部'],
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
      feedRowView({ seq: 1, kind: 'failed', ...film, reason: null, error: '' }),
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

// disc-2026-10-activity-subtitle-run-time-b AC #6 — one copy of the failure words:
// the raw-string matcher and the activity record's reason codes share objects.
describe('FAILURE_COPY_BY_REASON', () => {
  it.each([
    ['AI_BUDGET_EXCEEDED: ceiling', 'budget_exceeded'],
    ['no audio track found', 'no_audio_track'],
    ['ffmpeg not available', 'ffmpeg_missing'],
    ['API key not configured', 'asr_unconfigured'],
    ['AI_CREDIT_EXHAUSTED: openai', 'credit_exhausted'],
    ['status 401', 'unauthorized'],
    ['deadline exceeded', 'timeout'],
    ['AI_INVALID_RESPONSE: Cannot parse AI response', 'other'],
  ] as const)('failureCopy(%s) is the %s entry itself', (raw, reason) => {
    expect(failureCopy(raw)).toBe(FAILURE_COPY_BY_REASON[reason]);
  });

  it('has exactly the ten contract codes', () => {
    expect(Object.keys(FAILURE_COPY_BY_REASON).sort()).toEqual(
      [
        'asr_unconfigured',
        'budget_exceeded',
        'cancelled',
        'credit_exhausted',
        'ffmpeg_missing',
        'no_audio_track',
        'other',
        'target_not_writable',
        'timeout',
        'unauthorized',
      ].sort()
    );
  });

  it('an unknown, missing or prototype-named code falls back to 生成失敗', () => {
    expect(failureCopyForReason('nope').text).toBe('生成失敗');
    expect(failureCopyForReason(undefined).text).toBe('生成失敗');
    expect(failureCopyForReason('constructor').text).toBe('生成失敗');
    expect(failureCopyForReason('target_not_writable').text).toBe('影片所在的資料夾寫不進字幕');
    expect(failureCopyForReason('cancelled').text).toBe('中途停止');
  });
});
