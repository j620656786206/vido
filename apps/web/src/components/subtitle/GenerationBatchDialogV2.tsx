// Design ref: ux-design.pen Screen F8-D-v2 (i9Nun1)
/**
 * 批次生成字幕 dialog (Story ux3-subtitle-v2-batch, PH3-M5 slice 2 — FE half of
 * the 9R-16 pair). Screens: F8-D-v2 i9Nun1 / F8-M-v2 H717g (mobile bottom
 * sheet, same Radix Dialog) / F9-D-v2 JMqPg 預算上限.
 *
 * - Scope segments 範圍：缺字幕的項目 (count from GET
 *   /subtitles/generation-batch/preview?scope=missing — the ONLY 缺字幕 count
 *   source) + 已選項目 (rendered ONLY when opened with a non-empty selection).
 * - Queue rows come from the start-202 `items[]`; the active row joins the
 *   per-item `transcription_*` SSE stream on `current_media_id` (slice-1
 *   GenerationProgressV2 / useGenerationProgress reuse). ⚠️ 9R-16 CR caveat:
 *   on cancelled/budget_ceiling the interrupted in-flight item ALSO emits
 *   `transcription_failed` — the batch event's `status`/`paused_count` is
 *   AUTHORITATIVE for row rendering (paused/cancelled branches win over the
 *   recorded per-item failure).
 * - budget_ceiling (F9) is a NORMAL terminal state: warning-tint banner (per
 *   the drawn tokens), paused rows 已暫停 — 下次繼續, actions 關閉 + 下次繼續
 *   (= start a NEW scope=missing batch — 9R-16 resume-for-free; completed
 *   items self-exclude via re-enumeration).
 * - scope=selected sends MOVIE ids only — the backend REJECTS the whole
 *   request with 400 if ANY id is not a movie with a file (9R-16 AC 8); the
 *   caller pre-filters series ids and this dialog shows the visible note.
 * - 409 TRANSCRIPTION_BATCH_RUNNING on open/start → recover-and-attach
 *   (on-open GET .../status probe; BatchSubtitleDialog precedent). NOTE: a
 *   recovered batch has no `items[]` (the status probe carries only the
 *   progress snapshot) — the dialog falls back to the active-item card + counts.
 * - Escape is gated while running (fetch-dialog precedent); closing via ✕ only
 *   stops WATCHING — the batch continues server-side (recover-on-open re-attaches).
 *
 * Rule 23: zero wall-clock reads — progress/cost/counts are all SSE-supplied.
 * [@contract-v2] (9R-18): media ids are UUID STRINGS end-to-end — selection ids
 * pass through unconverted and SSE `current_media_id` joins rows directly.
 */
import { useCallback, useEffect, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Check, CircleAlert, CirclePause, Loader2, Radio } from 'lucide-react';
import { Dialog, DialogContent, DialogTitle } from '../ui/Dialog';
import { cn } from '../../lib/utils';
import {
  subtitleService,
  type GenerationBatchItem,
  type GenerationBatchScope,
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
import { libraryKeys } from '../../hooks/useLibrary';

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

// ---------------------------------------------------------------------------
// Presentational bits
// ---------------------------------------------------------------------------

function usd(v: number): string {
  return `$${v.toFixed(2)}`;
}

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
  /** Hook status ('idle' before a batch starts) — batch statuses are the wire enum. */
  status: GenerationBatchProgressState['status'];
  progress: GenerationBatchProgressState;
  /** Queue rows from the start-202 `items[]` (empty on 409/recover-attach). */
  items: GenerationBatchItem[];
  /** Media ids (UUID strings) whose per-item pipeline failed while the batch ran. */
  failedIds?: ReadonlySet<string>;
  /** Per-item stage detail for the active row (joined on current_media_id). */
  activeItemProgress?: GenerationProgressState | null;
  scope: GenerationBatchScope;
  onScopeChange: (scope: GenerationBatchScope) => void;
  /** 缺字幕 preview count — undefined while loading. */
  previewCount?: number;
  /** >0 → the 已選項目 segment renders (AC 1/5). */
  selectedCount?: number;
  /** Series excluded from the selection client-side (AC 5 visible note). */
  excludedSeriesCount?: number;
  /** Start resolved to total_items:0 → friendly empty state, not an error. */
  emptyScope?: boolean;
  starting?: boolean;
  startError?: string | null;
  onStart: () => void;
  onConfirmCancelAll: () => void;
  /** 下次繼續 — starts a NEW scope=missing batch (resume-for-free). */
  onResume: () => void;
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
  scope,
  onScopeChange,
  previewCount,
  selectedCount = 0,
  excludedSeriesCount = 0,
  emptyScope = false,
  starting = false,
  startError = null,
  onStart,
  onConfirmCancelAll,
  onResume,
  onClose,
}: GenerationBatchPanelV2Props) {
  const [confirmingCancel, setConfirmingCancel] = useState(false);

  const isRunning = status === 'running';
  const isIdle = status === 'idle';
  const isBudgetCeiling = status === 'budget_ceiling';
  const isTerminal = status === 'complete' || status === 'cancelled' || status === 'error';

  useEffect(() => {
    if (!isRunning) setConfirmingCancel(false);
  }, [isRunning]);

  const processed = progress.successCount + progress.failCount;
  const pct = progress.totalItems > 0 ? (processed / progress.totalItems) * 100 : 0;
  const rowStates = deriveRowStates(items, progress, failedIds);

  const showIdleEmpty = isIdle && scope === 'missing' && (emptyScope || previewCount === 0);
  const startDisabled =
    starting ||
    (scope === 'missing' && (previewCount === undefined || previewCount === 0)) ||
    (scope === 'selected' && selectedCount === 0);

  const statusAnnouncement = isBudgetCeiling
    ? `已達本次預算上限（${usd(progress.budgetUsd)}）— 已完成${processed}部，剩餘${progress.pausedCount}部下次繼續`
    : status === 'complete'
      ? '批次生成完成'
      : status === 'cancelled'
        ? '批次已取消'
        : status === 'error'
          ? '批次發生錯誤'
          : '';

  const scopeStatic =
    scope === 'missing' ? (
      <p className="flex items-center gap-[3px] text-[13px] text-[var(--text-secondary)]">
        範圍：缺字幕的項目（
        <span className="font-mono tabular-nums">{progress.totalItems}</span> 部）
      </p>
    ) : (
      <p className="flex items-center gap-[3px] text-[13px] text-[var(--text-secondary)]">
        範圍：已選項目（
        <span className="font-mono tabular-nums">{progress.totalItems}</span> 部）
      </p>
    );

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
          <DialogTitle className="truncate text-base font-semibold">批次生成字幕</DialogTitle>
        </div>

        {/* Body */}
        <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-6">
          {/* Status transitions announced to AT (AC 7). */}
          <p aria-live="polite" className="sr-only" data-testid="gen-batch-status-live">
            {statusAnnouncement}
          </p>

          {/* ---------- Scope line ---------- */}
          {isIdle ? (
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-[13px] text-[var(--text-secondary)]">範圍：</span>
              <button
                type="button"
                data-testid="gen-batch-scope-missing"
                aria-pressed={scope === 'missing'}
                onClick={() => onScopeChange('missing')}
                className={cn(
                  'flex h-11 items-center gap-1 rounded-[var(--radius-sm)] px-3 text-[13px] transition-colors',
                  scope === 'missing'
                    ? 'bg-[var(--accent-subtle)] font-semibold text-[var(--accent-text)]'
                    : 'bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
                )}
              >
                缺字幕的項目
                {previewCount === undefined ? (
                  <span
                    aria-hidden="true"
                    className="h-3.5 w-6 animate-pulse rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)] motion-reduce:animate-none"
                  />
                ) : (
                  <span className="font-mono font-semibold tabular-nums">{previewCount}</span>
                )}
              </button>
              {selectedCount > 0 && (
                <button
                  type="button"
                  data-testid="gen-batch-scope-selected"
                  aria-pressed={scope === 'selected'}
                  onClick={() => onScopeChange('selected')}
                  className={cn(
                    'flex h-11 items-center gap-1 rounded-[var(--radius-sm)] px-3 text-[13px] transition-colors',
                    scope === 'selected'
                      ? 'bg-[var(--accent-subtle)] font-semibold text-[var(--accent-text)]'
                      : 'bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
                  )}
                >
                  已選項目
                  <span className="font-mono font-semibold tabular-nums">{selectedCount}</span>
                </button>
              )}
            </div>
          ) : (
            scopeStatic
          )}

          {/* AC 5 capability-honor note: series excluded from the selection.
              NOT gated on scope==='selected' — when EVERY selected item was
              excluded the 已選項目 segment never renders (scope stays missing),
              and that is precisely when the exclusion must be visible. */}
          {isIdle && excludedSeriesCount > 0 && (
            <p
              data-testid="gen-batch-excluded-note"
              className="flex items-center gap-[3px] text-xs text-[var(--text-muted)]"
            >
              已排除 <span className="font-mono tabular-nums">{excludedSeriesCount}</span>
              部影集（影集字幕生成即將推出）
            </p>
          )}

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

          {/* ---------- Idle: empty-scope friendly state / start hint ---------- */}
          {isIdle && showIdleEmpty && (
            <p
              data-testid="gen-batch-empty-scope"
              className="rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] p-4 text-center text-sm text-[var(--text-secondary)]"
            >
              目前沒有缺字幕的項目
            </p>
          )}
          {isIdle && startError && (
            <p data-testid="gen-batch-start-error" className="text-sm text-[var(--error)]">
              {startError}
            </p>
          )}

          {/* ---------- Overall progress (running + terminal) ---------- */}
          {!isIdle && (
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
          )}

          {/* ---------- Queue rows ---------- */}
          {!isIdle &&
            (items.length > 0 ? (
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
            ))}

          {/* ---------- Cost row ---------- */}
          {!isIdle && (
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
          )}
        </div>

        {/* Footer */}
        <div className="flex shrink-0 items-center justify-end gap-3 border-t border-[var(--border-subtle)] px-6 py-3.5">
          {isIdle && (
            <button
              type="button"
              onClick={onStart}
              disabled={startDisabled}
              data-testid="gen-batch-start-btn"
              className="flex min-h-[44px] items-center gap-2 rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-6 text-sm font-medium text-[var(--text-on-accent)] transition-colors hover:bg-[var(--accent-pressed)] disabled:cursor-not-allowed disabled:opacity-50"
            >
              {starting && <Loader2 className="h-4 w-4 animate-spin motion-reduce:animate-none" />}
              開始生成
            </button>
          )}

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

          {(isTerminal || isBudgetCeiling || (isIdle && emptyScope)) && (
            <button
              type="button"
              onClick={onClose}
              data-testid="gen-batch-close-btn"
              className="flex min-h-[44px] items-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-5 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)]"
            >
              關閉
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
// Container — owns the queries, lazy-SSE hooks and the per-item join
// ---------------------------------------------------------------------------

export interface GenerationBatchDialogV2Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /**
   * Pre-filtered MOVIE media ids (UUID strings, [@contract-v2] — the selection
   * Set's ids pass through unconverted) when opened from a library selection.
   * Empty/absent → the 已選項目 segment is not rendered (AC 1).
   */
  selectedMovieIds?: string[];
  /** Series ids the caller excluded from the selection (AC 5 visible note). */
  excludedSeriesCount?: number;
}

export function GenerationBatchDialogV2({
  open,
  onOpenChange,
  selectedMovieIds,
  excludedSeriesCount = 0,
}: GenerationBatchDialogV2Props) {
  const queryClient = useQueryClient();
  const hasSelection = (selectedMovieIds?.length ?? 0) > 0;

  const [scope, setScope] = useState<GenerationBatchScope>(hasSelection ? 'selected' : 'missing');
  const [items, setItems] = useState<GenerationBatchItem[]>([]);
  const [failedIds, setFailedIds] = useState<Set<string>>(new Set());
  const [emptyScope, setEmptyScope] = useState(false);
  const [starting, setStarting] = useState(false);
  const [startError, setStartError] = useState<string | null>(null);

  const batch = useGenerationBatchProgress();
  const perItem = useGenerationProgress();
  const { startTracking: startBatchTracking, reset: resetBatch } = batch;
  const { startTracking: startItemTracking, reset: resetItem } = perItem;

  const isIdle = batch.status === 'idle';

  // 缺字幕 count comes ONLY from the 9R-16 preview endpoint (Rule 5 query).
  const previewQuery = useQuery({
    queryKey: generationBatchPreviewKey,
    queryFn: () => subtitleService.previewGenerationBatch(),
    enabled: open && isIdle,
  });

  // Reset the scope preselection whenever the dialog (re)opens (AC 1/5).
  useEffect(() => {
    if (open) setScope(hasSelection ? 'selected' : 'missing');
  }, [open, hasSelection]);

  // On open, recover an already-running batch (409-recover precedent): the
  // status probe lets us jump straight into the running view and attach SSE.
  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    subtitleService
      .getGenerationBatchStatus()
      .then((s) => {
        if (cancelled) return;
        if (s.running && s.progress) startBatchTracking(s.progress);
      })
      .catch(() => {
        // Best-effort recovery — a failed probe just leaves the panel idle.
      });
    return () => {
      cancelled = true;
    };
  }, [open, startBatchTracking]);

  // Per-item join: track the in-flight item's transcription_* stream on
  // current_media_id (slice-1 hook reuse; reconnects per item).
  const currentMediaId = batch.progress.currentMediaId;
  useEffect(() => {
    if (batch.status === 'running' && currentMediaId != null) {
      startItemTracking(currentMediaId);
    }
  }, [batch.status, currentMediaId, startItemTracking]);

  // Record per-item failures ONLY while the batch is still running — a
  // transcription_failed that coincides with a non-error terminal batch status
  // is the interrupted in-flight item (9R-16 CR caveat), and the paused/
  // cancelled row branches override it at render time regardless.
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
  // subtitle_status (9R-16 AC 12) → refresh library badges/counts + the
  // preview count for the next 下次繼續 round.
  const batchStatus = batch.status;
  useEffect(() => {
    if (batchStatus === 'idle' || batchStatus === 'running') return;
    resetItem();
    void queryClient.invalidateQueries({ queryKey: libraryKeys.all });
    void queryClient.invalidateQueries({ queryKey: generationBatchPreviewKey });
  }, [batchStatus, resetItem, queryClient]);

  const handleStart = useCallback(
    async (startScope: GenerationBatchScope) => {
      setStarting(true);
      setStartError(null);
      setEmptyScope(false);
      try {
        const outcome = await subtitleService.startGenerationBatch(
          startScope === 'selected'
            ? { scope: 'selected', mediaIds: selectedMovieIds ?? [] }
            : { scope: 'missing' }
        );
        if (outcome.conflict) {
          // A generation batch was already running (409) — attach to it. We did NOT
          // enumerate its items[], so clear any stale cache from a prior batch this
          // session — the ux3-ai-2 workspace then shows attach-degraded (honest),
          // never a previous batch's rows joined against this batch's progress.
          setItems([]);
          setFailedIds(new Set());
          queryClient.removeQueries({ queryKey: generationBatchItemsKey });
          startBatchTracking(outcome.progress);
        } else if (outcome.result.totalItems === 0) {
          setEmptyScope(true);
        } else {
          setItems(outcome.result.items);
          setFailedIds(new Set());
          // Cache items[] so the ux3-ai-2 workspace can render the full queue for
          // this session's batch (the status probe carries none).
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
    [selectedMovieIds, startBatchTracking]
  );

  // 下次繼續 = a NEW scope=missing batch (9R-16 resume-for-free ruling —
  // completed items self-exclude because the missing enumeration re-runs).
  const handleResume = useCallback(() => {
    setScope('missing');
    resetBatch();
    setItems([]);
    setFailedIds(new Set());
    void handleStart('missing');
  }, [handleStart, resetBatch]);

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
    setEmptyScope(false);
    setStarting(false);
    setStartError(null);
    onOpenChange(false);
  }, [resetBatch, resetItem, onOpenChange]);

  if (!open) return null;

  return (
    <GenerationBatchPanelV2
      open={open}
      status={batch.status}
      progress={batch.progress}
      items={items}
      failedIds={failedIds}
      activeItemProgress={perItem.progress}
      scope={scope}
      onScopeChange={setScope}
      previewCount={previewQuery.data?.totalItems}
      selectedCount={selectedMovieIds?.length ?? 0}
      excludedSeriesCount={excludedSeriesCount}
      emptyScope={emptyScope}
      starting={starting}
      startError={startError}
      onStart={() => void handleStart(scope)}
      onConfirmCancelAll={() => void handleConfirmCancelAll()}
      onResume={handleResume}
      onClose={handleClose}
    />
  );
}
