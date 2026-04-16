/**
 * Settings → 自訂首頁 — Story 10.3 management UI.
 */

import { useCallback, useEffect, useState } from 'react';
import { Plus, Pencil, Trash2, ArrowUp, ArrowDown } from 'lucide-react';
import {
  useExploreBlocks,
  useDeleteExploreBlock,
  useReorderExploreBlocks,
} from '../../hooks/useExploreBlocks';
import type { ExploreBlock } from '../../services/exploreBlockService';
import { ExploreBlockEditModal } from './ExploreBlockEditModal';

export function ExploreBlocksSettings() {
  const { data, isLoading, isError } = useExploreBlocks();
  const deleteBlock = useDeleteExploreBlock();
  const reorderBlocks = useReorderExploreBlocks();

  const [modalMode, setModalMode] = useState<
    { type: 'closed' } | { type: 'create' } | { type: 'edit'; block: ExploreBlock }
  >({ type: 'closed' });
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
  const [operationError, setOperationError] = useState<string | null>(null);

  const blocks = data?.blocks ?? [];

  const handleMove = async (index: number, direction: 'up' | 'down') => {
    const targetIndex = direction === 'up' ? index - 1 : index + 1;
    if (targetIndex < 0 || targetIndex >= blocks.length) return;
    const reordered = [...blocks];
    [reordered[index], reordered[targetIndex]] = [reordered[targetIndex], reordered[index]];
    try {
      setOperationError(null);
      await reorderBlocks.mutateAsync(reordered.map((b) => b.id));
    } catch (err) {
      setOperationError(err instanceof Error ? err.message : '排序失敗，請稍後再試');
    }
  };

  const handleConfirmDelete = async () => {
    if (!confirmDeleteId) return;
    try {
      setOperationError(null);
      await deleteBlock.mutateAsync(confirmDeleteId);
      setConfirmDeleteId(null);
    } catch (err) {
      setConfirmDeleteId(null);
      setOperationError(err instanceof Error ? err.message : '刪除失敗，請稍後再試');
    }
  };

  // L1 fix: close delete confirmation on Escape key
  const handleDeleteEscape = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape' && confirmDeleteId) setConfirmDeleteId(null);
    },
    [confirmDeleteId]
  );
  useEffect(() => {
    if (confirmDeleteId) {
      document.addEventListener('keydown', handleDeleteEscape);
      return () => document.removeEventListener('keydown', handleDeleteEscape);
    }
  }, [confirmDeleteId, handleDeleteEscape]);

  return (
    <div className="space-y-6" data-testid="explore-blocks-settings">
      <header className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-[var(--text-primary)]">自訂首頁區塊</h2>
          <p className="mt-1 text-sm text-[var(--text-secondary)]">
            管理首頁上的探索區塊。每個區塊會依條件從 TMDb 拉取推薦內容。
          </p>
        </div>
        <button
          type="button"
          onClick={() => setModalMode({ type: 'create' })}
          data-testid="explore-blocks-add-button"
          className="flex items-center gap-2 rounded-md bg-[var(--accent-primary)] px-3 py-2 text-sm font-medium text-white hover:bg-[var(--accent-pressed)]"
        >
          <Plus className="h-4 w-4" />
          新增區塊
        </button>
      </header>

      {isLoading && (
        <p className="text-sm text-[var(--text-muted)]" data-testid="explore-blocks-loading">
          載入中...
        </p>
      )}

      {isError && (
        <p className="text-sm text-[var(--error)]" role="alert">
          無法載入區塊列表，請稍後再試。
        </p>
      )}

      {operationError && (
        <div
          role="alert"
          data-testid="explore-blocks-operation-error"
          className="rounded-lg bg-red-900/30 px-3 py-2 text-sm text-[var(--error)]"
        >
          {operationError}
        </div>
      )}

      {!isLoading && !isError && blocks.length === 0 && (
        <p className="text-sm text-[var(--text-muted)]" data-testid="explore-blocks-empty">
          尚未建立任何區塊。點擊「新增區塊」開始自訂首頁。
        </p>
      )}

      <ul className="space-y-2">
        {blocks.map((block, index) => (
          <li
            key={block.id}
            data-testid={`explore-block-row-${block.id}`}
            className="flex items-center justify-between rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-4"
          >
            <div className="min-w-0 flex-1">
              <h3 className="truncate text-sm font-medium text-[var(--text-primary)]">
                {block.name}
              </h3>
              <p className="mt-0.5 text-xs text-[var(--text-muted)]">
                {block.contentType === 'movie' ? '🎬 電影' : '📺 影集'} · {block.maxItems} 個項目
                {block.genreIds && ` · 類型 ${block.genreIds}`}
                {block.region && ` · 地區 ${block.region}`}
              </p>
            </div>

            <div className="flex items-center gap-1 pl-3">
              <button
                type="button"
                onClick={() => handleMove(index, 'up')}
                disabled={index === 0 || reorderBlocks.isPending}
                aria-label={`上移 ${block.name}`}
                data-testid={`explore-block-move-up-${block.id}`}
                className="rounded p-1.5 text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)] disabled:opacity-30"
              >
                <ArrowUp className="h-4 w-4" />
              </button>
              <button
                type="button"
                onClick={() => handleMove(index, 'down')}
                disabled={index === blocks.length - 1 || reorderBlocks.isPending}
                aria-label={`下移 ${block.name}`}
                data-testid={`explore-block-move-down-${block.id}`}
                className="rounded p-1.5 text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)] disabled:opacity-30"
              >
                <ArrowDown className="h-4 w-4" />
              </button>
              <button
                type="button"
                onClick={() => setModalMode({ type: 'edit', block })}
                aria-label={`編輯 ${block.name}`}
                data-testid={`explore-block-edit-${block.id}`}
                className="rounded p-1.5 text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]"
              >
                <Pencil className="h-4 w-4" />
              </button>
              <button
                type="button"
                onClick={() => setConfirmDeleteId(block.id)}
                aria-label={`刪除 ${block.name}`}
                data-testid={`explore-block-delete-${block.id}`}
                className="rounded p-1.5 text-[var(--text-secondary)] hover:bg-red-900/30 hover:text-[var(--error)]"
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
          </li>
        ))}
      </ul>

      {modalMode.type !== 'closed' && (
        <ExploreBlockEditModal
          block={modalMode.type === 'edit' ? modalMode.block : undefined}
          onClose={() => setModalMode({ type: 'closed' })}
        />
      )}

      {confirmDeleteId && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
          data-testid="explore-block-delete-confirm"
          role="dialog"
          aria-modal="true"
          aria-labelledby="delete-confirm-title"
        >
          <div className="w-full max-w-sm rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-primary)] p-6 shadow-xl">
            <h3
              id="delete-confirm-title"
              className="text-lg font-semibold text-[var(--text-primary)]"
            >
              確認刪除
            </h3>
            <p className="mt-2 text-sm text-[var(--text-secondary)]">
              刪除後此區塊將從首頁移除。此動作無法復原。
            </p>
            <div className="mt-6 flex justify-end gap-3">
              <button
                type="button"
                onClick={() => setConfirmDeleteId(null)}
                className="rounded-lg border border-[var(--border-subtle)]/50 px-4 py-2 text-sm text-[var(--text-secondary)] hover:bg-[var(--bg-secondary)]"
              >
                取消
              </button>
              <button
                type="button"
                onClick={handleConfirmDelete}
                disabled={deleteBlock.isPending}
                data-testid="explore-block-delete-confirm-button"
                className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
              >
                {deleteBlock.isPending ? '刪除中...' : '確認刪除'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
