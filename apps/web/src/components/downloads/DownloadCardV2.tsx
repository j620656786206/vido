// Design ref: ux-design.pen Screen D1-D-v2 (cK1KF)
import type { Download } from '../../services/downloadService';
import { cn } from '../../lib/utils';
import { getDownloadStatus } from './downloadStatus';
import { formatSpeed, formatSize, formatETA, formatProgress } from './formatters';

interface DownloadCardV2Props {
  download: Download;
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
 * DownloadCard-v2 — the v2 restyle of one download row (ux3-4-3 AC2). Presentational: title (2-line
 * CJK clamp, Noto Sans TC), a source indicator, progress bar + percent, speed/ETA/size, and the
 * status token pill. All numerics are JetBrains Mono + tabular-nums (AC7). The status pill is a static
 * span (v2 convention, PosterCardV2 precedent); the progress bar carries the accessible role +
 * aria-valuenow so a screen reader reads progress without noisy aria-live announcements every poll.
 * Card actions + selection land in ux3-4-3b (GATE B).
 */
export function DownloadCardV2({ download }: DownloadCardV2Props) {
  const status = getDownloadStatus(download.status);
  const pct = Math.round(download.progress * 100);

  return (
    <article
      data-testid={`download-card-v2-${download.hash}`}
      className="rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-4"
    >
      <div className="flex items-start gap-3">
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
