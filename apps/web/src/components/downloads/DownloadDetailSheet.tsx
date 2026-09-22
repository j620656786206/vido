// Design ref: ux-design.pen Screen D9-M-v2 (DrYXb)
/**
 * The phone detail sheet for one download (dsr-4b-2) — the first place Hash and 儲存路徑 are
 * visible anywhere in the app. Every number comes from the list item the page already holds
 * (no extra request) and through the SAME formatters the card uses, so the two never disagree:
 * `meta.size` is progress × size, never `downloaded`. Content scrolls; the action bar stays put.
 * The primary button does not close the sheet — you watch the pill flip to 已暫停.
 */
import type { ComponentProps } from 'react';
import { Ellipsis } from 'lucide-react';
import { cn } from '../../lib/utils';
import type { Download } from '../../services/downloadService';
import { Sheet } from '../ui/Sheet';
import { DownloadStatusPill } from './DownloadCardV2';
import { getDownloadTone } from './downloadStatus';
import { stateActionFor } from './DownloadRowActions';
import { formatAddedOn, formatDownloadMeta, formatProgress } from './formatters';

interface DownloadDetailSheetProps {
  download: Download;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  finalFocus?: ComponentProps<typeof Sheet>['finalFocus'];
  /** Fires once the open/close transition has finished — the owner empties its slot on close. */
  onOpenChangeComplete?: (open: boolean) => void;
  onPause?: (hash: string) => void;
  onResume?: (hash: string) => void;
  /** ⋯ — hands off to the actions sheet. */
  onOpenActions: () => void;
}

export function DownloadDetailSheet({
  download,
  open,
  onOpenChange,
  finalFocus,
  onOpenChangeComplete,
  onPause,
  onResume,
  onOpenActions,
}: DownloadDetailSheetProps) {
  const pct = Math.round(download.progress * 100);
  const tone = getDownloadTone(download);
  const meta = formatDownloadMeta(download);
  const stateAction = stateActionFor(download, onPause, onResume);

  const cells: { label: string; value: string; wide?: boolean; mono?: boolean }[] = [
    { label: '下載速度', value: meta.downSpeed },
    { label: '上傳速度', value: meta.upSpeed },
    { label: '進度', value: meta.size },
    { label: '剩餘時間', value: meta.eta },
    { label: '來源', value: 'qBittorrent' },
    { label: '加入時間', value: formatAddedOn(download.addedOn) },
    { label: 'Hash', value: download.hash, wide: true, mono: true },
    { label: '儲存路徑', value: download.savePath, wide: true },
  ];

  return (
    <Sheet
      open={open}
      onOpenChange={onOpenChange}
      title={download.name}
      testId="download-detail-sheet"
      titleClassName="mb-2 px-5 text-lg font-bold break-words"
      className="flex flex-col overflow-hidden p-0 pt-2"
      finalFocus={finalFocus}
      onOpenChangeComplete={onOpenChangeComplete}
    >
      <div className="min-h-0 flex-1 overflow-y-auto px-5 pb-4">
        <div className="flex flex-col gap-4">
          <div>
            <DownloadStatusPill download={download} />
          </div>
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
            <span
              className={cn('shrink-0 font-mono text-base font-semibold tabular-nums', tone.text)}
            >
              {formatProgress(download.progress)}
            </span>
          </div>
          <h3 className="text-xs font-semibold text-[var(--text-secondary)]">詳細資訊</h3>
          <dl className="grid grid-cols-2 gap-3">
            {cells.map((c) => (
              <div key={c.label} className={cn('flex flex-col gap-0.5', c.wide && 'col-span-2')}>
                <dt className="text-xs text-[var(--text-secondary)]">{c.label}</dt>
                <dd
                  className={cn(
                    'text-sm text-[var(--text-primary)]',
                    c.wide && 'break-all',
                    c.mono && 'font-mono text-xs'
                  )}
                >
                  {c.value}
                </dd>
              </div>
            ))}
          </dl>
        </div>
      </div>

      <div className="flex shrink-0 items-center gap-3 border-t border-[var(--border-subtle)] px-5 pt-3 pb-[max(1.5rem,env(safe-area-inset-bottom))]">
        {stateAction ? (
          <>
            <button
              type="button"
              onClick={stateAction.run}
              aria-label={`${stateAction.label} ${download.name}`}
              className="flex h-12 flex-1 items-center justify-center gap-2 rounded-[var(--radius-md)] bg-[var(--accent-primary)] text-base font-semibold text-[var(--text-on-accent)]"
            >
              <stateAction.icon className="size-[18px]" aria-hidden="true" />
              {stateAction.label}
            </button>
            <button
              type="button"
              onClick={onOpenActions}
              aria-label={`更多動作：${download.name}`}
              className="flex size-12 shrink-0 items-center justify-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] text-[var(--text-secondary)]"
            >
              <Ellipsis className="size-5" aria-hidden="true" />
            </button>
          </>
        ) : (
          <button
            type="button"
            onClick={onOpenActions}
            aria-label={`更多動作：${download.name}`}
            className="flex h-12 flex-1 items-center justify-center gap-2 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] text-base font-semibold text-[var(--text-primary)]"
          >
            <Ellipsis className="size-[18px]" aria-hidden="true" />
            更多動作
          </button>
        )}
      </div>
    </Sheet>
  );
}
