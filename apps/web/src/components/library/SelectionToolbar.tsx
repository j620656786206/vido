import { Trash2, RefreshCw, Download, X } from 'lucide-react';

interface SelectionToolbarProps {
  selectedCount: number;
  onDelete: () => void;
  onReparse: () => void;
  onExport: () => void;
  onCancel: () => void;
  isProcessing?: boolean;
}

export function SelectionToolbar({
  selectedCount,
  onDelete,
  onReparse,
  onExport,
  onCancel,
  isProcessing,
}: SelectionToolbarProps) {
  return (
    <div
      data-testid="selection-toolbar"
      className="sticky top-0 z-20 flex flex-col gap-2 rounded-lg bg-[var(--bg-secondary)] px-4 py-3 sm:flex-row sm:items-center sm:justify-between"
    >
      <div className="flex items-center gap-4">
        <span className="text-sm font-medium text-white" data-testid="selected-count">
          已選取 {selectedCount} 項
        </span>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <button
          onClick={onReparse}
          disabled={isProcessing}
          data-testid="batch-reparse-btn"
          aria-label="重新解析"
          className="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-white disabled:opacity-50"
        >
          <RefreshCw size={14} />
          <span className="hidden sm:inline">重新解析</span>
        </button>
        <button
          onClick={onExport}
          disabled={isProcessing}
          data-testid="batch-export-btn"
          aria-label="匯出中繼資料"
          className="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-white disabled:opacity-50"
        >
          <Download size={14} />
          <span className="hidden sm:inline">匯出中繼資料</span>
        </button>
        <button
          onClick={onDelete}
          disabled={isProcessing}
          data-testid="batch-delete-btn"
          aria-label="刪除選取項目"
          className="flex items-center gap-1.5 rounded-lg bg-[var(--error)]/20 px-3 py-1.5 text-sm text-[var(--error)] transition-colors hover:bg-[var(--error)]/30 hover:text-red-300 disabled:opacity-50"
        >
          <Trash2 size={14} />
          <span className="hidden sm:inline">刪除選取項目</span>
        </button>
        <div className="mx-2 hidden h-5 w-px bg-[var(--bg-tertiary)] sm:block" />
        <button
          onClick={onCancel}
          data-testid="batch-cancel-btn"
          className="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-white"
        >
          <X size={14} />
          取消
        </button>
      </div>
    </div>
  );
}
