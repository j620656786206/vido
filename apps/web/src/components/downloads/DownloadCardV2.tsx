// Design ref: ux-design.pen Screen D1-D-v2 (cK1KF)
import { Pause, Play, Trash2 } from 'lucide-react';
import type { Download } from '../../services/downloadService';
import { cn } from '../../lib/utils';
import { Button } from '../ui/Button';
import {
  Dialog,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogFooter,
  DialogTitle,
  DialogDescription,
  DialogClose,
} from '../ui/Dialog';
import { getDownloadStatus } from './downloadStatus';
import { formatSpeed, formatSize, formatETA, formatProgress } from './formatters';

interface DownloadCardV2Props {
  download: Download;
  /** Select mode (ux3-4-3b AC5) — shows a per-card checkbox. */
  selectable?: boolean;
  selected?: boolean;
  onSelectChange?: (hash: string, selected: boolean) => void;
  /** Card actions (ux3-4-3b AC3). When provided, the pause/resume + remove cluster renders. */
  onPause?: (hash: string) => void;
  onResume?: (hash: string) => void;
  onRemove?: (hash: string, deleteFiles: boolean) => void;
}

// Progress-fill token by state: error → error, done/seeding → success, else accent. Token-only (AC7).
function progressFillClass(download: Download): string {
  if (download.status === 'error') return 'bg-[var(--error)]';
  if (download.progress >= 1 || download.status === 'completed' || download.status === 'seeding') {
    return 'bg-[var(--success)]';
  }
  return 'bg-[var(--accent-primary)]';
}

/**
 * DownloadCard-v2 — the v2 download row (ux3-4-3 AC2 display + AC3/AC5 interactions). Presentational:
 * all mutation/selection state lives in the parent and arrives via callbacks, so this stays testable
 * with a plain render + spies. Numerics are JetBrains Mono + tabular-nums; the status pill is a static
 * span (the progress bar carries role=progressbar + aria-valuenow for SRs). The destructive remove is
 * gated behind a Radix Dialog (focus-trapped + Escape + aria-modal for free).
 */
export function DownloadCardV2({
  download,
  selectable = false,
  selected = false,
  onSelectChange,
  onPause,
  onResume,
  onRemove,
}: DownloadCardV2Props) {
  const status = getDownloadStatus(download.status);
  const pct = Math.round(download.progress * 100);
  const showActions = Boolean(onPause || onResume || onRemove);

  return (
    <article
      data-testid={`download-card-v2-${download.hash}`}
      className={cn(
        'rounded-[var(--radius-lg)] border bg-[var(--bg-secondary)] p-4',
        selected ? 'border-[var(--accent-primary)]' : 'border-[var(--border-subtle)]'
      )}
    >
      <div className="flex items-start gap-3">
        {selectable && (
          <input
            type="checkbox"
            checked={selected}
            onChange={(e) => onSelectChange?.(download.hash, e.target.checked)}
            aria-label={`選取 ${download.name}`}
            className="mt-1 h-4 w-4 shrink-0 accent-[var(--accent-primary)]"
          />
        )}

        <div className="min-w-0 flex-1">
          <h3 className="line-clamp-2 text-sm font-medium text-[var(--text-primary)]">
            {download.name}
          </h3>
          <div className="mt-1">
            <span className="inline-flex items-center rounded-full bg-[var(--bg-tertiary)] px-2 py-0.5 text-[11px] font-medium text-[var(--text-muted)]">
              qBittorrent
            </span>
          </div>
        </div>

        <span
          data-testid={`download-status-${download.hash}`}
          className={cn(
            'shrink-0 rounded-full px-2 py-0.5 text-[11px] font-medium',
            status.className
          )}
        >
          {status.label}
        </span>

        {showActions && (
          <div className="flex shrink-0 items-center gap-1">
            {download.status === 'paused'
              ? onResume && (
                  <Button
                    size="icon"
                    variant="ghost"
                    aria-label={`繼續 ${download.name}`}
                    onClick={() => onResume(download.hash)}
                  >
                    <Play className="size-4" />
                  </Button>
                )
              : onPause && (
                  <Button
                    size="icon"
                    variant="ghost"
                    aria-label={`暫停 ${download.name}`}
                    onClick={() => onPause(download.hash)}
                  >
                    <Pause className="size-4" />
                  </Button>
                )}

            {onRemove && (
              <Dialog>
                <DialogTrigger asChild>
                  <Button size="icon" variant="ghost" aria-label={`移除 ${download.name}`}>
                    <Trash2 className="size-4 text-[var(--error-text)]" />
                  </Button>
                </DialogTrigger>
                <DialogContent aria-describedby={undefined}>
                  <DialogHeader>
                    <DialogTitle>移除下載</DialogTitle>
                    <DialogDescription>
                      「{download.name}」— 保留檔案只從 qBittorrent
                      移除任務；連同檔案刪除會一併刪除已下載的檔案，無法復原。
                    </DialogDescription>
                  </DialogHeader>
                  <DialogFooter>
                    <DialogClose asChild>
                      <Button variant="outline" onClick={() => onRemove(download.hash, false)}>
                        移除（保留檔案）
                      </Button>
                    </DialogClose>
                    <DialogClose asChild>
                      <Button variant="destructive" onClick={() => onRemove(download.hash, true)}>
                        移除（連同檔案刪除）
                      </Button>
                    </DialogClose>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
            )}
          </div>
        )}
      </div>

      {/* Progress bar + percent (Mono) */}
      <div className="mt-3 flex items-center gap-3">
        <div className="h-2 flex-1 overflow-hidden rounded-full bg-[var(--bg-tertiary)]">
          <div
            className={cn('h-full rounded-full transition-all', progressFillClass(download))}
            style={{ width: `${Math.min(pct, 100)}%` }}
            role="progressbar"
            aria-label={`${download.name} 下載進度`}
            aria-valuenow={pct}
            aria-valuemin={0}
            aria-valuemax={100}
          />
        </div>
        <span className="shrink-0 font-mono text-xs tabular-nums text-[var(--text-secondary)]">
          {formatProgress(download.progress)}
        </span>
      </div>

      {/* Meta: speed / ETA / size — all numerics Mono (AC7) */}
      <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-[var(--text-secondary)]">
        {download.status === 'downloading' && (
          <>
            <span className="font-mono tabular-nums text-[var(--success)]">
              ↓ {formatSpeed(download.downloadSpeed)}
            </span>
            <span className="font-mono tabular-nums">ETA {formatETA(download.eta)}</span>
          </>
        )}
        {download.status === 'seeding' && (
          <span className="font-mono tabular-nums text-[var(--accent-text)]">
            ↑ {formatSpeed(download.uploadSpeed)}
          </span>
        )}
        <span className="font-mono tabular-nums">{formatSize(download.size)}</span>
      </div>
    </article>
  );
}
