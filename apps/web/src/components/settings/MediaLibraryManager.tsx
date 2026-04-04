/**
 * Media Library Manager component for Settings page (Story 7b-4)
 * Replaces the env var display section in ScannerSettings.
 */

import { useState } from 'react';
import { Plus, Loader, AlertCircle } from 'lucide-react';
import { useMediaLibraries } from '../../hooks/useMediaLibrary';
import { LibraryCard } from './LibraryCard';
import { LibraryEditModal } from './LibraryEditModal';

export function MediaLibraryManager() {
  const { data, isLoading, error } = useMediaLibraries();
  const [editModal, setEditModal] = useState<{ open: boolean; libraryId?: string }>({
    open: false,
  });

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-[var(--text-secondary)]">
        <Loader className="h-4 w-4 animate-spin" />
        <span>載入媒體庫...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center gap-2 rounded-lg bg-red-900/30 px-4 py-3 text-sm text-[var(--error)]">
        <AlertCircle className="h-4 w-4 shrink-0" />
        無法載入媒體庫設定
      </div>
    );
  }

  const libraries = data?.libraries || [];

  return (
    <div className="space-y-3" data-testid="media-library-manager">
      <label className="text-sm font-medium text-[var(--text-secondary)]">媒體庫管理</label>

      {libraries.length === 0 ? (
        <div className="rounded-lg border border-dashed border-[var(--border-subtle)]/50 p-6 text-center text-sm text-[var(--text-muted)]">
          尚未設定任何媒體庫。請新增媒體庫以開始掃描。
        </div>
      ) : (
        <div className="space-y-3" data-testid="library-card-list">
          {libraries.map((lib) => (
            <LibraryCard
              key={lib.id}
              library={lib}
              onEdit={() => setEditModal({ open: true, libraryId: lib.id })}
            />
          ))}
        </div>
      )}

      <button
        type="button"
        onClick={() => setEditModal({ open: true })}
        className="flex w-full items-center justify-center gap-2 rounded-lg border border-dashed border-[var(--border-subtle)]/50 py-2.5 text-sm text-[var(--text-secondary)] transition-colors hover:border-[var(--accent-primary)]/50 hover:text-[var(--accent-primary)]"
        data-testid="add-library-button"
      >
        <Plus className="h-4 w-4" />
        新增媒體庫
      </button>

      {editModal.open && (
        <LibraryEditModal
          libraryId={editModal.libraryId}
          onClose={() => setEditModal({ open: false })}
        />
      )}
    </div>
  );
}
