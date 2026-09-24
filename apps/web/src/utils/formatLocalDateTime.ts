/**
 * `YYYY-MM-DD HH:mm` (or `… HH:mm:ss`) in the viewer's own time zone.
 *
 * `toLocaleString('zh-TW')` prints「2026/9/11 上午3:00:00」— unpadded, with a
 * 上午/下午 word, so a column of them never lines up and a 03:00 backup reads
 * like 3 in the afternoon at a glance. The design draws a fixed-width mono
 * stamp; this is that stamp. `precision` exists for dsr-3e's log viewer
 * (seconds) — one formatter, not two.
 *
 * Returns '' for an unparseable input rather than「NaN-NaN-NaN」.
 */
export function formatLocalDateTime(
  iso: string | null | undefined,
  precision: 'minute' | 'second' = 'minute'
): string {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  const p = (n: number) => String(n).padStart(2, '0');
  const date = `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
  const time = `${p(d.getHours())}:${p(d.getMinutes())}`;
  return precision === 'second' ? `${date} ${time}:${p(d.getSeconds())}` : `${date} ${time}`;
}
