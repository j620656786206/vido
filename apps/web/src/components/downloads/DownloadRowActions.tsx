// Design ref: ux-design.pen Screen D3-D-v2 (lCFq2) · D1-D-v2 (cK1KF) · D7-D-v2 (w3ipb)
import { useRef, useState } from 'react';
import * as DropdownMenu from '@radix-ui/react-dropdown-menu';
import {
  Ellipsis,
  File,
  FolderMinus,
  Pause,
  Play,
  RotateCw,
  Trash2,
  TriangleAlert,
  type LucideIcon,
} from 'lucide-react';
import type { Download } from '../../services/downloadService';
import { cn } from '../../lib/utils';
import { Button } from '../ui/Button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogFooter,
  DialogTitle,
  DialogDescription,
  DialogClose,
} from '../ui/Dialog';
import { formatSize } from './formatters';

interface DownloadRowActionsProps {
  download: Download;
  onPause?: (hash: string) => void;
  onResume?: (hash: string) => void;
  onRemove?: (hash: string, deleteFiles: boolean) => void;
  /** Card buttons sit on a filled square (D1-D-v2); table buttons are bare (D7-D-v2). */
  variant?: 'card' | 'table';
}

interface StateAction {
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
function stateActionFor(
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
}: DownloadRowActionsProps) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const moreRef = useRef<HTMLButtonElement>(null);
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

      {(stateAction || onRemove) && (
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
        <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
          {/* Controlled dialog with no DialogTrigger: Radix has nowhere to return focus on close, so a
              keyboard user would land on <body>. Hand focus back to the ⋯ that started the flow. */}
          <DialogContent
            className="max-w-[480px]"
            onCloseAutoFocus={(e) => {
              e.preventDefault();
              moreRef.current?.focus();
            }}
          >
            <DialogHeader>
              {/* A warning about what the button will do, not a report that something broke: no
                  semantic colour here (DESIGN.md §赭, 2026-09-11) — the destructive button carries it. */}
              <div className="flex items-center gap-3">
                <span className="flex size-12 shrink-0 items-center justify-center rounded-[var(--radius-lg)] bg-[var(--bg-tertiary)]">
                  <TriangleAlert className="size-6 text-[var(--text-primary)]" aria-hidden="true" />
                </span>
                <DialogTitle>移除並刪除檔案？</DialogTitle>
              </div>
              <DialogDescription>
                此動作會一併刪除硬碟上的檔案，無法復原。已下載的內容將從儲存空間中永久移除。
              </DialogDescription>
            </DialogHeader>
            <p className="flex items-center gap-2 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-3 py-2.5 text-sm text-[var(--text-secondary)]">
              <File className="size-4 shrink-0 text-[var(--text-muted)]" aria-hidden="true" />
              <span className="min-w-0 truncate">
                {download.name}
                {download.size > 0 && ` · ${formatSize(download.size)}`}
              </span>
            </p>
            <DialogFooter>
              <DialogClose asChild>
                <Button variant="secondary" className="h-11 font-semibold">
                  取消
                </Button>
              </DialogClose>
              <Button
                variant="destructive"
                className="h-11 font-semibold"
                onClick={() => {
                  onRemove(download.hash, true);
                  setConfirmOpen(false);
                }}
              >
                <Trash2 aria-hidden="true" />
                刪除檔案
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}
    </div>
  );
}
