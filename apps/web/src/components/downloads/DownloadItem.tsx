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
      className="rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-secondary)]/50 transition-colors hover:border-[var(--border-subtle)]"
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
            <p className="truncate font-medium text-[var(--text-primary)]">{download.name}</p>
            {/* Progress Bar */}
            <div className="mt-1.5 h-2 w-full overflow-hidden rounded-full bg-[var(--bg-tertiary)]">
              <div
                className={cn(
                  'h-full rounded-full transition-all',
                  download.progress >= 1
                    ? 'bg-emerald-500'
                    : download.status === 'error'
                      ? 'bg-[var(--error)]'
                      : 'bg-[var(--accent-primary)]'
                )}
                style={{ width: `${Math.min(download.progress * 100, 100)}%` }}
                role="progressbar"
                aria-valuenow={Math.round(download.progress * 100)}
                aria-valuemin={0}
                aria-valuemax={100}
              />
            </div>
            <div className="mt-1 flex gap-4 text-xs text-[var(--text-secondary)]">
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
                <p className="text-[var(--success)]">↓ {formatSpeed(download.downloadSpeed)}</p>
                <p className="text-[var(--text-secondary)]">{formatETA(download.eta)}</p>
              </>
            )}
            {download.status === 'seeding' && (
              <p className="text-[var(--accent-primary)]">↑ {formatSpeed(download.uploadSpeed)}</p>
            )}
            {download.status === 'completed' &&
              (download.parseStatus ? (
                <DownloadParseStatusBadge parseStatus={download.parseStatus} />
              ) : (
                <p className="text-emerald-400">完成</p>
              ))}
            {download.status === 'paused' && <p className="text-[var(--warning)]">已暫停</p>}
            {download.status === 'error' && <p className="text-[var(--error)]">錯誤</p>}
          </div>

          {/* Expand indicator */}
          <svg
            className={cn(
              'h-5 w-5 text-[var(--text-secondary)] transition-transform',
              expanded && 'rotate-180'
            )}
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
