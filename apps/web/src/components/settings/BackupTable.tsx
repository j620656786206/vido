// Design ref: ux-design.pen Screen 11 Backup Management Desktop (uhAKd)
import { Download, Trash2, ShieldCheck, RotateCcw } from 'lucide-react';
import type { Backup, BackupStatus } from '../../services/backupService';
import { backupService } from '../../services/backupService';
import { formatBytes } from '../../utils/formatBytes';

const statusConfig: Record<BackupStatus, { label: string; color: string; bg: string }> = {
  completed: { label: '完成', color: 'text-[var(--success-text)]', bg: 'bg-[var(--success-tint)]' },
  running: {
    label: '執行中',
    color: 'text-[var(--accent-text)]',
    bg: 'bg-[var(--accent-tint)]',
  },
  pending: { label: '等待中', color: 'text-[var(--warning-text)]', bg: 'bg-[var(--warning-tint)]' },
  failed: { label: '失敗', color: 'text-[var(--error-text)]', bg: 'bg-[var(--error-tint)]' },
  corrupted: {
    label: '已損壞',
    color: 'text-[var(--warning-text)]',
    bg: 'bg-[var(--warning-tint)]',
  },
};

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString('zh-TW');
}

interface BackupTableProps {
  backups: Backup[];
  onDelete: (id: string) => void;
  onVerify: (id: string) => void;
  onRestore: (id: string) => void;
  isDeleting: boolean;
  isVerifying: boolean;
  isRestoring: boolean;
}

export function BackupTable({
  backups,
  onDelete,
  onVerify,
  onRestore,
  isDeleting,
  isVerifying,
  isRestoring,
}: BackupTableProps) {
  return (
    <div
      className="overflow-hidden rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-secondary)]"
      data-testid="backup-table"
    >
      {/* Header */}
      <div className="flex items-center bg-[var(--bg-secondary)]/80 px-4 py-2.5 text-xs font-semibold text-[var(--text-secondary)]">
        <span className="w-[320px]">檔案名稱</span>
        <span className="w-[80px]">大小</span>
        <span className="w-[140px]">建立時間</span>
        <span className="w-[70px]">狀態</span>
        <span className="flex-1 text-right">操作</span>
      </div>

      {/* Rows */}
      {backups.map((backup, i) => {
        const config = statusConfig[backup.status] || statusConfig.failed;
        const isLast = i === backups.length - 1;

        return (
          <div
            key={backup.id}
            className={`flex items-center px-4 py-2.5 text-xs ${!isLast ? 'border-b border-[var(--border-subtle)]' : ''}`}
            data-testid={`backup-row-${backup.id}`}
          >
            <span
              className="w-[320px] truncate font-medium text-[var(--text-primary)]"
              title={backup.filename}
            >
              {backup.filename}
            </span>
            <span className="w-[80px] text-[var(--text-secondary)]">
              {formatBytes(backup.sizeBytes)}
            </span>
            <span className="w-[140px] text-[var(--text-secondary)]">
              {formatDate(backup.createdAt)}
            </span>
            <span className="w-[70px]">
              <span
                className={`inline-flex items-center gap-1 rounded px-2 py-0.5 text-[11px] font-medium ${config.bg} ${config.color}`}
              >
                <span
                  className={`inline-block h-1.5 w-1.5 rounded-full ${backup.status === 'completed' ? 'bg-[var(--success)]' : backup.status === 'running' ? 'bg-[var(--accent-primary)]' : backup.status === 'pending' ? 'bg-[var(--warning)]' : backup.status === 'corrupted' ? 'bg-[var(--warning)]' : 'bg-[var(--error)]'}`}
                />
                {config.label}
              </span>
            </span>
            <span className="flex flex-1 items-center justify-end gap-2">
              {backup.status === 'completed' && (
                <>
                  <button
                    onClick={() => onRestore(backup.id)}
                    disabled={isRestoring}
                    className="text-[var(--text-secondary)] transition-colors hover:text-[var(--warning-text)] disabled:opacity-50"
                    data-testid={`restore-btn-${backup.id}`}
                    title="還原"
                  >
                    <RotateCcw className="h-4 w-4" />
                  </button>
                  <button
                    onClick={() => onVerify(backup.id)}
                    disabled={isVerifying}
                    className="text-[var(--text-secondary)] transition-colors hover:text-[var(--accent-text)] disabled:opacity-50"
                    data-testid={`verify-btn-${backup.id}`}
                    title="驗證完整性"
                  >
                    <ShieldCheck className="h-4 w-4" />
                  </button>
                  <a
                    href={backupService.getDownloadUrl(backup.id)}
                    className="text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)]"
                    data-testid={`download-btn-${backup.id}`}
                    title="下載"
                  >
                    <Download className="h-4 w-4" />
                  </a>
                </>
              )}
              <button
                onClick={() => onDelete(backup.id)}
                disabled={isDeleting}
                className="text-[var(--text-secondary)] transition-colors hover:text-[var(--error-text)] disabled:opacity-50"
                data-testid={`delete-btn-${backup.id}`}
                title="刪除"
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </span>
          </div>
        );
      })}
    </div>
  );
}
