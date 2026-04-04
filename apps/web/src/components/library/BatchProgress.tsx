import { X } from 'lucide-react';
import type { BatchError } from '../../types/library';

interface BatchProgressProps {
  isOpen: boolean;
  current: number;
  total: number;
  action: string;
  errors?: BatchError[];
  isComplete: boolean;
  onClose: () => void;
  onCancel?: () => void;
}

export function BatchProgress({
  isOpen,
  current,
  total,
  action,
  errors,
  isComplete,
  onClose,
  onCancel,
}: BatchProgressProps) {
  if (!isOpen) return null;

  const progress = total > 0 ? (current / total) * 100 : 0;
  const hasErrors = errors && errors.length > 0;

  return (
    <div
      data-testid="batch-progress"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
      role="dialog"
      aria-modal="true"
    >
      <div className="mx-4 w-full max-w-sm rounded-xl bg-[var(--bg-secondary)] p-6 shadow-2xl">
        <h3 className="mb-4 text-lg font-semibold text-white">
          {isComplete ? '操作完成' : action}
        </h3>

        {/* Progress bar */}
        <div className="mb-2 h-2 overflow-hidden rounded-full bg-[var(--bg-tertiary)]">
          <div
            data-testid="progress-bar"
            className="h-full rounded-full bg-[var(--accent-primary)] transition-all duration-300"
            style={{ width: `${progress}%` }}
          />
        </div>

        <p className="mb-4 text-sm text-[var(--text-secondary)]" data-testid="progress-text">
          {isComplete ? `已完成 ${current} / ${total}` : `處理中 ${current} / ${total}...`}
        </p>

        {/* Error list */}
        {hasErrors && (
          <div className="mb-4 max-h-32 overflow-y-auto rounded-lg bg-[var(--bg-primary)] p-3">
            <p className="mb-2 text-xs font-medium text-[var(--error)]">
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
              className="flex items-center gap-1 rounded-lg px-3 py-1.5 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-white"
            >
              <X size={14} />
              取消
            </button>
          )}
          {isComplete && (
            <button
              onClick={onClose}
              data-testid="progress-close-btn"
              className="rounded-lg bg-[var(--bg-tertiary)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--bg-tertiary)]"
            >
              關閉
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
