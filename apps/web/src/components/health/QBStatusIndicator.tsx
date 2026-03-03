/**
 * qBittorrent connection status indicator (Story 4.6 - AC1, AC2)
 */

import { Wifi, WifiOff, AlertTriangle } from 'lucide-react';
import { cn } from '../../lib/utils';
import { useQBConnectionHealth } from '../../hooks/useConnectionHealth';

interface QBStatusIndicatorProps {
  onClick?: () => void;
}

function formatLastSuccess(lastSuccess?: string): string {
  if (!lastSuccess) return '';
  const date = new Date(lastSuccess);
  if (isNaN(date.getTime())) return '';
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);

  if (diffMins < 1) return '剛剛';
  if (diffMins < 60) return `${diffMins} 分鐘前`;
  const diffHours = Math.floor(diffMins / 60);
  if (diffHours < 24) return `${diffHours} 小時前`;
  return `${Math.floor(diffHours / 24)} 天前`;
}

const statusConfig = {
  healthy: {
    dotColor: 'bg-emerald-400',
    textColor: 'text-emerald-400',
    bgColor: 'bg-emerald-900/30',
    label: 'qBittorrent 已連線',
    Icon: Wifi,
  },
  degraded: {
    dotColor: 'bg-yellow-400',
    textColor: 'text-yellow-400',
    bgColor: 'bg-yellow-900/30',
    label: 'qBittorrent 連線不穩定',
    Icon: AlertTriangle,
  },
  down: {
    dotColor: 'bg-red-400',
    textColor: 'text-red-400',
    bgColor: 'bg-red-900/30',
    label: 'qBittorrent 未連線',
    Icon: WifiOff,
  },
} as const;

export function QBStatusIndicator({ onClick }: QBStatusIndicatorProps) {
  const { data: health, isLoading } = useQBConnectionHealth();

  if (isLoading) {
    return (
      <div
        className="flex items-center gap-1.5 rounded-full px-2 py-0.5"
        aria-label="載入連線狀態中"
      >
        <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-slate-500" />
        <span className="text-xs text-slate-500">連線中...</span>
      </div>
    );
  }

  const status = health?.status || 'down';
  const config = statusConfig[status] || statusConfig.down;
  const { Icon } = config;
  const lastSuccessText = formatLastSuccess(health?.lastSuccess);

  return (
    <button
      onClick={onClick}
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs transition-colors',
        config.bgColor,
        config.textColor,
        'hover:opacity-80'
      )}
      aria-label={config.label}
      title={
        status === 'down' && lastSuccessText
          ? `${config.label} — 上次連線：${lastSuccessText}`
          : config.label
      }
    >
      <span
        className={cn('h-1.5 w-1.5 rounded-full', config.dotColor)}
        aria-hidden="true"
      />
      <Icon className="h-3.5 w-3.5" aria-hidden="true" />
      {status === 'down' && lastSuccessText && (
        <span className="hidden sm:inline">上次：{lastSuccessText}</span>
      )}
    </button>
  );
}

export default QBStatusIndicator;
