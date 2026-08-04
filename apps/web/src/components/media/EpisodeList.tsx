// Implements: <screen-section — pending epic-19-8 mapping>
/**
 * EpisodeList (Story 12-2)
 *
 * Presentational list of a season's episodes. The parent (SeasonAccordion) owns
 * the TanStack Query and passes loading/error/retry down — mirroring the
 * DualRatingDisplay (Story 12-1) presentational pattern.
 *
 * Each episode row shows: SxxExx code, title, air date, runtime, and — only when
 * the episode has a local file — a subtitle status indicator (AC #4/#5/#6).
 * On mobile, rows stack (title/date line, metadata below) for readability (AC #8).
 */

import { CheckCircle2, XCircle, Loader2, Minus, CircleSlash } from 'lucide-react';
import { cn } from '../../lib/utils';
import type { MergedEpisode } from '../../types/library';

interface EpisodeListProps {
  episodes: MergedEpisode[];
  seasonNumber: number;
  isLoading?: boolean;
  isError?: boolean;
  onRetry?: () => void;
}

/** Formats a season/episode pair as SxxExx (e.g. S01E05). */
function episodeCode(seasonNumber: number, episodeNumber: number): string {
  const s = String(seasonNumber).padStart(2, '0');
  const e = String(episodeNumber).padStart(2, '0');
  return `S${s}E${e}`;
}

// The 9 `subtitle_status` values (sub-1-2 [@contract-v1]); treatment ratified by
// sub-1-7a, spec screen `flow-j-specs/j2-d`.
//
// `label` is the LONG form on purpose: the icon carries no visible text, so the
// accessible name is the only place the full explanation can live. That is where
// "已略過 must read as deliberate, not broken" is actually solved — the poster
// badge's 3–4-char label (libraryStatus.ts) cannot carry it.
//
// ICON GRAMMAR (J2-D, Sally 2026-08-04) — read this before adding a state:
//   CIRCLED glyph = a settled outcome. The pipeline has an answer for this file.
//     CheckCircle2 (found) · XCircle (not_found / no_text_source) · CircleSlash (skipped)
//   BARE glyph = not an outcome yet.
//     Minus (not_searched) · Loader2 (the four in-flight states)
// Colour is URGENCY, not outcome: --error = the user can do something about it,
// --text-muted = nothing to do, --success = done, --accent-text = in progress
// (accent stays reserved for in-progress — Sally 2026-07-05).
//
// `--accent-text` rather than `--accent-primary`: an icon is foreground, and
// #3b82f6 measures 3.04:1 on `--bg-tertiary` — at the WCAG 1.4.11 non-text
// threshold, so any hover surface puts it under. #60a5fa holds 4.40:1 worst case.
const SUBTITLE_STATUS: Record<
  string,
  { Icon: typeof CheckCircle2; color: string; label: string; spin?: boolean }
> = {
  found: { Icon: CheckCircle2, color: 'text-[var(--success)]', label: '已找到字幕' },
  not_found: { Icon: XCircle, color: 'text-[var(--error)]', label: '找不到字幕' },
  // Re-tinted --warning → --accent-text per sub-1-7a AC #5: `searching` IS an
  // in-progress state, and two colours for one meaning next to the three spinners
  // below would read as a distinction that does not exist.
  searching: { Icon: Loader2, color: 'text-[var(--accent-text)]', label: '字幕搜尋中', spin: true },
  not_searched: { Icon: Minus, color: 'text-[var(--text-muted)]', label: '尚未搜尋字幕' },
  probing: { Icon: Loader2, color: 'text-[var(--accent-text)]', label: '偵測字幕軌中', spin: true },
  extracting: {
    Icon: Loader2,
    color: 'text-[var(--accent-text)]',
    label: '抽取內嵌字幕中',
    spin: true,
  },
  translating: {
    Icon: Loader2,
    color: 'text-[var(--accent-text)]',
    label: '翻譯字幕中',
    spin: true,
  },
  // Terminal: the pipeline will not produce a subtitle for this file automatically.
  // Distinct from not_found (缺字幕) by recovery — only P2 ASR can change this one.
  no_text_source: {
    Icon: XCircle,
    color: 'text-[var(--text-muted)]',
    label: '無可用的文字字幕軌',
  },
  // Terminal and DELIBERATE: the track's language tag did not qualify (P0 — `und`
  // is never treated as English). Muted, not error-red: nothing went wrong.
  //
  // `CircleSlash`, not `Minus` (Sally's ruling 2026-08-04). A bare `Minus` put a
  // SETTLED verdict into the not-settled-yet family, which is why it rendered
  // identically to `not_searched`. The slash reads "excluded by rule", which is
  // exactly what happened here.
  skipped: {
    Icon: CircleSlash,
    color: 'text-[var(--text-muted)]',
    label: '已略過（字幕軌語言不符）',
  },
};

/** Subtitle status indicator — hidden entirely when no local file exists (AC #6). */
function SubtitleStatusIcon({ episode }: { episode: MergedEpisode }) {
  if (!episode.hasLocalFile) return null;

  const status = episode.subtitleStatus ?? 'not_searched';
  const meta = SUBTITLE_STATUS[status] ?? SUBTITLE_STATUS.not_searched;
  const { Icon } = meta;

  return (
    <span
      role="status"
      aria-label={meta.label}
      title={meta.label}
      className={cn('inline-flex shrink-0 items-center', meta.color)}
    >
      <Icon className={cn('h-4 w-4', meta.spin && 'animate-spin')} aria-hidden="true" />
    </span>
  );
}

function formatRuntime(runtime?: number): string | null {
  if (!runtime || runtime <= 0) return null;
  return `${runtime} 分鐘`;
}

/** Loading skeleton shown while a season's episodes are being fetched (AC #4, Task 7.5). */
function EpisodeListSkeleton() {
  return (
    <ul className="divide-y divide-[var(--border-subtle)]" data-testid="episode-list-skeleton">
      {[0, 1, 2, 3].map((i) => (
        <li key={i} className="flex items-center gap-3 px-4 py-3" aria-hidden="true">
          <div className="h-4 w-14 animate-pulse rounded bg-[var(--bg-secondary)]" />
          <div className="h-4 flex-1 animate-pulse rounded bg-[var(--bg-secondary)]" />
          <div className="h-4 w-10 animate-pulse rounded bg-[var(--bg-secondary)]" />
        </li>
      ))}
    </ul>
  );
}

/** Retry-able error state shown when the TMDb season fetch fails (AC #7, Task 7.6). */
function EpisodeListError({ onRetry }: { onRetry?: () => void }) {
  return (
    <div
      role="alert"
      className="flex flex-col items-center gap-3 px-4 py-6 text-center"
      data-testid="episode-list-error"
    >
      <p className="text-sm text-[var(--text-secondary)]">無法載入劇集列表，請稍後再試。</p>
      {onRetry && (
        <button
          type="button"
          onClick={onRetry}
          className="rounded-md border border-[var(--border-subtle)] px-3 py-1.5 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-secondary)]"
        >
          重試
        </button>
      )}
    </div>
  );
}

export function EpisodeList({
  episodes,
  seasonNumber,
  isLoading,
  isError,
  onRetry,
}: EpisodeListProps) {
  if (isLoading) return <EpisodeListSkeleton />;
  if (isError) return <EpisodeListError onRetry={onRetry} />;

  if (episodes.length === 0) {
    return (
      <p className="px-4 py-6 text-center text-sm text-[var(--text-secondary)]">
        此季沒有劇集資料。
      </p>
    );
  }

  return (
    <ul className="divide-y divide-[var(--border-subtle)]" data-testid="episode-list">
      {episodes.map((ep) => {
        const runtime = formatRuntime(ep.runtime);
        return (
          <li
            key={ep.episodeNumber}
            className="flex flex-col gap-1 px-4 py-3 sm:flex-row sm:items-center sm:gap-4"
            data-testid="episode-row"
          >
            {/* Title line: code + title + subtitle status */}
            <div className="flex min-w-0 flex-1 items-center gap-2">
              <span className="shrink-0 font-mono text-xs text-[var(--text-muted)]">
                {episodeCode(seasonNumber, ep.episodeNumber)}
              </span>
              <span className="truncate text-sm font-medium text-[var(--text-primary)]">
                {ep.name || `第 ${ep.episodeNumber} 集`}
              </span>
              <SubtitleStatusIcon episode={ep} />
            </div>

            {/* Metadata: air date + runtime (below on mobile, inline on desktop) */}
            <div className="flex shrink-0 items-center gap-3 text-xs text-[var(--text-secondary)]">
              {ep.airDate && <span>{ep.airDate}</span>}
              {runtime && <span>{runtime}</span>}
            </div>
          </li>
        );
      })}
    </ul>
  );
}
