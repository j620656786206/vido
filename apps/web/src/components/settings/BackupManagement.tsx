import { useState } from 'react';
import { HardDrive, Loader2, Plus } from 'lucide-react';
import {
  useBackups,
  useCreateBackup,
  useDeleteBackup,
  useVerifyBackup,
} from '../../hooks/useBackups';
import { BackupTable } from './BackupTable';
import { formatBytes } from '../../utils/formatBytes';

export function BackupManagement() {
  const { data, isLoading, error } = useBackups();
  const createBackup = useCreateBackup();
  const deleteBackup = useDeleteBackup();
  const verifyBackup = useVerifyBackup();
  const [createError, setCreateError] = useState<string | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [verifyMessage, setVerifyMessage] = useState<string | null>(null);

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
        setVerifyMessage('✅ 備份驗證通過，資料完整');
      } else {
        setVerifyMessage('⚠️ 備份校驗碼不符，檔案可能已損壞');
      }
    } catch (err) {
      setVerifyMessage(err instanceof Error ? err.message : '驗證失敗');
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20" data-testid="backup-loading">
        <Loader2 className="h-8 w-8 animate-spin text-slate-400" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="py-10 text-center" data-testid="backup-error">
        <p className="text-red-400">無法載入備份資料</p>
        <p className="mt-1 text-sm text-slate-500">{error.message}</p>
      </div>
    );
  }

  const backups = data?.backups ?? [];
  const totalSize = data?.totalSizeBytes ?? 0;

  return (
    <div className="space-y-6" data-testid="backup-management">
      {/* Header */}
      <div className="flex items-center gap-3">
        <HardDrive className="h-5 w-5 text-slate-400" />
        <div>
          <h2 className="text-lg font-semibold text-slate-200">備份與還原</h2>
          <p className="text-sm text-slate-400">建立與管理 Vido 資料庫備份，確保資料安全</p>
        </div>
      </div>

      {/* Action bar */}
      <div className="flex items-center justify-between">
        <div className="space-y-1">
          <p className="text-sm text-slate-300" data-testid="backup-summary">
            已使用 {formatBytes(totalSize)}（{backups.length} 個備份）
          </p>
        </div>
        <button
          onClick={handleCreate}
          disabled={createBackup.isPending}
          className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-500 disabled:opacity-50"
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
          className="rounded-lg border border-red-800 bg-red-900/20 px-4 py-3 text-sm text-red-400"
          role="alert"
          data-testid="create-error"
        >
          {createError}
        </div>
      )}

      {verifyMessage && (
        <div
          className="rounded-lg border border-blue-800 bg-blue-900/20 px-4 py-3 text-sm text-blue-300"
          role="status"
          data-testid="verify-message"
        >
          {verifyMessage}
        </div>
      )}

      {deleteError && (
        <div
          className="rounded-lg border border-red-800 bg-red-900/20 px-4 py-3 text-sm text-red-400"
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
          isDeleting={deleteBackup.isPending}
          isVerifying={verifyBackup.isPending}
        />
      ) : (
        <div
          className="rounded-lg border border-slate-700 bg-slate-800 py-16 text-center text-sm text-slate-500"
          data-testid="backup-empty"
        >
          尚未建立任何備份
        </div>
      )}
    </div>
  );
}
