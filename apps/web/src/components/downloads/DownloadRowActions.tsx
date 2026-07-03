// Design ref: ux-design.pen Screen D1-D-v2 (cK1KF)
import { Pause, Play, Trash2 } from 'lucide-react';
import type { Download } from '../../services/downloadService';
import { Button } from '../ui/Button';
import {
  Dialog,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogFooter,
  DialogTitle,
  DialogDescription,
  DialogClose,
} from '../ui/Dialog';

interface DownloadRowActionsProps {
  download: Download;
  onPause?: (hash: string) => void;
  onResume?: (hash: string) => void;
  onRemove?: (hash: string, deleteFiles: boolean) => void;
}

/**
 * The shared pause/resume + remove action cluster used by BOTH DownloadCardV2 (list) and
 * DownloadsTableV2 (table) — extracted so the two views don't fork the action affordances or the
 * destructive-remove confirm (ux3-4-4 AC5/AC7). The remove is gated behind a Radix Dialog
 * (focus-trap + Escape + aria-modal for free). Renders nothing if no handlers are provided.
 */
export function DownloadRowActions({
  download,
  onPause,
  onResume,
  onRemove,
}: DownloadRowActionsProps) {
  if (!onPause && !onResume && !onRemove) return null;

  return (
    <div className="flex shrink-0 items-center gap-1">
      {download.status === 'paused'
        ? onResume && (
            <Button
              size="icon"
              variant="ghost"
              aria-label={`繼續 ${download.name}`}
              onClick={() => onResume(download.hash)}
            >
              <Play className="size-4" />
            </Button>
          )
        : onPause && (
            <Button
              size="icon"
              variant="ghost"
              aria-label={`暫停 ${download.name}`}
              onClick={() => onPause(download.hash)}
            >
              <Pause className="size-4" />
            </Button>
          )}

      {onRemove && (
        <Dialog>
          <DialogTrigger asChild>
            <Button size="icon" variant="ghost" aria-label={`移除 ${download.name}`}>
              <Trash2 className="size-4 text-[var(--error-text)]" />
            </Button>
          </DialogTrigger>
          <DialogContent aria-describedby={undefined}>
            <DialogHeader>
              <DialogTitle>移除下載</DialogTitle>
              <DialogDescription>
                「{download.name}」— 保留檔案只從 qBittorrent
                移除任務；連同檔案刪除會一併刪除已下載的檔案，無法復原。
              </DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <DialogClose asChild>
                <Button variant="outline" onClick={() => onRemove(download.hash, false)}>
                  移除（保留檔案）
                </Button>
              </DialogClose>
              <DialogClose asChild>
                <Button variant="destructive" onClick={() => onRemove(download.hash, true)}>
                  移除（連同檔案刪除）
                </Button>
              </DialogClose>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}
    </div>
  );
}
