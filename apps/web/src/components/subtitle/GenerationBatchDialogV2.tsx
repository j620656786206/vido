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
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { Check, CircleAlert, CirclePause, Radio } from 'lucide-react';
import { Dialog, DialogContent, DialogTitle } from '../ui/Dialog';
import { cn } from '../../lib/utils';
import { subtitleService, type GenerationBatchItem } from '../../services/subtitleService';
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
import { libraryKeys } from '../../hooks/useLibrary';
import { usd } from '../../lib/currency';

export const generationBatchPreviewKey = ['subtitles', 'generation-batch', 'preview'] as const;

/**
 * Query key the launcher caches its start-202 `items[]` under, so the ux3-ai-2
 * WORKSPACE can render the full queue for a batch started this session (the status
 * probe carries no items[] — disc-2026-07-generation-batch-status-items). Cleared
 * on terminal; absent → the workspace falls back to attach-degraded.
 */
export const generationBatchItemsKey = ['subtitles', 'generation-batch', 'items'] as const;

// ---------------------------------------------------------------------------
// Row-state derivation (batch event is AUTHORITATIVE — 9R-16 CR caveat)
// ---------------------------------------------------------------------------

type RowState = 'done' | 'failed' | 'active' | 'queued' | 'paused' | 'stopped';

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

function RowStageLabel({ state }: { state: RowState }) {
  switch (state) {
    case 'done':
      return (
        <span className="flex shrink-0 items-center gap-1.5 text-xs text-[var(--success)]">
          <Check className="h-4 w-4" aria-hidden="true" />
          完成
        </span>
      );
    case 'failed':
      return (
        <span className="flex shrink-0 items-center gap-1.5 text-xs text-[var(--error)]">
          <CircleAlert className="h-4 w-4" aria-hidden="true" />
          失敗
        </span>
      );
    case 'active':
      return (
        <span className="shrink-0 text-xs font-semibold text-[var(--accent-text)]">轉錄中</span>
      );
    case 'paused':
      return (
        <span className="flex shrink-0 items-center gap-1.5 text-xs text-[var(--text-muted)]">
          <CirclePause className="h-3.5 w-3.5" aria-hidden="true" />
          已暫停 — 下次繼續
        </span>
      );
    case 'stopped':
      return <span className="shrink-0 text-xs text-[var(--text-muted)]">已取消</span>;
    default:
      return <span className="shrink-0 text-xs text-[var(--text-muted)]">排隊中</span>;
  }
}

function QueueRow({
  item,
  state,
  activeItemProgress,
}: {
  item: GenerationBatchItem;
  state: RowState;
  activeItemProgress?: GenerationProgressState | null;
}) {
  const showStepper = state === 'active';
  return (
    <li
      data-testid={`gen-batch-row-${item.mediaId}`}
      data-state={state}
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
          {item.title}
        </span>
        <RowStageLabel state={state} />
      </div>
      {showStepper && (
        <div className="sm:pl-[52px]">
          <GenerationProgressV2
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
  onConfirmCancelAll: () => void;
  /** 下次繼續 — back to the consent list to re-select/confirm (sub-4-3). */
  onResume: () => void;
  /**
   * 重試失敗項目 (sub-5-3 AC #3) — back to the consent list with the failed
   * rows preselected. undefined = no failed rows known (attach mode included):
   * the button is NOT rendered. Re-consent is structural: the only start path
   * remains the F16 confirm.
   */
  onRetryFailed?: () => void;
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
  onClose,
}: GenerationBatchPanelV2Props) {
  const [confirmingCancel, setConfirmingCancel] = useState(false);

  const isRunning = status === 'running';
  const isBudgetCeiling = status === 'budget_ceiling';
  const isTerminal = status === 'complete' || status === 'cancelled' || status === 'error';

  useEffect(() => {
    if (!isRunning) setConfirmingCancel(false);
  }, [isRunning]);

  const processed = progress.successCount + progress.failCount;
  const pct = progress.totalItems > 0 ? (processed / progress.totalItems) * 100 : 0;
  const rowStates = deriveRowStates(items, progress, failedIds);

  const statusAnnouncement = isBudgetCeiling
    ? `已達本次預算上限（${usd(progress.budgetUsd)}）— 已完成${processed}部，剩餘${progress.pausedCount}部下次繼續`
    : status === 'complete'
      ? '批次生成完成'
      : status === 'cancelled'
        ? '批次已取消'
        : status === 'error'
          ? '批次發生錯誤'
          : '';

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
          'bottom-0 left-0 right-0 top-auto w-full max-w-none translate-x-0 translate-y-0 rounded-b-none rounded-t-[var(--radius-xl)]',
          'sm:bottom-auto sm:left-1/2 sm:right-auto sm:top-1/2 sm:w-[calc(100vw-4rem)] sm:max-w-3xl sm:-translate-x-1/2 sm:-translate-y-1/2 sm:rounded-[var(--radius-lg)]'
        )}
      >
        {/* Mobile bottom-sheet drag handle (F8-M-v2 H717g handle `k46gFw`:
            36×4, fully-rounded, bg-tertiary). Hidden on the desktop dialog
            (sm+), so it changes no desktop baseline. */}
        <div
          data-testid="gen-batch-drag-handle"
          className="flex shrink-0 justify-center pb-1 pt-2 sm:hidden"
        >
          <span aria-hidden="true" className="h-1 w-9 rounded-full bg-[var(--bg-tertiary)]" />
        </div>

        {/* Title bar */}
        <div className="flex h-14 shrink-0 items-center justify-between border-b border-[var(--border-subtle)] pl-6 pr-12">
          <DialogTitle className="truncate text-base font-semibold">產生字幕</DialogTitle>
        </div>

        {/* Body */}
        <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-6">
          {/* Status transitions announced to AT (AC 7). */}
          <p aria-live="polite" className="sr-only" data-testid="gen-batch-status-live">
            {statusAnnouncement}
          </p>

          {/* ---------- Scope line (batch always starts from a consented
              explicit selection since sub-4-3) ---------- */}
          <p className="flex items-center gap-[3px] text-[13px] text-[var(--text-secondary)]">
            範圍：已選項目（
            <span className="font-mono tabular-nums">{progress.totalItems}</span> 部）
          </p>

          {/* ---------- F9 budget banner ---------- */}
          {isBudgetCeiling && (
            <div
              data-testid="gen-batch-budget-banner"
              className="flex items-center gap-2.5 rounded-[var(--radius-md)] bg-[var(--warning-tint)] p-3"
            >
              <CircleAlert className="h-4 w-4 shrink-0 text-[var(--warning)]" aria-hidden="true" />
              <p className="flex flex-wrap items-center gap-[3px] text-[13px] text-[var(--text-primary)]">
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
              <CircleAlert className="h-4 w-4 shrink-0 text-[var(--error)]" aria-hidden="true" />
              <p className="text-[13px] text-[var(--error-text)]">
                批次發生錯誤，已完成的字幕會保留
              </p>
            </div>
          )}

          {/* ---------- Overall progress (running + terminal) ---------- */}
          {
            <div className="flex flex-col gap-2">
              <div className="flex items-baseline gap-2.5">
                <span className="text-[13px] text-[var(--text-secondary)]">已完成</span>
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
                  className="h-full rounded-[var(--radius-sm)] bg-[var(--accent-primary)] transition-all duration-300"
                  style={{ width: `${pct}%` }}
                />
              </div>
            </div>
          }

          {/* ---------- Queue rows ---------- */}
          {items.length > 0 ? (
            <ul className="flex flex-col gap-2" data-testid="gen-batch-item-list">
              {items.map((item, i) => (
                <QueueRow
                  key={item.mediaId}
                  item={item}
                  state={rowStates[i]}
                  activeItemProgress={activeItemProgress}
                />
              ))}
            </ul>
          ) : (
            // 409/recover-attach fallback: the status probe has no items[] —
            // render the in-flight item card from the progress snapshot.
            progress.currentItem && (
              <ul className="flex flex-col gap-2" data-testid="gen-batch-item-list">
                <QueueRow
                  item={{
                    mediaId: progress.currentMediaId ?? '',
                    title: progress.currentItem,
                    // Attach-degraded card: the status probe carries no
                    // media_type — cosmetic placeholder only.
                    mediaType: 'movie',
                  }}
                  // Terminal semantics must hold here too (AC 2): the batch
                  // status is authoritative — budget_ceiling pauses the
                  // in-flight item (已暫停, never 已取消/失敗), complete
                  // resolves it via the failure record.
                  state={
                    isRunning
                      ? 'active'
                      : isBudgetCeiling
                        ? 'paused'
                        : status === 'complete'
                          ? progress.currentMediaId != null &&
                            failedIds.has(progress.currentMediaId)
                            ? 'failed'
                            : 'done'
                          : 'stopped'
                  }
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
                className="flex items-center gap-[3px] text-[13px] text-[var(--text-secondary)]"
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
                  className="flex items-center gap-1.5 rounded-[var(--radius-sm)] bg-[var(--info-tint)] px-2 py-1 text-[11px] text-[var(--info)]"
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
                onClick={() => setConfirmingCancel(true)}
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
                <span className="text-[13px] text-[var(--text-secondary)]">
                  確定要取消整個批次嗎？已完成的字幕會保留。
                </span>
                <button
                  type="button"
                  onClick={() => setConfirmingCancel(false)}
                  className="flex min-h-[44px] items-center rounded-[var(--radius-md)] px-4 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]"
                >
                  繼續生成
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setConfirmingCancel(false);
                    onConfirmCancelAll();
                  }}
                  data-testid="gen-batch-cancel-confirm-btn"
                  className="flex min-h-[44px] items-center rounded-[var(--radius-md)] bg-[var(--error-tint)] px-4 text-sm text-[var(--error)] transition-colors hover:opacity-80"
                >
                  確定取消
                </button>
              </div>
            ))}

          {(isTerminal || isBudgetCeiling) && (
            <button
              type="button"
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
              className="flex min-h-[44px] items-center rounded-[var(--radius-md)] bg-[var(--error-tint)] px-5 text-sm font-medium text-[var(--error)] transition-colors hover:opacity-80"
            >
              重試失敗項目
            </button>
          )}

          {isBudgetCeiling && (
            <button
              type="button"
              onClick={onResume}
              data-testid="gen-batch-resume-btn"
              className="flex min-h-[44px] items-center rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-6 text-sm font-medium text-[var(--text-on-accent)] transition-colors hover:bg-[var(--accent-pressed)]"
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

  const batch = useGenerationBatchProgress();
  const perItem = useGenerationProgress();
  const { startTracking: startBatchTracking, reset: resetBatch } = batch;
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
        if (s.running && s.progress) startBatchTracking(s.progress);
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
  }, [open, startBatchTracking]);

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
  useEffect(() => {
    if (perItemPhase === 'failed' && batch.status === 'running' && currentMediaId != null) {
      setFailedIds((prev) => {
        if (prev.has(currentMediaId)) return prev;
        const next = new Set(prev);
        next.add(currentMediaId);
        return next;
      });
    }
  }, [perItemPhase, batch.status, currentMediaId]);

  // Terminal: stop watching the per-item stream; completed items wrote back
  // subtitle_status → refresh library badges/counts, the F17 toast count and
  // the (now stale) candidate snapshot for the next consent round.
  const batchStatus = batch.status;
  useEffect(() => {
    if (batchStatus === 'idle' || batchStatus === 'running') return;
    resetItem();
    setPostTerminal(true); // CR H2: next consent render must re-analyze
    void queryClient.invalidateQueries({ queryKey: libraryKeys.all });
    void queryClient.invalidateQueries({ queryKey: generationBatchPreviewKey });
  }, [batchStatus, resetItem, queryClient]);

  /**
   * The consent flow's confirmed start (sub-4-3 AC #4): explicit ids in list
   * order + the user-approved on-screen ceiling — ALWAYS scope=selected,
   * ALWAYS budget_usd (WYSIWYG consent; 9R-16 AC #1 [@contract-v3]).
   */
  const handleStartConsented = useCallback(
    async (mediaIds: string[], budgetUsd: number) => {
      setStarting(true);
      setStartError(null);
      try {
        const outcome = await subtitleService.startGenerationBatch({
          scope: 'selected',
          mediaIds,
          budgetUsd,
        });
        setRetryIds(undefined); // consumed — the next consent render starts clean
        if (outcome.conflict) {
          // A batch was already running (409) — attach to it. We did NOT
          // enumerate its items[], so clear any stale cache from a prior batch
          // this session — the workspace then shows attach-degraded (honest).
          setItems([]);
          setFailedIds(new Set());
          queryClient.removeQueries({ queryKey: generationBatchItemsKey });
          startBatchTracking(outcome.progress);
        } else {
          setItems(outcome.result.items);
          setFailedIds(new Set());
          // Cache items[] so the ux3-ai-2 workspace can render the full queue
          // for this session's batch (the status probe carries none).
          queryClient.setQueryData(generationBatchItemsKey, outcome.result.items);
          startBatchTracking({
            batchId: outcome.result.batchId ?? '',
            totalItems: outcome.result.totalItems,
          });
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
  const handleResume = useCallback(() => {
    const remaining = remainingIds(items, batch.progress, failedIds);
    setRetryIds(remaining.length > 0 ? remaining : undefined);
    resetBatch();
    setItems([]);
    setFailedIds(new Set());
  }, [items, batch.progress, failedIds, resetBatch]);

  // 重試失敗項目 (sub-5-3 AC #3) — same mechanism, failed rows only. Consent
  // is structural: this NEVER calls startGenerationBatch — the only start
  // path stays the F16 confirm inside GenerationConsentView.
  // CR L1: memoized — a fresh array every render defeated handleRetryFailed's
  // own useCallback (the bugfix-19-4b-1 unstable-callback-prop class) and
  // re-walked every row on each SSE tick.
  const failedRows = useMemo(
    () => failedRowIds(items, batch.progress, failedIds),
    [items, batch.progress, failedIds]
  );
  const handleRetryFailed = useCallback(() => {
    setRetryIds(failedRows);
    resetBatch();
    setItems([]);
    setFailedIds(new Set());
  }, [failedRows, resetBatch]);

  const handleConfirmCancelAll = useCallback(async () => {
    try {
      await subtitleService.cancelGenerationBatch();
      // The terminal `cancelled` SSE event flips the status — no manual change.
    } catch {
      // Best-effort cancel — leave the panel as-is on failure.
    }
  }, []);

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
        onStartBatch={(mediaIds, budgetUsd) => void handleStartConsented(mediaIds, budgetUsd)}
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
      onConfirmCancelAll={() => void handleConfirmCancelAll()}
      onResume={handleResume}
      onRetryFailed={failedRows.length > 0 ? handleRetryFailed : undefined}
      onClose={handleClose}
    />
  );
}
