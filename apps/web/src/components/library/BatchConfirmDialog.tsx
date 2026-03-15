import { useEffect, useRef } from 'react';
import { AlertTriangle } from 'lucide-react';

interface BatchConfirmDialogProps {
  isOpen: boolean;
  itemCount: number;
  action: 'delete' | 'reparse' | 'export';
  onConfirm: () => void;
  onCancel: () => void;
}

const ACTION_CONFIG = {
  delete: {
    title: '確認刪除',
    message: (count: number) => `確定要刪除 ${count} 個項目嗎？`,
    warning: '此操作無法復原',
    confirmText: '刪除',
    confirmClass: 'bg-red-600 hover:bg-red-700 text-white',
  },
  reparse: {
    title: '確認重新解析',
    message: (count: number) => `確定要重新解析 ${count} 個項目嗎？`,
    warning: '現有元數據將被重置',
    confirmText: '重新解析',
    confirmClass: 'bg-blue-600 hover:bg-blue-700 text-white',
  },
  export: {
    title: '確認匯出',
    message: (count: number) => `確定要匯出 ${count} 個項目的元數據嗎？`,
    warning: '',
    confirmText: '匯出',
    confirmClass: 'bg-blue-600 hover:bg-blue-700 text-white',
  },
};

export function BatchConfirmDialog({
  isOpen,
  itemCount,
  action,
  onConfirm,
  onCancel,
}: BatchConfirmDialogProps) {
  const dialogRef = useRef<HTMLDivElement>(null);
  const config = ACTION_CONFIG[action];

  useEffect(() => {
    if (!isOpen) return;

    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCancel();
    };
    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [isOpen, onCancel]);

  if (!isOpen) return null;

  return (
    <div
      data-testid="batch-confirm-dialog"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
      onClick={(e) => {
        if (e.target === e.currentTarget) onCancel();
      }}
      role="dialog"
      aria-modal="true"
      aria-labelledby="batch-confirm-title"
    >
      <div ref={dialogRef} className="mx-4 w-full max-w-sm rounded-xl bg-slate-800 p-6 shadow-2xl">
        {action === 'delete' && (
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-red-600/20">
            <AlertTriangle className="h-6 w-6 text-red-400" />
          </div>
        )}

        <h3 id="batch-confirm-title" className="mb-2 text-center text-lg font-semibold text-white">
          {config.title}
        </h3>

        <p className="mb-2 text-center text-sm text-slate-300" data-testid="confirm-message">
          {config.message(itemCount)}
        </p>

        {config.warning && (
          <p className="mb-4 text-center text-xs text-red-400" data-testid="confirm-warning">
            {config.warning}
          </p>
        )}

        <div className="flex gap-3">
          <button
            onClick={onCancel}
            data-testid="confirm-cancel-btn"
            className="flex-1 rounded-lg bg-slate-700 px-4 py-2 text-sm font-medium text-slate-300 transition-colors hover:bg-slate-600"
          >
            取消
          </button>
          <button
            onClick={onConfirm}
            data-testid="confirm-action-btn"
            className={`flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-colors ${config.confirmClass}`}
          >
            {config.confirmText}
          </button>
        </div>
      </div>
    </div>
  );
}
