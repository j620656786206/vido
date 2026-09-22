// Design ref: ux-design.pen Screen D8-M-v2 (jDgxJ)
/**
 * The phone actions sheet for one download (dsr-4b-2): what the desktop ⋯ menu holds, as
 * thumb-sized rows, plus a 詳細資訊 entry that only the phone has. Presentational — the page owns
 * which sheet is open. Rows 2 and 3 do their thing and close; 詳細資訊 and 連同檔案刪除 are
 * HANDOFFS (another overlay opens next), so the owner decides when this one goes.
 */
import type { ComponentProps } from 'react';
import { FolderMinus, Info, Trash2 } from 'lucide-react';
import { cn } from '../../lib/utils';
import type { Download } from '../../services/downloadService';
import { Sheet } from '../ui/Sheet';
import { getDownloadStatus, getDownloadTone } from './downloadStatus';
import { stateActionFor } from './DownloadRowActions';
import { formatProgress } from './formatters';

interface DownloadActionsSheetProps {
  download: Download;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  finalFocus?: ComponentProps<typeof Sheet>['finalFocus'];
  /** Fires once the open/close transition has finished — the owner empties its slot on close. */
  onOpenChangeComplete?: (open: boolean) => void;
  onPause?: (hash: string) => void;
  onResume?: (hash: string) => void;
  onRemove?: (hash: string, deleteFiles: boolean) => void;
  /** When given, the first row opens the detail sheet. */
  onShowDetails?: () => void;
  /** The owner closes this sheet and opens the confirm dialog — nothing is removed here. */
  onRequestDeleteWithFiles: () => void;
}

const ROW =
  'flex min-h-[52px] w-full items-center gap-3 rounded-[var(--radius-md)] px-3 text-left text-base font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-tertiary)] [&_svg]:size-5 [&_svg]:shrink-0';

export function DownloadActionsSheet({
  download,
  open,
  onOpenChange,
  finalFocus,
  onOpenChangeComplete,
  onPause,
  onResume,
  onRemove,
  onShowDetails,
  onRequestDeleteWithFiles,
}: DownloadActionsSheetProps) {
  const stateAction = stateActionFor(download, onPause, onResume);
  const status = getDownloadStatus(download.status);
  const tone = getDownloadTone(download);

  return (
    <Sheet
      open={open}
      onOpenChange={onOpenChange}
      title={download.name}
      testId="download-actions-sheet"
      titleClassName="mb-1 px-5 line-clamp-2"
      className="p-0 pt-2 pb-[max(1.25rem,env(safe-area-inset-bottom))]"
      finalFocus={finalFocus}
      onOpenChangeComplete={onOpenChangeComplete}
    >
      <p className={cn('px-5 pb-3 text-sm font-medium', tone.text)}>
        {status.label} · {formatProgress(download.progress)}
      </p>
      <div aria-hidden="true" className="h-px bg-[var(--border-subtle)]" />
      <div className="flex flex-col gap-0.5 px-2 py-1">
        {onShowDetails && (
          <button type="button" onClick={onShowDetails} className={ROW}>
            <Info aria-hidden="true" className="text-[var(--text-secondary)]" />
            詳細資訊
          </button>
        )}
        {stateAction && (
          <button
            type="button"
            onClick={() => {
              stateAction.run();
              onOpenChange(false);
            }}
            className={ROW}
          >
            <stateAction.icon aria-hidden="true" className="text-[var(--text-secondary)]" />
            {stateAction.label}
          </button>
        )}
        {onRemove && (
          <button
            type="button"
            onClick={() => {
              onRemove(download.hash, false);
              onOpenChange(false);
            }}
            className={ROW}
          >
            <FolderMinus aria-hidden="true" className="text-[var(--text-secondary)]" />
            移除（保留檔案）
          </button>
        )}
        <div aria-hidden="true" className="my-1 h-px bg-[var(--border-subtle)]" />
        <button
          type="button"
          onClick={onRequestDeleteWithFiles}
          className={cn(ROW, 'text-[var(--error-text)] [&_svg]:text-[var(--error-text)]')}
        >
          <Trash2 aria-hidden="true" />
          移除（連同檔案刪除）
        </button>
      </div>
    </Sheet>
  );
}
