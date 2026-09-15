// Design ref: ux-design.pen Screen D12-D-v2 (tvp15)
import { Link } from '@tanstack/react-router';
import { CircleAlert, EyeOff, Hourglass, Library, ScanSearch, type LucideIcon } from 'lucide-react';
import { cn } from '../../lib/utils';
import type { DownloadImportStatus } from '../../services/downloadService';

const SOURCE_NAME: Record<string, string> = {
  radarr: 'Radarr',
  sonarr: 'Sonarr',
};

const TONE = {
  // 青碧 — it arrived.
  success: 'bg-[var(--success-tint)] text-[var(--success-text)]',
  // 靛青 — imported, Vido just does not have it yet: information, not a fault.
  info: 'bg-[var(--info-tint)] text-[var(--info-text)]',
  // 赭 — Sonarr/Radarr was asked to bring it in and marked it failed.
  warning: 'bg-[var(--warning-tint)] text-[var(--warning-text)]',
  // Waiting a minute after the download, or someone chose to skip it: neither is a state to colour.
  neutral: 'bg-[var(--bg-tertiary)] text-[var(--text-secondary)]',
} as const;

export interface ImportStatusCopy {
  /** Card chip text. */
  label: string;
  /** Dense table chip text. */
  short: string;
  /** The whole sentence, for screen readers. */
  sentence: string;
  icon: LucideIcon;
  tone: keyof typeof TONE;
}

/** What the chip says for a status — or null for a state this build does not know (say nothing). */
export function describeImportStatus(status: DownloadImportStatus): ImportStatusCopy | null {
  const source = SOURCE_NAME[status.source] ?? 'Sonarr／Radarr';
  switch (status.state) {
    case 'in_library':
      return {
        label: '已入庫',
        short: '已入庫',
        sentence: `${source} 已匯入，Vido 媒體庫裡有`,
        icon: Library,
        tone: 'success',
      };
    case 'awaiting_scan': {
      const total = status.episodesImported ?? 0;
      const have = status.episodesInLibrary ?? 0;
      if (status.mediaType === 'tv' && total > 0 && have > 0) {
        return {
          label: `Vido 有 ${have}/${total} 集`,
          short: `${have}/${total} 集`,
          sentence: `${source} 匯入了 ${total} 集，Vido 目前有其中 ${have} 集`,
          icon: ScanSearch,
          tone: 'info',
        };
      }
      return {
        label: `${source} 已匯入 · Vido 還沒有`,
        short: '已匯入',
        sentence: `${source} 已匯入，但 Vido 媒體庫裡還沒有`,
        icon: ScanSearch,
        tone: 'info',
      };
    }
    case 'awaiting_import':
      return {
        label: `等 ${source} 匯入`,
        short: '待匯入',
        sentence: `下載完了，等 ${source} 匯入`,
        icon: Hourglass,
        tone: 'neutral',
      };
    case 'import_failed':
      return {
        label: `${source} 匯入失敗`,
        short: '匯入失敗',
        sentence: `${source} 把這個下載標記為失敗，要到 ${source} 看原因`,
        icon: CircleAlert,
        tone: 'warning',
      };
    case 'import_ignored':
      return {
        label: '已略過匯入',
        short: '已略過',
        sentence: `有人在 ${source} 略過了這個下載`,
        icon: EyeOff,
        tone: 'neutral',
      };
    default:
      return null;
  }
}

/**
 * Where the chip can take you: the title's detail page, only when what it says and where it goes
 * agree — in the library, or a season pack Vido already has part of. "Vido 還沒有" is never a link.
 */
function detailTarget(status: DownloadImportStatus): string | null {
  if (!status.mediaId || (status.mediaType !== 'movie' && status.mediaType !== 'tv')) return null;
  if (status.state === 'in_library') return status.mediaId;
  if (
    status.state === 'awaiting_scan' &&
    status.mediaType === 'tv' &&
    (status.episodesInLibrary ?? 0) > 0
  ) {
    return status.mediaId;
  }
  return null;
}

/**
 * The import status of a finished download (dl-import-2), on a card or in the table. The accessible
 * name starts with the visible words (WCAG 2.5.3), so「點已入庫」works for voice control; a link's
 * hit area is padded out to 44px.
 */
export function ImportStatusChip({
  status,
  variant = 'card',
}: {
  status: DownloadImportStatus;
  variant?: 'card' | 'table';
}) {
  const copy = describeImportStatus(status);
  if (!copy) return null;

  const Icon = copy.icon;
  const text = variant === 'card' ? copy.label : copy.short;
  const target = detailTarget(status);
  const className = cn(
    'inline-flex min-w-0 max-w-full items-center gap-1 rounded-full text-xs font-semibold',
    // A card chip may be the longest thing on a narrow phone card: it truncates instead of pushing
    // past the edge. The table chip is always short.
    variant === 'card' ? 'px-2.5 py-1' : 'shrink-0 whitespace-nowrap px-2 py-0.5',
    TONE[copy.tone]
  );
  const visible = <span className={cn(variant === 'card' && 'truncate')}>{text}</span>;

  if (target) {
    return (
      <Link
        to="/media/$type/$id"
        params={{ type: status.mediaType, id: target }}
        data-testid="download-import-status"
        data-state={status.state}
        aria-label={`${text}：${copy.sentence}，查看詳情`}
        className={cn(
          className,
          "relative after:absolute after:-inset-x-1 after:content-[''] hover:underline",
          variant === 'card' ? 'after:-inset-y-2.5' : 'after:-inset-y-3'
        )}
      >
        <Icon className="size-3 shrink-0" aria-hidden="true" />
        {visible}
      </Link>
    );
  }

  return (
    <span data-testid="download-import-status" data-state={status.state} className={className}>
      <Icon className="size-3 shrink-0" aria-hidden="true" />
      {visible}
      <span className="sr-only">：{copy.sentence}</span>
    </span>
  );
}
