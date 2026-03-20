import { Download, Trash2 } from 'lucide-react';
import type { Backup, BackupStatus } from '../../services/backupService';
import { backupService } from '../../services/backupService';
import { formatBytes } from '../../utils/formatBytes';

const statusConfig: Record<BackupStatus, { label: string; color: string; bg: string }> = {
  completed: { label: '完成', color: 'text-green-400', bg: 'bg-green-400/10' },
  running: { label: '執行中', color: 'text-blue-400', bg: 'bg-blue-400/10' },
  pending: { label: '等待中', color: 'text-yellow-400', bg: 'bg-yellow-400/10' },
  failed: { label: '失敗', color: 'text-red-400', bg: 'bg-red-400/10' },
};

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString('zh-TW');
}

interface BackupTableProps {
  backups: Backup[];
  onDelete: (id: string) => void;
  isDeleting: boolean;
}

export function BackupTable({ backups, onDelete, isDeleting }: BackupTableProps) {
  return (
    <div
      className="overflow-hidden rounded-lg border border-slate-700 bg-slate-800"
      data-testid="backup-table"
    >
      {/* Header */}
      <div className="flex items-center bg-slate-800/80 px-4 py-2.5 text-xs font-semibold text-slate-400">
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
            className={`flex items-center px-4 py-2.5 text-xs ${!isLast ? 'border-b border-slate-700' : ''}`}
            data-testid={`backup-row-${backup.id}`}
          >
            <span className="w-[320px] truncate font-medium text-slate-200" title={backup.filename}>
              {backup.filename}
            </span>
            <span className="w-[80px] text-slate-400">{formatBytes(backup.sizeBytes)}</span>
            <span className="w-[140px] text-slate-400">{formatDate(backup.createdAt)}</span>
            <span className="w-[70px]">
              <span
                className={`inline-flex items-center gap-1 rounded px-2 py-0.5 text-[11px] font-medium ${config.bg} ${config.color}`}
              >
                <span
                  className={`inline-block h-1.5 w-1.5 rounded-full ${backup.status === 'completed' ? 'bg-green-400' : backup.status === 'running' ? 'bg-blue-400' : backup.status === 'pending' ? 'bg-yellow-400' : 'bg-red-400'}`}
                />
                {config.label}
              </span>
            </span>
            <span className="flex flex-1 items-center justify-end gap-2">
              {backup.status === 'completed' && (
                <a
                  href={backupService.getDownloadUrl(backup.id)}
                  className="text-slate-400 transition-colors hover:text-slate-200"
                  data-testid={`download-btn-${backup.id}`}
                  title="下載"
                >
                  <Download className="h-4 w-4" />
                </a>
              )}
              <button
                onClick={() => onDelete(backup.id)}
                disabled={isDeleting}
                className="text-slate-400 transition-colors hover:text-red-400 disabled:opacity-50"
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
