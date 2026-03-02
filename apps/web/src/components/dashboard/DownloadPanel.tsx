import { useState } from 'react';
import { Link } from '@tanstack/react-router';
import { ChevronDown, Download as DownloadIcon } from 'lucide-react';
import { cn } from '../../lib/utils';
import { useDownloads } from '../../hooks/useDownloads';
import { useQBittorrentConfig } from '../../hooks/useQBittorrent';
import type { Download } from '../../services/downloadService';
import { formatProgress } from '../downloads/formatters';
import { StatusIcon } from '../downloads/StatusIcon';

interface DownloadPanelProps {
  className?: string;
}

export function DownloadPanel({ className }: DownloadPanelProps) {
  const { data: config, isLoading: configLoading } = useQBittorrentConfig();
  const { data: downloads, isLoading: downloadsLoading } = useDownloads();
  const [isExpanded, setIsExpanded] = useState(true);

  const isConnected = config?.configured === true;
  const isLoading = configLoading || (isConnected && downloadsLoading);
  const downloadCount = isConnected && downloads ? downloads.length : 0;

  return (
    <div
      className={cn('rounded-lg border border-slate-700 bg-slate-800/50', className)}
      data-testid="download-panel"
    >
      {/* Collapsible Header (AC4) */}
      <button
        type="button"
        onClick={() => setIsExpanded(!isExpanded)}
        className="flex w-full items-center justify-between border-b border-slate-700 px-4 py-3 text-left lg:cursor-default"
        aria-expanded={isExpanded}
        aria-controls="download-panel-content"
      >
        <div className="flex items-center gap-2">
          <DownloadIcon className="h-5 w-5 text-slate-300" />
          <h2 className="text-lg font-semibold text-slate-100">下載中</h2>
          {isConnected && downloadCount > 0 && (
            <span className="rounded-full bg-slate-700 px-2 py-0.5 text-xs text-slate-300">
              {downloadCount}
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          <ConnectionStatusBadge connected={isConnected} loading={configLoading} />
          {/* Chevron only visible on mobile */}
          <ChevronDown
            className={cn(
              'h-5 w-5 text-slate-400 transition-transform lg:hidden',
              isExpanded && 'rotate-180'
            )}
          />
        </div>
      </button>

      {/* Collapsible Content (AC4) */}
      <div
        id="download-panel-content"
        className={cn(
          'overflow-hidden transition-all duration-300',
          isExpanded
            ? 'max-h-[2000px] opacity-100'
            : 'max-h-0 opacity-0 lg:max-h-none lg:opacity-100'
        )}
      >
        {/* Content */}
        <div className="px-4 py-3">
          {isLoading ? (
            <div
              className="flex items-center justify-center py-6"
              data-testid="download-panel-loading"
            >
              <div className="h-6 w-6 animate-spin rounded-full border-2 border-blue-500 border-t-transparent" />
              <span className="ml-2 text-sm text-slate-400">載入中...</span>
            </div>
          ) : !isConnected ? (
            <DisconnectedState />
          ) : downloads?.length === 0 ? (
            <div className="py-6 text-center text-sm text-slate-400">目前沒有下載任務</div>
          ) : (
            <div className="space-y-2">
              {downloads?.slice(0, 5).map((download) => (
                <CompactDownloadItem key={download.hash} download={download} />
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="border-t border-slate-700 px-4 py-2">
          <Link
            to="/downloads"
            className="text-sm text-blue-400 hover:text-blue-300 hover:underline"
          >
            查看全部下載 →
          </Link>
        </div>
      </div>
    </div>
  );
}

function ConnectionStatusBadge({ connected, loading }: { connected: boolean; loading: boolean }) {
  if (loading) return null;

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs',
        connected ? 'bg-emerald-900/30 text-emerald-400' : 'bg-red-900/30 text-red-400'
      )}
    >
      <span
        className={cn('h-1.5 w-1.5 rounded-full', connected ? 'bg-emerald-400' : 'bg-red-400')}
      />
      {connected ? '已連線' : '未連線'}
    </span>
  );
}

function DisconnectedState() {
  return (
    <div className="flex flex-col items-center py-6 text-center">
      <span className="mb-2 text-3xl">⚠</span>
      <p className="text-sm text-slate-300">qBittorrent 未連線</p>
      <Link
        to="/settings/qbittorrent"
        className="mt-2 text-sm text-blue-400 hover:text-blue-300 hover:underline"
      >
        前往設定
      </Link>
    </div>
  );
}

function CompactDownloadItem({ download }: { download: Download }) {
  return (
    <div className="group flex items-center gap-3 rounded-lg p-2 transition-colors hover:bg-slate-700/50">
      <StatusIcon status={download.status} className="shrink-0 text-xs" />
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm text-slate-200">{download.name}</p>
        <div className="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-slate-700">
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
      </div>
      <span className="shrink-0 text-xs text-slate-400">{formatProgress(download.progress)}</span>
      <Link
        to="/downloads"
        className="shrink-0 opacity-0 transition-opacity group-hover:opacity-100"
        aria-label={`查看 ${download.name} 詳情`}
      >
        <svg
          className="h-4 w-4 text-slate-400"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
        </svg>
      </Link>
    </div>
  );
}
