/**
 * Formatting utilities for download display (Story 4.2)
 */

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

export function formatDate(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleDateString('zh-TW', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}
