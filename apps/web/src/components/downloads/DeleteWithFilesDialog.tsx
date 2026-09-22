// Design ref: ux-design.pen Screen D3-D-v2 (lCFq2)
/**
 * The one confirmation in the downloads flow: 移除（連同檔案刪除）cannot be undone, so it asks
 * first. Lifted out of DownloadRowActions (dsr-4b-2) so the desktop ⋯ menu and the phone actions
 * sheet open the SAME dialog — copy, layout and buttons unchanged.
 *
 * `download` is a snapshot object, not a hash: removal is optimistic, so by the time the dialog
 * animates out the row is already gone from the list and a lookup would come back empty.
 */
import { File, Trash2, TriangleAlert } from 'lucide-react';
import type { Download } from '../../services/downloadService';
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

interface DeleteWithFilesDialogProps {
  download: Pick<Download, 'hash' | 'name' | 'size'> | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: (hash: string) => void;
  /** Radix's close-focus hook — controlled dialogs have no trigger to return focus to. */
  onCloseAutoFocus?: (e: Event) => void;
}

export function DeleteWithFilesDialog({
  download,
  open,
  onOpenChange,
  onConfirm,
  onCloseAutoFocus,
}: DeleteWithFilesDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[480px]" onCloseAutoFocus={onCloseAutoFocus}>
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
            {download?.name}
            {download && download.size > 0 && ` · ${formatSize(download.size)}`}
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
              if (download) onConfirm(download.hash);
              onOpenChange(false);
            }}
          >
            <Trash2 aria-hidden="true" />
            刪除檔案
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
