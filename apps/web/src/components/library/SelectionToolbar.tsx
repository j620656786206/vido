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
      className="sticky top-0 z-20 flex items-center justify-between rounded-lg bg-slate-800 px-4 py-3"
    >
      <div className="flex items-center gap-4">
        <span className="text-sm font-medium text-white" data-testid="selected-count">
          已選取 {selectedCount} 項
        </span>
      </div>
      <div className="flex items-center gap-2">
        <button
          onClick={onReparse}
          disabled={isProcessing}
          data-testid="batch-reparse-btn"
          className="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm text-slate-300 transition-colors hover:bg-slate-700 hover:text-white disabled:opacity-50"
        >
          <RefreshCw size={14} />
          重新解析
        </button>
        <button
          onClick={onExport}
          disabled={isProcessing}
          data-testid="batch-export-btn"
          className="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm text-slate-300 transition-colors hover:bg-slate-700 hover:text-white disabled:opacity-50"
        >
          <Download size={14} />
          匯出元數據
        </button>
        <button
          onClick={onDelete}
          disabled={isProcessing}
          data-testid="batch-delete-btn"
          className="flex items-center gap-1.5 rounded-lg bg-red-600/20 px-3 py-1.5 text-sm text-red-400 transition-colors hover:bg-red-600/30 hover:text-red-300 disabled:opacity-50"
        >
          <Trash2 size={14} />
          刪除選取項目
        </button>
        <div className="mx-2 h-5 w-px bg-slate-600" />
        <button
          onClick={onCancel}
          data-testid="batch-cancel-btn"
          className="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm text-slate-400 transition-colors hover:bg-slate-700 hover:text-white"
        >
          <X size={14} />
          取消
        </button>
      </div>
    </div>
  );
}
