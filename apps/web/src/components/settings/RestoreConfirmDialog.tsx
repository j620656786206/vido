// Design ref: ux-design.pen Screen C19-D (G8BYO) · C19-M (gPZU6)
/**
 * Restore overwrites ALL of Vido's data, so this is the confirmation that most
 * needs to behave like a real dialog. It used to be a hand-rolled `fixed` div:
 * no role="dialog", no focus trap, Esc did nothing. Now it is ui/Dialog — the
 * same Radix primitive every other confirmation uses — and on a phone the same
 * content slides up as a bottom sheet (C19-M), following ConfirmGenerationDialog.
 */
import { useRef, useState } from 'react';
import { Info, Loader2, RotateCcw } from 'lucide-react';
import type { Backup } from '../../services/backupService';
import { Dialog, DialogContent, DialogDescription, DialogTitle } from '../ui/Dialog';
import { MOBILE_SHEET_CLOSE, MOBILE_SHEET_CONTENT, SheetGrabber } from '../ui/mobileSheet';
import { useIsPhone } from '../../hooks/useIsPhone';
import { formatBytes } from '../../utils/formatBytes';
import { formatLocalDateTime } from '../../utils/formatLocalDateTime';
import { cn } from '../../lib/utils';

interface RestoreConfirmDialogProps {
  backup: Backup;
  isRestoring: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function RestoreConfirmDialog({
  backup,
  isRestoring,
  onConfirm,
  onCancel,
}: RestoreConfirmDialogProps) {
  const isPhone = useIsPhone();
  const cancelRef = useRef<HTMLButtonElement>(null);
  // Opened by a row's 還原 button, not by a Radix Trigger, so Radix has nothing
  // to hand focus back to and drops it on <body>. Remember what had focus when
  // the dialog mounted (the 還原 button) and return there on close.
  const [returnFocusTo] = useState(() =>
    typeof document === 'undefined' ? null : (document.activeElement as HTMLElement | null)
  );

  const cancel = (
    <button
      ref={cancelRef}
      type="button"
      onClick={() => {
        if (!isRestoring) onCancel();
      }}
      // aria-disabled, not disabled: disabling the focused 取消 would blur it
      // and leave a keyboard user on <body> inside the modal (dsr-3f CR #4).
      aria-disabled={isRestoring || undefined}
      className="h-11 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-5 text-sm font-semibold text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)] aria-disabled:cursor-not-allowed aria-disabled:opacity-50 max-sm:h-12 max-sm:w-full"
      data-testid="restore-cancel-btn"
    >
      取消
    </button>
  );
  const confirm = (
    <button
      type="button"
      onClick={() => {
        if (!isRestoring) onConfirm();
      }}
      aria-disabled={isRestoring || undefined}
      className="flex h-11 items-center justify-center gap-2 rounded-[var(--radius-md)] bg-[var(--error)] px-5 text-sm font-semibold text-[var(--text-on-scrim)] transition-colors hover:bg-[var(--error-pressed)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)] aria-disabled:cursor-not-allowed aria-disabled:opacity-50 max-sm:h-12 max-sm:w-full"
      data-testid="restore-confirm-btn"
    >
      {isRestoring ? (
        <>
          <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
          還原中...
        </>
      ) : (
        '確認還原'
      )}
    </button>
  );

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        // A restore in flight cannot be backed out of; Esc / ✕ / the scrim wait.
        if (!open && !isRestoring) onCancel();
      }}
    >
      <DialogContent
        data-testid="restore-confirm-dialog"
        // The safe answer takes focus, never the destructive one.
        onOpenAutoFocus={(e) => {
          e.preventDefault();
          cancelRef.current?.focus();
        }}
        onCloseAutoFocus={(e) => {
          e.preventDefault();
          // The saved button can be gone if the list switched between table
          // and cards while the dialog was open (a phone rotated) — find the
          // same backup's 還原 in whichever layout is showing now.
          const target = returnFocusTo?.isConnected
            ? returnFocusTo
            : document.querySelector<HTMLElement>(`[data-testid="restore-btn-${backup.id}"]`);
          target?.focus();
        }}
        // top-[38px]: centred on the 40px title row that sits under the grabber.
        // Hidden while restoring — it could not close the dialog anyway.
        closeClassName={cn(MOBILE_SHEET_CLOSE, 'max-sm:top-[38px]', isRestoring && 'hidden')}
        className={cn(
          'flex flex-col gap-4 border border-[var(--border-subtle)] p-6 max-sm:px-6 max-sm:pb-8 max-sm:pt-2',
          MOBILE_SHEET_CONTENT,
          'sm:bottom-auto sm:left-1/2 sm:right-auto sm:top-1/2 sm:w-full sm:max-w-[496px] sm:-translate-x-1/2 sm:-translate-y-1/2 sm:rounded-[var(--radius-lg)]'
        )}
      >
        {isPhone && <SheetGrabber data-testid="restore-sheet-grabber" />}

        <div className="flex items-center gap-3 pr-10">
          <span className="flex size-10 shrink-0 items-center justify-center rounded-full bg-[var(--error-tint)]">
            <RotateCcw className="size-5 text-[var(--error-text)]" aria-hidden="true" />
          </span>
          <DialogTitle asChild>
            <h3 className="text-lg font-bold text-[var(--text-primary)] sm:text-xl">確認還原</h3>
          </DialogTitle>
        </div>

        <DialogDescription className="text-sm text-[var(--text-secondary)]">
          即將從以下備份還原資料，
          {/* Kept deliberately (C19 now draws it too): in a confirmation that
              overwrites everything, the consequence is the sentence to see. */}
          <strong className="text-[var(--text-primary)]">這將會取代目前所有的資料</strong>。
        </DialogDescription>

        <div
          className="space-y-0.5 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] p-3 font-mono text-xs"
          data-testid="restore-file"
        >
          <p className="break-all text-[var(--text-primary)]" data-testid="restore-filename">
            {backup.filename}
          </p>
          <p className="text-[var(--text-muted)]" data-testid="restore-file-meta">
            {formatBytes(backup.sizeBytes)} · {formatLocalDateTime(backup.createdAt)}
          </p>
        </div>

        <p
          className="flex items-start gap-2 rounded-[var(--radius-md)] bg-[var(--info-tint)] p-3 text-xs text-[var(--info-text)]"
          data-testid="restore-snapshot-note"
        >
          <Info className="mt-px size-4 shrink-0" aria-hidden="true" />
          系統會在還原前自動建立目前資料的快照，以便在需要時復原。
        </p>

        {/* Phone (C19-M): stacked, full width, 確認還原 on top within thumb reach.
            Desktop: 取消 then 確認還原, right-aligned. Swapped in the DOM, not with
            CSS order, so tab order matches what is on screen. */}
        <div
          className={cn('flex gap-3', isPhone ? 'flex-col gap-2' : 'justify-end')}
          data-testid="restore-actions"
        >
          {isPhone ? (
            <>
              {confirm}
              {cancel}
            </>
          ) : (
            <>
              {cancel}
              {confirm}
            </>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
