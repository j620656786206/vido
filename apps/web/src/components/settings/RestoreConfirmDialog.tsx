import { AlertTriangle, Loader2 } from 'lucide-react';
import type { Backup } from '../../services/backupService';

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
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
      data-testid="restore-confirm-dialog"
    >
      <div className="mx-4 w-full max-w-md rounded-xl border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-6 shadow-2xl">
        <div className="mb-4 flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-amber-500/10">
            <AlertTriangle className="h-5 w-5 text-amber-400" />
          </div>
          <h3 className="text-lg font-semibold text-[var(--text-primary)]">確認還原</h3>
        </div>

        <div className="mb-6 space-y-3">
          <p className="text-sm text-[var(--text-secondary)]">
            即將從以下備份還原資料，
            <strong className="text-amber-400">這將會取代目前所有的資料</strong>。
          </p>
          <div className="rounded-lg bg-[var(--bg-primary)] px-3 py-2 text-xs text-[var(--text-secondary)]">
            <p data-testid="restore-filename">{backup.filename}</p>
          </div>
          <p className="text-xs text-[var(--text-muted)]">
            系統會在還原前自動建立目前資料的快照，以便在需要時復原。
          </p>
        </div>

        <div className="flex justify-end gap-3">
          <button
            onClick={onCancel}
            disabled={isRestoring}
            className="rounded-lg border border-[var(--border-subtle)] px-4 py-2 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] disabled:opacity-50"
            data-testid="restore-cancel-btn"
          >
            取消
          </button>
          <button
            onClick={onConfirm}
            disabled={isRestoring}
            className="flex items-center gap-2 rounded-lg bg-amber-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-amber-500 disabled:opacity-50"
            data-testid="restore-confirm-btn"
          >
            {isRestoring ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                還原中...
              </>
            ) : (
              '確認還原'
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
