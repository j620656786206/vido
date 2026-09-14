// Implements: <utility — no .pen counterpart>
/**
 * Download status → v2 token descriptor (DL-v2 §2.5).
 *
 * Reuses the N1 lifecycle TINT token pairs from `libraryStatus.ts` (ux3-0-2) rather than inventing a
 * download-specific palette (ux3-4-1 decision #5), so the download status pill and the library poster
 * badge read as one system. Rendered as a static `<span>` pill (the v2 convention — PosterCardV2:73;
 * ARIA lives on the filter tab controls, not the pill). Colored text uses the AA-safe `*-text`
 * variants throughout (TC-2) — a base token as pill TEXT on its own tint is the sub-AA case those
 * twins exist to replace; CJK labels stay in the default Noto Sans TC (TY-1).
 */
import type { Download, TorrentStatus } from '../../services/downloadService';

export interface DownloadStatusDescriptor {
  label: string;
  /** Tailwind token classes: tint background + AA-safe text color (§2.5, mirrors libraryStatus TINT). */
  className: string;
}

const TINT = {
  success: 'bg-[var(--success-tint)] text-[var(--success-text)]',
  accent: 'bg-[var(--accent-tint)] text-[var(--accent-text)]',
  error: 'bg-[var(--error-tint)] text-[var(--error-text)]',
  info: 'bg-[var(--info-tint)] text-[var(--info-text)]',
  // Nothing is running and nothing went wrong.
  neutral: 'bg-[var(--bg-tertiary)] text-[var(--text-secondary)]',
} as const;

// Total map over TorrentStatus (8 states — the 6 live filter values plus the transient
// stalled/queued/checking qBittorrent reports). A total Record means a new backend status is a
// compile error here, not a silently-unstyled pill.
const STATUS_TOKENS: Record<TorrentStatus, DownloadStatusDescriptor> = {
  downloading: { label: '下載中', className: TINT.accent },
  // Neutral, not 赭 (dsr-4, D1-D-v2). 赭 means「你要求了，但它沒發生」; a paused torrent is
  // exactly what the user asked for.
  paused: { label: '已暫停', className: TINT.neutral },
  seeding: { label: '做種', className: TINT.info },
  completed: { label: '已完成', className: TINT.success },
  stalled: { label: '停滯', className: TINT.neutral },
  error: { label: '錯誤', className: TINT.error },
  queued: { label: '佇列中', className: TINT.neutral },
  checking: { label: '檢查中', className: TINT.info },
};

/** The one download status descriptor for a torrent's current state. Total — never returns null. */
export function getDownloadStatus(status: TorrentStatus): DownloadStatusDescriptor {
  return STATUS_TOKENS[status];
}

export interface DownloadTone {
  /** Progress-bar fill (a base token — a bar is not text). */
  fill: string;
  /** Percent text (the AA-safe `*-text` twin). */
  text: string;
}

/**
 * Progress bar + percent colour, shared by the card and the table (D1-D-v2 / D7-D-v2). Gold only
 * while bytes are actually moving down; a paused or waiting torrent is neutral, not「正在跑」.
 */
export function getDownloadTone(download: Pick<Download, 'status' | 'progress'>): DownloadTone {
  switch (download.status) {
    case 'error':
      return { fill: 'bg-[var(--error)]', text: 'text-[var(--error-text)]' };
    case 'downloading':
      return { fill: 'bg-[var(--accent-primary)]', text: 'text-[var(--accent-text)]' };
    case 'seeding':
    case 'checking':
      return { fill: 'bg-[var(--info)]', text: 'text-[var(--info-text)]' };
    default:
      return download.progress >= 1 || download.status === 'completed'
        ? { fill: 'bg-[var(--success)]', text: 'text-[var(--success-text)]' }
        : { fill: 'bg-[var(--text-muted)]', text: 'text-[var(--text-secondary)]' };
  }
}
