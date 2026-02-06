import { cn } from '../../lib/utils';
import type { DegradationLevel, ServicesHealth } from './types';

export interface ServiceHealthBannerProps {
  level: DegradationLevel;
  message?: string;
  services?: ServicesHealth;
  onDismiss?: () => void;
  className?: string;
}

const levelConfig: Record<
  DegradationLevel,
  { bgColor: string; borderColor: string; textColor: string; icon: string }
> = {
  normal: {
    bgColor: 'bg-green-900/20',
    borderColor: 'border-green-500/30',
    textColor: 'text-green-200',
    icon: '✓',
  },
  partial: {
    bgColor: 'bg-yellow-900/20',
    borderColor: 'border-yellow-500/30',
    textColor: 'text-yellow-200',
    icon: '⚠️',
  },
  minimal: {
    bgColor: 'bg-orange-900/20',
    borderColor: 'border-orange-500/30',
    textColor: 'text-orange-200',
    icon: '⚡',
  },
  offline: {
    bgColor: 'bg-red-900/20',
    borderColor: 'border-red-500/30',
    textColor: 'text-red-200',
    icon: '🔴',
  },
};

const defaultMessages: Record<DegradationLevel, string> = {
  normal: '',
  partial: '部分服務暫時降級中，功能可能受到影響',
  minimal: '多項服務無法使用，僅提供基本功能',
  offline: '無法連線到外部服務，使用本地快取資料',
};

export function ServiceHealthBanner({
  level,
  message,
  services,
  onDismiss,
  className,
}: ServiceHealthBannerProps) {
  // Don't show banner for normal status
  if (level === 'normal') {
    return null;
  }

  const config = levelConfig[level];
  const displayMessage = message || defaultMessages[level];

  // Get affected services list
  const affectedServices: string[] = [];
  if (services) {
    if (services.tmdb.status !== 'healthy')
      affectedServices.push(services.tmdb.displayName);
    if (services.douban.status !== 'healthy')
      affectedServices.push(services.douban.displayName);
    if (services.wikipedia.status !== 'healthy')
      affectedServices.push(services.wikipedia.displayName);
    if (services.ai.status !== 'healthy')
      affectedServices.push(services.ai.displayName);
  }

  return (
    <div
      className={cn(
        'flex items-center justify-between gap-4 rounded-lg border p-3',
        config.bgColor,
        config.borderColor,
        className
      )}
      role="alert"
      aria-live="polite"
    >
      <div className="flex items-center gap-3">
        <span className="text-lg" aria-hidden="true">
          {config.icon}
        </span>
        <div className={cn('text-sm', config.textColor)}>
          <p className="font-medium">{displayMessage}</p>
          {affectedServices.length > 0 && (
            <p className="mt-1 text-xs opacity-80">
              受影響服務：{affectedServices.join('、')}
            </p>
          )}
        </div>
      </div>

      {onDismiss && (
        <button
          onClick={onDismiss}
          className={cn(
            'rounded p-1 transition-colors hover:bg-white/10',
            config.textColor
          )}
          aria-label="關閉通知"
        >
          <svg
            className="h-4 w-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M6 18L18 6M6 6l12 12"
            />
          </svg>
        </button>
      )}
    </div>
  );
}
