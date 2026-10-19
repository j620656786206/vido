import { describe, it, expect } from 'vitest';
import {
  queueRowLabel,
  queueRowTitle,
  queueRowBadge,
  queueRowSubStatus,
  type QueueRowView,
} from './generationQueueRow';
import { GENERATION_STAGES } from './GenerationProgressV2';

const view = (over: Partial<QueueRowView> = {}): QueueRowView => ({
  status: 'queued',
  reason: '',
  ...over,
});

describe('queueRowLabel (dsr-6d-b AC #3 — the row says what the backend says)', () => {
  it('done → 完成 in success text', () => {
    expect(queueRowLabel(view({ status: 'done' }))).toEqual({
      text: '完成',
      className: 'text-[var(--success-text)]',
      icon: 'check',
    });
  });

  it('queued → 排隊中, muted, no icon', () => {
    expect(queueRowLabel(view({ status: 'queued' }))).toEqual({
      text: '排隊中',
      className: 'text-[var(--text-muted)]',
      icon: null,
    });
  });

  it('paused → 已暫停 — 下次繼續, muted, pause icon', () => {
    expect(queueRowLabel(view({ status: 'paused' }))).toEqual({
      text: '已暫停 — 下次繼續',
      className: 'text-[var(--text-muted)]',
      icon: 'pause',
    });
  });

  it('cancelled → 已取消, muted, no icon', () => {
    expect(queueRowLabel(view({ status: 'cancelled' }))).toEqual({
      text: '已取消',
      className: 'text-[var(--text-muted)]',
      icon: null,
    });
  });

  describe('failed — the reason is the sentence (AC #3)', () => {
    it('skipped never says 已略過 (legacy unconfigured keys land here too)', () => {
      const label = queueRowLabel(view({ status: 'failed', reason: 'skipped' }));
      expect(label.text).toBe('沒有可用的字幕來源');
      expect(label.text.startsWith('已略過')).toBe(false);
      expect(label.className).toBe('text-[var(--error-text)]');
      expect(label.icon).toBe('alert');
    });

    it('busy_elsewhere names the other job', () => {
      expect(queueRowLabel(view({ status: 'failed', reason: 'busy_elsewhere' })).text).toBe(
        '這部正在別處處理'
      );
    });

    it('error → 生成失敗', () => {
      expect(queueRowLabel(view({ status: 'failed', reason: 'error' })).text).toBe('生成失敗');
    });

    it('empty reason (pre-dsr-6d-a server) falls back to 生成失敗', () => {
      expect(queueRowLabel(view({ status: 'failed', reason: '' })).text).toBe('生成失敗');
    });
  });

  // disc-2026-10-generation-queue-failure-reason AC #2/#12: the run's failure
  // code words an `error` row — the same sentence as the live log beside it.
  describe('failed + error — worded by the run failure code', () => {
    const failed = (failureReason: string | null | undefined) =>
      queueRowLabel(view({ status: 'failed', reason: 'error', failureReason }));

    it.each([
      ['target_not_writable', '影片所在的資料夾寫不進字幕'],
      ['cancelled', '中途停止'],
      ['no_audio_track', '找不到可用的音軌'],
    ])('%s → %s, red with the alert glyph', (code, text) => {
      expect(failed(code)).toEqual({
        text,
        className: 'text-[var(--error-text)]',
        icon: 'alert',
      });
    });

    it('a money code is still a red failure — `stopped` is not read here', () => {
      // A real budget / credit stop pauses the item; only a code that lost its
      // type on the way lands on failed + error, and the queue says 失敗.
      expect(failed('budget_exceeded')).toEqual({
        text: '已達預算上限',
        className: 'text-[var(--error-text)]',
        icon: 'alert',
      });
      expect(failed('credit_exhausted').className).toBe('text-[var(--error-text)]');
    });

    it.each([null, undefined, '', 'other', 'nope', 'constructor', '__proto__'])(
      'no code / unknown / prototype name %s → 生成失敗',
      (code) => {
        expect(failed(code).text).toBe('生成失敗');
      }
    );

    it('a code never rewords skipped / busy_elsewhere / a reason-less failure', () => {
      const code = 'target_not_writable';
      expect(
        queueRowLabel(view({ status: 'failed', reason: 'busy_elsewhere', failureReason: code }))
          .text
      ).toBe('這部正在別處處理');
      expect(
        queueRowLabel(view({ status: 'failed', reason: 'skipped', failureReason: code })).text
      ).toBe('沒有可用的字幕來源');
      expect(queueRowLabel(view({ status: 'failed', reason: '', failureReason: code })).text).toBe(
        '生成失敗'
      );
    });

    it('the sub-status line under the title repeats the worded reason', () => {
      const v = view({ status: 'failed', reason: 'error', failureReason: 'target_not_writable' });
      expect(queueRowSubStatus(v, queueRowLabel(v).text)).toBe('影片所在的資料夾寫不進字幕');
    });

    it('a code on a non-failed row changes nothing', () => {
      expect(
        queueRowLabel(view({ status: 'done', reason: 'error', failureReason: 'timeout' })).text
      ).toBe('完成');
    });
  });

  describe('running — the label is the CURRENT stage, verbatim from the frozen list', () => {
    it.each([
      ['extracting', GENERATION_STAGES[0]],
      ['transcribing', GENERATION_STAGES[1]],
      ['translating', GENERATION_STAGES[2]],
    ] as const)('%s → %s', (phase, text) => {
      const label = queueRowLabel(view({ status: 'running' }), phase);
      expect(label.text).toBe(text);
      expect(label.className).toBe('text-[var(--accent-text)] font-semibold');
      expect(label.icon).toBe(null);
    });

    it.each(['idle', 'complete', 'failed'] as const)(
      'phase %s carries no usable stage → 處理中',
      (phase) => {
        expect(queueRowLabel(view({ status: 'running' }), phase).text).toBe('處理中');
      }
    );

    it('no phase at all (just attached, no per-item event yet) → 處理中', () => {
      expect(queueRowLabel(view({ status: 'running' })).text).toBe('處理中');
      expect(queueRowLabel(view({ status: 'running' }), null).text).toBe('處理中');
    });

    it('the stage names are the frozen GENERATION_STAGES, not a private copy', () => {
      expect([
        queueRowLabel(view({ status: 'running' }), 'extracting').text,
        queueRowLabel(view({ status: 'running' }), 'transcribing').text,
        queueRowLabel(view({ status: 'running' }), 'translating').text,
      ]).toEqual([GENERATION_STAGES[0], GENERATION_STAGES[1], GENERATION_STAGES[2]]);
    });
  });

  it('a reason on a non-failed status never leaks into the label', () => {
    expect(queueRowLabel(view({ status: 'done', reason: 'skipped' })).text).toBe('完成');
  });
});

describe('queueRowTitle (dsr-6d-b AC #2 — 劇名 + 集數標題)', () => {
  it('prefixes the series title for an episode', () => {
    expect(queueRowTitle({ title: 'S04E07 第七章', seriesTitle: '怪奇物語' })).toBe(
      '怪奇物語 S04E07 第七章'
    );
  });

  it('a movie (empty series title) is just the title', () => {
    expect(queueRowTitle({ title: '沙丘：第二部', seriesTitle: '' })).toBe('沙丘：第二部');
  });

  it('a missing seriesTitle key (e2e mocks, old fixtures) never renders undefined', () => {
    expect(queueRowTitle({ title: '奧本海默' })).toBe('奧本海默');
    expect(queueRowTitle({ title: '奧本海默', seriesTitle: undefined })).toBe('奧本海默');
  });

  it('a whitespace-only series title is not a prefix', () => {
    expect(queueRowTitle({ title: '奧本海默', seriesTitle: '   ' })).toBe('奧本海默');
  });
});

describe('queueRowBadge / queueRowSubStatus (dsr-6d-c-1 — one vocabulary, two lengths)', () => {
  it.each([
    ['done', '完成'],
    ['failed', '失敗'],
    ['paused', '已暫停'],
    ['cancelled', '已取消'],
    ['queued', '排隊中'],
  ] as const)('badge for %s is %s', (status, text) => {
    expect(queueRowBadge(view({ status }), '轉錄中')).toBe(text);
  });

  it('a running badge shows the CURRENT stage', () => {
    expect(queueRowBadge(view({ status: 'running' }), '翻譯中')).toBe('翻譯中');
  });

  it('[P0] a cancelled row never claims 未處理 — the in-flight item may already be paid for', () => {
    const text = queueRowSubStatus(view({ status: 'cancelled' }), '已取消');
    expect(text).toBe('已取消');
    expect(text).not.toContain('未處理');
  });

  it('[P0] a done row never claims 繁中 — a run can keep English blocks or 簡體', () => {
    const text = queueRowSubStatus(view({ status: 'done' }), '完成');
    expect(text).toBe('已完成，字幕已寫入檔案');
    expect(text).not.toContain('繁中');
  });

  it('[P0] a failed row carries the REASON, not a generic sentence', () => {
    const v = view({ status: 'failed', reason: 'busy_elsewhere' });
    expect(queueRowSubStatus(v, queueRowLabel(v).text)).toBe('這部正在別處處理');
  });

  it('a running row appends the ellipsis to the stage; a queued row explains the wait', () => {
    expect(queueRowSubStatus(view({ status: 'running' }), '轉錄中')).toBe('轉錄中…');
    expect(queueRowSubStatus(view({ status: 'queued' }), '排隊中')).toBe('等待前面項目完成');
  });
});
