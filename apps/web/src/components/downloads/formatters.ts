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
): { down: string; up: string; eta: string; size: string; sizeCompact: string } {
  const moving = d.status === 'downloading';
  const sharing = moving || d.status === 'seeding';
  const complete = d.progress >= 1;
  // Size 0 is qBittorrent still fetching a magnet's metadata — unknown, not empty.
  const known = d.size > 0;
  const done = Math.round(Math.min(d.progress, 1) * d.size);
  return {
    down: `↓ ${moving ? formatSpeed(d.downloadSpeed) : DASH}`,
    up: `↑ ${sharing ? formatSpeed(d.uploadSpeed) : DASH}`,
    eta: moving ? formatETA(d.eta) : DASH,
    size: !known
      ? DASH
      : complete
        ? formatSize(d.size)
        : `${formatSize(done)} / ${formatSize(d.size)}`,
    sizeCompact: !known ? DASH : complete ? formatSize(d.size) : compactSizePair(done, d.size),
  };
}
