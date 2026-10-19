// Implements: <utility — no .pen counterpart>
/**
 * Words and colours for the generation workspace's live log (Story dsr-6d-c-2
 * AC #2/#5). The hook (useGenerationJobsFeed) records what happened; this turns
 * one recorded row into what the pane draws. The row kinds are specified on
 * the .pen spec screen F11-SPEC-LOG (tT3JX) — change them there too. A row is
 * two lines at most: stage · parts on the first, and for "one film + why"
 * (④ 失敗, ⑤ / ⑤b 已停止) the reason on its own second line (`detail`), so a
 * long reason never squeezes the film name away
 * (disc-2026-10-generation-queue-failure-reason).
 *
 * Why a run failed: the backend decides the reason code itself
 * (`models.SubtitleRunFailureReason`) and sends it as `failure_reason` on the
 * live events (`transcription_failed`, the batch's `changed_item`) and on the
 * activity record alike (disc-2026-10-live-activity-failure-reasons-a/-b). The
 * client only looks the code up in FAILURE_COPY_BY_REASON — it never matches
 * raw error strings, so the live log and the activity page always word the
 * same failure the same way. No code (or an unknown one) says 生成失敗.
 *
 * A batch item's `skipped` / `busy_elsewhere` is the structured
 * `changed_item.reason`, worded by generationQueueRow's `queueRowLabel`; the
 * reason code only refines `reason: "error"`.
 */
import type { FeedRow, FeedStage } from '../../hooks/useGenerationJobsFeed';
import { queueRowLabel, queueRowTitle } from './generationQueueRow';
import { GENERATION_STAGES } from './GenerationProgressV2';
import { usd } from '../../lib/currency';
import { creditProviderName } from './creditStop';
import { failureCopyForReason } from './runFailureCopy';

// The failure words live in a leaf module (AC #1 of
// disc-2026-10-generation-queue-failure-reason); re-exported so existing
// importers keep working.
export {
  FAILURE_COPY_BY_REASON,
  failureCopyForReason,
  type FailureCopy,
  type RunFailureReason,
} from './runFailureCopy';

export type FeedGlyph =
  | 'loader'
  | 'check'
  | 'circle-dashed'
  | 'triangle-alert'
  | 'circle-alert'
  | 'circle-pause';

export interface FeedRowView {
  glyph: FeedGlyph;
  /** Token colour for the glyph — always a `-text` twin or a `--text-*` step. */
  glyphClass: string;
  /** Spinning is a claim that work is running now (DESIGN.md Motion). */
  spin: boolean;
  stage: string;
  stageClass: string;
  /** Shown after the stage, each preceded by a `·`. */
  parts: string[];
  /**
   * The second line: shown whole under the stage word, with NO `·` in front —
   * the reason of ④ 失敗 / ⑤ / ⑤b 已停止 (F11-SPEC-LOG tT3JX). null = one line.
   */
  detail: string | null;
  trail: string | null;
  trailClass: string;
  /** What the log's live region reads for this row; null = not announced. */
  announce: string | null;
  /**
   * A stage row's state in words, for screen readers only — the glyph and the
   * colour that carry it are hidden from them (CR L10).
   */
  srState: string | null;
}

const ACCENT = 'text-[var(--accent-text)]';
const MUTED = 'text-[var(--text-muted)]';
const SECONDARY = 'text-[var(--text-secondary)]';
const PRIMARY = 'text-[var(--text-primary)]';
const SUCCESS = 'text-[var(--success-text)]';
const WARNING = 'text-[var(--warning-text)]';
const ERROR = 'text-[var(--error-text)]';

/** The stepper's frozen words for the three audio stages; `track` is the pipeline's. */
const STAGE_WORD: Record<FeedStage, string> = {
  extracting: GENERATION_STAGES[0],
  transcribing: GENERATION_STAGES[1],
  translating: GENERATION_STAGES[2],
  track: '抽取字幕',
};

const UNKNOWN_TITLE = '處理中的項目';
const THIS_BATCH = '本批次';

/** A row's film name; never a raw id (the hook already refuses id-shaped titles). */
function filmName(row: { title: string; seriesTitle: string }): string {
  return row.title ? queueRowTitle(row) : UNKNOWN_TITLE;
}

function view(
  v: Omit<FeedRowView, 'trail' | 'trailClass' | 'spin' | 'srState' | 'detail'> &
    Partial<FeedRowView>
): FeedRowView {
  return { spin: false, trail: null, trailClass: SECONDARY, srState: null, detail: null, ...v };
}

export function feedRowView(row: FeedRow): FeedRowView {
  switch (row.kind) {
    case 'stage': {
      const name = filmName(row);
      if (row.state === 'passed') {
        return view({
          glyph: 'check',
          glyphClass: MUTED,
          stage: STAGE_WORD[row.stage],
          stageClass: SECONDARY,
          parts: [name],
          announce: null,
          srState: '（這一步已完成）',
        });
      }
      if (row.state === 'stopped') {
        // Not a tick: the step did not finish, or we lost sight of it (a failure,
        // the batch ending, a stream gap). A tick here would claim it was done.
        return view({
          glyph: 'circle-dashed',
          glyphClass: MUTED,
          stage: STAGE_WORD[row.stage],
          stageClass: SECONDARY,
          parts: [name],
          announce: null,
          srState: '（已中斷）',
        });
      }
      return view({
        glyph: 'loader',
        glyphClass: ACCENT,
        spin: true,
        stage: STAGE_WORD[row.stage],
        stageClass: ACCENT,
        parts: [name],
        trail: row.stage === 'translating' && row.percentage !== null ? `${row.percentage}%` : null,
        trailClass: ACCENT,
        announce: null,
        srState: '（進行中）',
      });
    }
    case 'done': {
      const name = filmName(row);
      return view({
        glyph: 'check',
        glyphClass: SUCCESS,
        stage: '完成',
        stageClass: SUCCESS,
        parts: [name],
        announce: `完成：${name}`,
      });
    }
    case 'failed': {
      const name = filmName(row);
      if (row.reason === null) {
        const copy = failureCopyForReason(row.failureReason);
        if (copy.stopped) {
          return view({
            glyph: 'circle-alert',
            glyphClass: WARNING,
            stage: '已停止',
            stageClass: WARNING,
            parts: [name],
            detail: copy.text,
            announce: `已停止：${name}，${copy.text}`,
          });
        }
        return failedView(name, copy.text);
      }
      // A batch item that failed with `error`: the code says why — always as a
      // failure (AC #3). A real money stop does not end up here: the batch
      // itself ends as budget_ceiling / credit_exhausted and pauses the item.
      // Same call as the queue row on the left, so the two sides of the
      // workspace word one failure the same way.
      const text = queueRowLabel({
        status: 'failed',
        reason: row.reason,
        failureReason: row.failureReason,
      }).text;
      return failedView(name, text);
    }
    case 'batch': {
      const { status, successCount, failCount } = row;
      if (status === 'budget_ceiling') {
        return view({
          glyph: 'circle-alert',
          glyphClass: WARNING,
          stage: '已達預算上限',
          stageClass: WARNING,
          parts: [THIS_BATCH],
          // Money is a fact, not a state (DESIGN.md) — neutral, whatever the row says.
          trail: usd(row.budgetUsd),
          trailClass: PRIMARY,
          announce: '已達預算上限，批次已停止',
        });
      }
      if (status === 'credit_exhausted') {
        // F11-SPEC-LOG ⑪: the same family as ⑩ — whose account is a fact, so
        // the trail is neutral.
        const name = creditProviderName(row.creditProvider);
        return view({
          glyph: 'circle-alert',
          glyphClass: WARNING,
          stage: 'AI 帳號沒有額度',
          stageClass: WARNING,
          parts: [THIS_BATCH],
          trail: name,
          trailClass: PRIMARY,
          announce: `${name} 帳號沒有額度，批次已停止`,
        });
      }
      if (status === 'error') {
        return view({
          glyph: 'circle-alert',
          glyphClass: ERROR,
          stage: '批次發生錯誤',
          stageClass: ERROR,
          parts: [THIS_BATCH],
          announce: '批次發生錯誤',
        });
      }
      if (status === 'cancelled') {
        const done = `完成 ${successCount} 部`;
        return view({
          glyph: 'circle-pause',
          glyphClass: MUTED,
          stage: '批次已取消',
          stageClass: SECONDARY,
          parts: [THIS_BATCH, done],
          announce: `批次已取消：${done}`,
        });
      }
      if (failCount > 0) {
        // Same words as the verdict line; a run with failures is not green.
        const counts = `完成 ${successCount} 部、失敗 ${failCount} 部`;
        return view({
          glyph: 'check',
          glyphClass: MUTED,
          stage: '批次完成',
          stageClass: SECONDARY,
          parts: [THIS_BATCH, counts],
          announce: `批次完成：${counts}`,
        });
      }
      return view({
        glyph: 'check',
        glyphClass: SUCCESS,
        stage: '批次完成',
        stageClass: SUCCESS,
        parts: [THIS_BATCH],
        announce: '批次完成',
      });
    }
  }
}

function failedView(name: string, reason: string): FeedRowView {
  return view({
    glyph: 'triangle-alert',
    glyphClass: ERROR,
    stage: '失敗',
    stageClass: ERROR,
    parts: [name],
    detail: reason,
    announce: `失敗：${name}，${reason}`,
  });
}
