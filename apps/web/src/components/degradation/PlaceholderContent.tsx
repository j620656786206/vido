import { cn } from '../../lib/utils';

export interface PlaceholderContentProps {
  field: 'title' | 'overview' | 'posterUrl' | 'year' | 'genres' | 'cast';
  className?: string;
}

const placeholderConfig: Record<
  PlaceholderContentProps['field'],
  { label: string; message: string }
> = {
  title: {
    label: '標題',
    message: '未知標題',
  },
  overview: {
    label: '簡介',
    message: '暫無簡介',
  },
  posterUrl: {
    label: '海報',
    message: '無法載入海報',
  },
  year: {
    label: '年份',
    message: '—',
  },
  genres: {
    label: '類型',
    message: '未知',
  },
  cast: {
    label: '演員',
    message: '暫無資料',
  },
};

export function PlaceholderContent({
  field,
  className,
}: PlaceholderContentProps) {
  const config = placeholderConfig[field];

  return (
    <span
      className={cn(
        'inline-block text-gray-500 italic',
        className
      )}
      title={`${config.label}暫時無法取得`}
    >
      {config.message}
    </span>
  );
}

export interface PlaceholderPosterProps {
  className?: string;
  size?: 'sm' | 'md' | 'lg';
}

const sizeClasses = {
  sm: 'h-24 w-16',
  md: 'h-48 w-32',
  lg: 'h-72 w-48',
};

export function PlaceholderPoster({
  className,
  size = 'md',
}: PlaceholderPosterProps) {
  return (
    <div
      className={cn(
        'flex items-center justify-center rounded-lg bg-gray-800',
        sizeClasses[size],
        className
      )}
      role="img"
      aria-label="海報無法載入"
    >
      <svg
        className="h-12 w-12 text-gray-600"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={1.5}
          d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
        />
      </svg>
    </div>
  );
}

export interface DegradationMessageProps {
  message: string;
  missingFields?: string[];
  className?: string;
}

const fieldLabels: Record<string, string> = {
  title: '標題',
  year: '年份',
  overview: '簡介',
  posterUrl: '海報',
  genres: '類型',
  cast: '演員',
};

export function DegradationMessage({
  message,
  missingFields,
  className,
}: DegradationMessageProps) {
  return (
    <div
      className={cn(
        'rounded-lg border border-yellow-500/20 bg-yellow-900/10 p-3 text-sm text-yellow-200',
        className
      )}
      role="status"
    >
      <p>{message}</p>
      {missingFields && missingFields.length > 0 && (
        <p className="mt-1 text-xs opacity-70">
          無法取得：
          {missingFields.map((f) => fieldLabels[f] || f).join('、')}
        </p>
      )}
    </div>
  );
}
