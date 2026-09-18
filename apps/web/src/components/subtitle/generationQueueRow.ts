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
 */
import type {
  GenerationBatchItemReason,
  GenerationBatchItemStatus,
} from '../../services/subtitleService';
import type { GenerationPhase } from '../../hooks/useGenerationProgress';
import { GENERATION_STAGES } from './GenerationProgressV2';

/** The two backend-owned fields a row's label is derived from. */
export interface QueueRowView {
  status: GenerationBatchItemStatus;
  reason: GenerationBatchItemReason;
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
 * Failure sentences per backend reason. `skipped` deliberately does NOT open
 * with 已略過: legacy runs with no API key configured land on the same reason
 * (dsr-6d-a CR L8), and "skipped" would read as a decision the system made on
 * purpose rather than something it could not do.
 */
const FAILURE_TEXT: Record<GenerationBatchItemReason, string> = {
  skipped: '沒有可用的字幕來源',
  busy_elsewhere: '這部正在別處處理',
  error: '生成失敗',
  // Pre-dsr-6d-a servers send no reason at all.
  '': '生成失敗',
};

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
      return {
        text: FAILURE_TEXT[view.reason] ?? FAILURE_TEXT[''],
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
      return '失敗';
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
 *    (`englishKeptBlocks`) or keep 簡體 under the CN policy.
 */
export function queueRowSubStatus(view: QueueRowView, labelText: string): string {
  switch (view.status) {
    case 'done':
      return '已完成，字幕已寫入檔案';
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
