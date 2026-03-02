import { CheckCircle, XCircle } from 'lucide-react';
import { cn } from '../../lib/utils';

interface ParseCompleteToastProps {
  title: string;
  posterUrl?: string;
  mediaType: 'movie' | 'tv';
  status?: 'success' | 'failed';
  errorMessage?: string;
}

const mediaTypeLabels: Record<string, string> = {
  movie: '電影',
  tv: '影集',
};

export function ParseCompleteToast({
  title,
  posterUrl,
  mediaType,
  status = 'success',
  errorMessage,
}: ParseCompleteToastProps) {
  const isFailed = status === 'failed';

  return (
    <div
      className="flex items-center gap-3 rounded-lg bg-slate-800 p-3 shadow-lg"
      data-testid="parse-complete-toast"
    >
      {/* Poster thumbnail */}
      <div className="h-12 w-8 shrink-0 overflow-hidden rounded bg-slate-700">
        {posterUrl ? (
          <img src={posterUrl} alt={title} className="h-full w-full object-cover" />
        ) : (
          <div
            className="flex h-full items-center justify-center text-slate-500"
            data-testid="parse-complete-poster-placeholder"
          >
            <span className="text-xs">🎬</span>
          </div>
        )}
      </div>

      {/* Content */}
      <div className="min-w-0 flex-1">
        <p className={cn('text-sm font-medium', isFailed ? 'text-red-400' : 'text-slate-100')}>
          {isFailed ? '解析失敗' : '解析完成'}
        </p>
        <p className="truncate text-xs text-slate-300">{title}</p>
        {isFailed && errorMessage ? (
          <p className="truncate text-[10px] text-red-400/70">{errorMessage}</p>
        ) : (
          <span className="text-[10px] text-slate-400">{mediaTypeLabels[mediaType]}</span>
        )}
      </div>

      {/* Status indicator */}
      {isFailed ? (
        <XCircle className="h-5 w-5 shrink-0 text-red-400" />
      ) : (
        <CheckCircle className="h-5 w-5 shrink-0 text-emerald-400" />
      )}
    </div>
  );
}
