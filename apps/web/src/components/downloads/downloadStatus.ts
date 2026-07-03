// Implements: <utility — no .pen counterpart>
/**
 * Download status → v2 token descriptor (DL-v2 §2.5).
 *
 * Reuses the N1 lifecycle TINT token pairs from `libraryStatus.ts` (ux3-0-2) rather than inventing a
 * download-specific palette (ux3-4-1 decision #5), so the download status pill and the library poster
 * badge read as one system. Rendered as a static `<span>` pill (the v2 convention — PosterCardV2:73;
 * ARIA lives on the filter tab controls, not the pill). Colored text uses the AA-safe `*-text`
 * variants for accent/error (TC-2); CJK labels stay in the default Noto Sans TC (TY-1).
 */
import type { TorrentStatus } from '../../services/downloadService';

export interface DownloadStatusDescriptor {
  label: string;
  /** Tailwind token classes: tint background + AA-safe text color (§2.5, mirrors libraryStatus TINT). */
  className: string;
}

// Token strings kept identical to libraryStatus.ts TINT so the two badge systems stay visually unified.
const TINT = {
  success: 'bg-[var(--success-tint)] text-[var(--success)]',
  accent: 'bg-[var(--accent-tint)] text-[var(--accent-text)]',
  warning: 'bg-[var(--warning-tint)] text-[var(--warning)]',
  error: 'bg-[var(--error-tint)] text-[var(--error-text)]',
  info: 'bg-[var(--info-tint)] text-[var(--info)]',
  neutral: 'bg-[var(--bg-tertiary)] text-[var(--text-muted)]',
} as const;

// Total map over TorrentStatus (8 states — the 6 live filter values plus the transient
// stalled/queued/checking qBittorrent reports). A total Record means a new backend status is a
// compile error here, not a silently-unstyled pill.
const STATUS_TOKENS: Record<TorrentStatus, DownloadStatusDescriptor> = {
  downloading: { label: '下載中', className: TINT.accent },
  paused: { label: '已暫停', className: TINT.warning },
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
