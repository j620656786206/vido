// Implements: <utility — no .pen counterpart>
/**
 * Formatting utilities for download display (Story 4.2)
 */

import type { Download } from '../../services/downloadService';

export function formatSpeed(bytesPerSec: number): string {
  if (bytesPerSec <= 0) return '0 B/s';
  if (bytesPerSec >= 1073741824) {
    return `${(bytesPerSec / 1073741824).toFixed(1)} GB/s`;
  }
  if (bytesPerSec >= 1048576) {
    return `${(bytesPerSec / 1048576).toFixed(1)} MB/s`;
  }
  if (bytesPerSec >= 1024) {
    return `${(bytesPerSec / 1024).toFixed(1)} KB/s`;
  }
  return `${bytesPerSec} B/s`;
}

export function formatSize(bytes: number): string {
  if (bytes <= 0) return '0 B';
  if (bytes >= 1099511627776) {
    return `${(bytes / 1099511627776).toFixed(2)} TB`;
  }
  if (bytes >= 1073741824) {
    return `${(bytes / 1073741824).toFixed(2)} GB`;
  }
  if (bytes >= 1048576) {
    return `${(bytes / 1048576).toFixed(1)} MB`;
  }
  if (bytes >= 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`;
  }
  return `${bytes} B`;
}

export function formatETA(seconds: number): string {
  if (seconds < 0 || seconds === 8640000) return '∞';
  if (seconds === 0) return '0s';
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) {
    return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
  }
  const hours = Math.floor(seconds / 3600);
  const mins = Math.floor((seconds % 3600) / 60);
  if (hours >= 24) {
    const days = Math.floor(hours / 24);
    const remainHours = hours % 24;
    return `${days}d ${remainHours}h`;
  }
  return `${hours}h ${mins}m`;
}

export function formatProgress(progress: number): string {
  return `${(progress * 100).toFixed(1)}%`;
}

const DASH = '—';

/** "5.05 GB / 8.10 GB" → "5.05 / 8.10 GB" when both sides share a unit (the dense table cell). */
function compactSizePair(done: number, total: number): string {
  const left = formatSize(done);
  const right = formatSize(total);
  const [leftValue, leftUnit] = left.split(' ');
  return leftUnit === right.split(' ')[1] ? `${leftValue} / ${right}` : `${left} / ${right}`;
}

/**
 * The card footer / table cells of D1-D-v2 and D7-D-v2. Every field is always present and says
 * "—" when it does not apply, so rows line up and a missing number never reads as zero.
 * ↓ and ETA only while downloading; ↑ while downloading or seeding; size is "done / total"
 * until the torrent is complete.
 */
export function formatDownloadMeta(
  d: Pick<Download, 'status' | 'downloadSpeed' | 'uploadSpeed' | 'eta' | 'size' | 'progress'>
): {
  down: string;
  up: string;
  /** The speeds without their arrows — the detail sheet labels them instead (dsr-4b-2). */
  downSpeed: string;
  upSpeed: string;
  eta: string;
  size: string;
  sizeCompact: string;
} {
  const moving = d.status === 'downloading';
  const sharing = moving || d.status === 'seeding';
  const complete = d.progress >= 1;
  // Size 0 is qBittorrent still fetching a magnet's metadata — unknown, not empty.
  const known = d.size > 0;
  const done = Math.round(Math.min(d.progress, 1) * d.size);
  const downSpeed = moving ? formatSpeed(d.downloadSpeed) : DASH;
  const upSpeed = sharing ? formatSpeed(d.uploadSpeed) : DASH;
  return {
    down: `↓ ${downSpeed}`,
    up: `↑ ${upSpeed}`,
    downSpeed,
    upSpeed,
    eta: moving ? formatETA(d.eta) : DASH,
    size: !known
      ? DASH
      : complete
        ? formatSize(d.size)
        : `${formatSize(done)} / ${formatSize(d.size)}`,
    sizeCompact: !known ? DASH : complete ? formatSize(d.size) : compactSizePair(done, d.size),
  };
}

/**
 * `YYYY-MM-DD HH:mm` for the detail sheet's 加入時間 (dsr-4b-2 D9-M). `addedOn` arrives as an
 * RFC3339 UTC string; shown in `timeZone`, or the browser's zone when omitted. `hourCycle: 'h23'`
 * rather than `hour12: false` — some engines print midnight as 24:xx under the latter. Anything
 * unparsable is「—」.
 */
const addedOnFormatters = new Map<string | undefined, Intl.DateTimeFormat>();

export function formatAddedOn(iso: string, timeZone?: string): string {
  const date = new Date(iso);
  if (!iso || Number.isNaN(date.getTime())) return DASH;
  // The sheet re-renders on every poll; building a formatter each time is the one cost here.
  let fmt = addedOnFormatters.get(timeZone);
  if (!fmt) {
    fmt = new Intl.DateTimeFormat('en-CA', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hourCycle: 'h23',
      timeZone,
    });
    addedOnFormatters.set(timeZone, fmt);
  }
  const parts = fmt.formatToParts(date);
  const get = (type: Intl.DateTimeFormatPartTypes) =>
    parts.find((p) => p.type === type)?.value ?? '';
  return `${get('year')}-${get('month')}-${get('day')} ${get('hour')}:${get('minute')}`;
}
