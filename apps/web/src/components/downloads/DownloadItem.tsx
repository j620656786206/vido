import type { Download } from '../../services/downloadService';
import { cn } from '../../lib/utils';
import { StatusIcon } from './StatusIcon';
import { DownloadParseStatusBadge } from './DownloadParseStatusBadge';
import { formatSpeed, formatSize, formatETA, formatProgress } from './formatters';

interface DownloadItemProps {
  download: Download;
  expanded: boolean;
  onToggleExpand: () => void;
}

export function DownloadItem({ download, expanded, onToggleExpand }: DownloadItemProps) {
  return (
    <div
      className="rounded-lg border border-slate-700 bg-slate-800/50 transition-colors hover:border-slate-600"
      data-testid={`download-item-${download.hash}`}
    >
      <button
        type="button"
        className="w-full p-4 text-left"
        onClick={onToggleExpand}
        aria-expanded={expanded}
      >
        <div className="flex items-center gap-4">
          {/* Status Icon */}
          <StatusIcon status={download.status} />

          {/* Name and Progress */}
          <div className="min-w-0 flex-1">
            <p className="truncate font-medium text-slate-100">{download.name}</p>
            {/* Progress Bar */}
            <div className="mt-1.5 h-2 w-full overflow-hidden rounded-full bg-slate-700">
              <div
                className={cn(
                  'h-full rounded-full transition-all',
                  download.progress >= 1
                    ? 'bg-emerald-500'
                    : download.status === 'error'
                      ? 'bg-red-500'
                      : 'bg-blue-500'
                )}
                style={{ width: `${Math.min(download.progress * 100, 100)}%` }}
                role="progressbar"
                aria-valuenow={Math.round(download.progress * 100)}
                aria-valuemin={0}
                aria-valuemax={100}
              />
            </div>
            <div className="mt-1 flex gap-4 text-xs text-slate-400">
              <span>{formatProgress(download.progress)}</span>
              <span>
                {formatSize(download.downloaded)} / {formatSize(download.size)}
              </span>
            </div>
          </div>

          {/* Speed and ETA */}
          <div className="text-right text-sm">
            {download.status === 'downloading' && (
              <>
                <p className="text-green-400">↓ {formatSpeed(download.downloadSpeed)}</p>
                <p className="text-slate-400">{formatETA(download.eta)}</p>
              </>
            )}
            {download.status === 'seeding' && (
              <p className="text-blue-400">↑ {formatSpeed(download.uploadSpeed)}</p>
            )}
            {download.status === 'completed' &&
              (download.parseStatus ? (
                <DownloadParseStatusBadge parseStatus={download.parseStatus} />
              ) : (
                <p className="text-emerald-400">完成</p>
              ))}
            {download.status === 'paused' && <p className="text-yellow-400">已暫停</p>}
            {download.status === 'error' && <p className="text-red-400">錯誤</p>}
          </div>

          {/* Expand indicator */}
          <svg
            className={cn('h-4 w-4 text-slate-400 transition-transform', expanded && 'rotate-180')}
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </div>
      </button>
    </div>
  );
}
