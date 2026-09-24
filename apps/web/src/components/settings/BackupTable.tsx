// Design ref: ux-design.pen Screen C5-D (uhAKd) · C5-M (gEQX4)
import {
  CircleCheck,
  CircleX,
  Clock,
  Download,
  Loader,
  RotateCcw,
  ShieldCheck,
  Trash2,
  TriangleAlert,
} from 'lucide-react';
import type { Backup, BackupStatus } from '../../services/backupService';
import { backupService } from '../../services/backupService';
import { formatBytes } from '../../utils/formatBytes';
import { formatLocalDateTime } from '../../utils/formatLocalDateTime';
import { useIsPhone } from '../../hooks/useIsPhone';
import { cn } from '../../lib/utils';

const statusConfig: Record<
  BackupStatus,
  { label: string; className: string; icon: React.ElementType }
> = {
  // 固定詞彙: a finished backup is DONE, and done wears neutral — every row in
  // this list is normally 完成, and a column of green says nothing. Running
  // keeps accent (泥金＝正在跑).
  completed: {
    label: '完成',
    className: 'bg-[var(--bg-tertiary)] text-[var(--text-secondary)]',
    icon: CircleCheck,
  },
  running: {
    label: '執行中',
    className: 'bg-[var(--accent-tint)] text-[var(--accent-text)]',
    icon: Loader,
  },
  pending: {
    label: '等待中',
    className: 'bg-[var(--warning-tint)] text-[var(--warning-text)]',
    icon: Clock,
  },
  failed: {
    label: '失敗',
    className: 'bg-[var(--error-tint)] text-[var(--error-text)]',
    icon: CircleX,
  },
  corrupted: {
    label: '已損壞',
    className: 'bg-[var(--warning-tint)] text-[var(--warning-text)]',
    icon: TriangleAlert,
  },
};

/**
 * One column template for the header AND every row, so they cannot drift.
 * C5-D draws 440 / 120 / 200 / 120 / 176 with 16 gaps across a 1152 column;
 * those fixed widths only fit from `xl`. Between 640 and 1280 the settings
 * column is ~600–1000px, so the fixed columns shrink and the file name (the
 * one column that can truncate) takes whatever is left.
 */
export const BACKUP_GRID =
  'grid grid-cols-[minmax(0,1fr)_80px_128px_80px_auto] items-center gap-x-3 xl:grid-cols-[minmax(0,1fr)_120px_200px_120px_176px] xl:gap-x-4';

interface BackupTableProps {
  backups: Backup[];
  onDelete: (id: string) => void;
  onVerify: (id: string) => void;
  onRestore: (id: string) => void;
  isDeleting: boolean;
  isVerifying: boolean;
  isRestoring: boolean;
}

type ActionsProps = Omit<BackupTableProps, 'backups'> & {
  backup: Backup;
  /** `row`: 32px squares at the end of a table row. `card`: equal-width 44px buttons across a phone card. */
  variant: 'row' | 'card';
};

/** The one set of action buttons — the table row and the phone card both render this. */
function BackupActions({
  backup,
  variant,
  onDelete,
  onVerify,
  onRestore,
  isDeleting,
  isVerifying,
  isRestoring,
}: ActionsProps) {
  const btn = cn(
    'flex items-center justify-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)] disabled:opacity-50',
    variant === 'row' ? 'size-8' : 'h-11 flex-1'
  );
  return (
    <div
      className={cn('flex items-center gap-2', variant === 'row' && 'justify-end')}
      data-testid={`backup-actions-${backup.id}`}
    >
      {/* Only a finished backup can be restored, checked or downloaded — a
          running one is not a backup yet. It can still be deleted. */}
      {backup.status === 'completed' && (
        <>
          <button
            type="button"
            onClick={() => onRestore(backup.id)}
            disabled={isRestoring}
            className={btn}
            data-testid={`restore-btn-${backup.id}`}
            aria-label="還原"
            title="還原"
          >
            <RotateCcw className="size-3.5" aria-hidden="true" />
          </button>
          <button
            type="button"
            onClick={() => onVerify(backup.id)}
            disabled={isVerifying}
            className={btn}
            data-testid={`verify-btn-${backup.id}`}
            aria-label="驗證完整性"
            title="驗證完整性"
          >
            <ShieldCheck className="size-3.5" aria-hidden="true" />
          </button>
          <a
            href={backupService.getDownloadUrl(backup.id)}
            className={btn}
            data-testid={`download-btn-${backup.id}`}
            aria-label="下載"
            title="下載"
          >
            <Download className="size-3.5" aria-hidden="true" />
          </a>
        </>
      )}
      <button
        type="button"
        onClick={() => onDelete(backup.id)}
        disabled={isDeleting}
        className={cn(btn, 'text-[var(--error-text)] hover:text-[var(--error-text)]')}
        data-testid={`delete-btn-${backup.id}`}
        aria-label="刪除"
        title="刪除"
      >
        <Trash2 className="size-3.5" aria-hidden="true" />
      </button>
    </div>
  );
}

function StatusPill({ status }: { status: BackupStatus }) {
  const config = statusConfig[status] || statusConfig.failed;
  const Icon = config.icon;
  return (
    <span
      className={cn(
        'inline-flex shrink-0 items-center gap-1 rounded-full px-2 py-0.5 text-xs font-semibold',
        config.className
      )}
    >
      <Icon className="size-3" aria-hidden="true" />
      {config.label}
    </span>
  );
}

function BackupTime({ iso, className }: { iso: string; className?: string }) {
  return (
    <time dateTime={iso} className={className}>
      {formatLocalDateTime(iso)}
    </time>
  );
}

export function BackupTable({ backups, ...handlers }: BackupTableProps) {
  // Phone: one card per backup. The desktop table is ~610px of fixed columns
  // plus the actions, and inside `overflow-hidden` the actions were simply cut
  // off at 390 — restore and delete could not be reached on a phone at all.
  // useIsPhone (not a CSS swap) so there is ONE copy of each button in the DOM.
  const isPhone = useIsPhone();

  if (isPhone) {
    return (
      <ul
        className="overflow-hidden rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)]"
        data-testid="backup-table"
      >
        {backups.map((backup) => (
          <li
            key={backup.id}
            className="flex flex-col gap-2 p-4"
            data-testid={`backup-row-${backup.id}`}
          >
            <div className="flex items-center gap-3">
              <div className="min-w-0 flex-1 font-mono text-xs">
                <BackupTime iso={backup.createdAt} className="block text-[var(--text-primary)]" />
                <p className="truncate text-[var(--text-muted)]" title={backup.filename}>
                  {formatBytes(backup.sizeBytes)} · {backup.filename}
                </p>
              </div>
              <StatusPill status={backup.status} />
            </div>
            <BackupActions backup={backup} variant="card" {...handlers} />
          </li>
        ))}
      </ul>
    );
  }

  return (
    <div
      className="overflow-hidden rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)]"
      data-testid="backup-table"
    >
      <div
        className={cn(
          BACKUP_GRID,
          'bg-[var(--bg-tertiary)] px-4 py-3 text-xs font-semibold text-[var(--text-muted)]'
        )}
        data-testid="backup-table-head"
      >
        <span>檔案名稱</span>
        <span>大小</span>
        <span>建立時間</span>
        <span>狀態</span>
        <span>操作</span>
      </div>

      {backups.map((backup) => (
        <div
          key={backup.id}
          className={cn(BACKUP_GRID, 'min-h-14 px-4 py-3 font-mono text-xs')}
          data-testid={`backup-row-${backup.id}`}
        >
          <span className="truncate text-[var(--text-primary)]" title={backup.filename}>
            {backup.filename}
          </span>
          <span className="text-[var(--text-secondary)]">{formatBytes(backup.sizeBytes)}</span>
          <BackupTime iso={backup.createdAt} className="text-[var(--text-secondary)]" />
          <span className="font-sans">
            <StatusPill status={backup.status} />
          </span>
          <BackupActions backup={backup} variant="row" {...handlers} />
        </div>
      ))}
    </div>
  );
}
