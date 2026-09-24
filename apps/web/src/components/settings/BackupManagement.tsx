// Design ref: ux-design.pen Screen C5-D (uhAKd) · C5-M (gEQX4)；建立失敗見 C20-D (v2C4xr)
import { useState } from 'react';
import { Link } from '@tanstack/react-router';
import { AlertTriangle, Check, CircleAlert, HardDrive, Loader2, Plus, XCircle } from 'lucide-react';
import { cn } from '../../lib/utils';
import {
  useBackups,
  useCreateBackup,
  useDeleteBackup,
  useVerifyBackup,
  useRestoreBackup,
} from '../../hooks/useBackups';
import { BackupTable } from './BackupTable';
import { RestoreConfirmDialog } from './RestoreConfirmDialog';
import { BackupScheduleConfig } from './BackupScheduleConfig';
import { SettingsErrorState } from './SettingsErrorState';
import { formatBytes } from '../../utils/formatBytes';
import type { Backup } from '../../services/backupService';

/** 固定詞彙: ok = done-ness = NEUTRAL; warn = 你要求了但沒發生; error = 壞掉了. */
const TONE_CLASSES = {
  ok: 'bg-[var(--bg-tertiary)] text-[var(--text-secondary)]',
  warn: 'bg-[var(--warning-tint)] text-[var(--warning-text)]',
  error: 'bg-[var(--error-tint)] text-[var(--error-text)]',
} as const;

const TONE_ICONS = {
  ok: <Check className="h-4 w-4 shrink-0" aria-hidden="true" />,
  warn: <AlertTriangle className="h-4 w-4 shrink-0" aria-hidden="true" />,
  error: <XCircle className="h-4 w-4 shrink-0" aria-hidden="true" />,
} as const;

export function BackupManagement() {
  const { data, isFetched, isFetching, error, refetch } = useBackups();
  const createBackup = useCreateBackup();
  const deleteBackup = useDeleteBackup();
  const verifyBackup = useVerifyBackup();
  const restoreBackup = useRestoreBackup();
  // Only whether it failed: the backend neither pre-checks disk space nor
  // classifies the error, so its message (English) says nothing a user can act on.
  const [createFailed, setCreateFailed] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  // Feedback carries its OUTCOME so the banner can wear the vocabulary's colour:
  // ok → neutral (done-ness never wears green/gold), warn → --warning-*, error →
  // --error-*. A corruption warning in gold and a success in amber were both
  // lying in the product's own colour language at its two highest-stakes moments.
  type Feedback = { tone: 'ok' | 'warn' | 'error'; text: string };
  const [verifyMessage, setVerifyMessage] = useState<Feedback | null>(null);
  const [restoreMessage, setRestoreMessage] = useState<Feedback | null>(null);
  const [restoreTarget, setRestoreTarget] = useState<Backup | null>(null);

  const handleCreate = async () => {
    if (createBackup.isPending) return;
    // The banner stays up while 重試 runs (its button keeps focus) and goes
    // away only once a backup is actually made.
    try {
      await createBackup.mutateAsync();
      setCreateFailed(false);
    } catch {
      setCreateFailed(true);
    }
  };

  const handleDelete = async (id: string) => {
    setDeleteError(null);
    try {
      await deleteBackup.mutateAsync(id);
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : '刪除備份失敗');
    }
  };

  const handleVerify = async (id: string) => {
    setVerifyMessage(null);
    try {
      const result = await verifyBackup.mutateAsync(id);
      if (result.match) {
        setVerifyMessage({ tone: 'ok', text: '備份驗證通過，資料完整' });
      } else {
        setVerifyMessage({ tone: 'warn', text: '備份校驗碼不符，檔案可能已損壞' });
      }
    } catch (err) {
      setVerifyMessage({ tone: 'error', text: err instanceof Error ? err.message : '驗證失敗' });
    }
  };

  const handleRestoreClick = (id: string) => {
    const backup = data?.backups?.find((b) => b.id === id);
    if (backup) {
      setRestoreTarget(backup);
    }
  };

  const handleRestoreConfirm = async () => {
    if (!restoreTarget || restoreBackup.isPending) return;
    setRestoreMessage(null);
    try {
      const result = await restoreBackup.mutateAsync(restoreTarget.id);
      setRestoreTarget(null);
      if (result.status === 'completed') {
        setRestoreMessage({ tone: 'ok', text: '還原完成，資料庫已恢復' });
      } else {
        setRestoreMessage({ tone: 'error', text: `還原失敗：${result.error || '未知錯誤'}` });
      }
    } catch (err) {
      setRestoreTarget(null);
      setRestoreMessage({ tone: 'error', text: err instanceof Error ? err.message : '還原失敗' });
    }
  };

  // Keyed on data, not isLoading: a failed read with nothing cached drops back
  // to pending on every refetch (project_tanstack_refetch_no_data_resets_pending),
  // which would swap the error page for the spinner the moment 重試 is pressed.
  if (!data && !error && !isFetched) {
    return (
      <div className="flex items-center justify-center py-20" data-testid="backup-loading">
        <Loader2 className="h-8 w-8 animate-spin text-[var(--text-secondary)]" />
      </div>
    );
  }

  if (!data) {
    return (
      <SettingsErrorState
        testId="backup-error"
        title="無法載入備份資訊"
        description="與後端的連線中斷了。已存在的備份檔不受影響。"
        isRetrying={isFetching}
        onRetry={() => void refetch()}
      />
    );
  }

  const backups = data?.backups ?? [];
  const totalSize = data?.totalSizeBytes ?? 0;

  return (
    <div className="space-y-6" data-testid="backup-management">
      {/* C20-D: the failure sits ABOVE the summary — it is about the button the
          user just pressed, and the list below is still worth showing. */}
      {createFailed && (
        <CreateBackupFailed onRetry={handleCreate} isRetrying={createBackup.isPending} />
      )}

      {/* Action bar */}
      <div className="flex items-center justify-between gap-3">
        <p
          className="flex min-w-0 items-center gap-2 text-sm text-[var(--text-secondary)]"
          data-testid="backup-summary"
        >
          <HardDrive className="size-4 shrink-0" aria-hidden="true" />
          {backups.length} 份備份 · 已使用 {formatBytes(totalSize)}
        </p>
        <button
          onClick={handleCreate}
          disabled={createBackup.isPending}
          className="flex h-11 shrink-0 items-center gap-2 rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-4 text-sm font-semibold text-[var(--text-on-accent)] transition-colors hover:bg-[var(--accent-hover)] disabled:opacity-50 sm:h-10"
          data-testid="create-backup-btn"
        >
          {createBackup.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Plus className="h-4 w-4" />
          )}
          建立備份
        </button>
      </div>

      {verifyMessage && (
        <div
          className={cn(
            'flex items-center gap-2 rounded-lg px-4 py-3 text-sm',
            TONE_CLASSES[verifyMessage.tone]
          )}
          role={verifyMessage.tone === 'error' ? 'alert' : 'status'}
          data-testid="verify-message"
        >
          {TONE_ICONS[verifyMessage.tone]}
          {verifyMessage.text}
        </div>
      )}

      {restoreMessage && (
        <div
          className={cn(
            'flex items-center gap-2 rounded-lg px-4 py-3 text-sm',
            TONE_CLASSES[restoreMessage.tone]
          )}
          role={restoreMessage.tone === 'error' ? 'alert' : 'status'}
          data-testid="restore-message"
        >
          {TONE_ICONS[restoreMessage.tone]}
          {restoreMessage.text}
        </div>
      )}

      {deleteError && (
        <div
          className="rounded-lg bg-[var(--error-tint)] px-4 py-3 text-sm text-[var(--error-text)]"
          role="alert"
          data-testid="delete-error"
        >
          {deleteError}
        </div>
      )}

      {/* Backup table */}
      {backups.length > 0 ? (
        <BackupTable
          backups={backups}
          onDelete={handleDelete}
          onVerify={handleVerify}
          onRestore={handleRestoreClick}
          isDeleting={deleteBackup.isPending}
          isVerifying={verifyBackup.isPending}
          isRestoring={restoreBackup.isPending}
        />
      ) : (
        <div
          className="rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-secondary)] py-16 text-center text-sm text-[var(--text-muted)]"
          data-testid="backup-empty"
        >
          尚未建立任何備份
        </div>
      )}

      {/* Schedule config */}
      <BackupScheduleConfig />

      {/* Restore confirmation dialog */}
      {restoreTarget && (
        <RestoreConfirmDialog
          backup={restoreTarget}
          isRestoring={restoreBackup.isPending}
          onConfirm={handleRestoreConfirm}
          onCancel={() => setRestoreTarget(null)}
        />
      )}
    </div>
  );
}

/**
 * C20-D. Only says that it failed and where to look: the backend neither
 * pre-checks disk space nor classifies the error, so C20's「磁碟空間不足：需要
 * 52 MB…」is not something this page can know (disc-2026-09-backup-disk-space-precheck).
 */
export function CreateBackupFailed({
  onRetry,
  isRetrying,
}: {
  onRetry: () => void;
  isRetrying: boolean;
}) {
  return (
    <div
      className="flex items-center gap-2 rounded-[var(--radius-md)] bg-[var(--error-tint)] p-4 text-[var(--error-text)]"
      role="alert"
      data-testid="create-error"
    >
      <CircleAlert className="size-[18px] shrink-0" aria-hidden="true" />
      <div className="min-w-0 flex-1">
        <p className="text-sm font-semibold">建立備份失敗</p>
        <p className="mt-0.5 text-xs">
          備份沒有建立成功。請稍後再試；若持續失敗，到「
          <Link to="/settings/logs" className="underline underline-offset-2">
            系統日誌
          </Link>
          」查看原因。
        </p>
      </div>
      <button
        type="button"
        onClick={onRetry}
        aria-disabled={isRetrying || undefined}
        className="h-9 shrink-0 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-3 text-xs font-semibold text-[var(--text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)] aria-disabled:opacity-50 max-sm:h-11"
        data-testid="create-retry-btn"
      >
        重試
      </button>
    </div>
  );
}
