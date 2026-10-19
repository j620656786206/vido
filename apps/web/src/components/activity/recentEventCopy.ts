// Implements: <utility — no .pen counterpart>
/**
 * Words for a subtitle-generation row in the activity record
 * (disc-2026-10-activity-subtitle-run-time-b; ⚖️ Sally 2026-10-19 rulings 2–4,
 * spec note `spec-note-k-run-time` XYUmV). The backend sends copy-free fields
 * (`[@contract-v1]`, the -a story's AC #1); every Chinese word is built here.
 *
 *   detail: 「末日光明 S01E02 · 翻譯 568 句」 / 「末日光明 S01E01 · 影片所在的資料夾寫不進字幕」
 *   meta:   「3 分鐘 · US$0.27」 / 「4 秒 · 沒花錢」
 *
 * A missing field drops its part rather than inventing a value: no
 * `durationSeconds` → no time; no `spentUsd` → no money (not recorded ≠ $0).
 */
import type { RecentEvent } from '../../services/activityService';
import { gtUsd, ltUsd, usd } from '../../lib/currency';
import { formatRuntime } from '../../lib/formatMedia';
import { failureCopyForReason } from '../subtitle/generationEventCopy';

const SEP = ' · ';

/** ⚖️ Sally 2026-10-19 (追認 3): the run's item has left the library. */
export const MISSING_TITLE = '已不在媒體庫的項目';

/** Route word before the cue count (追認 1; asr is ruling 2). */
const ROUTE_WORD: Record<string, string> = {
  translate: '翻譯',
  asr: '聽寫＋翻譯',
  deliver_direct: '片內中文字幕',
  convert_then_deliver: '簡體轉繁體',
  online: '線上字幕',
};

/** 「N 秒」 under a minute, then the runtime words: 「4 分鐘」「1 小時 5 分」. */
export function formatRunDuration(seconds: number | null | undefined): string | null {
  if (seconds == null || !Number.isFinite(seconds) || seconds < 0) return null;
  if (seconds < 1) return '不到 1 秒';
  if (seconds < 60) return `${Math.floor(seconds)} 秒`;
  return formatRuntime(Math.round(seconds / 60));
}

/** 「US$0.27」; 0 → 「沒花錢」; a sub-cent charge → 「不到 US$0.01」, never 「US$0.00」. */
export function formatRunSpend(spentUsd: number | null | undefined): string | null {
  // ⚠️ A number today; backlog-money-string-on-the-wire would make this a string —
  // widen the guard then, or the money part silently disappears.
  if (spentUsd == null || !Number.isFinite(spentUsd)) return null;
  // A negative spend is not a thing the ledger records; saying nothing beats
  // printing a refund that never happened.
  if (ltUsd(spentUsd, 0)) return null;
  if (!gtUsd(spentUsd, 0)) return '沒花錢';
  // Ask usd() itself whether the amount rounds away, so this can never drift from
  // its rounding rule (a string check — no money is compared here).
  const shown = usd(spentUsd);
  if (shown === '$0.00') return '不到 US$0.01';
  return `US${shown}`;
}

function pad2(n: number): string {
  return String(n).padStart(2, '0');
}

const isCount = (n: number | undefined): n is number => Number.isInteger(n) && (n as number) >= 0;

/** 「末日光明 S01E02」; a movie or a whole series has no episode tag. */
export function recentSubtitleName(ev: RecentEvent): string {
  const title = ev.mediaTitle?.trim();
  // The contract always sends the key; "" means the item left the library, and
  // then there is no episode to name either.
  if (!title) return MISSING_TITLE;
  if (ev.mediaType === 'episode' && isCount(ev.seasonNumber) && isCount(ev.episodeNumber)) {
    return `${title} S${pad2(ev.seasonNumber)}E${pad2(ev.episodeNumber)}`;
  }
  return title;
}

/** The row's description line. */
export function recentSubtitleDetail(ev: RecentEvent): string {
  const name = recentSubtitleName(ev);
  if (ev.result === 'failed') {
    return `${name}${SEP}${failureCopyForReason(ev.failureReason).text}`;
  }
  if (!isCount(ev.cueCount) || ev.cueCount === 0) return name;
  // Own keys only: a route named `constructor` must not print a function.
  const route = ev.route ?? '';
  const word = Object.prototype.hasOwnProperty.call(ROUTE_WORD, route)
    ? ROUTE_WORD[route]
    : undefined;
  return `${name}${SEP}${word ? `${word} ` : ''}${ev.cueCount} 句`;
}

/** 「花多久 · 花多少錢」, either part dropped when unknown; null when both are. */
export function recentSubtitleMeta(ev: RecentEvent): string | null {
  const parts = [formatRunDuration(ev.durationSeconds), formatRunSpend(ev.spentUsd)].filter(
    (p): p is string => p !== null
  );
  return parts.length > 0 ? parts.join(SEP) : null;
}
