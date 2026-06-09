// Design ref: ux-design.pen Screen G4 Batch Subtitle Processing Desktop (NXijD)
// Mobile peek per Screen G6 (fUtqO). Screen-section component — no Reusable
// Component node in the .pen; G4/G6 are screen frames (Rule 21 Design-ref form).
import { useState, useCallback, useEffect } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { X, Loader2, Search } from 'lucide-react';
import { subtitleService, type BatchScope } from '../../services/subtitleService';
import {
  useSubtitleBatchProgress,
  type BatchStatus,
  type BatchProgressState,
} from '../../hooks/useSubtitleBatchProgress';

// ---------------------------------------------------------------------------
// Presentational panel (prop-driven so all three states are visual-testable)
// ---------------------------------------------------------------------------

interface BatchSubtitlePanelProps {
  status: BatchStatus;
  progress: BatchProgressState;
  /** When provided, the 「整季」 scope option is offered (AC #2). */
  seasonId?: string;
  starting?: boolean;
  startError?: string | null;
  onStart: (scope: BatchScope) => void;
  onConfirmCancel: () => void;
  onViewNotFound: () => void;
  onClose: () => void;
}

export function BatchSubtitlePanel({
  status,
  progress,
  seasonId,
  starting = false,
  startError = null,
  onStart,
  onConfirmCancel,
  onViewNotFound,
  onClose,
}: BatchSubtitlePanelProps) {
  const [scope, setScope] = useState<BatchScope>('library');
  const [confirmingCancel, setConfirmingCancel] = useState(false);

  const isProcessing = status === 'running';
  const isTerminal = status === 'complete' || status === 'cancelled' || status === 'error';
  const pct = progress.totalItems > 0 ? (progress.currentIndex / progress.totalItems) * 100 : 0;

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center bg-black/60 sm:items-center"
      onClick={(e) => {
        // Backdrop click closes only when not mid-run.
        if (e.target === e.currentTarget && !isProcessing) onClose();
      }}
      role="dialog"
      aria-modal="true"
      aria-labelledby="batch-subtitle-title"
      data-testid="batch-subtitle-dialog"
    >
      <div className="w-full rounded-t-2xl bg-[var(--bg-secondary)] shadow-2xl sm:mx-4 sm:max-w-md sm:rounded-xl">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-[var(--border-subtle)] px-5 py-4">
          <h2 id="batch-subtitle-title" className="text-base font-semibold text-white">
            {status === 'complete' ? '批次字幕搜尋完成' : '批次字幕搜尋'}
          </h2>
          {!isProcessing && (
            <button
              onClick={onClose}
              className="rounded-lg p-1 text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] hover:text-white"
              aria-label="關閉"
            >
              <X className="h-5 w-5" />
            </button>
          )}
        </div>

        <div className="p-5">
          {/* ---------- Idle: scope selector + start ---------- */}
          {status === 'idle' && (
            <div className="space-y-4">
              <fieldset className="space-y-2">
                <legend className="mb-1 text-sm font-medium text-[var(--text-secondary)]">
                  搜尋範圍
                </legend>
                <label className="flex cursor-pointer items-center gap-2">
                  <input
                    type="radio"
                    name="batch-scope"
                    value="library"
                    checked={scope === 'library'}
                    onChange={() => setScope('library')}
                    className="text-[var(--accent-primary)]"
                    data-testid="batch-subtitle-scope-library"
                  />
                  <span className="text-sm text-white">整個媒體庫</span>
                </label>
                {seasonId && (
                  <label className="flex cursor-pointer items-center gap-2">
                    <input
                      type="radio"
                      name="batch-scope"
                      value="season"
                      checked={scope === 'season'}
                      onChange={() => setScope('season')}
                      className="text-[var(--accent-primary)]"
                      data-testid="batch-subtitle-scope-season"
                    />
                    <span className="text-sm text-white">整季</span>
                  </label>
                )}
              </fieldset>

              {startError && (
                <p className="text-sm text-[var(--error)]" data-testid="batch-subtitle-error">
                  {startError}
                </p>
              )}

              <button
                onClick={() => onStart(scope)}
                disabled={starting}
                data-testid="batch-subtitle-start-btn"
                className="flex w-full items-center justify-center gap-2 rounded-lg bg-[var(--accent-primary)] px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-[var(--accent-pressed)] disabled:opacity-50"
              >
                {starting ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Search className="h-4 w-4" />
                )}
                開始批次搜尋
              </button>
            </div>
          )}

          {/* ---------- Processing ---------- */}
          {isProcessing && (
            <div className="space-y-3">
              {/* Progress bar */}
              <div className="h-1.5 overflow-hidden rounded-full bg-[var(--bg-tertiary)]">
                <div
                  data-testid="batch-subtitle-progress-bar"
                  className="h-full rounded-full bg-[var(--accent-primary)] transition-all duration-300"
                  style={{ width: `${pct}%` }}
                />
              </div>

              <div className="flex items-center justify-between">
                <span
                  className="font-mono text-sm text-[var(--text-secondary)]"
                  data-testid="batch-subtitle-counter"
                >
                  {progress.currentIndex} / {progress.totalItems}
                </span>
                <div className="flex gap-3 text-sm">
                  <span className="text-[var(--success)]" data-testid="batch-subtitle-found">
                    找到 {progress.successCount}
                  </span>
                  <span className="text-[var(--error)]" data-testid="batch-subtitle-notfound">
                    未找到 {progress.failCount}
                  </span>
                </div>
              </div>

              {progress.currentItem && (
                <p
                  className="truncate text-sm text-[var(--text-secondary)]"
                  title={progress.currentItem}
                >
                  正在搜尋：{progress.currentItem}
                </p>
              )}

              {!confirmingCancel ? (
                <button
                  onClick={() => setConfirmingCancel(true)}
                  data-testid="batch-subtitle-cancel-btn"
                  className="flex w-full items-center justify-center gap-1 rounded-lg px-3 py-2 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-white"
                >
                  <X size={14} />
                  取消
                </button>
              ) : (
                <div
                  className="rounded-lg bg-[var(--bg-primary)] p-3"
                  data-testid="batch-subtitle-cancel-confirm"
                >
                  <p className="mb-3 text-sm text-[var(--text-secondary)]">
                    確定要取消嗎？已處理的結果會保留。
                  </p>
                  <div className="flex justify-end gap-2">
                    <button
                      onClick={() => setConfirmingCancel(false)}
                      className="rounded-lg px-3 py-1.5 text-sm text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] hover:text-white"
                    >
                      繼續搜尋
                    </button>
                    <button
                      onClick={() => {
                        setConfirmingCancel(false);
                        onConfirmCancel();
                      }}
                      data-testid="batch-subtitle-cancel-confirm-btn"
                      className="rounded-lg bg-[var(--error)]/20 px-3 py-1.5 text-sm text-[var(--error)] hover:bg-[var(--error)]/30"
                    >
                      確定取消
                    </button>
                  </div>
                </div>
              )}
            </div>
          )}

          {/* ---------- Terminal: complete / cancelled ---------- */}
          {isTerminal && (
            <div className="space-y-4">
              <p
                className="text-sm text-[var(--text-secondary)]"
                data-testid="batch-subtitle-summary"
              >
                {status === 'cancelled' ? '已取消 · ' : ''}
                找到 {progress.successCount} · 未找到 {progress.failCount} · 共{' '}
                {progress.totalItems}
              </p>

              <div className="flex items-center justify-between gap-2">
                <button
                  onClick={onViewNotFound}
                  data-testid="batch-subtitle-view-notfound"
                  className="text-sm text-[var(--accent-primary)] hover:text-blue-300"
                >
                  查看未找到項目
                </button>
                <button
                  onClick={onClose}
                  data-testid="batch-subtitle-close-btn"
                  className="rounded-lg bg-[var(--bg-tertiary)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--bg-tertiary)]"
                >
                  關閉
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Container — owns the lazy-SSE hook + service calls
// ---------------------------------------------------------------------------

interface BatchSubtitleDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** When provided, the 「整季」 scope option is offered (AC #2). */
  seasonId?: string;
}

export function BatchSubtitleDialog({ open, onOpenChange, seasonId }: BatchSubtitleDialogProps) {
  const navigate = useNavigate();
  const { progress, status, startTracking, reset } = useSubtitleBatchProgress();

  const [starting, setStarting] = useState(false);
  const [startError, setStartError] = useState<string | null>(null);

  const isProcessing = status === 'running';

  const handleClose = useCallback(() => {
    reset();
    setStarting(false);
    setStartError(null);
    onOpenChange(false);
  }, [reset, onOpenChange]);

  // Escape closes only when not mid-run (AC #5/#8).
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !isProcessing) handleClose();
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [open, isProcessing, handleClose]);

  const handleStart = useCallback(
    async (scope: BatchScope) => {
      setStarting(true);
      setStartError(null);
      try {
        const outcome = await subtitleService.startBatch({
          scope,
          seasonId: scope === 'season' ? seasonId : undefined,
        });
        if (outcome.conflict) {
          // A batch was already running (409) — recover by tracking it (AC #7).
          startTracking(outcome.progress);
        } else {
          startTracking({
            batchId: outcome.result.batchId,
            totalItems: outcome.result.totalItems,
          });
        }
      } catch (err) {
        setStartError(err instanceof Error ? err.message : '批次搜尋啟動失敗');
      } finally {
        setStarting(false);
      }
    },
    [seasonId, startTracking]
  );

  const handleConfirmCancel = useCallback(async () => {
    try {
      await subtitleService.cancelBatch();
      // The terminal `cancelled` SSE event flips `status`; no manual change.
    } catch {
      // Best-effort cancel — leave the panel as-is on failure.
    }
  }, []);

  const handleViewNotFound = useCallback(() => {
    handleClose();
    // Forward-compatible deep-link. NOTE: backend list filtering by
    // subtitle_status is NOT yet supported (tracked backlog); the param is
    // preserved by the route's validateSearch for when it lands.
    navigate({ to: '/library', search: (prev) => ({ ...prev, subtitleStatus: 'not_found' }) });
  }, [navigate, handleClose]);

  if (!open) return null;

  return (
    <BatchSubtitlePanel
      status={status}
      progress={progress}
      seasonId={seasonId}
      starting={starting}
      startError={startError}
      onStart={handleStart}
      onConfirmCancel={handleConfirmCancel}
      onViewNotFound={handleViewNotFound}
      onClose={handleClose}
    />
  );
}
