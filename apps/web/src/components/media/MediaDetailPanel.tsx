import { useState, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { getImageUrl } from '../../lib/image';
import { TrailerEmbed } from './TrailerEmbed';
import { MetadataSourceBadge } from './MetadataSourceBadge';
import { FileInfo } from './FileInfo';
import { DetailPanelMenu } from './DetailPanelMenu';
import { SubtitleSearchDialog } from '../subtitle/SubtitleSearchDialog';
import { useMediaTrailers, libraryKeys } from '../../hooks/useLibrary';
import type { MovieDetails, TVShowDetails, Credits } from '../../types/tmdb';
import type { TMDbVideo } from '../../types/library';

export interface MediaDetailPanelProps {
  type: 'movie' | 'tv';
  details: MovieDetails | TVShowDetails | null;
  credits?: Credits | null;
  isLoading?: boolean;
  libraryId?: string;
  metadataSource?: string;
  filePath?: string;
  fileSize?: number;
  createdAt?: string;
  onPlay?: () => void;
  onAddToList?: () => void;
  onReparse?: () => void;
  onExport?: () => void;
  onDelete?: () => void;
}

export function MediaDetailPanel({
  type,
  details,
  credits,
  isLoading,
  libraryId,
  metadataSource,
  filePath,
  fileSize,
  createdAt,
  onPlay,
  onAddToList,
  onReparse,
  onExport,
  onDelete,
}: MediaDetailPanelProps) {
  // Hooks must be called before any early return (rules-of-hooks)
  const queryClient = useQueryClient();
  const [subtitleDialogOpen, setSubtitleDialogOpen] = useState(false);
  const handleSubtitleDownloadSuccess = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: libraryKeys.all });
  }, [queryClient]);

  if (isLoading || !details) {
    return <MediaDetailSkeleton />;
  }

  const isMovie = type === 'movie';
  const movie = isMovie ? (details as MovieDetails) : null;
  const tvShow = !isMovie ? (details as TVShowDetails) : null;

  const title = isMovie ? movie!.title : tvShow!.name;
  const originalTitle = isMovie ? movie!.originalTitle : tvShow!.originalName;

  const year = isMovie ? movie!.releaseDate?.slice(0, 4) : tvShow!.firstAirDate?.slice(0, 4);
  const runtime = isMovie ? movie!.runtime : tvShow!.episodeRunTime?.[0];

  const posterUrl = getImageUrl(details.posterPath, 'w500');
  const backdropUrl = getImageUrl(details.backdropPath, 'w780');

  const director = credits?.crew?.find((c) => c.job === 'Director');
  const topCast = credits?.cast?.slice(0, 5) ?? [];

  const hasContextMenu = onReparse && onExport && onDelete;

  const productionCountryStr = details.productionCountries?.map((c) => c.iso31661).join(',') ?? '';

  return (
    <div className="flex flex-col" data-testid="media-detail-panel">
      {/* Backdrop header */}
      {backdropUrl && (
        <div className="relative h-48 w-full">
          <img src={backdropUrl} alt="" className="h-full w-full object-cover" loading="lazy" />
          <div className="absolute inset-0 bg-gradient-to-t from-[var(--bg-primary)] to-transparent" />
        </div>
      )}

      <div className="-mt-12 relative z-10 p-4">
        {/* Context menu */}
        {hasContextMenu && (
          <div className="mb-2 flex justify-end">
            <DetailPanelMenu onReparse={onReparse} onExport={onExport} onDelete={onDelete} />
          </div>
        )}

        {/* Poster and basic info */}
        <div className="flex gap-4">
          {posterUrl && (
            <img
              src={posterUrl}
              alt={title}
              className="h-48 w-32 flex-shrink-0 rounded-lg object-cover shadow-lg"
              loading="lazy"
              data-testid="detail-poster"
            />
          )}
          <div className="min-w-0 flex-1">
            <h1 className="text-xl font-bold text-white" data-testid="detail-title">
              {title}
            </h1>
            {originalTitle && originalTitle !== title && (
              <p
                className="truncate text-sm text-[var(--text-secondary)]"
                data-testid="detail-original-title"
              >
                {originalTitle}
              </p>
            )}

            {/* Year, runtime, rating */}
            <div className="mt-2 flex flex-wrap items-center gap-3 text-sm text-[var(--text-secondary)]">
              {year && <span data-testid="detail-year">{year}</span>}
              {runtime && runtime > 0 && <span data-testid="detail-runtime">{runtime} 分鐘</span>}
              {details.voteAverage > 0 && (
                <span
                  className="flex items-center gap-1 text-[var(--warning)]"
                  data-testid="detail-rating"
                >
                  ⭐ {details.voteAverage.toFixed(1)}
                </span>
              )}
            </div>

            {/* Genre tags */}
            <div className="mt-3 flex flex-wrap gap-2" data-testid="detail-genres">
              {details.genres?.map((genre) => (
                <span
                  key={genre.id}
                  className="rounded-full bg-[var(--bg-secondary)] px-3 py-1 text-xs text-[var(--text-secondary)]"
                >
                  {genre.name}
                </span>
              ))}
            </div>

            {/* Metadata source badge (AC3) */}
            {metadataSource && (
              <div className="mt-2">
                <MetadataSourceBadge source={metadataSource} fetchDate={createdAt} />
              </div>
            )}
          </div>
        </div>

        {/* Overview */}
        <div className="mt-6">
          <h3 className="mb-2 text-sm font-semibold text-[var(--text-secondary)]">劇情簡介</h3>
          <p
            className="text-sm leading-relaxed text-[var(--text-secondary)]"
            data-testid="detail-overview"
          >
            {details.overview || '暫無簡介'}
          </p>
        </div>

        {/* Director / Created by */}
        {director && (
          <div className="mt-4">
            <span className="text-sm text-[var(--text-secondary)]">導演：</span>
            <span className="ml-2 text-sm text-white">{director.name}</span>
          </div>
        )}
        {tvShow?.createdBy && tvShow.createdBy.length > 0 && (
          <div className="mt-4">
            <span className="text-sm text-[var(--text-secondary)]">創作者：</span>
            <span className="ml-2 text-sm text-white">
              {tvShow.createdBy.map((c) => c.name).join(', ')}
            </span>
          </div>
        )}

        {/* Cast (AC2) */}
        {topCast.length > 0 && (
          <div className="mt-4" data-testid="detail-cast">
            <span className="text-sm text-[var(--text-secondary)]">主演：</span>
            <span className="ml-2 text-sm text-white">{topCast.map((c) => c.name).join('、')}</span>
          </div>
        )}

        {/* CTA Buttons (AC1) */}
        {(onPlay || onAddToList) && (
          <div className="mt-6 flex gap-3" data-testid="detail-cta-buttons">
            {onPlay && (
              <button
                onClick={onPlay}
                className="flex-1 rounded-lg bg-emerald-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-emerald-500"
                data-testid="detail-play-button"
              >
                播放
              </button>
            )}
            {onAddToList && (
              <button
                onClick={onAddToList}
                className="flex-1 rounded-lg border border-[var(--accent-primary)] px-4 py-2.5 text-sm font-medium text-[var(--accent-primary)] transition-colors hover:bg-[var(--accent-hover)]/10"
                data-testid="detail-add-to-list-button"
              >
                加入清單
              </button>
            )}
          </div>
        )}

        {/* Search Subtitles button (Story 8-8 AC #1) */}
        {filePath && (
          <div className="mt-3">
            <button
              onClick={() => setSubtitleDialogOpen(true)}
              className="w-full rounded-lg border border-[var(--border-subtle)] px-4 py-2 text-sm font-medium text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)]"
              data-testid="search-subtitles-button"
            >
              搜尋字幕
            </button>
          </div>
        )}

        {/* Subtitle Search Dialog (Story 8-8) */}
        <SubtitleSearchDialog
          mediaId={libraryId ?? ''}
          mediaType={type === 'movie' ? 'movie' : 'series'}
          mediaTitle={title}
          mediaFilePath={filePath ?? ''}
          productionCountry={productionCountryStr}
          open={subtitleDialogOpen}
          onOpenChange={setSubtitleDialogOpen}
          onDownloadSuccess={handleSubtitleDownloadSuccess}
        />

        {/* TV Show enhanced details (AC5) */}
        {tvShow && <TVShowSeasons tvShow={tvShow} />}

        {/* File info section (AC4) */}
        {(filePath || fileSize) && (
          <div className="mt-6">
            <h3 className="mb-2 text-sm font-semibold text-[var(--text-secondary)]">檔案資訊</h3>
            <FileInfo filePath={filePath} fileSize={fileSize} />
          </div>
        )}

        {/* Date added */}
        {createdAt && (
          <div className="mt-4 text-sm text-[var(--text-muted)]" data-testid="detail-date-added">
            加入日期：{new Date(createdAt).toLocaleDateString('zh-TW')}
          </div>
        )}

        {/* Trailer section (AC2) */}
        {libraryId && (
          <TrailerSection
            type={type === 'movie' ? 'movie' : 'series'}
            id={libraryId}
            title={title}
          />
        )}
      </div>
    </div>
  );
}

function TrailerSection({
  type,
  id,
  title,
}: {
  type: 'movie' | 'series';
  id: string;
  title: string;
}) {
  const [showTrailers, setShowTrailers] = useState(false);
  const { data: videosData } = useMediaTrailers(type, id, showTrailers);

  const trailers: TMDbVideo[] =
    videosData?.results?.filter(
      (v) => v.site === 'YouTube' && (v.type === 'Trailer' || v.type === 'Teaser')
    ) ?? [];

  if (!showTrailers) {
    return (
      <div className="mt-6">
        <button
          onClick={() => setShowTrailers(true)}
          className="flex w-full items-center justify-center gap-2 rounded-lg bg-[var(--bg-secondary)] px-4 py-3 text-sm text-white transition-colors hover:bg-[var(--bg-tertiary)]"
          data-testid="load-trailers-button"
        >
          <span className="text-lg">▶</span>
          觀看預告片
        </button>
      </div>
    );
  }

  if (trailers.length === 0 && videosData) {
    return (
      <div className="mt-6 text-center text-sm text-[var(--text-muted)]" data-testid="no-trailers">
        暫無預告片
      </div>
    );
  }

  return (
    <div className="mt-6 space-y-3" data-testid="trailer-section">
      <h3 className="text-sm font-semibold text-[var(--text-secondary)]">預告片</h3>
      {trailers.slice(0, 3).map((trailer) => (
        <TrailerEmbed
          key={trailer.id}
          videoKey={trailer.key}
          title={`${title} - ${trailer.name}`}
        />
      ))}
    </div>
  );
}

function TVShowSeasons({ tvShow }: { tvShow: TVShowDetails }) {
  if (!tvShow.seasons || tvShow.seasons.length === 0) return null;

  return (
    <div className="mt-6" data-testid="tv-seasons">
      <h3 className="mb-2 text-sm font-semibold text-[var(--text-secondary)]">
        季數資訊 ({tvShow.numberOfSeasons} 季 · {tvShow.numberOfEpisodes} 集)
      </h3>
      {tvShow.productionCountries && tvShow.productionCountries.length > 0 && (
        <p className="mb-2 text-xs text-[var(--text-muted)]">
          製作國家：{tvShow.productionCountries.map((c) => c.name).join(', ')}
        </p>
      )}
      <div className="space-y-1">
        {tvShow.seasons
          .filter((s) => s.seasonNumber > 0)
          .map((season) => (
            <div
              key={season.id}
              className="flex items-center justify-between rounded bg-[var(--bg-secondary)]/50 px-3 py-1.5 text-sm"
            >
              <span className="text-[var(--text-secondary)]">{season.name}</span>
              <span className="text-[var(--text-muted)]">{season.episodeCount} 集</span>
            </div>
          ))}
      </div>
    </div>
  );
}

function MediaDetailSkeleton() {
  return (
    <div className="animate-pulse" data-testid="media-detail-skeleton">
      <div className="h-48 w-full bg-[var(--bg-tertiary)]" />
      <div className="p-4">
        <div className="flex gap-4">
          <div className="h-48 w-32 flex-shrink-0 rounded-lg bg-[var(--bg-tertiary)]" />
          <div className="flex-1 space-y-3">
            <div className="h-6 w-3/4 rounded bg-[var(--bg-tertiary)]" />
            <div className="h-4 w-1/2 rounded bg-[var(--bg-tertiary)]" />
            <div className="flex gap-3">
              <div className="h-4 w-12 rounded bg-[var(--bg-tertiary)]" />
              <div className="h-4 w-16 rounded bg-[var(--bg-tertiary)]" />
              <div className="h-4 w-10 rounded bg-[var(--bg-tertiary)]" />
            </div>
            <div className="flex gap-2">
              <div className="h-6 w-16 rounded-full bg-[var(--bg-tertiary)]" />
              <div className="h-6 w-20 rounded-full bg-[var(--bg-tertiary)]" />
              <div className="h-6 w-14 rounded-full bg-[var(--bg-tertiary)]" />
            </div>
          </div>
        </div>
        <div className="mt-6 space-y-2">
          <div className="h-4 w-20 rounded bg-[var(--bg-tertiary)]" />
          <div className="h-4 w-full rounded bg-[var(--bg-tertiary)]" />
          <div className="h-4 w-full rounded bg-[var(--bg-tertiary)]" />
          <div className="h-4 w-2/3 rounded bg-[var(--bg-tertiary)]" />
        </div>
      </div>
    </div>
  );
}

export default MediaDetailPanel;
