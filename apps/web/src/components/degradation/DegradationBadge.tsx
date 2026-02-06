import { cn } from '../../lib/utils';
import type { DegradationLevel } from './types';

export interface DegradationBadgeProps {
  level: DegradationLevel;
  className?: string;
  showLabel?: boolean;
}

const levelConfig: Record<
  DegradationLevel,
  { color: string; bgColor: string; label: string; icon: string }
> = {
  normal: {
    color: 'text-green-400',
    bgColor: 'bg-green-400/10',
    label: '正常',
    icon: '✓',
  },
  partial: {
    color: 'text-yellow-400',
    bgColor: 'bg-yellow-400/10',
    label: '部分降級',
    icon: '⚠',
  },
  minimal: {
    color: 'text-orange-400',
    bgColor: 'bg-orange-400/10',
    label: '功能受限',
    icon: '⚡',
  },
  offline: {
    color: 'text-red-400',
    bgColor: 'bg-red-400/10',
    label: '離線模式',
    icon: '⚫',
  },
};

export function DegradationBadge({
  level,
  className,
  showLabel = true,
}: DegradationBadgeProps) {
  const config = levelConfig[level];

  if (level === 'normal') {
    return null;
  }

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium',
        config.bgColor,
        config.color,
        className
      )}
      role="status"
      aria-label={`系統狀態：${config.label}`}
    >
      <span aria-hidden="true">{config.icon}</span>
      {showLabel && <span>{config.label}</span>}
    </span>
  );
}
