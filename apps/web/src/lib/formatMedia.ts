/**
 * Pure formatting helpers for the PosterCard below-image metadata line (zh-TW).
 *
 * bugfix-10-7: the card shows `${year} · ${extra}` where `extra` is the running
 * time (movies) or the season + episode count (series). These helpers are kept
 * side-effect-free so they can be unit-tested at the boundary (Rule 16).
 */

/**
 * Format a movie running time (minutes) as a zh-TW string.
 *
 * - falsy / `<= 0` → `''`
 * - `< 60` → `${minutes} 分鐘`
 * - `>= 60` → `${h} 小時 ${m} 分`, dropping the ` ${m} 分` when `m === 0`
 *
 * Examples: `47` → `47 分鐘`, `60` → `1 小時`, `120` → `2 小時`,
 * `125` → `2 小時 5 分`, `139` → `2 小時 19 分`, `0` / `undefined` / `null` → `''`.
 */
export function formatRuntime(minutes?: number | null): string {
  if (!minutes || minutes <= 0) return '';
  if (minutes < 60) return `${minutes} 分鐘`;
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  return m === 0 ? `${h} 小時` : `${h} 小時 ${m} 分`;
}

/**
 * Format a series' season + episode count as a zh-TW string.
 *
 * - `seasons` falsy / `<= 0` → `''`
 * - `episodes` falsy / `<= 0` → `${seasons} 季`
 * - otherwise → `${seasons} 季 ${episodes} 集`
 *
 * Examples: `(4, 34)` → `4 季 34 集`, `(1, undefined)` → `1 季`,
 * `(1, 0)` → `1 季`, `(0, …)` → `''`.
 */
export function formatSeriesCount(seasons?: number | null, episodes?: number | null): string {
  if (!seasons || seasons <= 0) return '';
  if (!episodes || episodes <= 0) return `${seasons} 季`;
  return `${seasons} 季 ${episodes} 集`;
}

/**
 * Compose the PosterCard metadata line from the release year and an `extra`
 * fragment (the output of {@link formatRuntime} or {@link formatSeriesCount}).
 *
 * - `year && extra` → `${year} · ${extra}`
 * - `year` only → `${year}`
 * - `extra` only → `extra`
 * - neither → `''`
 */
export function formatPosterMeta(year: number | null, extra: string): string {
  if (year && extra) return `${year} · ${extra}`;
  if (year) return `${year}`;
  if (extra) return extra;
  return '';
}
