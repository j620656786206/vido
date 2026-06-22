/**
 * Compact zh-TW relative-time label (剛剛 / N 分鐘前 / N 小時前 / N 天前).
 * Used by the Activity hub recent-events feed (ux3-2-3). Returns '' for an
 * unparseable timestamp (never throws — F3).
 */
export function formatRelativeTime(iso: string | undefined, now: number = Date.now()): string {
  if (!iso) return '';
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return '';
  const sec = Math.floor((now - then) / 1000);
  if (sec < 45) return '剛剛';
  const min = Math.floor(sec / 60);
  if (min < 60) return `${Math.max(min, 1)} 分鐘前`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr} 小時前`;
  const day = Math.floor(hr / 24);
  return `${day} 天前`;
}
