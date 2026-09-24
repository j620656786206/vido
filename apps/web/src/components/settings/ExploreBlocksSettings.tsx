// Design ref: ux-design.pen Screen C10-D (wnmGh) · C10-M (ZjsVs) · H9-SPEC (Y5XvRv) · H3 (Paqlk)
/**
 * Settings → 自訂首頁 — Story 10.3 management UI.
 *
 * H9-SPEC (Y5XvRv, renamed from H5-D by dsr-7), section C PhBJ8 — bugfix-10-6
 * polish (lucide content-type icons in place of 🎬/📺 emoji).
 *
 * dsr-3d: C10 used to draw a per-block on/off switch and drag handles; the
 * product has neither (ExploreBlock has no enabled field; order is 上移／下移)
 * — the design now draws what is here (disc-2026-09-explore-block-toggle-and-drag).
 * Genres still print as TMDb IDs (disc-2026-09-explore-block-genre-ids-raw).
 */

import { useCallback, useEffect, useState } from 'react';
import { Plus, Pencil, Trash2, ArrowUp, ArrowDown, Film, Tv } from 'lucide-react';
import {
  useExploreBlocks,
  useDeleteExploreBlock,
  useReorderExploreBlocks,
} from '../../hooks/useExploreBlocks';
import type { ExploreBlock } from '../../services/exploreBlockService';
import { sortLabel } from './exploreBlockSort';
import { ExploreBlockEditModal } from './ExploreBlockEditModal';

/** C10 rRMJl: 32 solid squares on desktop, 44 on a phone (touch). */
const ACTION_BTN =
  'flex size-11 items-center justify-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)] disabled:opacity-30 sm:size-8';

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
    <div className="space-y-4" data-testid="explore-blocks-settings">
      {/* Title/description live at the route level; this row is the count + the action. */}
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm text-[var(--text-secondary)]" data-testid="explore-blocks-count">
          {blocks.length} 個區塊
        </p>
        <button
          type="button"
          onClick={() => setModalMode({ type: 'create' })}
          data-testid="explore-blocks-add-button"
          className="flex h-11 items-center gap-2 rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-4 text-sm font-semibold text-[var(--text-on-accent)] hover:bg-[var(--accent-pressed)] sm:h-10"
        >
          <Plus className="h-4 w-4" />
          新增區塊
        </button>
      </div>

      {isLoading && (
        <p className="text-sm text-[var(--text-muted)]" data-testid="explore-blocks-loading">
          載入中...
        </p>
      )}

      {isError && (
        <p className="text-sm text-[var(--error-text)]" role="alert">
          無法載入區塊列表，請稍後再試。
        </p>
      )}

      {operationError && (
        <div
          role="alert"
          data-testid="explore-blocks-operation-error"
          className="rounded-lg bg-[var(--error-tint)] px-3 py-2 text-sm text-[var(--error-text)]"
        >
          {operationError}
        </div>
      )}

      {!isLoading && !isError && blocks.length === 0 && (
        <p className="text-sm text-[var(--text-muted)]" data-testid="explore-blocks-empty">
          尚未建立任何區塊。點擊「新增區塊」開始自訂首頁。
        </p>
      )}

      <ul className="space-y-3">
        {blocks.map((block, index) => {
          const TypeIcon = block.contentType === 'movie' ? Film : Tv;
          // 電影 · 熱門度（高→低） · 20 部 · zh-TW · 地區 TW · 類型 16
          const parts = [
            block.contentType === 'movie' ? '電影' : '影集',
            sortLabel(block.sortBy),
            `${block.maxItems} 部`,
            block.language,
            block.region && `地區 ${block.region}`,
            block.genreIds && `類型 ${block.genreIds}`,
          ].filter(Boolean);
          return (
            <li
              key={block.id}
              data-testid={`explore-block-row-${block.id}`}
              // Phone: the four 44px buttons take their own line (C10-M), so
              // the name is not squeezed to ~100px beside them.
              className="flex flex-wrap items-center gap-x-4 gap-y-3 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-4 sm:flex-nowrap"
            >
              <TypeIcon
                className="size-[18px] shrink-0 text-[var(--text-secondary)]"
                aria-hidden="true"
                data-testid={`explore-block-type-icon-${block.id}`}
              />
              <div className="min-w-0 flex-1">
                <h3 className="truncate text-sm font-semibold text-[var(--text-primary)]">
                  {block.name}
                </h3>
                <p
                  className="mt-0.5 text-xs text-[var(--text-muted)]"
                  data-testid={`explore-block-desc-${block.id}`}
                >
                  {parts.join(' · ')}
                </p>
              </div>

              <div className="flex basis-full items-center justify-end gap-1 sm:basis-auto sm:gap-2">
                <button
                  type="button"
                  onClick={() => handleMove(index, 'up')}
                  disabled={index === 0 || reorderBlocks.isPending}
                  aria-label={`上移 ${block.name}`}
                  data-testid={`explore-block-move-up-${block.id}`}
                  className={ACTION_BTN}
                >
                  <ArrowUp className="size-3.5" aria-hidden="true" />
                </button>
                <button
                  type="button"
                  onClick={() => handleMove(index, 'down')}
                  disabled={index === blocks.length - 1 || reorderBlocks.isPending}
                  aria-label={`下移 ${block.name}`}
                  data-testid={`explore-block-move-down-${block.id}`}
                  className={ACTION_BTN}
                >
                  <ArrowDown className="size-3.5" aria-hidden="true" />
                </button>
                <button
                  type="button"
                  onClick={() => setModalMode({ type: 'edit', block })}
                  aria-label={`編輯 ${block.name}`}
                  data-testid={`explore-block-edit-${block.id}`}
                  className={ACTION_BTN}
                >
                  <Pencil className="size-3.5" aria-hidden="true" />
                </button>
                <button
                  type="button"
                  onClick={() => setConfirmDeleteId(block.id)}
                  aria-label={`刪除 ${block.name}`}
                  data-testid={`explore-block-delete-${block.id}`}
                  className={`${ACTION_BTN} hover:bg-[var(--error-tint)] hover:text-[var(--error-text)]`}
                >
                  <Trash2 className="size-3.5" aria-hidden="true" />
                </button>
              </div>
            </li>
          );
        })}
      </ul>

      {blocks.length > 0 && (
        <p className="text-xs text-[var(--text-muted)]" data-testid="explore-blocks-owned-note">
          已擁有的作品不會出現在首頁。
        </p>
      )}

      {modalMode.type !== 'closed' && (
        <ExploreBlockEditModal
          block={modalMode.type === 'edit' ? modalMode.block : undefined}
          onClose={() => setModalMode({ type: 'closed' })}
        />
      )}

      {confirmDeleteId && (
        /* --overlay-scrim is the modal-backdrop token and stays DARK in both
           themes: a paper modal on paper ground needs the same boundary a dark
           one does. Was black/60; the token is 70%. */
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-[var(--overlay-scrim)]"
          data-testid="explore-block-delete-confirm"
          role="dialog"
          aria-modal="true"
          aria-labelledby="delete-confirm-title"
        >
          <div className="w-full max-w-sm rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-primary)] p-6 shadow-[var(--shadow-xl)]">
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
                className="rounded-lg bg-[var(--error)] px-4 py-2 text-sm font-medium text-[var(--text-on-scrim)] hover:bg-[var(--error-pressed)] disabled:opacity-50"
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
