// Design ref: ux-design.pen Screen D3-D-v2 (lCFq2) · D1-D-v2 (cK1KF) · D7-D-v2 (w3ipb) · D8-M-v2 (jDgxJ)
import { useEffect, useRef, useState } from 'react';
import * as DropdownMenu from '@radix-ui/react-dropdown-menu';
import {
  Ellipsis,
  FolderMinus,
  Pause,
  Play,
  RotateCw,
  Trash2,
  type LucideIcon,
} from 'lucide-react';
import type { Download } from '../../services/downloadService';
import { cn } from '../../lib/utils';
import { useIsPhone } from '../../hooks/useIsPhone';
import { DeleteWithFilesDialog } from './DeleteWithFilesDialog';

interface DownloadRowActionsProps {
  download: Download;
  onPause?: (hash: string) => void;
  onResume?: (hash: string) => void;
  onRemove?: (hash: string, deleteFiles: boolean) => void;
  /** Card buttons sit on a filled square (D1-D-v2); table buttons are bare (D7-D-v2). */
  variant?: 'card' | 'table';
  /**
   * Phone (dsr-4b-2 D8-M-v2): when given, on a phone the card's ⋯ reports the tap upward instead
   * of opening the dropdown — the page owns the actions sheet. Desktop, the table, and a card
   * rendered without it keep the dropdown, which works at any width.
   */
  onOpenActions?: (hash: string, trigger: HTMLElement) => void;
}

export interface StateAction {
  label: string;
  icon: LucideIcon;
  tone?: 'error';
  run: () => void;
}

/**
 * The one state-dependent action a row shows next to its ⋯ menu. A completed torrent has none.
 * 重試 on an errored torrent is qBittorrent's resume — the same endpoint, named for what it does
 * from the user's side.
 */
export function stateActionFor(
  download: Download,
  onPause?: (hash: string) => void,
  onResume?: (hash: string) => void
): StateAction | null {
  switch (download.status) {
    case 'completed':
      return null;
    case 'paused':
      return onResume ? { label: '繼續', icon: Play, run: () => onResume(download.hash) } : null;
    case 'error':
      return onResume
        ? { label: '重試', icon: RotateCw, tone: 'error', run: () => onResume(download.hash) }
        : null;
    default:
      return onPause ? { label: '暫停', icon: Pause, run: () => onPause(download.hash) } : null;
  }
}

const MENU_ITEM =
  'flex min-h-11 cursor-default select-none items-center gap-3 rounded-[var(--radius-sm)] px-3 text-sm font-medium text-[var(--text-primary)] outline-none data-[highlighted]:bg-[var(--accent-subtle)] [&_svg]:size-[18px] [&_svg]:shrink-0';

/**
 * The action cluster shared by DownloadCardV2 (list) and DownloadsTableV2 (table), so the two
 * views cannot fork the affordances (ux3-4-4 AC5/AC7): one state button + a ⋯ menu with the full
 * set (D3-D-v2). Only 移除（連同檔案刪除）asks for confirmation — it is the one action that cannot be
 * undone; 移除（保留檔案）leaves the files on disk. Renders nothing if no handlers are provided.
 */
export function DownloadRowActions({
  download,
  onPause,
  onResume,
  onRemove,
  variant = 'card',
  onOpenActions,
}: DownloadRowActionsProps) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const moreRef = useRef<HTMLButtonElement>(null);
  const isPhone = useIsPhone();
  const useSheet = isPhone && variant === 'card' && !!onOpenActions;
  // Crossing below `sm` with the desktop confirm open unmounts the dropdown, not the confirm:
  // forget the answer, or a resize back up would re-open「移除並刪除檔案？」on its own.
  useEffect(() => {
    if (useSheet) setConfirmOpen(false);
  }, [useSheet]);
  if (!onPause && !onResume && !onRemove) return null;

  const stateAction = stateActionFor(download, onPause, onResume);
  const square = cn(
    'flex size-11 shrink-0 items-center justify-center rounded-[var(--radius-md)] text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)] active:bg-[var(--bg-tertiary)]',
    variant === 'card'
      ? 'bg-[var(--bg-tertiary)] hover:bg-[var(--border-subtle)]'
      : 'hover:bg-[var(--bg-tertiary)]'
  );

  return (
    <div className="flex shrink-0 items-center gap-1">
      {stateAction && (
        <button
          type="button"
          onClick={stateAction.run}
          aria-label={`${stateAction.label} ${download.name}`}
          className={cn(
            square,
            stateAction.tone === 'error' &&
              'text-[var(--error-text)] hover:text-[var(--error-text)]'
          )}
        >
          <stateAction.icon className="size-[18px]" aria-hidden="true" />
        </button>
      )}

      {(stateAction || onRemove) && useSheet && (
        <button
          type="button"
          ref={moreRef}
          aria-label={`更多動作：${download.name}`}
          aria-haspopup="dialog"
          onClick={(e) => onOpenActions(download.hash, e.currentTarget)}
          className={square}
        >
          <Ellipsis className="size-[18px]" aria-hidden="true" />
        </button>
      )}

      {(stateAction || onRemove) && !useSheet && (
        // modal={false}: a modal menu leaves `pointer-events: none` on <body> for a frame while
        // the confirm dialog mounts, which can swallow the first click inside that dialog.
        <DropdownMenu.Root modal={false}>
          <DropdownMenu.Trigger asChild>
            <button
              type="button"
              ref={moreRef}
              aria-label={`更多動作：${download.name}`}
              className={square}
            >
              <Ellipsis className="size-[18px]" aria-hidden="true" />
            </button>
          </DropdownMenu.Trigger>
          <DropdownMenu.Portal>
            <DropdownMenu.Content
              align="end"
              sideOffset={4}
              className="z-50 min-w-64 rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-tertiary)] p-1.5 shadow-[var(--shadow-lg)]"
            >
              {stateAction && (
                <DropdownMenu.Item
                  onSelect={stateAction.run}
                  className={cn(MENU_ITEM, '[&_svg]:text-[var(--text-secondary)]')}
                >
                  <stateAction.icon aria-hidden="true" />
                  {stateAction.label}
                </DropdownMenu.Item>
              )}
              {onRemove && (
                <>
                  <DropdownMenu.Item
                    onSelect={() => onRemove(download.hash, false)}
                    className={cn(MENU_ITEM, '[&_svg]:text-[var(--text-secondary)]')}
                  >
                    <FolderMinus aria-hidden="true" />
                    移除（保留檔案）
                  </DropdownMenu.Item>
                  <DropdownMenu.Separator className="my-1 h-px bg-[var(--border-subtle)]" />
                  <DropdownMenu.Item
                    onSelect={() => setConfirmOpen(true)}
                    className={cn(
                      MENU_ITEM,
                      'text-[var(--error-text)] [&_svg]:text-[var(--error-text)]'
                    )}
                  >
                    <Trash2 aria-hidden="true" />
                    移除（連同檔案刪除）
                  </DropdownMenu.Item>
                </>
              )}
            </DropdownMenu.Content>
          </DropdownMenu.Portal>
        </DropdownMenu.Root>
      )}

      {onRemove && (
        <DeleteWithFilesDialog
          download={download}
          open={confirmOpen}
          onOpenChange={setConfirmOpen}
          onConfirm={(hash) => onRemove(hash, true)}
          // Controlled dialog with no DialogTrigger: Radix has nowhere to return focus on close, so a
          // keyboard user would land on <body>. Hand focus back to the ⋯ that started the flow.
          onCloseAutoFocus={(e) => {
            e.preventDefault();
            moreRef.current?.focus();
          }}
        />
      )}
    </div>
  );
}
