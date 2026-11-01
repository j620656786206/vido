/**
 * How long a run took, from two ISO timestamps, in plain zh-TW:
 * 「不到 1 秒」「45 秒」「8 分 3 秒」「1 小時 5 分」. Undefined when either end is missing or
 * unreadable, or the end is before the start — the caller writes 「沒記錄」,
 * never 「0 秒」 (disc-2026-11-spend-history-totals-and-duration).
 */
/**
 * The API sends Go's nanosecond timestamps (…09.558765419Z). The ECMAScript
 * date format has exactly three fraction digits, and Safari's engine (every
 * iPhone browser) is not obliged to read more, so trim to milliseconds first.
 */
function parseIso(s: string): number {
  return Date.parse(s.replace(/(\.\d{3})\d+/, '$1'));
}

export function formatRunDuration(
  startedAt: string | null | undefined,
  completedAt: string | null | undefined
): string | undefined {
  if (!startedAt || !completedAt) return undefined;
  const ms = parseIso(completedAt) - parseIso(startedAt);
  if (!Number.isFinite(ms) || ms < 0) return undefined;
  const total = Math.round(ms / 1000);
  if (total < 1) return '不到 1 秒'; // a cache hit: real, but 「0 秒」 reads like no data
  if (total < 60) return `${total} 秒`;
  if (total < 3600) return `${Math.floor(total / 60)} 分 ${total % 60} 秒`;
  return `${Math.floor(total / 3600)} 小時 ${Math.floor((total % 3600) / 60)} 分`;
}
