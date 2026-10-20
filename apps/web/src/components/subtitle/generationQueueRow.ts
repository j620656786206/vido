// Implements: <utility — no .pen counterpart>
/**
 * Display vocabulary for ONE generation-batch queue row (Story dsr-6d-b AC
 * #2/#3). Pure functions, kept out of the component so the sentence a row
 * shows is unit-testable on its own.
 *
 * Why this file exists: before dsr-6d-a the dialog GUESSED each row's state
 * from counters and an index, so an item the pipeline refused (already busy
 * elsewhere, nothing to generate from) rendered 完成 and vanished from the
 * retry set. The backend now ships `status` + `reason` per item — this module
 * is the single place that turns that pair into words.
 *
 * Stage names are re-exported from the FROZEN `GENERATION_STAGES` list rather
 * than re-typed: renaming them would break the fixture↔baseline mapping, and a
 * private copy would silently drift from the stepper drawn right below the row.
 *
 * A failed item with `reason: "error"` is worded from its `failure_reason` code
 * through runFailureCopy's FAILURE_COPY_BY_REASON — the same sentence the live
 * log and the activity record show for that failure
 * (disc-2026-10-generation-queue-failure-reason). `FAILURE_TEXT.error` is only
 * the fallback when no code came with it.
 *
 * A SKIPPED item (`reason: "skipped"`, or a code whose `skipped` flag is set —
 * `no_text_source`) is not a failure: the film is fine, there was nothing to
 * generate from. It is a grey 已略過 with the reason as its sentence
 * (disc-2026-10-no-text-source-skip-b). `queueRowIsSkipped` is the one place
 * that decides it — the workspace queue, the dialog and the live log all ask it.
 */
import type {
  GenerationBatchItemReason,
  GenerationBatchItemStatus,
} from '../../services/subtitleService';
import type { GenerationPhase } from '../../hooks/useGenerationProgress';
import { GENERATION_STAGES } from './GenerationProgressV2';
import {
  FAILURE_COPY_BY_REASON,
  failureCopyForReason,
  type RunFailureReason,
} from './runFailureCopy';

/** The backend-owned fields a row's label is derived from. */
export interface QueueRowView {
  status: GenerationBatchItemStatus;
  reason: GenerationBatchItemReason;
  /**
   * The run's failure code (`failure_reason`). Only read for `failed` +
   * `reason: "error"` or `"skipped"` (never for `busy_elsewhere`); absent on
   * older servers and on the dialog's fallback rows, which then say 生成失敗.
   */
  failureReason?: string | null;
}

/** Which lucide glyph precedes the label (the component owns the rendering). */
export type QueueRowIcon = 'check' | 'alert' | 'pause' | null;

export interface QueueRowLabel {
  text: string;
  /** Token-based colour class — verbatim, so specs can assert it literally. */
  className: string;
  icon: QueueRowIcon;
}

const MUTED = 'text-[var(--text-muted)]';

/**
 * Failure sentences per backend reason. `skipped` is the fallback reason line
 * of a grey 已略過 row (a skipped item that came with no usable code).
 *
 * ⚖️ History: dsr-6d-b AC #3 / dsr-6d-a CR L8 forbade 已略過 here — legacy runs
 * with no API key landed on `skipped` too, and 略過 read as the system choosing
 * not to. Overturned by the 2026-10-19 party mode (Sally 追認 #2,
 * disc-2026-10-no-text-source-skip-b): the second line now says exactly what is
 * missing (「也沒有設定語音辨識」), and the no-key legacy path sends
 * `asr_unconfigured` (語音辨識未設定) instead.
 *
 * `error` is refined by the run's failure code (FAILURE_COPY_BY_REASON, shared
 * with the live log and the activity record); the `error` entry here is only
 * the fallback for a failure that came with no code.
 */
const FAILURE_TEXT: Record<GenerationBatchItemReason, string> = {
  skipped: '沒有可用的字幕來源',
  busy_elsewhere: '這部正在別處處理',
  error: '生成失敗',
  // Pre-dsr-6d-a servers send no reason at all.
  '': '生成失敗',
};

/**
 * Is this row a grey 已略過 rather than a failure? `reason: "skipped"`, or an
 * `error` whose code is flagged `skipped` (`no_text_source`). `busy_elsewhere`
 * never is — its code, if any, is not read (the film is busy, not missing
 * anything). The single judge for the queue, the dialog and the live log.
 */
export function queueRowIsSkipped(view: QueueRowView): boolean {
  if (view.status !== 'failed') return false;
  if (view.reason === 'skipped') return true;
  return view.reason === 'error' && failureCopyForReason(view.failureReason).skipped;
}

/**
 * A skipped row's reason line: the code's own sentence when it came with a
 * known one (`no_text_source`, `asr_unconfigured`, …), else 沒有可用的字幕來源 —
 * a skipped row never says 生成失敗.
 */
function skippedText(failureReason: string | null | undefined): string {
  const code = failureReason ?? '';
  if (code !== 'other' && Object.prototype.hasOwnProperty.call(FAILURE_COPY_BY_REASON, code)) {
    return FAILURE_COPY_BY_REASON[code as RunFailureReason].text;
  }
  return FAILURE_TEXT.skipped;
}

/** Live phases that name a stage; everything else has no stage to show yet. */
const PHASE_STAGE: Partial<Record<GenerationPhase, string>> = {
  extracting: GENERATION_STAGES[0],
  transcribing: GENERATION_STAGES[1],
  translating: GENERATION_STAGES[2],
};

/**
 * The sentence for one row. `phase` is the per-item stream's current phase and
 * is only consulted while the item is `running` — with no phase yet (just
 * attached, no per-item event seen) the row says 處理中 instead of claiming a
 * stage it cannot know.
 */
export function queueRowLabel(view: QueueRowView, phase?: GenerationPhase | null): QueueRowLabel {
  switch (view.status) {
    case 'done':
      return { text: '完成', className: 'text-[var(--success-text)]', icon: 'check' };
    case 'failed':
      if (queueRowIsSkipped(view)) {
        // Grey, no glyph — the same neutral as 已取消 (Sally 追認 #1).
        return { text: skippedText(view.failureReason), className: MUTED, icon: null };
      }
      return {
        // Always red 失敗 — a code's `stopped` flag is not read here: a real
        // budget / credit stop pauses the item, it does not fail it.
        text:
          view.reason === 'error'
            ? failureCopyForReason(view.failureReason).text
            : (FAILURE_TEXT[view.reason] ?? FAILURE_TEXT['']),
        className: 'text-[var(--error-text)]',
        icon: 'alert',
      };
    case 'running':
      return {
        text: (phase && PHASE_STAGE[phase]) || '處理中',
        className: 'text-[var(--accent-text)] font-semibold',
        icon: null,
      };
    case 'paused':
      return { text: '已暫停 — 下次繼續', className: MUTED, icon: 'pause' };
    case 'cancelled':
      return { text: '已取消', className: MUTED, icon: null };
    default:
      return { text: '排隊中', className: MUTED, icon: null };
  }
}

/**
 * Short word for a status BADGE (the workspace's `aw4Qr` → `OLg1J`). The badge
 * is the label; the sentence is `queueRowSubStatus`. Kept here, not in the
 * component, so the dialog and the workspace can never drift apart on the words
 * (dsr-6d-c-1 建單裁定 #2).
 */
export function queueRowBadge(view: QueueRowView, stage: string): string {
  switch (view.status) {
    case 'done':
      return '完成';
    case 'failed':
      return queueRowIsSkipped(view) ? '已略過' : '失敗';
    case 'running':
      return stage;
    case 'paused':
      return '已暫停';
    case 'cancelled':
      return '已取消';
    default:
      return '排隊中';
  }
}

/**
 * The explanatory line under a row's title (`lUZol`). A FAILED row shows the
 * backend's reason — that is the whole point of the items[] contract.
 *
 * ⚠️ Two sentences deliberately UNDERCLAIM:
 *  - `cancelled` does NOT say 未處理. `finish()` marks the in-flight item
 *    cancelled too, and that item may already have been paid for.
 *  - `done` does NOT say 繁中. A run can deliver a partial translation
 *    (`englishKeptBlocks`) or keep 簡體 under the CN policy. Nor does it
 *    repeat 已完成 — the 完成 badge already says it, and the shorter line
 *    fits whole at 1024 (disc-2026-10-gen-queue-done-row-1024).
 */
export function queueRowSubStatus(view: QueueRowView, labelText: string): string {
  switch (view.status) {
    case 'done':
      return '字幕已寫入檔案';
    case 'failed':
      return labelText;
    case 'running':
      return `${labelText}…`;
    case 'paused':
      return '已暫停 — 下次繼續';
    case 'cancelled':
      return '已取消';
    default:
      return '等待前面項目完成';
  }
}

/**
 * The row's visible name. An episode's `title` is just "S04E07 第七章", so the
 * show it belongs to is prefixed when the backend supplied one (dsr-6d-a AC
 * #7). `seriesTitle` is read defensively — e2e mocks and older visual fixtures
 * may omit the key entirely, and "undefined 第七章" is worse than no prefix.
 */
export function queueRowTitle(item: { title: string; seriesTitle?: string }): string {
  const series = item.seriesTitle?.trim();
  return series ? `${series} ${item.title}` : item.title;
}
