import type { TorrentStatus } from '../../services/downloadService';
import { cn } from '../../lib/utils';

interface StatusIconProps {
  status: TorrentStatus;
  className?: string;
}

const statusConfig: Record<TorrentStatus, { label: string; color: string; icon: string }> = {
  downloading: { label: '下載中', color: 'text-[var(--success)]', icon: '↓' },
  paused: { label: '已暫停', color: 'text-[var(--warning)]', icon: '⏸' },
  seeding: { label: '做種中', color: 'text-[var(--accent-primary)]', icon: '↑' },
  completed: { label: '已完成', color: 'text-emerald-400', icon: '✓' },
  stalled: { label: '停滯中', color: 'text-[var(--warning)]', icon: '⏳' },
  error: { label: '錯誤', color: 'text-[var(--error)]', icon: '✗' },
  queued: { label: '排隊中', color: 'text-[var(--text-secondary)]', icon: '⏱' },
  checking: { label: '檢查中', color: 'text-purple-400', icon: '⟳' },
};

export function StatusIcon({ status, className }: StatusIconProps) {
  const config = statusConfig[status] || statusConfig.downloading;

  return (
    <span
      className={cn('inline-flex items-center gap-1 text-sm font-medium', config.color, className)}
      title={config.label}
    >
      <span className="text-base">{config.icon}</span>
      <span>{config.label}</span>
    </span>
  );
}
