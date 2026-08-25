// Design ref: ux-design.pen Screen 11 Backup Management Desktop (uhAKd)
import { useState } from 'react';
import { AlertTriangle, Check, Loader2, Plus, XCircle } from 'lucide-react';
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
import { MetadataExport } from './MetadataExport';
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
  const { data, isLoading, error } = useBackups();
  const createBackup = useCreateBackup();
  const deleteBackup = useDeleteBackup();
  const verifyBackup = useVerifyBackup();
  const restoreBackup = useRestoreBackup();
  const [createError, setCreateError] = useState<string | null>(null);
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
    setCreateError(null);
    try {
      await createBackup.mutateAsync();
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : '建立備份失敗');
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

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20" data-testid="backup-loading">
        <Loader2 className="h-8 w-8 animate-spin text-[var(--text-secondary)]" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="py-10 text-center" data-testid="backup-error">
        <p className="text-[var(--error-text)]">無法載入備份資料</p>
        <p className="mt-1 text-sm text-[var(--text-muted)]">{error.message}</p>
      </div>
    );
  }

  const backups = data?.backups ?? [];
  const totalSize = data?.totalSizeBytes ?? 0;

  return (
    <div className="space-y-6" data-testid="backup-management">
      {/* Action bar */}
      <div className="flex items-center justify-between">
        <div className="space-y-1">
          <p className="text-sm text-[var(--text-secondary)]" data-testid="backup-summary">
            已使用 {formatBytes(totalSize)}（{backups.length} 個備份）
          </p>
        </div>
        <button
          onClick={handleCreate}
          disabled={createBackup.isPending}
          className="flex items-center gap-2 rounded-lg bg-[var(--accent-primary)] px-4 py-2 text-sm font-medium text-[var(--text-on-accent)] transition-colors hover:bg-[var(--accent-hover)] disabled:opacity-50"
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

      {/* Error display */}
      {createError && (
        <div
          className="rounded-lg bg-[var(--error-tint)] px-4 py-3 text-sm text-[var(--error-text)]"
          role="alert"
          data-testid="create-error"
        >
          {createError}
        </div>
      )}

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

      {/* Metadata export */}
      <MetadataExport />

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
