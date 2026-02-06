/**
 * ParseStatusBadge Component (Story 3.10 - Task 6)
 * Displays a status icon with tooltip for parse status
 * AC1: Status Icons in File List (⏳ ✅ ⚠️ ❌)
 */

import { Clock, Loader2, CheckCircle, AlertTriangle, XCircle } from 'lucide-react';
import { cn } from '../../lib/utils';
import type { ParseStatus } from './types';

export interface ParseStatusBadgeProps {
  /** Current parse status */
  status: ParseStatus;
  /** Optional tooltip text (defaults to status label) */
  tooltip?: string;
  /** Size variant */
  size?: 'sm' | 'md' | 'lg';
  /** Show text label alongside icon */
  showLabel?: boolean;
  /** Additional CSS classes */
  className?: string;
}

const STATUS_CONFIG: Record<
  ParseStatus,
  {
    icon: typeof Clock;
    color: string;
    bgColor: string;
    label: string;
    animate?: boolean;
  }
> = {
  pending: {
    icon: Clock,
    color: 'text-slate-400',
    bgColor: 'bg-slate-500/10',
    label: '等待中',
  },
  parsing: {
    icon: Loader2,
    color: 'text-blue-500',
    bgColor: 'bg-blue-500/10',
    label: '解析中',
    animate: true,
  },
  success: {
    icon: CheckCircle,
    color: 'text-green-500',
    bgColor: 'bg-green-500/10',
    label: '已完成',
  },
  needs_ai: {
    icon: AlertTriangle,
    color: 'text-yellow-500',
    bgColor: 'bg-yellow-500/10',
    label: '需要處理',
  },
  failed: {
    icon: XCircle,
    color: 'text-red-500',
    bgColor: 'bg-red-500/10',
    label: '失敗',
  },
};

// For in-progress/parsing state, we use a special indicator
const PARSING_CONFIG = {
  icon: Loader2,
  color: 'text-blue-500',
  bgColor: 'bg-blue-500/10',
  label: '解析中',
  animate: true,
};

const SIZE_CONFIG: Record<string, { iconSize: string; textSize: string; padding: string }> = {
  sm: { iconSize: 'h-3 w-3', textSize: 'text-xs', padding: 'px-1.5 py-0.5' },
  md: { iconSize: 'h-4 w-4', textSize: 'text-sm', padding: 'px-2 py-1' },
  lg: { iconSize: 'h-5 w-5', textSize: 'text-base', padding: 'px-2.5 py-1.5' },
};

/**
 * Badge component for displaying parse status with icon
 * Implements AC1: Status Icons in File List
 */
export function ParseStatusBadge({
  status,
  tooltip,
  size = 'md',
  showLabel = false,
  className,
}: ParseStatusBadgeProps) {
  const config = STATUS_CONFIG[status];
  const sizeConfig = SIZE_CONFIG[size];
  const Icon = config.icon;
  const displayLabel = tooltip || config.label;

  return (
    <div
      className={cn(
        'inline-flex items-center gap-1 rounded-full',
        config.bgColor,
        sizeConfig.padding,
        className
      )}
      title={displayLabel}
      role="status"
      aria-label={`解析狀態: ${config.label}`}
      data-testid="parse-status-badge"
      data-status={status}
    >
      <Icon
        className={cn(sizeConfig.iconSize, config.color, config.animate && 'animate-spin')}
        aria-hidden="true"
      />
      {showLabel && <span className={cn(sizeConfig.textSize, config.color)}>{config.label}</span>}
    </div>
  );
}

/**
 * Parsing-specific badge that shows spinner animation
 */
export function ParsingStatusBadge({
  tooltip = '解析中...',
  size = 'md',
  showLabel = false,
  className,
}: Omit<ParseStatusBadgeProps, 'status'>) {
  const config = PARSING_CONFIG;
  const sizeConfig = SIZE_CONFIG[size];
  const Icon = config.icon;

  return (
    <div
      className={cn(
        'inline-flex items-center gap-1 rounded-full',
        config.bgColor,
        sizeConfig.padding,
        className
      )}
      title={tooltip}
      role="status"
      aria-label="解析進度: 解析中"
      aria-live="polite"
      data-testid="parsing-status-badge"
    >
      <Icon className={cn(sizeConfig.iconSize, config.color, 'animate-spin')} aria-hidden="true" />
      {showLabel && <span className={cn(sizeConfig.textSize, config.color)}>{config.label}</span>}
    </div>
  );
}

export default ParseStatusBadge;
