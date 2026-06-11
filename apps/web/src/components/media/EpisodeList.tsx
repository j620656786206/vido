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

import { CheckCircle2, XCircle, Loader2, Minus } from 'lucide-react';
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

const SUBTITLE_STATUS: Record<
  string,
  { Icon: typeof CheckCircle2; color: string; label: string; spin?: boolean }
> = {
  found: { Icon: CheckCircle2, color: 'text-[var(--success)]', label: '已找到字幕' },
  not_found: { Icon: XCircle, color: 'text-[var(--error)]', label: '找不到字幕' },
  searching: { Icon: Loader2, color: 'text-[var(--warning)]', label: '字幕搜尋中', spin: true },
  not_searched: { Icon: Minus, color: 'text-[var(--text-muted)]', label: '尚未搜尋字幕' },
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
