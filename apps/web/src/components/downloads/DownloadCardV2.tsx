// Design ref: ux-design.pen Screen D1-D-v2 (cK1KF) · D2-D-v2 (tx6U1) · D12-D-v2 (tvp15)
import { Magnet, Plus, Timer } from 'lucide-react';
import type { Download } from '../../services/downloadService';
import { cn } from '../../lib/utils';
import { getDownloadStatus, getDownloadTone } from './downloadStatus';
import { DownloadRowActions } from './DownloadRowActions';
import { ImportStatusChip } from './ImportStatusChip';
import { formatDownloadMeta, formatProgress } from './formatters';

interface DownloadCardV2Props {
  download: Download;
  /** Select mode (ux3-4-3b AC5) — shows a per-card checkbox. */
  selectable?: boolean;
  selected?: boolean;
  onSelectChange?: (hash: string, selected: boolean) => void;
  /** Card actions (ux3-4-3b AC3). When provided, the state button + ⋯ menu render. */
  onPause?: (hash: string) => void;
  onResume?: (hash: string) => void;
  onRemove?: (hash: string, deleteFiles: boolean) => void;
  /** Phone (dsr-4b-2): ⋯ reports upward so the page can open its actions sheet. */
  onOpenActions?: (hash: string, trigger: HTMLElement) => void;
}

/** Status token pill with its dot — shared by the card and the table (D1-D-v2 / D7-D-v2). */
export function DownloadStatusPill({ download }: { download: Pick<Download, 'hash' | 'status'> }) {
  const status = getDownloadStatus(download.status);
  return (
    <span
      data-testid={`download-status-${download.hash}`}
      className={cn(
        'inline-flex shrink-0 items-center gap-1.5 whitespace-nowrap rounded-full px-2.5 py-1 text-xs font-semibold',
        status.className
      )}
    >
      <span className="size-[7px] rounded-full bg-current" aria-hidden="true" />
      {status.label}
    </span>
  );
}

/**
 * DownloadCard-v2 (Component/DownloadCard-v2, Mz428) — presentational: all mutation/selection
 * state lives in the parent and arrives via callbacks. Numerics are JetBrains Mono + tabular-nums;
 * the progress bar carries role=progressbar + aria-valuenow for screen readers.
 */
export function DownloadCardV2({
  download,
  selectable = false,
  selected = false,
  onSelectChange,
  onPause,
  onResume,
  onRemove,
  onOpenActions,
}: DownloadCardV2Props) {
  const pct = Math.round(download.progress * 100);
  const tone = getDownloadTone(download);
  const meta = formatDownloadMeta(download);

  return (
    <article
      data-testid={`download-card-v2-${download.hash}`}
      className={cn(
        'flex flex-col gap-3 rounded-[var(--radius-lg)] border bg-[var(--bg-secondary)] p-4',
        selected ? 'border-[var(--accent-primary)]' : 'border-[var(--border-subtle)]'
      )}
    >
      <div className="flex items-start gap-3">
        {selectable && (
          <label className="-my-2.5 -ml-2.5 flex size-11 shrink-0 cursor-pointer items-center justify-center">
            <input
              type="checkbox"
              checked={selected}
              onChange={(e) => onSelectChange?.(download.hash, e.target.checked)}
              aria-label={`選取 ${download.name}`}
              className="size-5 accent-[var(--accent-primary)]"
            />
          </label>
        )}

        <div className="flex min-w-0 flex-1 flex-col gap-2">
          <h3 className="line-clamp-2 text-base font-semibold text-[var(--text-primary)]">
            {download.name}
          </h3>
          <div className="flex flex-wrap items-center gap-2">
            <span className="inline-flex items-center gap-1 rounded-full bg-[var(--bg-tertiary)] px-2.5 py-1 text-xs font-medium text-[var(--text-secondary)]">
              <Magnet className="size-3" aria-hidden="true" />
              qBittorrent
            </span>
            {/* Reserved-inert source slot (ux3-4-1 decision #4): NZBGet is not built, so the
                slot is drawn but never live. */}
            <span
              aria-disabled="true"
              title="尚未支援 NZBGet"
              className="inline-flex items-center gap-1 rounded-full bg-[var(--bg-tertiary)] px-2.5 py-1 text-xs font-medium text-[var(--text-disabled)]"
            >
              <Plus className="size-3" aria-hidden="true" />
              NZBGet
              <span className="sr-only">（尚未支援）</span>
            </span>
            {download.importStatus && <ImportStatusChip status={download.importStatus} />}
          </div>
        </div>

        <DownloadStatusPill download={download} />
      </div>

      {/* Progress bar + percent (Mono, coloured by state) */}
      <div className="flex items-center gap-3">
        <div className="h-2 flex-1 overflow-hidden rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)]">
          <div
            className={cn('h-full rounded-[var(--radius-sm)] transition-[width]', tone.fill)}
            style={{ width: `${Math.min(pct, 100)}%` }}
            role="progressbar"
            aria-label={`${download.name} 下載進度`}
            aria-valuenow={pct}
            aria-valuemin={0}
            aria-valuemax={100}
          />
        </div>
        <span className={cn('shrink-0 font-mono text-sm font-semibold tabular-nums', tone.text)}>
          {formatProgress(download.progress)}
        </span>
      </div>

      {/* Meta + actions. Speeds are neutral: a number going up is information, not a state
          (closes disc-2026-08-download-speed-wears-completed-green). */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div
          className="flex flex-wrap items-center gap-x-4 gap-y-1 font-mono text-xs tabular-nums text-[var(--text-secondary)]"
          data-testid={`download-meta-${download.hash}`}
        >
          <span>{meta.down}</span>
          <span>{meta.up}</span>
          <span className="flex items-center gap-1">
            <Timer className="size-3.5 text-[var(--text-muted)]" aria-hidden="true" />
            <span className="sr-only">預估剩餘</span>
            {meta.eta}
          </span>
          <span>{meta.size}</span>
        </div>

        <DownloadRowActions
          download={download}
          onPause={onPause}
          onResume={onResume}
          onRemove={onRemove}
          onOpenActions={onOpenActions}
        />
      </div>
    </article>
  );
}
