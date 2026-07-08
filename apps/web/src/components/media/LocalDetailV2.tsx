// Implements: Component/Detail-Movie-v2 (uRGu2) + Component/Detail-TV-v2 (N2fmG6)
/**
 * v2 library detail page (UX Redesign Phase 2 — UX2-3). The pilot's most
 * satisfying surface (perfect zh-TW metadata) made the most capable. Backdrop
 * hero + N1 status + RESTORED actions, then overview / 檔案資訊 / cast / trailer /
 * providers / recommendations / Douban — each Epic-12 section reused and failing
 * soft independently (F3, preserved through the restyle). TV adds the season
 * accordion (AC #5).
 *
 * Actions (Rule 24 / brief P8 — capability verified): Vido has NO playback path
 * (no in-app player / external-player / Plex-Jellyfin deep-link / file-serve), so
 * there is NO 播放 button. The real capabilities are surfaced instead: primary
 * 管理字幕 (the subtitle differentiator, gated on a local filePath), secondary
 * 修改資訊, and 複製檔案路徑.
 */
import { useState, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { Pencil, Subtitles, Copy, Check } from 'lucide-react';
import {
  detailKeys,
  useLocalMovieDetails,
  useLocalSeriesDetails,
  useMovieCredits,
  useTVShowCredits,
  useSeriesSeasons,
  useRecommendations,
  useWatchProviders,
} from '../../hooks/useMediaDetails';
import { libraryKeys } from '../../hooks/useLibrary';
import { useDoubanRating } from '../../hooks/useDoubanRating';
import { useDoubanReviewSummary } from '../../hooks/useDoubanReviewSummary';
import { CreditsSection } from './CreditsSection';
import { SeasonAccordion } from './SeasonAccordion';
import { RelatedContent } from './RelatedContent';
import { StreamingAvailability } from './StreamingAvailability';
import { TrailerSection } from './TrailerSection';
import { DoubanSection } from './DoubanSection';
import { DualRatingDisplay } from './DualRatingDisplay';
import { MetadataEditorDialog } from '../metadata-editor';
import type { MediaMetadata } from '../metadata-editor';
import { ManageSubtitleDialogV2 } from '../subtitle/ManageSubtitleDialogV2';
import { DetailHeroV2 } from './DetailHeroV2';
import { DetailTechInfoV2 } from './DetailTechInfoV2';
import { DetailSkeletonV2, DetailNotFoundV2 } from './DetailStatesV2';
import { deriveLifecycleStatus, deriveSubtitleStatus } from '../../utils/libraryStatus';

const WATCH_REGION = 'TW';

export function LocalDetailV2({ type, id }: { type: 'movie' | 'tv'; id: string }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const isMovie = type === 'movie';
  const [editorOpen, setEditorOpen] = useState(false);
  const [subtitleOpen, setSubtitleOpen] = useState(false);
  const [copied, setCopied] = useState(false);

  const localMovie = useLocalMovieDetails(isMovie ? id : '');
  const localSeries = useLocalSeriesDetails(!isMovie ? id : '');
  const data = isMovie ? localMovie.data : localSeries.data;
  const isLoading = isMovie ? localMovie.isLoading : localSeries.isLoading;
  const isError = isMovie ? localMovie.isError : localSeries.isError;

  // §9b CN-subtitle policy source (movies only; series has no production_countries).
  // Flatten to the comma-joined ISO string ManageSubtitleDialogV2 expects, mirroring
  // MediaDetailPanel's TMDb path. disc-2026-07-production-countries-detail-api.
  const productionCountryStr =
    localMovie.data?.productionCountries?.map((c) => c.iso31661).join(',') ?? '';

  const tmdbId = data?.tmdbId ?? 0;
  const movieCredits = useMovieCredits(isMovie && tmdbId > 0 ? tmdbId : 0);
  const tvCredits = useTVShowCredits(!isMovie && tmdbId > 0 ? tmdbId : 0);
  const credits = isMovie ? movieCredits : tvCredits;
  // disc-2026-07-credits-spoken-languages-persist: a manual metadata edit is an intentional
  // override, so prefer the persisted local credits when metadataSource === 'manual';
  // otherwise fall back to the live TMDb credits (credits.data). data?.credits is the local
  // movie/series payload — absent for never-edited items, so TMDb wins by default.
  const effectiveCredits =
    data?.metadataSource === 'manual' && data.credits ? data.credits : credits.data;
  const douban = useDoubanRating(id, isMovie ? 'movie' : 'series', tmdbId > 0);
  const doubanReview = useDoubanReviewSummary(
    id,
    isMovie ? 'movie' : 'series',
    Boolean(douban.data?.doubanId)
  );
  const seasons = useSeriesSeasons(id, !isMovie && tmdbId > 0);
  const recs = useRecommendations(tmdbId, isMovie ? 'movie' : 'tv', tmdbId > 0);
  const watch = useWatchProviders(tmdbId, isMovie ? 'movie' : 'tv', tmdbId > 0, WATCH_REGION);

  const onBack = useCallback(() => navigate({ to: '/library' }), [navigate]);

  // AC 6 lifecycle consistency (N1): on transcription_complete, refetch the media
  // detail + library lists so poster badges (deriveSubtitleStatus) refresh without
  // reload. NOTE (annotated 2026-07-05): the transcription path does not yet write
  // movies.subtitle_status/subtitle_language — until 9R-16 AC 12 lands this refetches
  // unchanged data and the badge stays 缺字幕 (a rescan fixes it). Correct FE
  // behavior as specced; NO client-side badge override.
  const onGenerationComplete = useCallback(() => {
    queryClient.invalidateQueries({
      queryKey: isMovie ? detailKeys.localMovie(id) : detailKeys.localSeries(id),
    });
    queryClient.invalidateQueries({ queryKey: libraryKeys.all });
  }, [queryClient, isMovie, id]);

  const buildEditorMetadata = useCallback((): MediaMetadata | null => {
    if (!data) return null;
    const date = isMovie ? localMovie.data?.releaseDate : localSeries.data?.firstAirDate;
    return {
      id,
      mediaType: isMovie ? 'movie' : 'series',
      title: data.title,
      titleEnglish: data.originalTitle,
      year: parseInt(date?.slice(0, 4) || '0', 10),
      genres: data.genres || [],
      director: isMovie
        ? effectiveCredits?.crew?.find((c) => c.job === 'Director')?.name
        : undefined,
      cast: effectiveCredits?.cast?.slice(0, 10).map((c) => c.name) || [],
      overview: data.overview,
    };
  }, [data, isMovie, id, effectiveCredits, localMovie.data, localSeries.data]);

  if (isLoading) return <DetailSkeletonV2 />;
  if (isError || !data) return <DetailNotFoundV2 onBack={onBack} />;

  const date = isMovie ? localMovie.data?.releaseDate : localSeries.data?.firstAirDate;
  const year = date?.slice(0, 4);
  const runtimeMeta = isMovie
    ? localMovie.data?.runtime
      ? `${localMovie.data.runtime} 分`
      : null
    : localSeries.data?.numberOfSeasons
      ? `${localSeries.data.numberOfSeasons} 季 · ${localSeries.data.numberOfEpisodes ?? '?'} 集`
      : null;
  const director = isMovie ? effectiveCredits?.crew?.find((c) => c.job === 'Director') : undefined;
  const filePath = data.filePath;
  const editorMetadata = buildEditorMetadata();

  const copyPath = () => {
    if (!filePath) return;
    navigator.clipboard?.writeText(filePath).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  };

  const actions = (
    <>
      {filePath && (
        <button
          type="button"
          onClick={() => setSubtitleOpen(true)}
          data-testid="action-manage-subtitle"
          className="flex min-h-[44px] items-center gap-2 rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-4 text-sm font-medium text-[var(--text-on-accent)] transition-colors hover:bg-[var(--accent-pressed)]"
        >
          <Subtitles className="h-4 w-4" aria-hidden="true" />
          管理字幕
        </button>
      )}
      <button
        type="button"
        onClick={() => setEditorOpen(true)}
        data-testid="action-edit-metadata"
        className="flex min-h-[44px] items-center gap-2 rounded-[var(--radius-md)] bg-[var(--bg-secondary)] px-4 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-tertiary)]"
      >
        <Pencil className="h-4 w-4" aria-hidden="true" />
        修改資訊
      </button>
      {filePath && (
        <button
          type="button"
          onClick={copyPath}
          aria-label="複製檔案路徑"
          data-testid="action-copy-path"
          className="flex min-h-[44px] w-11 items-center justify-center rounded-[var(--radius-md)] bg-[var(--bg-secondary)] text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]"
        >
          {copied ? (
            <Check className="h-4 w-4 text-[var(--success)]" />
          ) : (
            <Copy className="h-4 w-4" />
          )}
        </button>
      )}
    </>
  );

  const meta = (
    <>
      {year && <span className="font-mono">{year}</span>}
      {runtimeMeta && <span className="font-mono">{runtimeMeta}</span>}
      {data.genres?.length > 0 && <span>{data.genres.join(' / ')}</span>}
      <DualRatingDisplay
        tmdbRating={data.voteAverage}
        tmdbVoteCount={data.voteCount}
        doubanRating={douban.data?.doubanRating}
        doubanVoteCount={douban.data?.doubanVoteCount}
        doubanLoading={douban.isLoading}
      />
    </>
  );

  return (
    <div className="min-h-screen bg-[var(--bg-primary)]" data-testid="local-detail-v2">
      <DetailHeroV2
        backdropPath={data.backdropPath}
        posterPath={data.posterPath}
        title={data.title}
        originalTitle={data.originalTitle}
        badges={[deriveLifecycleStatus(data), deriveSubtitleStatus(data)]}
        meta={meta}
        actions={actions}
        onBack={onBack}
      />

      <div className="mx-auto max-w-5xl space-y-8 px-4 pb-16 pt-8 sm:px-8">
        {data.overview && (
          <section data-testid="detail-overview">
            <h2 className="mb-2 text-lg font-semibold text-[var(--text-primary)]">簡介</h2>
            <p className="leading-relaxed text-[var(--text-secondary)]">{data.overview}</p>
          </section>
        )}

        {/* TV: seasons/episodes are the core series content → placed high, right
            after the overview (matches the .pen TV detail N2fmG6 body order). */}
        {!isMovie && (
          <SeasonAccordion
            seasons={seasons.data ?? []}
            seriesId={id}
            tmdbId={tmdbId}
            isLoading={seasons.isLoading}
            isError={seasons.isError}
            onRetry={() => seasons.refetch()}
          />
        )}

        <DetailTechInfoV2
          videoResolution={data.videoResolution}
          videoCodec={data.videoCodec}
          audioCodec={data.audioCodec}
          audioChannels={data.audioChannels}
          hdrFormat={data.hdrFormat}
          subtitleTracks={data.subtitleTracks}
          fileSize={data.fileSize}
          filePath={filePath}
        />

        {effectiveCredits && (
          <CreditsSection director={director} cast={effectiveCredits.cast?.slice(0, 8)} />
        )}

        {tmdbId > 0 && <TrailerSection tmdbId={tmdbId} type={type} title={data.title} />}

        {tmdbId > 0 && (
          <StreamingAvailability
            region={watch.data?.results?.[WATCH_REGION]}
            isLoading={watch.isLoading}
            isError={watch.isError}
            onRetry={() => watch.refetch()}
          />
        )}

        {douban.data?.doubanId && (
          <DoubanSection
            doubanId={douban.data.doubanId}
            summary={doubanReview.data}
            isLoading={doubanReview.isLoading}
            isError={doubanReview.isError}
          />
        )}

        {tmdbId > 0 && (
          <RelatedContent
            items={recs.data?.results ?? []}
            isLoading={recs.isLoading}
            isError={recs.isError}
            onRetry={() => recs.refetch()}
          />
        )}
      </div>

      {editorMetadata && (
        <MetadataEditorDialog
          isOpen={editorOpen}
          onClose={() => setEditorOpen(false)}
          mediaId={id}
          mediaType={isMovie ? 'movie' : 'series'}
          initialData={editorMetadata}
          onSuccess={() => (isMovie ? localMovie.refetch() : localSeries.refetch())}
        />
      )}

      {filePath && (
        <ManageSubtitleDialogV2
          mediaId={id}
          mediaType={isMovie ? 'movie' : 'series'}
          mediaTitle={data.title}
          mediaFilePath={filePath}
          mediaResolution={data.videoResolution}
          productionCountry={productionCountryStr}
          subtitleTracks={data.subtitleTracks}
          subtitleStatus={data.subtitleStatus}
          subtitleLanguage={data.subtitleLanguage}
          open={subtitleOpen}
          onOpenChange={setSubtitleOpen}
          onGenerationComplete={onGenerationComplete}
          onDownloadSuccess={() => (isMovie ? localMovie.refetch() : localSeries.refetch())}
        />
      )}
    </div>
  );
}
