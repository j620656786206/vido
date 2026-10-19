// Implements: <utility — no .pen counterpart>
/**
 * Words and colours for the generation workspace's live log (Story dsr-6d-c-2
 * AC #2/#5). The hook (useGenerationJobsFeed) records what happened; this turns
 * one recorded row into what the pane draws. The ten row kinds are specified on
 * the .pen spec screen F11-SPEC-LOG (tT3JX) — change them there too.
 *
 * `failureCopy` — the Chinese sentence for a failed generation run.
 *
 * `transcription_failed` carries the backend's raw error — English, wrapped in
 * stage prefixes, and for a filesystem failure it includes a server path
 * (`failJob`, transcription_service.go). There is no machine-readable reason
 * code on that event; the only structure is the upper-case `AI_*` prefixes the
 * AI layer embeds in the string. So this maps known substrings to one sentence
 * each and says 生成失敗 for everything else. The raw string is never returned
 * (the server log already has it).
 *
 * Batch items do not come through here first: their reason is the structured
 * `changed_item.reason`, worded by generationQueueRow's `queueRowLabel`. This
 * table only refines the vague `reason: "error"` case.
 */
import type { FeedRow, FeedStage } from '../../hooks/useGenerationJobsFeed';
import { queueRowLabel, queueRowTitle } from './generationQueueRow';
import { GENERATION_STAGES } from './GenerationProgressV2';
import { usd } from '../../lib/currency';
import { creditProviderName } from './creditStop';

export interface FailureCopy {
  text: string;
  /**
   * The run stopped for money — the AI budget ceiling, or an AI account with no
   * credit left. That is "you asked and it did not happen" (ochre), not "it
   * broke" (cinnabar) — the log renders it as 已停止 (F11-SPEC-LOG ⑤ / ⑤b).
   */
  stopped: boolean;
}

/**
 * The reason codes a finished subtitle run carries on GET /api/v1/activity
 * (`failure_reason`, disc-2026-10-activity-subtitle-run-time-a AC #1
 * `[@contract-v1]`). The backend classifies the raw error itself
 * (`models.SubtitleRunFailureReason`); its rules are a SUPERSET of TABLE below —
 * it also knows `cancelled` and `target_not_writable`, and checks them in its
 * own order. The WORDS are shared; the matching is not (yet). The live log
 * therefore still says 生成失敗 for a write failure the activity record names.
 */
export type RunFailureReason =
  | 'cancelled'
  | 'budget_exceeded'
  | 'credit_exhausted'
  | 'target_not_writable'
  | 'no_audio_track'
  | 'ffmpeg_missing'
  | 'asr_unconfigured'
  | 'unauthorized'
  | 'timeout'
  | 'other';

/**
 * One sentence per reason — the single copy of the failure words. TABLE (the
 * raw-string matcher for the live log) points at these same frozen objects, so
 * the two surfaces never word the same reason two ways.
 */
export const FAILURE_COPY_BY_REASON: Readonly<Record<RunFailureReason, FailureCopy>> = {
  // ⚖️ Sally 2026-10-19: a user cancel or a container restart mid-run.
  cancelled: Object.freeze({ text: '中途停止', stopped: false }),
  budget_exceeded: Object.freeze({ text: '已達預算上限', stopped: true }),
  // An empty provider account (disc-2026-09-asr-quota-429-treated-as-transient):
  // not a failure — the film is fine, the account needs a top-up (F11-SPEC-LOG ⑤b).
  credit_exhausted: Object.freeze({ text: 'AI 帳號沒有額度', stopped: true }),
  // The first half of the 409 sentence (transcription_handler.go). Never the
  // second half 「這次沒有花到錢」: a run can fail to write AFTER translating
  // (正式機 和平使者 S01E01 spent $0.196).
  target_not_writable: Object.freeze({ text: '影片所在的資料夾寫不進字幕', stopped: false }),
  no_audio_track: Object.freeze({ text: '找不到可用的音軌', stopped: false }),
  ffmpeg_missing: Object.freeze({ text: '伺服器缺少 ffmpeg', stopped: false }),
  asr_unconfigured: Object.freeze({ text: '語音辨識未設定', stopped: false }),
  unauthorized: Object.freeze({ text: 'API 金鑰無效', stopped: false }),
  timeout: Object.freeze({ text: '處理逾時', stopped: false }),
  other: Object.freeze({ text: '生成失敗', stopped: false }),
};

/** Copy for a wire reason code; an unknown or missing code says 生成失敗. */
export function failureCopyForReason(reason: string | null | undefined): FailureCopy {
  return Object.prototype.hasOwnProperty.call(FAILURE_COPY_BY_REASON, reason ?? '')
    ? FAILURE_COPY_BY_REASON[reason as RunFailureReason]
    : FAILURE_COPY_BY_REASON.other;
}

/** First match wins; order matters (a budget stop can mention a timeout). */
const TABLE: ReadonlyArray<{ match: readonly string[]; copy: FailureCopy }> = [
  { match: ['AI_BUDGET_EXCEEDED'], copy: FAILURE_COPY_BY_REASON.budget_exceeded },
  { match: ['no audio track found'], copy: FAILURE_COPY_BY_REASON.no_audio_track },
  { match: ['ffmpeg not available'], copy: FAILURE_COPY_BY_REASON.ffmpeg_missing },
  {
    match: ['transcription unavailable', 'API key not configured'],
    copy: FAILURE_COPY_BY_REASON.asr_unconfigured,
  },
  // The solo runner's sentence says 帳號沒有額度, the legacy path keeps the
  // AI_CREDIT_EXHAUSTED code in its raw error.
  {
    match: ['AI_CREDIT_EXHAUSTED', '帳號沒有額度'],
    copy: FAILURE_COPY_BY_REASON.credit_exhausted,
  },
  // The LLM side says AI_UNAUTHORIZED; the Whisper side says `status 401`.
  { match: ['AI_UNAUTHORIZED', 'status 401'], copy: FAILURE_COPY_BY_REASON.unauthorized },
  {
    match: ['AI_TIMEOUT', 'timed out', 'deadline exceeded'],
    copy: FAILURE_COPY_BY_REASON.timeout,
  },
];

const FALLBACK: FailureCopy = FAILURE_COPY_BY_REASON.other;

export function failureCopy(error: string | null | undefined): FailureCopy {
  if (!error) return FALLBACK;
  for (const row of TABLE) {
    if (row.match.some((m) => error.includes(m))) return row.copy;
  }
  return FALLBACK;
}

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
  v: Omit<FeedRowView, 'trail' | 'trailClass' | 'spin' | 'srState'> & Partial<FeedRowView>
): FeedRowView {
  return { spin: false, trail: null, trailClass: SECONDARY, srState: null, ...v };
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
        const copy = failureCopy(row.error);
        if (copy.stopped) {
          return view({
            glyph: 'circle-alert',
            glyphClass: WARNING,
            stage: '已停止',
            stageClass: WARNING,
            parts: [name, copy.text],
            announce: `已停止：${name}，${copy.text}`,
          });
        }
        return failedView(name, copy.text);
      }
      const text =
        row.reason === 'error' && row.error
          ? failureCopy(row.error).text
          : queueRowLabel({ status: 'failed', reason: row.reason }).text;
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
    parts: [name, reason],
    announce: `失敗：${name}，${reason}`,
  });
}
