import { useState, useRef, useEffect } from 'react';
import { MoreHorizontal, RefreshCw, Download, Trash2 } from 'lucide-react';

export interface DetailPanelMenuProps {
  onReparse: () => void;
  onExport: () => void;
  onDelete: () => void;
}

export function DetailPanelMenu({ onReparse, onExport, onDelete }: DetailPanelMenuProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setIsOpen(false);
        setShowConfirm(false);
      }
    }
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
    }
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [isOpen]);

  return (
    <div className="relative" ref={menuRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-slate-700 hover:text-white"
        aria-label="更多操作"
        aria-expanded={isOpen}
        aria-haspopup="menu"
        data-testid="detail-menu-trigger"
      >
        <MoreHorizontal size={20} />
      </button>

      {isOpen && (
        <div
          className="absolute right-0 top-full z-50 mt-1 w-48 rounded-lg border border-slate-700 bg-slate-800 py-1 shadow-xl"
          role="menu"
          data-testid="detail-menu-dropdown"
        >
          <button
            onClick={() => {
              onReparse();
              setIsOpen(false);
            }}
            className="flex w-full items-center gap-2 px-3 py-2 text-sm text-gray-300 hover:bg-slate-700"
            role="menuitem"
            data-testid="menu-reparse"
          >
            <RefreshCw size={16} />
            重新解析
          </button>
          <button
            onClick={() => {
              onExport();
              setIsOpen(false);
            }}
            className="flex w-full items-center gap-2 px-3 py-2 text-sm text-gray-300 hover:bg-slate-700"
            role="menuitem"
            data-testid="menu-export"
          >
            <Download size={16} />
            匯出中繼資料
          </button>
          <div className="my-1 border-t border-slate-700" role="separator" />
          {!showConfirm ? (
            <button
              onClick={() => setShowConfirm(true)}
              className="flex w-full items-center gap-2 px-3 py-2 text-sm text-red-400 hover:bg-slate-700"
              role="menuitem"
              data-testid="menu-delete"
            >
              <Trash2 size={16} />
              刪除
            </button>
          ) : (
            <div className="px-3 py-2">
              <p className="mb-2 text-xs text-gray-400">確定要刪除嗎？</p>
              <div className="flex gap-2">
                <button
                  onClick={() => {
                    onDelete();
                    setIsOpen(false);
                    setShowConfirm(false);
                  }}
                  className="rounded bg-red-600 px-3 py-1 text-xs text-white hover:bg-red-500"
                  data-testid="confirm-delete"
                >
                  確定
                </button>
                <button
                  onClick={() => setShowConfirm(false)}
                  className="rounded bg-slate-700 px-3 py-1 text-xs text-gray-300 hover:bg-slate-600"
                  data-testid="cancel-delete"
                >
                  取消
                </button>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
