// Design ref: ux-design.pen Screen C24-D 批次重新解析 — 比對進度 spec (L7EVR)
import { X } from 'lucide-react';
import type { BatchError } from '../../types/library';
import type { EnrichmentMatching } from '../../hooks/useEnrichmentRefresh';

interface BatchProgressProps {
  isOpen: boolean;
  current: number;
  total: number;
  action: string;
  errors?: BatchError[];
  isComplete: boolean;
  /**
   * Batch 重新解析 only (disc-2026-09-batch-reparse-progress-in-dialog): the
   * request returned but the match keeps running in the background. The dialog
   * then walks C24-D's three states — 重新解析中 (queued) → 比對中 (running) →
   * 比對完成 (done) — with every number coming from the SSE events. `total` is
   * the whole pass (every pending row), so the copy says 含你勾的 N 部 and never
   * 你勾的 T 部. Absent for delete/export, and when nothing was queued.
   */
  matching?: EnrichmentMatching | null;
  onClose: () => void;
  onCancel?: () => void;
}

function matchingTitle(m: EnrichmentMatching): string {
  return m.phase === 'queued' ? '重新解析中' : m.phase === 'running' ? '比對中' : '比對完成';
}

export function BatchProgress({
  isOpen,
  current,
  total,
  action,
  errors,
  isComplete,
  matching,
  onClose,
  onCancel,
}: BatchProgressProps) {
  if (!isOpen) return null;

  const m = isComplete && matching ? matching : null;
  const progress = m
    ? m.phase === 'done'
      ? 100
      : m.total > 0
        ? (m.processed / m.total) * 100
        : 0
    : total > 0
      ? (current / total) * 100
      : 0;
  const hasErrors = errors && errors.length > 0;

  return (
    <div
      data-testid="batch-progress"
      // Modal backdrop = the scrim token's documented second role; stays ink in both
      // themes so a paper modal keeps a boundary. Alpha 60% → the token's 70%.
      className="fixed inset-0 z-50 flex items-center justify-center bg-[var(--overlay-scrim)]"
      role="dialog"
      aria-modal="true"
      data-matching-phase={m?.phase}
    >
      <div className="mx-4 w-full max-w-sm rounded-xl bg-[var(--bg-secondary)] p-6 shadow-[var(--shadow-xl)]">
        <h3 className="mb-4 text-lg font-semibold text-[var(--text-primary)]">
          {m ? matchingTitle(m) : isComplete ? '操作完成' : action}
        </h3>

        {/* Progress bar */}
        <div className="mb-2 h-2 overflow-hidden rounded-full bg-[var(--bg-tertiary)]">
          <div
            data-testid="progress-bar"
            className="h-full rounded-full bg-[var(--accent-primary)] transition-all duration-[var(--motion-move)]"
            style={{ width: `${progress}%` }}
          />
        </div>

        {m ? (
          <div className="mb-4 flex flex-col gap-1 text-sm text-[var(--text-secondary)]">
            <p data-testid="progress-text">
              {m.phase === 'queued'
                ? `已排入比對 ${current} 項，等待開始…`
                : m.phase === 'running'
                  ? `本輪整理 ${m.total} 部（含你勾的 ${current} 部）`
                  : `成功 ${m.succeeded}・失敗 ${m.failed} — 清單已更新`}
            </p>
            {m.phase === 'queued' && (
              <p data-testid="matching-note">本輪會整理所有待整理的片，不只你勾的 {current} 部。</p>
            )}
            {m.phase === 'running' && (
              <>
                <p className="flex items-baseline justify-between gap-3">
                  <span className="min-w-0 truncate" data-testid="matching-current">
                    目前：{m.currentTitle || '—'}
                  </span>
                  <span
                    className="shrink-0 font-mono tabular-nums text-[var(--text-primary)]"
                    data-testid="matching-count"
                  >
                    {m.processed} / {m.total}
                  </span>
                </p>
                <p data-testid="matching-tally">
                  成功 {m.succeeded}・失敗 {m.failed}・略過 {m.skipped}
                </p>
              </>
            )}
          </div>
        ) : (
          <p className="mb-4 text-sm text-[var(--text-secondary)]" data-testid="progress-text">
            {isComplete ? `已完成 ${current} / ${total}` : `處理中 ${current} / ${total}...`}
          </p>
        )}

        {/* Error list */}
        {hasErrors && (
          <div className="mb-4 max-h-32 overflow-y-auto rounded-lg bg-[var(--bg-primary)] p-3">
            <p className="mb-2 text-xs font-medium text-[var(--error-text)]">
              {errors.length} 個項目失敗：
            </p>
            {errors.map((err) => (
              <p key={err.id} className="text-xs text-[var(--text-secondary)]">
                {err.id}: {err.message}
              </p>
            ))}
          </div>
        )}

        <div className="flex justify-end gap-2">
          {!isComplete && onCancel && (
            <button
              onClick={onCancel}
              data-testid="progress-cancel-btn"
              className="flex items-center gap-1 rounded-lg px-3 py-1.5 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]"
            >
              <X size={14} />
              取消
            </button>
          )}
          {isComplete && (
            <button
              onClick={onClose}
              data-testid="progress-close-btn"
              // --bg-tertiary is a page GROUND, not a semantic fill, so this label is
              // --text-primary and not --text-on-accent (which is cut for gold/cinnabar).
              className="rounded-lg bg-[var(--bg-tertiary)] px-4 py-2 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-tertiary)]"
            >
              關閉
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
