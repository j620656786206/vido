// Implements: <screen-section — pending epic-19-8 mapping>
/**
 * SeasonAccordion (Story 12-2)
 *
 * Expandable per-season accordion shown on a TV series detail page. Collapsed
 * rows show the season poster, name, episode count, and air date (AC #2). On
 * expand, the season's episodes are fetched lazily (TanStack Query `enabled`
 * gate, AC #3) and rendered via EpisodeList, which receives the query's
 * loading/error/retry state.
 *
 * Multiple seasons may be open at once (Task 6.4 — default multi-open).
 */

import { useState } from 'react';
import { ChevronDown } from 'lucide-react';
import { cn } from '../../lib/utils';
import { getImageUrl } from '../../lib/image';
import { useSeasonEpisodes } from '../../hooks/useMediaDetails';
import type { MergedEpisode, SeasonSummary } from '../../types/library';
import { ManageSubtitleDialogV2 } from '../subtitle/ManageSubtitleDialogV2';
import { EpisodeList, canManageEpisodeSubtitle, episodeCode } from './EpisodeList';

interface SeasonAccordionProps {
  seasons: SeasonSummary[];
  seriesId: string;
  /** Show name. The per-episode subtitle dialog titles itself with the SHOW and
   *  puts SxxExx in a separate code chip (design nodes Aodey + tO72N) — titling
   *  it with the episode name instead loses the show and duplicates the chip. */
  seriesTitle: string;
  tmdbId: number;
  /** Season-list fetch state (Story 12-2 M2) — when the season summaries are
   *  still loading or failed, the accordion shows a skeleton / retry instead of
   *  silently rendering nothing. */
  isLoading?: boolean;
  isError?: boolean;
  onRetry?: () => void;
}

interface SeasonAccordionItemProps {
  season: SeasonSummary;
  seriesId: string;
  seriesTitle: string;
}

function SeasonAccordionItem({ season, seriesId, seriesTitle }: SeasonAccordionItemProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  // 9R-10c: EpisodeList is presentational, so the per-episode dialog state
  // lives here — this is also the level that already knows the seriesId, which
  // is the GLOSSARY key (the dialog's mediaId stays the episode row id).
  const [subtitleEpisode, setSubtitleEpisode] = useState<MergedEpisode | null>(null);

  // Lazy fetch: query stays disabled until the season is expanded (AC #3).
  const { data, isLoading, isError, refetch } = useSeasonEpisodes(
    seriesId,
    season.seasonNumber,
    isExpanded
  );

  const posterUrl = getImageUrl(season.posterPath ?? null, 'w92');
  const contentId = `season-${season.seasonNumber}-content`;
  const seasonName = season.name || `第 ${season.seasonNumber} 季`;

  return (
    <div className="overflow-hidden rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-secondary)]/50">
      <button
        type="button"
        onClick={() => setIsExpanded((prev) => !prev)}
        aria-expanded={isExpanded}
        aria-controls={contentId}
        className="flex w-full items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-[var(--bg-secondary)]"
        data-testid={`season-header-${season.seasonNumber}`}
      >
        {/* Poster thumbnail (56×84) */}
        <div className="h-[84px] w-[56px] shrink-0 overflow-hidden rounded bg-[var(--bg-secondary)]">
          {posterUrl ? (
            <img
              src={posterUrl}
              alt={`${seasonName} 海報`}
              className="h-full w-full object-cover"
              loading="lazy"
            />
          ) : (
            <div className="flex h-full w-full items-center justify-center text-xs text-[var(--text-muted)]">
              無圖
            </div>
          )}
        </div>

        {/* Season meta */}
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <span className="truncate text-base font-semibold text-[var(--text-primary)]">
            {seasonName}
          </span>
          <span className="text-xs text-[var(--text-secondary)]">
            {season.episodeCount ? `${season.episodeCount} 集` : '劇集數未知'}
            {season.airDate ? ` · ${season.airDate}` : ''}
          </span>
        </div>

        <ChevronDown
          className={cn(
            'h-5 w-5 shrink-0 text-[var(--text-secondary)] transition-transform',
            isExpanded && 'rotate-180'
          )}
          aria-hidden="true"
        />
      </button>

      {isExpanded && (
        <div id={contentId} className="border-t border-[var(--border-subtle)]">
          <EpisodeList
            episodes={data?.episodes ?? []}
            seasonNumber={season.seasonNumber}
            isLoading={isLoading}
            isError={isError}
            onRetry={() => refetch()}
            onManageSubtitle={setSubtitleEpisode}
          />
        </div>
      )}

      {/* 9R-10c: mediaId is the EPISODE row id (the transcribe target);
          glossaryMediaId is the SERIES id (the glossary is per-show).
          subtitleTracks is deliberately NOT passed — episodes carry no embedded
          track data and probing each one would turn a season expand into a disk
          storm (red line 3); the authoritative subtitleStatus wins anyway.
          onGenerationComplete uses THIS query's own refetch — narrower than
          invalidating by key, and it needs no QueryClient of its own. */}
      {subtitleEpisode && canManageEpisodeSubtitle(subtitleEpisode) && (
        <ManageSubtitleDialogV2
          mediaId={subtitleEpisode.episodeId}
          mediaType="episode"
          glossaryMediaId={seriesId}
          mediaTitle={seriesTitle}
          mediaCode={episodeCode(season.seasonNumber, subtitleEpisode.episodeNumber)}
          mediaFilePath={subtitleEpisode.filePath}
          subtitleStatus={subtitleEpisode.subtitleStatus}
          subtitleLanguage={subtitleEpisode.subtitleLanguage}
          open={true}
          onOpenChange={(next) => !next && setSubtitleEpisode(null)}
          onGenerationComplete={() => refetch()}
        />
      )}
    </div>
  );
}

export function SeasonAccordion({
  seasons,
  seriesId,
  seriesTitle,
  tmdbId,
  isLoading,
  isError,
  onRetry,
}: SeasonAccordionProps) {
  // AC #1: the accordion is only shown for TMDb-linked series — episode data
  // is resolved from TMDb, so without a tmdb_id there is nothing to expand.
  if (tmdbId <= 0) return null;

  // M2: the season-list fetch itself can be loading or fail. Mirror AC #7's
  // graceful-degradation contract for the season list (not just per-season
  // episodes) so a transient failure shows a retry instead of nothing.
  if (isLoading) {
    return (
      <section aria-label="季與劇集" className="flex flex-col gap-3" data-testid="season-accordion">
        <h2 className="text-lg font-semibold text-[var(--text-primary)]">季與劇集</h2>
        <ul className="flex flex-col gap-3" data-testid="season-accordion-skeleton">
          {[0, 1, 2].map((i) => (
            <li
              key={i}
              className="h-[60px] animate-pulse rounded-lg bg-[var(--bg-secondary)]/50"
              aria-hidden="true"
            />
          ))}
        </ul>
      </section>
    );
  }

  if (isError) {
    return (
      <section aria-label="季與劇集" className="flex flex-col gap-3" data-testid="season-accordion">
        <h2 className="text-lg font-semibold text-[var(--text-primary)]">季與劇集</h2>
        <div
          role="alert"
          className="flex flex-col items-center gap-3 rounded-lg border border-[var(--border-subtle)] px-4 py-6 text-center"
          data-testid="season-accordion-error"
        >
          <p className="text-sm text-[var(--text-secondary)]">無法載入季列表,請稍後再試。</p>
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
      </section>
    );
  }

  if (!seasons || seasons.length === 0) return null;

  return (
    <section aria-label="季與劇集" className="flex flex-col gap-3" data-testid="season-accordion">
      <h2 className="text-lg font-semibold text-[var(--text-primary)]">季與劇集</h2>
      {seasons.map((season) => (
        <SeasonAccordionItem
          key={season.seasonNumber}
          season={season}
          seriesId={seriesId}
          seriesTitle={seriesTitle}
        />
      ))}
    </section>
  );
}
