// Design ref: ux-design.pen Screen E1-D (KvZSc) · E1-M (uABWl) · J4-D (sPzZT) · J5-D (alrIw)
/**
 * Library Card component for displaying a media library in Settings (Story 7b-4)
 */

import { useRef, useState } from 'react';
import * as DropdownMenu from '@radix-ui/react-dropdown-menu';
import { Film, Tv, MoreVertical, Trash2, Pencil, FolderOpen } from 'lucide-react';
import { cn } from '../../lib/utils';
import { useDeleteLibrary } from '../../hooks/useMediaLibrary';
import type { MediaLibraryWithPaths } from '../../services/mediaLibraryService';

interface LibraryCardProps {
  library: MediaLibraryWithPaths;
  /**
   * Whether THIS deployment actually runs the free auto-generation lane
   * (9R-10b-M4). `auto_subtitle` is writable in every mode, but the generator
   * that honours it is built only when the API runs in `pipeline` mode — so a
   * library left opted in after a switch back to `legacy` would otherwise keep
   * announcing work nobody is doing.
   *
   * Supplied by MediaLibraryManager from the same `useMediaLibraries()` query
   * that provides the libraries themselves — no extra request, no new hook.
   */
  autoSubtitleSupported: boolean;
  onEdit: () => void;
}

const STATUS_CONFIG = {
  accessible: { color: 'text-[var(--success-text)]', bg: 'bg-[var(--success)]', label: '已連線' },
  not_found: { color: 'text-[var(--error-text)]', bg: 'bg-[var(--error)]', label: '無法存取' },
  not_readable: { color: 'text-[var(--error-text)]', bg: 'bg-[var(--error)]', label: '無法讀取' },
  not_directory: { color: 'text-[var(--error-text)]', bg: 'bg-[var(--error)]', label: '非目錄' },
  unknown: { color: 'text-[var(--text-secondary)]', bg: 'bg-[var(--text-muted)]', label: '未檢查' },
} as const;

export function LibraryCard({ library, autoSubtitleSupported, onEdit }: LibraryCardProps) {
  const [confirmDelete, setConfirmDelete] = useState(false);
  // 編輯 opens a dialog that takes focus itself; the menu handing focus back to
  // ⋮ first would make the dialog's trap re-select 名稱's whole text.
  const openingEditor = useRef(false);
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
      // bg-primary, not bg-secondary: the card sits INSIDE the settings form card,
      // which is already bg-secondary — same ground plus a border read as a
      // box drawn on nothing (E1-D draws it one step darker, like its rows).
      className="rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-primary)] p-4"
      data-testid={`library-card-${library.id}`}
    >
      {/* Header */}
      <div className="mb-3 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <TypeIcon className="h-4 w-4 text-[var(--text-secondary)]" />
          <span className="text-sm font-medium text-[var(--text-primary)]">{library.name}</span>
          <span className="rounded-full bg-[var(--bg-tertiary)] px-2 py-0.5 text-xs text-[var(--text-secondary)]">
            {typeLabel}
          </span>
        </div>
        {/* disc-2026-09-scanner-custom-modals-a11y: a real menu (Esc, a click
            outside, arrow keys). modal={false}: a modal menu leaves
            pointer-events:none on <body> for a frame while 編輯's dialog mounts,
            which can swallow the first click inside it (DownloadRowActions). */}
        {/* The block wrapper keeps the header row's height: an inline button in
            a block box gets the line's strut (E1-D baseline, 2px). */}
        <div>
          <DropdownMenu.Root modal={false}>
            <DropdownMenu.Trigger asChild>
              <button
                type="button"
                className="-m-3.5 rounded-[var(--radius-sm)] p-3.5 text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-secondary)] sm:m-0 sm:p-1"
                aria-label={`${library.name} 的操作`}
                data-testid="library-menu-button"
              >
                <MoreVertical className="h-4 w-4" />
              </button>
            </DropdownMenu.Trigger>
            <DropdownMenu.Portal>
              <DropdownMenu.Content
                align="end"
                sideOffset={4}
                onCloseAutoFocus={(event) => {
                  if (openingEditor.current) {
                    openingEditor.current = false;
                    event.preventDefault();
                  }
                }}
                className="z-50 w-32 rounded-md border border-[var(--border-subtle)] bg-[var(--bg-secondary)] py-1 shadow-[var(--shadow-lg)]"
              >
                <DropdownMenu.Item
                  onSelect={() => {
                    openingEditor.current = true;
                    onEdit();
                  }}
                  className="flex min-h-11 w-full cursor-default items-center gap-2 px-3 py-1.5 text-sm text-[var(--text-secondary)] outline-none hover:bg-[var(--bg-tertiary)] data-[highlighted]:bg-[var(--bg-tertiary)] sm:min-h-0"
                >
                  <Pencil className="h-3 w-3" aria-hidden="true" /> 編輯
                </DropdownMenu.Item>
                <DropdownMenu.Item
                  onSelect={() => setConfirmDelete(true)}
                  className="flex min-h-11 w-full cursor-default items-center gap-2 px-3 py-1.5 text-sm text-[var(--error-text)] outline-none hover:bg-[var(--bg-tertiary)] data-[highlighted]:bg-[var(--bg-tertiary)] sm:min-h-0"
                >
                  <Trash2 className="h-3 w-3" aria-hidden="true" /> 刪除
                </DropdownMenu.Item>
              </DropdownMenu.Content>
            </DropdownMenu.Portal>
          </DropdownMenu.Root>
        </div>
      </div>

      {/* Paths */}
      <div className="mb-3 space-y-1.5">
        {(library.paths || []).map((p) => {
          const statusCfg = STATUS_CONFIG[p.status] || STATUS_CONFIG.unknown;
          return (
            <div key={p.id} className="flex items-center gap-2 text-sm">
              <FolderOpen className="h-3.5 w-3.5 shrink-0 text-[var(--text-muted)]" />
              <span className="flex-1 truncate font-mono text-xs text-[var(--text-secondary)]">
                {p.path}
              </span>
              <span className={cn('flex items-center gap-1 text-xs', statusCfg.color)}>
                <span className={cn('inline-block h-1.5 w-1.5 rounded-full', statusCfg.bg)} />
                {statusCfg.label}
              </span>
            </div>
          );
        })}
      </div>

      {/* Footer — Story 9R-10b. The opt-in state rides the EXISTING dot-separated
          grammar instead of getting a new badge: the path rows above already own a
          coloured-dot status vocabulary, and a second one would compete with it for
          the same glance. Success green is borrowed from the consent flow, where it
          means exactly this: costs nothing. Absent entirely when the library is off.

          9R-10b-M4 (design J5-D block E) — colour here is a RULE, not three
          separate decisions:
            success = it IS happening
            warning = you asked for it, and it is NOT happening
            absent  = you did not ask
          The parenthetical names WHO is not enabled. Dropped, the line reads as
          "you didn't tick the box" — and the user DID tick it, which is the
          worst available misreading. */}
      <div className="text-xs text-[var(--text-muted)]" data-testid="library-card-footer">
        {(library.paths || []).length} 個資料夾 · {library.mediaCount} 個項目
        {library.autoSubtitle && (
          <span
            className={`font-medium ${
              autoSubtitleSupported ? 'text-[var(--success-text)]' : 'text-[var(--warning-text)]'
            }`}
            data-testid="library-card-auto-subtitle-status"
          >
            {autoSubtitleSupported ? ' · 自動處理免費字幕' : ' · 自動處理免費字幕（伺服器未啟用）'}
          </span>
        )}
      </div>

      {/* Delete Confirmation */}
      {confirmDelete && (
        <div className="mt-3 rounded-lg bg-[var(--error-tint)] p-3">
          <p className="mb-2 text-sm text-[var(--error-text)]">確定要刪除「{library.name}」嗎？</p>
          <label className="mb-3 flex items-center gap-2 text-xs text-[var(--text-secondary)]">
            <input
              type="checkbox"
              checked={removeMedia}
              onChange={(e) => setRemoveMedia(e.target.checked)}
              className="rounded border-[var(--border-subtle)]"
            />
            同時移除已掃描的媒體資料
          </label>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => setConfirmDelete(false)}
              className="rounded px-3 py-1 text-xs text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)]"
            >
              取消
            </button>
            <button
              type="button"
              onClick={handleDelete}
              disabled={deleteLibrary.isPending}
              className="rounded bg-[var(--error)] px-3 py-1 text-xs text-[var(--text-on-scrim)] hover:bg-[var(--error-pressed)] disabled:opacity-50"
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
