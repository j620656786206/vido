import { Loader2, CheckCircle, XCircle, SkipForward } from 'lucide-react';
import { cn } from '../../lib/utils';
import type { DownloadParseStatus } from '../../services/downloadService';

interface DownloadParseStatusBadgeProps {
  parseStatus?: DownloadParseStatus;
  className?: string;
}

const STATUS_CONFIG: Record<
  string,
  {
    icon: typeof Loader2;
    color: string;
    bgColor: string;
    label: string;
    animate?: boolean;
  }
> = {
  pending: {
    icon: Loader2,
    color: 'text-blue-400',
    bgColor: 'bg-blue-500/10',
    label: '解析中...',
    animate: true,
  },
  processing: {
    icon: Loader2,
    color: 'text-blue-400',
    bgColor: 'bg-blue-500/10',
    label: '解析中...',
    animate: true,
  },
  completed: {
    icon: CheckCircle,
    color: 'text-green-500',
    bgColor: 'bg-green-500/10',
    label: '已解析',
  },
  completed_with_media: {
    icon: CheckCircle,
    color: 'text-emerald-400',
    bgColor: 'bg-emerald-500/10',
    label: '已入庫',
  },
  failed: {
    icon: XCircle,
    color: 'text-red-500',
    bgColor: 'bg-red-500/10',
    label: '解析失敗',
  },
  skipped: {
    icon: SkipForward,
    color: 'text-slate-400',
    bgColor: 'bg-slate-500/10',
    label: '已跳過',
  },
};

export function DownloadParseStatusBadge({
  parseStatus,
  className,
}: DownloadParseStatusBadgeProps) {
  if (!parseStatus) return null;

  let configKey = parseStatus.status as string;
  if (parseStatus.status === 'completed' && parseStatus.mediaId) {
    configKey = 'completed_with_media';
  }

  const config = STATUS_CONFIG[configKey];
  if (!config) return null;

  const Icon = config.icon;

  return (
    <div
      className={cn(
        'inline-flex items-center gap-1 rounded-full px-2 py-0.5',
        config.bgColor,
        className
      )}
      role="status"
      aria-label={`解析狀態: ${config.label}`}
      data-testid="download-parse-status-badge"
      data-status={parseStatus.status}
    >
      <Icon
        className={cn('h-3 w-3', config.color, config.animate && 'animate-spin')}
        aria-hidden="true"
      />
      <span className={cn('text-xs', config.color)}>{config.label}</span>
    </div>
  );
}
