/**
 * Shared relative time formatting utilities (zh-TW)
 */

export function formatRelativeTimeZh(isoStringOrDate?: string | Date): string {
  if (!isoStringOrDate) return '';
  const date = typeof isoStringOrDate === 'string' ? new Date(isoStringOrDate) : isoStringOrDate;
  if (isNaN(date.getTime())) return '';
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);

  if (diffMins < 1) return '剛剛';
  if (diffMins < 60) return `${diffMins} 分鐘前`;
  const diffHours = Math.floor(diffMins / 60);
  if (diffHours < 24) return `${diffHours} 小時前`;
  return `${Math.floor(diffHours / 24)} 天前`;
}
