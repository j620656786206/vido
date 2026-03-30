/**
 * Library Card component for displaying a media library in Settings (Story 7b-4)
 */

import { useState } from 'react';
import { Film, Tv, MoreVertical, Trash2, Pencil, FolderOpen } from 'lucide-react';
import { cn } from '../../lib/utils';
import { useDeleteLibrary } from '../../hooks/useMediaLibrary';
import type { MediaLibraryWithPaths } from '../../services/mediaLibraryService';

interface LibraryCardProps {
  library: MediaLibraryWithPaths;
  onEdit: () => void;
}

const STATUS_CONFIG = {
  accessible: { color: 'text-green-400', bg: 'bg-green-400', label: '已連線' },
  not_found: { color: 'text-red-400', bg: 'bg-red-400', label: '無法存取' },
  not_readable: { color: 'text-red-400', bg: 'bg-red-400', label: '無法讀取' },
  not_directory: { color: 'text-red-400', bg: 'bg-red-400', label: '非目錄' },
  unknown: { color: 'text-slate-400', bg: 'bg-slate-400', label: '未檢查' },
} as const;

export function LibraryCard({ library, onEdit }: LibraryCardProps) {
  const [menuOpen, setMenuOpen] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [removeMedia, setRemoveMedia] = useState(false);
  const deleteLibrary = useDeleteLibrary();

  const TypeIcon = library.contentType === 'movie' ? Film : Tv;
  const typeLabel = library.contentType === 'movie' ? '電影' : '影集';

  const handleDelete = async () => {
    await deleteLibrary.mutateAsync({ id: library.id, removeMedia });
    setConfirmDelete(false);
  };

  return (
    <div
      className="rounded-lg border border-slate-700 bg-slate-800 p-4"
      data-testid={`library-card-${library.id}`}
    >
      {/* Header */}
      <div className="mb-3 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <TypeIcon className="h-4 w-4 text-slate-400" />
          <span className="text-sm font-medium text-slate-200">{library.name}</span>
          <span className="rounded bg-slate-700 px-1.5 py-0.5 text-xs text-slate-400">
            {typeLabel}
          </span>
        </div>
        <div className="relative">
          <button
            type="button"
            onClick={() => setMenuOpen(!menuOpen)}
            className="rounded p-1 text-slate-500 transition-colors hover:bg-slate-700 hover:text-slate-300"
            data-testid="library-menu-button"
          >
            <MoreVertical className="h-4 w-4" />
          </button>
          {menuOpen && (
            <div className="absolute right-0 z-10 mt-1 w-32 rounded-md border border-slate-600 bg-slate-800 py-1 shadow-lg">
              <button
                type="button"
                onClick={() => {
                  setMenuOpen(false);
                  onEdit();
                }}
                className="flex w-full items-center gap-2 px-3 py-1.5 text-sm text-slate-300 hover:bg-slate-700"
              >
                <Pencil className="h-3 w-3" /> 編輯
              </button>
              <button
                type="button"
                onClick={() => {
                  setMenuOpen(false);
                  setConfirmDelete(true);
                }}
                className="flex w-full items-center gap-2 px-3 py-1.5 text-sm text-red-400 hover:bg-slate-700"
              >
                <Trash2 className="h-3 w-3" /> 刪除
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Paths */}
      <div className="mb-3 space-y-1.5">
        {(library.paths || []).map((p) => {
          const statusCfg = STATUS_CONFIG[p.status] || STATUS_CONFIG.unknown;
          return (
            <div key={p.id} className="flex items-center gap-2 text-sm">
              <FolderOpen className="h-3.5 w-3.5 shrink-0 text-slate-500" />
              <span className="flex-1 truncate font-mono text-xs text-slate-400">{p.path}</span>
              <span className={cn('flex items-center gap-1 text-xs', statusCfg.color)}>
                <span className={cn('inline-block h-1.5 w-1.5 rounded-full', statusCfg.bg)} />
                {statusCfg.label}
              </span>
            </div>
          );
        })}
      </div>

      {/* Footer */}
      <div className="text-xs text-slate-500">
        {(library.paths || []).length} 個資料夾 · {library.mediaCount} 個項目
      </div>

      {/* Delete Confirmation */}
      {confirmDelete && (
        <div className="mt-3 rounded-lg border border-red-800/50 bg-red-950/30 p-3">
          <p className="mb-2 text-sm text-red-300">確定要刪除「{library.name}」嗎？</p>
          <label className="mb-3 flex items-center gap-2 text-xs text-slate-400">
            <input
              type="checkbox"
              checked={removeMedia}
              onChange={(e) => setRemoveMedia(e.target.checked)}
              className="rounded border-slate-600"
            />
            同時移除已掃描的媒體資料
          </label>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => setConfirmDelete(false)}
              className="rounded px-3 py-1 text-xs text-slate-400 hover:bg-slate-700"
            >
              取消
            </button>
            <button
              type="button"
              onClick={handleDelete}
              disabled={deleteLibrary.isPending}
              className="rounded bg-red-600 px-3 py-1 text-xs text-white hover:bg-red-700 disabled:opacity-50"
              data-testid="confirm-delete-button"
            >
              {deleteLibrary.isPending ? '刪除中...' : '確認刪除'}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
