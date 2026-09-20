// Design ref: ux-design.pen Screen F8-D-v2 (i9Nun1)
/**
 * 產生字幕 dialog (Story ux3-subtitle-v2-batch F8 execution shell; idle branch
 * REPLACED by the sub-4-3 consent flow). Screens: F8-D-v2 i9Nun1 / F8-M-v2
 * H717g (mobile bottom sheet, same Radix Dialog) / F9-D-v2 JMqPg 預算上限.
 *
 * - IDLE = the consent flow (GenerationConsentView: F14 analyze → F15 list →
 *   F16/F19 confirm → F20 empty). The old scope-chips + aggregate-count idle
 *   state is history (2026-08-07 三件一體 ruling): nothing starts without an
 *   explicit, priced, consented selection. The batch always starts as
 *   `{scope:'selected', media_ids, budget_usd}` (9R-16 AC #1 [@contract-v3] —
 *   mixed movie/episode UUIDs, WYSIWYG user-approved ceiling).
 * - Queue rows come from the start-202 `items[]`; the active row joins the
 *   per-item streams on `current_media_id` — BOTH families since sub-4-3 AC #8:
 *   `transcription_*` (ASR route) and D6 `subtitle_progress` (extract route;
 *   pipeline mode drives ProcessItem directly). ⚠️ 9R-16 CR caveat: on
 *   cancelled/budget_ceiling the interrupted in-flight item ALSO emits a
 *   terminal per-item event — the batch event's `status`/`paused_count` is
 *   AUTHORITATIVE for row rendering.
 * - budget_ceiling (F9) is a NORMAL terminal state: warning-tint banner,
 *   paused rows 已暫停 — 下次繼續, actions 關閉 + 下次繼續 (= back to the
 *   consent list to re-select/confirm — the paused items are still candidates,
 *   and a resume is a NEW consent, not an un-consented restart).
 * - 409 TRANSCRIPTION_BATCH_RUNNING on open/start → recover-and-attach
 *   (on-open GET .../status probe). A recovered batch has no `items[]` — the
 *   dialog falls back to the active-item card + counts.
 * - Escape is gated while running; closing via ✕ only stops WATCHING — the
 *   batch continues server-side (recover-on-open re-attaches).
 *
 * Rule 23: zero wall-clock reads — progress/cost/counts are all SSE-supplied.
 * Media ids are UUID STRINGS end-to-end (9R-18; movie OR episode row ids).
 */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { Check, CircleAlert, CirclePause, Radio } from 'lucide-react';
import { Dialog, DialogContent, DialogTitle } from '../ui/Dialog';
import { MOBILE_SHEET_CLOSE, MOBILE_SHEET_CONTENT, SheetGrabber } from '../ui/mobileSheet';
import { cn } from '../../lib/utils';
import {
  subtitleService,
  type GenerationBatchItem,
  type GenerationBatchItemState,
} from '../../services/subtitleService';
import {
  useGenerationBatchProgress,
  type GenerationBatchProgressState,
} from '../../hooks/useGenerationBatchProgress';
import {
  useGenerationProgress,
  type GenerationProgressState,
} from '../../hooks/useGenerationProgress';
import { GenerationProgressV2 } from './GenerationProgressV2';
import { GenerationConsentView } from './consent/GenerationConsentView';
import { queueRowLabel, queueRowTitle } from './generationQueueRow';
import { libraryKeys } from '../../hooks/useLibrary';
import { detailKeys } from '../../hooks/useMediaDetails';
import { activityKeys } from '../../hooks/useActivity';
import { transcriptionEstimateKeys } from '../../hooks/useTranscriptionEstimate';
import { usd } from '../../lib/currency';

export const generationBatchPreviewKey = ['subtitles', 'generation-batch', 'preview'] as const;

/**
 * Query key the launcher caches its start-202 `items[]` under, so the ux3-ai-2
 * WORKSPACE can render the full queue for a batch started this session (the status
 * probe carries no items[] — disc-2026-07-generation-batch-status-items). Cleared
 * on terminal; absent → the workspace falls back to attach-degraded.
 */
export const generationBatchItemsKey = ['subtitles', 'generation-batch', 'items'] as const;

/**
 * Query key for the on-open status probe. EXPORTED (dsr-6d-c-1) because the
 * workspace watches the same endpoint: while it was an inline literal here,
 * nothing in the app could invalidate it, so starting a batch from this dialog
 * left a workspace open underneath showing 0 / 0 until its 5-minute cache expired.
 */
export const generationBatchStatusKey = ['subtitles', 'generation-batch', 'status'] as const;

// ---------------------------------------------------------------------------
// Row-state derivation (batch event is AUTHORITATIVE — 9R-16 CR caveat)
// ---------------------------------------------------------------------------

type RowState = 'done' | 'failed' | 'active' | 'queued' | 'paused' | 'stopped';

/**
 * FALLBACK ONLY (dsr-6d-b): used when no `progress.items[]` snapshot is
 * available — a pre-dsr-6d-a server, or the moments before the first snapshot
 * lands. It can only GUESS from counters and an index, which is exactly how a
 * refused item used to render 完成. `dsr-6d-c` re-evaluates removing it.
 */
export function deriveRowStates(
  items: GenerationBatchItem[],
  progress: GenerationBatchProgressState,
  failedIds: ReadonlySet<string>
): RowState[] {
  const { status, currentMediaId, totalItems, pausedCount } = progress;
  const idxOfCurrent = items.findIndex((it) => it.mediaId === currentMediaId);
  return items.map((it, i) => {
    const resolved: RowState = failedIds.has(it.mediaId) ? 'failed' : 'done';
    if (status === 'complete') return resolved;
    if (status === 'budget_ceiling') {
      // paused_count is authoritative: the last `pausedCount` rows (incl. the
      // interrupted in-flight item) are 已暫停, NEVER 失敗 — even if the
      // per-item pipeline reported its own abort as transcription_failed.
      return i >= totalItems - pausedCount ? 'paused' : resolved;
    }
    if (status === 'cancelled' || status === 'error') {
      // The in-flight item (current_media_id) and everything after it never
      // finished — batch status wins over a racing per-item failed event.
      return idxOfCurrent >= 0 && i >= idxOfCurrent ? 'stopped' : resolved;
    }
    // running (or idle seed before the first SSE event)
    if (idxOfCurrent < 0) return i === 0 ? 'active' : 'queued';
    if (i < idxOfCurrent) return resolved;
    if (i === idxOfCurrent) return failedIds.has(it.mediaId) ? 'failed' : 'active';
    return 'queued';
  });
}

/**
 * Ids whose RENDERED terminal state is 失敗 (sub-5-3 AC #3). Derived from
 * deriveRowStates — NOT from failedIds directly — so the retry preselection
 * matches exactly the rows the user sees marked failed (a budget_ceiling
 * interrupted item renders 已暫停 and belongs to remainingIds instead).
 */
export function failedRowIds(
  items: GenerationBatchItem[],
  progress: GenerationBatchProgressState,
  failedIds: ReadonlySet<string>
): string[] {
  const states = deriveRowStates(items, progress, failedIds);
  return items.filter((_, i) => states[i] === 'failed').map((it) => it.mediaId);
}

/**
 * Ids the batch did NOT finish (sub-5-3 AC #4) — 已暫停/停止/失敗 rows. The
 * 下次繼續 preselection: the user's consented picks must survive the ceiling
 * instead of silently falling back to the extract-only default.
 */
export function remainingIds(
  items: GenerationBatchItem[],
  progress: GenerationBatchProgressState,
  failedIds: ReadonlySet<string>
): string[] {
  const states = deriveRowStates(items, progress, failedIds);
  return items
    .filter((_, i) => states[i] === 'paused' || states[i] === 'stopped' || states[i] === 'failed')
    .map((it) => it.mediaId);
}

// ---------------------------------------------------------------------------
// Presentational bits
// ---------------------------------------------------------------------------

/**
 * Fallback vocabulary → the backend's, so ONE renderer draws both paths. The
 * fallback knows no failure reason, so it lands on the generic 生成失敗.
 */
const FALLBACK_STATUS: Record<RowState, GenerationBatchItemState['status']> = {
  done: 'done',
  failed: 'failed',
  active: 'running',
  queued: 'queued',
  paused: 'paused',
  stopped: 'cancelled',
};

function RowStageLabel({
  view,
  phase,
}: {
  view: { status: GenerationBatchItemState['status']; reason: GenerationBatchItemState['reason'] };
  phase?: GenerationProgressState['phase'] | null;
}) {
  const label = queueRowLabel(view, phase);
  return (
    <span className={cn('flex shrink-0 items-center gap-1.5 text-xs', label.className)}>
      {label.icon === 'check' && <Check className="h-4 w-4" aria-hidden="true" />}
      {label.icon === 'alert' && <CircleAlert className="h-4 w-4" aria-hidden="true" />}
      {label.icon === 'pause' && <CirclePause className="h-3.5 w-3.5" aria-hidden="true" />}
      {label.text}
    </span>
  );
}

function QueueRow({
  item,
  activeItemProgress,
}: {
  item: GenerationBatchItemState;
  activeItemProgress?: GenerationProgressState | null;
}) {
  // The stepper is the claim "this is being worked on right now" — only a
  // running row may make it. A failed row used to draw 提取音訊中 underneath
  // its own 失敗 label (dsr-6d-b 🔴 #9).
  const showStepper = item.status === 'running';
  return (
    <li
      data-testid={`gen-batch-row-${item.mediaId}`}
      data-state={item.status}
      className={cn(
        'flex flex-col gap-3 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-4 py-3.5'
      )}
    >
      <div className="flex items-center gap-3.5">
        <span
          aria-hidden="true"
          className="h-[54px] w-[38px] shrink-0 rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)]"
        />
        <span className="min-w-0 flex-1 truncate text-sm text-[var(--text-primary)]">
          {queueRowTitle(item)}
        </span>
        <RowStageLabel
          view={{ status: item.status, reason: item.reason }}
          phase={activeItemProgress?.phase}
        />
      </div>
      {showStepper && (
        <div className="sm:pl-12">
          <GenerationProgressV2
            align="start"
            phase={
              activeItemProgress && activeItemProgress.phase !== 'failed'
                ? activeItemProgress.phase
                : 'idle'
            }
            percentage={activeItemProgress?.percentage}
            message={activeItemProgress?.message}
          />
        </div>
      )}
    </li>
  );
}

// ---------------------------------------------------------------------------
// Presentational panel (prop-driven so every state is fixture-/visual-testable)
// ---------------------------------------------------------------------------

export interface GenerationBatchPanelV2Props {
  open: boolean;
  /**
   * Batch statuses (the wire enum). 'idle' is type-compatible but the
   * container never renders this panel when idle — the sub-4-3 consent flow
   * (GenerationConsentView) owns that phase.
   */
  status: GenerationBatchProgressState['status'];
  progress: GenerationBatchProgressState;
  /** Queue rows from the start-202 `items[]` (empty on 409/recover-attach). */
  items: GenerationBatchItem[];
  /** Media ids (UUID strings) whose per-item pipeline failed while the batch ran. */
  failedIds?: ReadonlySet<string>;
  /** Per-item stage detail for the active row (joined on current_media_id). */
  activeItemProgress?: GenerationProgressState | null;
  /**
   * Resolves when the cancel request succeeded; REJECTS when it failed — the
   * panel keeps the confirm row up and says so (dsr-6d-b 🔴 #5: a swallowed
   * error used to look exactly like a successful cancel).
   */
  onConfirmCancelAll: () => Promise<void>;
  /** 下次繼續 — back to the consent list to re-select/confirm (sub-4-3). */
  onResume: () => void;
  /**
   * 重試失敗項目 (sub-5-3 AC #3) — back to the consent list with the failed
   * rows preselected. undefined = no failed rows known (attach mode included):
   * the button is NOT rendered. Re-consent is structural: the only start path
   * remains the F16 confirm.
   */
  onRetryFailed?: () => void;
  /**
   * 再產生字幕 — the ONLY way out of a terminal panel that has no failed rows
   * to retry. Without it, a `last` snapshot attached on open would pin the
   * dialog to yesterday's result forever (the backend keeps `last` until the
   * next batch starts or the workspace dismisses it).
   */
  onRestart?: () => void;
  onClose: () => void;
}

const EMPTY_FAILED: ReadonlySet<string> = new Set();

export function GenerationBatchPanelV2({
  open,
  status,
  progress,
  items,
  failedIds = EMPTY_FAILED,
  activeItemProgress = null,
  onConfirmCancelAll,
  onResume,
  onRetryFailed,
  onRestart,
  onClose,
}: GenerationBatchPanelV2Props) {
  const [confirmingCancel, setConfirmingCancel] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  const [cancelFailed, setCancelFailed] = useState(false);

  const cancelAllRef = useRef<HTMLButtonElement | null>(null);
  const keepGoingRef = useRef<HTMLButtonElement | null>(null);
  const closeRef = useRef<HTMLButtonElement | null>(null);
  /** Where focus must land once that control exists (🔴 #12 — it fell to <body>). */
  const pendingFocusRef = useRef<'keepGoing' | 'cancelAll' | 'close' | null>(null);

  const isRunning = status === 'running';
  const isBudgetCeiling = status === 'budget_ceiling';
  const isTerminal = status === 'complete' || status === 'cancelled' || status === 'error';

  useEffect(() => {
    if (!isRunning) {
      setConfirmingCancel(false);
      setCancelFailed(false);
      setCancelling(false);
    }
  }, [isRunning]);

  // Runs after EVERY render: the control we want focused (關閉 after a
  // successful cancel) may not exist until the terminal event lands.
  useEffect(() => {
    const target = pendingFocusRef.current;
    if (!target) return;
    const el =
      target === 'keepGoing'
        ? keepGoingRef.current
        : target === 'cancelAll'
          ? cancelAllRef.current
          : closeRef.current;
    if (el) {
      el.focus();
      pendingFocusRef.current = null;
    }
  });

  const processed = progress.successCount + progress.failCount;
  const pct = progress.totalItems > 0 ? (processed / progress.totalItems) * 100 : 0;

  /**
   * Row source, in a fixed priority (dsr-6d-b AC #2):
   *  1. `progress.items` — the backend's own queue, status and reason per item;
   *  2. the 202 `items[]` prop through deriveRowStates — the legacy GUESS;
   *  3. nothing but an in-flight title — one honest degraded card.
   */
  const backendQueue = progress.items && progress.items.length > 0 ? progress.items : null;
  const fallbackStates = backendQueue ? null : deriveRowStates(items, progress, failedIds);
  const rows: GenerationBatchItemState[] = backendQueue
    ? backendQueue
    : items.map((it, i) => ({
        ...it,
        status: FALLBACK_STATUS[fallbackStates![i]],
        reason: '' as const,
      }));

  const handleConfirmCancel = async () => {
    if (cancelling) return;
    setCancelFailed(false);
    setCancelling(true);
    try {
      await onConfirmCancelAll();
      // Accepted — but NOT over: the server still has to stop the in-flight
      // ffmpeg/ASR job, which takes seconds. We deliberately keep the confirm
      // row (and 取消中…) up until the batch reports a terminal status, so the
      // focused button never vanishes into <body> and the panel never pretends
      // the batch already stopped. The !isRunning effect above clears both.
      pendingFocusRef.current = 'close';
    } catch {
      // Keep the confirm row up: the batch is still running, and the alert
      // below is the only thing telling the user that.
      setCancelFailed(true);
      setCancelling(false);
      pendingFocusRef.current = null;
    }
  };

  /**
   * One sentence for the whole batch (🔴 #10). Green is "there is an answer and
   * it is the good one" — a run with failures does not qualify, so it stays
   * neutral. Rendered VISIBLY with aria-live, which is why the sr-only line
   * goes quiet for these statuses (no double announcement).
   */
  const summary: { text: string; className: string } | null =
    status === 'complete'
      ? progress.failCount > 0
        ? {
            text: `完成 ${progress.successCount} 部、失敗 ${progress.failCount} 部`,
            className: 'text-[var(--text-secondary)]',
          }
        : {
            text: `全部完成（${progress.successCount} 部）`,
            className: 'text-[var(--success-text)]',
          }
      : status === 'cancelled'
        ? {
            text: `已取消：完成 ${progress.successCount} 部`,
            className: 'text-[var(--text-secondary)]',
          }
        : null;

  /**
   * ONE announcer: the sr-only region below is mounted for the panel's whole
   * life, so a screen reader registers it up front and speaks the text when it
   * changes. The visible verdict line is deliberately inert (no aria-live) —
   * a live region injected together with its own content is the canonical
   * case AT does NOT announce, and two regions would say it twice.
   */
  const statusAnnouncement = isBudgetCeiling
    ? `已達本次預算上限（${usd(progress.budgetUsd)}）— 已完成${processed}部，剩餘${progress.pausedCount}部下次繼續`
    : status === 'error'
      ? '批次發生錯誤'
      : (summary?.text ?? '');

  /**
   * The bar is not text (downloads precedent): 泥金 means RUNNING, so a
   * finished batch must stop glowing (dsr-4:78). Never --bg-tertiary — that is
   * the track's own colour, i.e. an invisible bar.
   */
  const barColor =
    status === 'running'
      ? 'bg-[var(--accent-primary)]'
      : status === 'complete' && progress.failCount === 0
        ? 'bg-[var(--success)]'
        : status === 'error'
          ? 'bg-[var(--error)]'
          : 'bg-[var(--text-muted)]';

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
    >
      <DialogContent
        data-testid="generation-batch-dialog-v2"
        aria-describedby={undefined}
        closeClassName={cn(MOBILE_SHEET_CLOSE, 'max-sm:top-[22px]')}
        onEscapeKeyDown={(e) => {
          // Escape gated while running (fetch-dialog precedent) — close via ✕/關閉.
          if (isRunning) e.preventDefault();
        }}
        onPointerDownOutside={(e) => {
          if (isRunning) e.preventDefault();
        }}
        onInteractOutside={(e) => {
          if (isRunning) e.preventDefault();
        }}
        className={cn(
          'flex max-h-[85vh] flex-col gap-0 overflow-hidden p-0',
          // Mobile: bottom sheet (F8-M-v2 H717g). Desktop: centered dialog (F8-D-v2 i9Nun1).
          MOBILE_SHEET_CONTENT,
          'sm:bottom-auto sm:left-1/2 sm:right-auto sm:top-1/2 sm:w-[calc(100vw-4rem)] sm:max-w-[880px] sm:-translate-x-1/2 sm:-translate-y-1/2 sm:rounded-[var(--radius-lg)] sm:border sm:border-[var(--border-subtle)]'
        )}
      >
        {/* Mobile bottom-sheet drag handle (F8-M-v2 H717g handle `k46gFw`:
            36×4, fully-rounded, bg-tertiary). Hidden on the desktop dialog
            (sm+), so it changes no desktop baseline. */}
        <SheetGrabber data-testid="gen-batch-drag-handle" />

        {/* Title bar */}
        <div className="flex h-14 shrink-0 items-center justify-between border-b border-[var(--border-subtle)] pl-6 pr-12">
          <DialogTitle className="truncate text-base font-semibold">產生字幕</DialogTitle>
        </div>

        {/* Body */}
        <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-6 py-5">
          {/* Status transitions announced to AT (AC 7). */}
          <p aria-live="polite" className="sr-only" data-testid="gen-batch-status-live">
            {statusAnnouncement}
          </p>

          {/* ---------- Scope line (batch always starts from a consented
              explicit selection since sub-4-3) ---------- */}
          <p className="flex items-center gap-1 text-sm text-[var(--text-secondary)]">
            範圍：已選項目（
            <span className="font-mono tabular-nums">{progress.totalItems}</span> 部）
          </p>

          {/* ---------- F9 budget banner ---------- */}
          {isBudgetCeiling && (
            <div
              data-testid="gen-batch-budget-banner"
              className="flex items-center gap-2.5 rounded-[var(--radius-md)] bg-[var(--warning-tint)] p-3"
            >
              <CircleAlert
                className="h-4 w-4 shrink-0 text-[var(--warning-text)]"
                aria-hidden="true"
              />
              <p className="flex flex-wrap items-center gap-1 text-sm text-[var(--text-primary)]">
                已達本次預算上限（
                <span className="font-mono font-semibold tabular-nums">
                  {usd(progress.budgetUsd)}
                </span>
                ）— 已完成
                <span className="font-mono font-semibold tabular-nums">{processed}</span>
                部，剩餘
                <span className="font-mono font-semibold tabular-nums">{progress.pausedCount}</span>
                部下次繼續
              </p>
            </div>
          )}

          {status === 'error' && (
            <div
              data-testid="gen-batch-error-banner"
              className="flex items-center gap-2.5 rounded-[var(--radius-md)] bg-[var(--error-tint)] p-3"
            >
              <CircleAlert
                className="h-4 w-4 shrink-0 text-[var(--error-text)]"
                aria-hidden="true"
              />
              <p className="text-sm text-[var(--error-text)]">批次發生錯誤，已完成的字幕會保留</p>
            </div>
          )}

          {/* One-line verdict for the whole batch — the VISIBLE aria-live
              source (the sr-only line stays empty for these statuses). */}
          {summary && (
            <p data-testid="gen-batch-summary" className={cn('text-sm', summary.className)}>
              {summary.text}
            </p>
          )}

          {/* ---------- Overall progress (running + terminal) ---------- */}
          {
            <div className="flex flex-col gap-2">
              <div className="flex items-center gap-2.5">
                <span className="text-sm text-[var(--text-secondary)]">已完成</span>
                <span
                  data-testid="gen-batch-counter"
                  className="font-mono text-xl font-semibold tabular-nums text-[var(--text-primary)]"
                >
                  {processed} / {progress.totalItems}
                </span>
              </div>
              <div
                role="progressbar"
                aria-valuemin={0}
                aria-valuemax={progress.totalItems}
                aria-valuenow={processed}
                aria-label="批次生成進度"
                className="h-1.5 overflow-hidden rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)]"
              >
                <div
                  data-testid="gen-batch-progress-bar"
                  className={cn(
                    'h-full rounded-[var(--radius-sm)] transition-all duration-[var(--motion-move)]',
                    barColor
                  )}
                  style={{ width: `${pct}%` }}
                />
              </div>
            </div>
          }

          {/* ---------- Queue rows ---------- */}
          {rows.length > 0 ? (
            <ul className="flex flex-col gap-2" data-testid="gen-batch-item-list">
              {rows.map((item) => (
                <QueueRow
                  key={item.mediaId}
                  item={item}
                  // Per-item stage detail belongs to the in-flight row ONLY.
                  activeItemProgress={
                    item.mediaId === progress.currentMediaId ? activeItemProgress : null
                  }
                />
              ))}
            </ul>
          ) : (
            // Nothing enumerable at all (a pre-dsr-6d-a server on the 409 path):
            // one honest card for the item we know is in flight.
            progress.currentItem && (
              <ul className="flex flex-col gap-2" data-testid="gen-batch-item-list">
                <QueueRow
                  item={{
                    mediaId: progress.currentMediaId ?? '',
                    title: progress.currentItem,
                    // Attach-degraded card: the status probe carries no
                    // media_type/series — cosmetic placeholders only.
                    mediaType: 'movie',
                    seriesTitle: '',
                    // Terminal semantics must hold here too (AC 2): the batch
                    // status is authoritative — budget_ceiling pauses the
                    // in-flight item (已暫停, never 已取消/失敗).
                    status: isRunning
                      ? 'running'
                      : isBudgetCeiling
                        ? 'paused'
                        : status === 'complete'
                          ? progress.currentMediaId != null &&
                            failedIds.has(progress.currentMediaId)
                            ? 'failed'
                            : 'done'
                          : 'cancelled',
                    reason: '',
                  }}
                  activeItemProgress={activeItemProgress}
                />
              </ul>
            )
          )}

          {/* ---------- Cost row ---------- */}
          {
            <div className="flex items-center gap-2">
              <p
                data-testid="gen-batch-cost-line"
                className="flex items-center gap-1 text-sm text-[var(--text-secondary)]"
              >
                本次用量：
                <span className="font-mono font-semibold tabular-nums text-[var(--text-primary)]">
                  {usd(progress.spentUsd)}
                </span>
                <span> / 上限 </span>
                <span className="font-mono tabular-nums">{usd(progress.budgetUsd)}</span>
              </p>
              <span className="flex-1" />
              {isRunning && (
                <span
                  data-testid="gen-batch-sse-chip"
                  className="flex items-center gap-1.5 rounded-[var(--radius-sm)] bg-[var(--info-tint)] px-2 py-1 text-[11px] text-[var(--info-text)]"
                >
                  <Radio className="h-3 w-3" aria-hidden="true" />
                  即時更新（SSE）
                </span>
              )}
            </div>
          }
        </div>

        {/* Footer */}
        <div className="flex shrink-0 items-center justify-end gap-3 border-t border-[var(--border-subtle)] px-6 py-3.5">
          {isRunning &&
            (!confirmingCancel ? (
              <button
                type="button"
                ref={cancelAllRef}
                onClick={() => {
                  setConfirmingCancel(true);
                  pendingFocusRef.current = 'keepGoing';
                }}
                data-testid="gen-batch-cancel-all"
                className="flex min-h-[44px] items-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-5 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)]"
              >
                全部取消
              </button>
            ) : (
              <div
                data-testid="gen-batch-cancel-confirm"
                className="flex flex-wrap items-center gap-3"
              >
                <span className="text-sm text-[var(--text-secondary)]">
                  確定要取消整個批次嗎？已完成的字幕會保留。
                </span>
                {cancelFailed && (
                  <span
                    role="alert"
                    data-testid="gen-batch-cancel-error"
                    className="text-sm text-[var(--error-text)]"
                  >
                    取消失敗，批次仍在進行。請再試一次。
                  </span>
                )}
                <button
                  type="button"
                  ref={keepGoingRef}
                  onClick={() => {
                    setConfirmingCancel(false);
                    // Drop the previous attempt's alert — otherwise reopening
                    // the confirm row re-announces a failure nobody retried.
                    setCancelFailed(false);
                    pendingFocusRef.current = 'cancelAll';
                  }}
                  className="flex min-h-[44px] items-center rounded-[var(--radius-md)] px-4 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]"
                >
                  繼續生成
                </button>
                {/* Neutral Secondary, not 硃砂: cancelling a batch keeps every
                    subtitle already produced — nothing here is unrecoverable
                    (DESIGN.md:300 — a state colour makes a claim about state). */}
                <button
                  type="button"
                  // aria-disabled, not disabled: a disabled button drops focus
                  // to <body> mid-request and breaks Escape handling (dsr-6c).
                  aria-disabled={cancelling || undefined}
                  onClick={() => void handleConfirmCancel()}
                  data-testid="gen-batch-cancel-confirm-btn"
                  className={cn(
                    'flex min-h-[44px] items-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-4 text-sm text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)]',
                    cancelling && 'opacity-60'
                  )}
                >
                  {cancelling ? '取消中…' : '確定取消'}
                </button>
              </div>
            ))}

          {(isTerminal || isBudgetCeiling) && (
            <button
              type="button"
              ref={closeRef}
              onClick={onClose}
              data-testid="gen-batch-close-btn"
              className="flex min-h-[44px] items-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-5 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)]"
            >
              關閉
            </button>
          )}

          {/* CR M1: NOT at budget_ceiling. There 下次繼續 already carries the
              failed rows PLUS the paused ones, so a narrower 重試失敗項目 next
              to it would silently drop selections the user already consented
              to — the very loss AC #4 exists to stop. */}
          {isTerminal && onRetryFailed && (
            <button
              type="button"
              onClick={onRetryFailed}
              data-testid="gen-batch-retry-failed-btn"
              // Neutral, not 硃砂: retrying is RECOVERY, the opposite of the
              // destructive act a red button claims (DESIGN.md:296/300).
              className="flex min-h-[44px] items-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-5 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)]"
            >
              重試失敗項目
            </button>
          )}

          {/* The way out of an attached `last` result with nothing to retry —
              without it the dialog can never reach the consent flow again. */}
          {isTerminal && !onRetryFailed && onRestart && (
            <button
              type="button"
              onClick={onRestart}
              data-testid="gen-batch-restart-btn"
              className="flex min-h-[44px] items-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-5 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)]"
            >
              再產生字幕
            </button>
          )}

          {isBudgetCeiling && (
            <button
              type="button"
              onClick={onResume}
              data-testid="gen-batch-resume-btn"
              className="flex min-h-[44px] items-center rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-5 text-sm font-semibold text-[var(--text-on-accent)] transition-colors hover:bg-[var(--accent-pressed)]"
            >
              下次繼續
            </button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Container — consent flow (idle) ⇄ execution panel (running+), lazy-SSE hooks
// and the dual-family per-item join
// ---------------------------------------------------------------------------

export interface GenerationBatchDialogV2Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /**
   * Media ids (movie AND episode UUIDs — mixed since sub-4-2 D1) pre-checked
   * in the consent list when opened from a library selection. Intersected with
   * the candidate list once analysis is ready; empty/absent → the default
   * (extract-only) selection applies.
   */
  selectedMediaIds?: string[];
  /**
   * CR sub-4-3 H2: force a fresh candidate analysis on open (F17 post-scan
   * deep link — the library just changed, a ready snapshot is stale).
   */
  forceAnalyze?: boolean;
}

export function GenerationBatchDialogV2({
  open,
  onOpenChange,
  selectedMediaIds,
  forceAnalyze = false,
}: GenerationBatchDialogV2Props) {
  const queryClient = useQueryClient();

  const [items, setItems] = useState<GenerationBatchItem[]>([]);
  const [failedIds, setFailedIds] = useState<Set<string>>(new Set());
  /**
   * Consent preselection carried across a terminal→consent transition
   * (sub-5-3 AC #3/#4): the failed rows (重試失敗項目) or the unfinished rows
   * (下次繼續). State — not a derived value — because the preselectedIds prop
   * demands a render-stable array, and because it must survive the resets
   * that clear items/failedIds on the way back to the consent phase.
   */
  const [retryIds, setRetryIds] = useState<string[] | undefined>(undefined);
  const [starting, setStarting] = useState(false);
  const [startError, setStartError] = useState<string | null>(null);
  // CR M4: the consent flow must not bootstrap (and possibly kick a full
  // library probe sweep) before we know whether a batch is already running.
  const [probed, setProbed] = useState(false);
  // CR H2: after a batch terminal the candidate snapshot is stale (completed
  // items still listed, quotes wrong) — the next consent render re-analyzes.
  const [postTerminal, setPostTerminal] = useState(false);

  // Re-run the status probe on demand (409 with an empty body).
  const [reprobeTick, setReprobeTick] = useState(0);
  /**
   * The `last` terminal snapshot we have already shown. The backend keeps
   * `last` until the next batch starts or the workspace dismisses it, so
   * attaching it unconditionally would pin 產生字幕 to a finished batch for
   * ever — the user could never start a new one.
   */
  const seenLastBatchIdRef = useRef<string | null>(null);
  /** Terminal batches whose cache invalidation already ran (once per batch). */
  const handledTerminalRef = useRef<string | null>(null);

  const batch = useGenerationBatchProgress();
  const perItem = useGenerationProgress();
  const {
    startTracking: startBatchTracking,
    attachSnapshot,
    reset: resetBatch,
    connectionEpoch,
  } = batch;
  const { startTracking: startItemTracking, reset: resetItem } = perItem;

  const isIdle = batch.status === 'idle';

  // On open, recover an already-running batch (409-recover precedent): the
  // status probe lets us jump straight into the running view and attach SSE —
  // skipping the consent flow, because THAT batch was already consented.
  useEffect(() => {
    if (!open) {
      setProbed(false);
      return;
    }
    let cancelled = false;
    subtitleService
      .getGenerationBatchStatus()
      .then((s) => {
        if (cancelled) return;
        if (s.running && s.progress) {
          setStartError(null);
          startBatchTracking(s.progress);
          return;
        }
        // Not running: the most recent terminal snapshot still explains how the
        // batch ended (budget ceiling, failures) even if its SSE event was lost
        // or the dialog was closed. Shown ONCE per batch — see the ref's note.
        const last = s.last;
        if (last?.batchId && seenLastBatchIdRef.current !== last.batchId) {
          setStartError(null);
          seenLastBatchIdRef.current = last.batchId;
          attachSnapshot(last);
        }
      })
      .catch(() => {
        // Best-effort recovery — a failed probe just leaves the consent flow up.
      })
      .finally(() => {
        if (!cancelled) setProbed(true);
      });
    return () => {
      cancelled = true;
    };
    // connectionEpoch: an SSE gap may have swallowed the terminal event, so a
    // backoff reconnect re-reads status — the only cure for a view stuck on
    // 進行中 (9R-16 CR L1).
  }, [open, reprobeTick, connectionEpoch, startBatchTracking, attachSnapshot]);

  // Per-item join: track the in-flight item's per-item streams (BOTH families
  // since sub-4-3 AC #8) on current_media_id; reconnects per item.
  const currentMediaId = batch.progress.currentMediaId;
  useEffect(() => {
    if (batch.status === 'running' && currentMediaId != null) {
      startItemTracking(currentMediaId);
    }
  }, [batch.status, currentMediaId, startItemTracking]);

  // Record per-item failures ONLY while the batch is still running — a
  // terminal per-item event that coincides with a non-error terminal batch
  // status is the interrupted in-flight item (9R-16 CR caveat), and the
  // paused/cancelled row branches override it at render time regardless.
  const perItemPhase = perItem.progress.phase;
  const hasBackendQueue = (batch.progress.items?.length ?? 0) > 0;
  useEffect(() => {
    // With a backend queue the per-item stream is NOT a source of truth for
    // which row failed — items[].status is (dsr-6d-b 🔴 #1/#2).
    if (hasBackendQueue) return;
    if (perItemPhase === 'failed' && batch.status === 'running' && currentMediaId != null) {
      setFailedIds((prev) => {
        if (prev.has(currentMediaId)) return prev;
        const next = new Set(prev);
        next.add(currentMediaId);
        return next;
      });
    }
  }, [hasBackendQueue, perItemPhase, batch.status, currentMediaId]);

  // Terminal: stop watching the per-item stream; completed items wrote back
  // subtitle_status → refresh library badges/counts, the F17 toast count and
  // the (now stale) candidate snapshot for the next consent round.
  const batchStatus = batch.status;
  const batchId = batch.progress.batchId;
  useEffect(() => {
    if (batchStatus === 'idle' || batchStatus === 'running') return;
    resetItem();
    // CR H2: the candidate snapshot is stale after ANY terminal — including a
    // re-attached `last` — so this must NOT sit behind the once-per-batch
    // guard (handleClose resets it, and the next open would otherwise re-quote
    // items the batch already finished).
    setPostTerminal(true);
    // Watching a batch finish COUNTS as having seen its result: without this,
    // reopening the dialog would re-attach the same `last` snapshot and its
    // 重試失敗項目 would hijack a brand-new library selection (preselectedIds
    // prefers retryIds).
    seenLastBatchIdRef.current = batchId;
    // ONCE per batch: with `last` attached on open, the refresh below would
    // otherwise sweep library/detail/activity every time the dialog opens.
    if (handledTerminalRef.current === batchId) return;
    handledTerminalRef.current = batchId;
    void queryClient.invalidateQueries({ queryKey: libraryKeys.all });
    void queryClient.invalidateQueries({ queryKey: generationBatchPreviewKey });
    // Completed items wrote subtitle_status back: the open detail page still
    // says 缺字幕, the activity row is stale, and the per-title quote changed.
    void queryClient.invalidateQueries({ queryKey: detailKeys.all });
    void queryClient.invalidateQueries({ queryKey: activityKeys.all });
    void queryClient.invalidateQueries({ queryKey: transcriptionEstimateKeys.all });
    // The header promise: the cached queue is cleared ON TERMINAL (not on
    // close — a closed dialog does not stop the batch, and the workspace is
    // still drawing that queue).
    queryClient.removeQueries({ queryKey: generationBatchItemsKey });
    // …and the workspace must learn it ended (it reads `last` from this probe).
    void queryClient.invalidateQueries({ queryKey: generationBatchStatusKey });
  }, [batchStatus, batchId, resetItem, queryClient]);

  /**
   * The consent flow's confirmed start (sub-4-3 AC #4): explicit ids in list
   * order + the user-approved on-screen ceiling — ALWAYS scope=selected,
   * ALWAYS budget_usd (WYSIWYG consent; 9R-16 AC #1 [@contract-v3]).
   *
   * sub-6-8b extends the same rule to the model: an explicit `modelId` is the
   * one whose price the user just read, so it travels with the batch. An empty
   * string is omitted rather than sent — the server then uses its own default,
   * which is exactly what the priced rows fell back to.
   */
  const handleStartConsented = useCallback(
    async (mediaIds: string[], budgetUsd: number, modelId: string) => {
      setStarting(true);
      setStartError(null);
      try {
        const outcome = await subtitleService.startGenerationBatch({
          scope: 'selected',
          mediaIds,
          budgetUsd,
          ...(modelId ? { modelId } : {}),
        });
        setRetryIds(undefined); // consumed — the next consent render starts clean
        if (outcome.conflict) {
          // A batch was already running (409) — attach to it. We did NOT
          // enumerate its items[], so clear any stale cache from a prior batch
          // this session — the workspace then shows attach-degraded (honest).
          setItems([]);
          setFailedIds(new Set());
          queryClient.removeQueries({ queryKey: generationBatchItemsKey });
          if (outcome.progress) {
            startBatchTracking(outcome.progress);
          } else {
            // The batch ended between the 409 and the read (the server sends
            // its live snapshot, which may already be gone). Seeding an empty
            // snapshot would pin the panel at 0 / 0 「進行中」 for ever — ask
            // status instead, which answers running / last / nothing. The
            // message stands only if that re-read finds nothing either (the
            // probe clears it as soon as it attaches something).
            setStartError('剛才那個批次已經結束了，請再按一次開始');
            setReprobeTick((n) => n + 1);
          }
        } else if (outcome.result.batchId == null) {
          // Empty scope 200: nothing to do is not an error, but it is also not
          // a batch — stay in the consent flow and say so.
          setStartError('沒有可以生成的項目');
        } else {
          setItems(outcome.result.items);
          setFailedIds(new Set());
          // Cache items[] so the ux3-ai-2 workspace can render the full queue
          // for this session's batch (the status probe carries none).
          queryClient.setQueryData(generationBatchItemsKey, outcome.result.items);
          // A workspace open underneath must learn a batch just started.
          void queryClient.invalidateQueries({ queryKey: generationBatchStatusKey });
          // dsr-6d-a AC #5: the 202 carries the started batch's own snapshot
          // (real ceiling + the queue), so the first paint needs no SSE event
          // — which may already have been broadcast and missed.
          const started = outcome.result.progress;
          if (started && started.status !== 'running') {
            // A batch short enough to finish before we read the response (one
            // title the pipeline refuses outright — the very case this story
            // exists for). startTracking would force `status: 'running'` and
            // open a stream for a batch that is already over, leaving the panel
            // glowing 進行中 with a 全部取消 that can never do anything.
            attachSnapshot(started);
          } else {
            // Without a snapshot we fall back to the 202 items[] + deriveRowStates.
            startBatchTracking(
              started ?? {
                batchId: outcome.result.batchId,
                totalItems: outcome.result.totalItems,
              }
            );
          }
        }
      } catch (err) {
        setStartError(err instanceof Error ? err.message : '批次生成啟動失敗');
      } finally {
        setStarting(false);
      }
    },
    [queryClient, startBatchTracking]
  );

  // 下次繼續 (budget_ceiling) — back to the consent list: the paused items are
  // still candidates, and a resume is a NEW consent (re-select, re-price,
  // re-confirm), never an un-consented auto-restart. sub-5-3 AC #4: the
  // unfinished rows are PRESELECTED — the user's consented picks must not
  // silently fall back to the extract-only default.
  const backendQueue = batch.progress.items;
  const handleResume = useCallback(() => {
    const remaining =
      backendQueue && backendQueue.length > 0
        ? backendQueue
            .filter(
              (it) => it.status === 'failed' || it.status === 'paused' || it.status === 'cancelled'
            )
            .map((it) => it.mediaId)
        : remainingIds(items, batch.progress, failedIds);
    setRetryIds(remaining.length > 0 ? remaining : undefined);
    resetBatch();
    setItems([]);
    setFailedIds(new Set());
  }, [backendQueue, items, batch.progress, failedIds, resetBatch]);

  // 重試失敗項目 (sub-5-3 AC #3) — same mechanism, failed rows only. Consent
  // is structural: this NEVER calls startGenerationBatch — the only start
  // path stays the F16 confirm inside GenerationConsentView.
  // CR L1: memoized so handleRetryFailed's own useCallback is not defeated by
  // a fresh array every render (the bugfix-19-4b-1 unstable-callback-prop
  // class). ⚠️ `batch.progress` is a NEW object on every SSE tick, so this
  // still recomputes per tick — cheap for the items-first branch (one filter),
  // and the deriveRowStates branch only runs on servers with no items[].
  const failedRows = useMemo(
    () =>
      backendQueue && backendQueue.length > 0
        ? backendQueue.filter((it) => it.status === 'failed').map((it) => it.mediaId)
        : failedRowIds(items, batch.progress, failedIds),
    [backendQueue, items, batch.progress, failedIds]
  );
  const handleRetryFailed = useCallback(() => {
    setRetryIds(failedRows);
    resetBatch();
    setItems([]);
    setFailedIds(new Set());
  }, [failedRows, resetBatch]);

  /**
   * The terminal `cancelled` SSE event flips the status — no manual change.
   * A failure is RETHROWN on purpose: the panel needs it to keep the confirm
   * row up and say the batch is still running (dsr-6d-b 🔴 #5).
   */
  const handleConfirmCancelAll = useCallback(async () => {
    const result = await subtitleService.cancelGenerationBatch();
    if (!result.cancelled && !result.running) {
      // Nothing to cancel — the batch had already finished and we missed the
      // terminal event. Re-read status so the panel shows how it actually
      // ended instead of sitting in 取消中… for ever.
      setReprobeTick((n) => n + 1);
    }
  }, []);

  /**
   * 再產生字幕 — leave an attached terminal result and go back to the consent
   * flow. Like 下次繼續/重試失敗項目 it starts NOTHING; only the F16 confirm can.
   */
  const handleRestart = useCallback(() => {
    setRetryIds(undefined);
    resetBatch();
    setItems([]);
    setFailedIds(new Set());
  }, [resetBatch]);

  const handleClose = useCallback(() => {
    // Closing only stops WATCHING — a running batch continues server-side.
    resetBatch();
    resetItem();
    setItems([]);
    setFailedIds(new Set());
    setRetryIds(undefined);
    setStarting(false);
    setStartError(null);
    setPostTerminal(false);
    onOpenChange(false);
  }, [resetBatch, resetItem, onOpenChange]);

  if (!open) return null;

  if (isIdle) {
    // CR M4: hold the consent flow until the recovery probe settles — its
    // bootstrap could otherwise kick a full-library probe sweep (with no
    // visible F14/取消) while a batch is mid-execution.
    if (!probed) return null;
    return (
      <GenerationConsentView
        open={open}
        preselectedIds={retryIds ?? selectedMediaIds}
        forceAnalyze={forceAnalyze || postTerminal}
        starting={starting}
        startError={startError}
        onStartBatch={(mediaIds, budgetUsd, modelId) =>
          void handleStartConsented(mediaIds, budgetUsd, modelId)
        }
        onClose={handleClose}
      />
    );
  }

  return (
    <GenerationBatchPanelV2
      open={open}
      status={batch.status}
      progress={batch.progress}
      items={items}
      failedIds={failedIds}
      activeItemProgress={perItem.progress}
      onConfirmCancelAll={handleConfirmCancelAll}
      onResume={handleResume}
      onRetryFailed={failedRows.length > 0 ? handleRetryFailed : undefined}
      onRestart={handleRestart}
      onClose={handleClose}
    />
  );
}
