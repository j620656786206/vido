// Design ref: ux-design.pen Screen 4 Detail Panel Desktop (RgSxQ)
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
  // `color` dresses the whole pill — glyph AND label — over the matching `-tint`, which
  // is the exact surface styles.css measured the base tokens as sub-AA on. Body text
  // takes the AA-safe twins; the `-tint` backgrounds are unaffected.
  normal: {
    color: 'text-[var(--success-text)]',
    bgColor: 'bg-[var(--success-tint)]',
    label: '正常',
    icon: '✓',
  },
  partial: {
    color: 'text-[var(--warning-text)]',
    bgColor: 'bg-[var(--warning-tint)]',
    label: '部分降級',
    icon: '⚠',
  },
  minimal: {
    color: 'text-[var(--warning-text)]',
    bgColor: 'bg-[var(--warning-tint)]',
    label: '功能受限',
    icon: '⚡',
  },
  offline: {
    color: 'text-[var(--error-text)]',
    bgColor: 'bg-[var(--error-tint)]',
    label: '離線模式',
    icon: '⚫',
  },
};

export function DegradationBadge({ level, className, showLabel = true }: DegradationBadgeProps) {
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
