// Design ref: ux-design.pen Screen B3p-D (uRGu2) + Screen B4p-D (N2fmG6) + Screen B3p-M (SzNRb) + Screen B10p-D (p3qEc) + Screen B11p-D (V56cx)
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
import { libraryKeys, useReparseItem } from '../../hooks/useLibrary';
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
import { NfoLocalizeAction } from './NfoLocalizeAction';
import { DetailHeroV2 } from './DetailHeroV2';
import { DetailTechInfoV2 } from './DetailTechInfoV2';
import { DetailNoMetadataV2, type NoMetadataVariant } from './DetailNoMetadataV2';
import { ManualMatchDialogV2 } from './ManualMatchDialogV2';
import { cleanFilenameForSearch } from '../../utils/cleanFilenameForSearch';
import { DetailSkeletonV2, DetailNotFoundV2, DetailLoadErrorV2 } from './DetailStatesV2';
import { isNotFoundError } from '../../lib/apiError';
import { TmdbAttribution } from '../ui/TmdbAttribution';
import { deriveLifecycleStatus, deriveSubtitleStatus } from '../../utils/libraryStatus';

const WATCH_REGION = 'TW';

// 9R-13b / 9R-UX: four actions. On mobile they wrap to two rows of two equal
// halves (B3p-M) — every one keeps its text label, because an icon-only button
// that overwrites a file is a guessing game. From `sm` up they sit on one row
// at their natural widths (B3p-D / B4p-D).
const actionBasis =
  'flex min-h-[44px] grow basis-[calc(50%-0.25rem)] items-center sm:grow-0 sm:basis-auto';

export function LocalDetailV2({ type, id }: { type: 'movie' | 'tv'; id: string }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const isMovie = type === 'movie';
  const [editorOpen, setEditorOpen] = useState(false);
  const [subtitleOpen, setSubtitleOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const [matchOpen, setMatchOpen] = useState(false);
  const reparse = useReparseItem();

  const localMovie = useLocalMovieDetails(isMovie ? id : '');
  const localSeries = useLocalSeriesDetails(!isMovie ? id : '');
  const data = isMovie ? localMovie.data : localSeries.data;
  const isLoading = isMovie ? localMovie.isLoading : localSeries.isLoading;
  const isError = isMovie ? localMovie.isError : localSeries.isError;
  const loadError = isMovie ? localMovie.error : localSeries.error;

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
  // 短評摘要已停用(⚖️ Alexyu 2026-08-30,bugfix-douban-sec-gate-liveness):
  // movie.douban.com 對匿名爬蟲 18/19 回 302 → sec.douban.com,每次開頁都是
  // 一次註定失敗的爬蟲 + 骨架閃爍。停打 API、只留評分與外連;豆瓣通了以後
  // 把 useDoubanReviewSummary 的呼叫接回來即可(hook 與後端端點都保留)。
  const doubanReview = useDoubanReviewSummary(id, isMovie ? 'movie' : 'series', false);
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
  // dsr-2 AC #8: only a real 404 means "removed"; any other failure is a failed load —
  // the item (and its files) may be perfectly fine. Only when there is NOTHING to
  // show: React Query keeps cached data when a background refetch fails (e.g. the
  // refetch after subtitle generation hits a locked DB), and the loaded page — plus
  // any open dialog — must survive that.
  if (!data) {
    if (isError && !isNotFoundError(loadError)) {
      const query = isMovie ? localMovie : localSeries;
      return (
        <DetailLoadErrorV2
          onBack={onBack}
          onRetry={() => query.refetch()}
          retrying={query.isFetching}
          code={(loadError as { code?: string } | null)?.code}
          reassureFiles
        />
      );
    }
    return <DetailNotFoundV2 onBack={onBack} />;
  }

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

  // dsr-2b-b AC #2 / #3: read from parseStatus, never from tmdbId — a Douban/NFO
  // match or a manual edit has no tmdb id and IS matched. '' (API-created rows,
  // usually already matched) is not "pending" either.
  const noMetadata: NoMetadataVariant | null =
    data.parseStatus === 'failed' ? 'failed' : data.parseStatus === 'pending' ? 'pending' : null;
  const mediaKind = isMovie ? 'movie' : 'series';
  // Only a re-match for THIS item speaks here (the hook instance survives
  // navigation between detail pages).
  const ownRematch = reparse.variables?.id === id;
  // Only once the page itself reads failed: between the re-match answering and the
  // detail refetch landing, a pending block would otherwise say "still not found"
  // under 「資料還在整理」.
  const lastRematch =
    noMetadata === 'failed' &&
    ownRematch &&
    !reparse.isPending &&
    reparse.data?.parseStatus === 'failed'
      ? 'still-failed'
      : null;

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
          // One solid accent per screen: with a no-metadata block the primary
          // action is 手動選片 / 立即比對 (dsr-2b-b B10p-D / B11p-D).
          className={
            noMetadata
              ? `${actionBasis} justify-center gap-2 rounded-[var(--radius-md)] bg-[var(--bg-secondary)] px-4 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-tertiary)]`
              : `${actionBasis} justify-center gap-2 rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-4 text-sm font-medium text-[var(--text-on-accent)] transition-colors hover:bg-[var(--accent-pressed)]`
          }
        >
          <Subtitles className="h-4 w-4" aria-hidden="true" />
          管理字幕
        </button>
      )}
      <button
        type="button"
        onClick={() => setEditorOpen(true)}
        data-testid="action-edit-metadata"
        className={`${actionBasis} order-3 justify-center gap-2 rounded-[var(--radius-md)] bg-[var(--bg-secondary)] px-4 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-tertiary)] sm:order-2`}
      >
        <Pencil className="h-4 w-4" aria-hidden="true" />
        修改資訊
      </button>
      {/* 9R-13b: order-2 on mobile so the two rows read
          「管理字幕｜在地化資訊」/「修改資訊｜複製路徑」 per the B3p-M design;
          sm:order-3 restores the desktop order of B3p-D / B4p-D. */}
      {/* dsr-2b-b AC #5: without metadata the "title" is the file name, and
          localizing would translate it into the NFO. */}
      {!noMetadata && (
        <NfoLocalizeAction
          mediaType={type}
          id={id}
          hasFilePath={Boolean(filePath)}
          className={`${actionBasis} order-2 justify-center sm:order-3`}
        />
      )}
      {filePath && (
        <button
          type="button"
          onClick={copyPath}
          aria-label="複製檔案路徑"
          data-testid="action-copy-path"
          className={`${actionBasis} order-4 justify-center gap-2 rounded-[var(--radius-md)] bg-[var(--bg-secondary)] text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)] sm:w-11 sm:grow-0 sm:basis-auto sm:px-0`}
        >
          {copied ? (
            <Check className="h-4 w-4 text-[var(--success-text)]" />
          ) : (
            <Copy className="h-4 w-4" />
          )}
          <span className="text-sm font-medium sm:hidden">複製路徑</span>
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

  const techInfoBlock = (
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
  );
  const creditsBlock = effectiveCredits ? (
    <CreditsSection director={director} cast={effectiveCredits.cast?.slice(0, 8)} />
  ) : null;

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
        {noMetadata && (
          <DetailNoMetadataV2
            variant={noMetadata}
            mediaType={mediaKind}
            onManualMatch={() => setMatchOpen(true)}
            onRematch={() => reparse.mutate({ type: mediaKind, id })}
            rematching={ownRematch && reparse.isPending}
            lastRematch={lastRematch}
            rematchError={ownRematch && !reparse.isPending ? reparse.error : null}
          />
        )}

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
            seriesTitle={data.title}
            tmdbId={tmdbId}
            isLoading={seasons.isLoading}
            isError={seasons.isError}
            onRetry={() => seasons.refetch()}
          />
        )}

        {/* Movie: cast before file facts (B3p-D, ux2-3 AC #4). Series keeps file facts
            above cast — the season list already sits between overview and both. */}
        {isMovie ? (
          <>
            {creditsBlock}
            {techInfoBlock}
          </>
        ) : (
          <>
            {techInfoBlock}
            {creditsBlock}
          </>
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

        {/* 相關推薦 before 豆瓣 (B8p-D, ux2-3 AC #4). */}
        {tmdbId > 0 && (
          <RelatedContent
            items={recs.data?.results ?? []}
            isLoading={recs.isLoading}
            isError={recs.isError}
            onRetry={() => recs.refetch()}
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

        {/* TMDB API Terms of Use §3 (sub-6-9). Gated on tmdbId because an
            unmatched local file shows no TMDB data at all — attributing a
            source we did not use would be its own kind of lie. Page footer
            rather than inside StreamingAvailability: see TMDbDetailV2. */}
        {tmdbId > 0 && <TmdbAttribution variant="inline" />}
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

      {noMetadata && (
        <ManualMatchDialogV2
          open={matchOpen}
          onOpenChange={setMatchOpen}
          mediaId={id}
          mediaType={mediaKind}
          // Movies: the file's own name — after an edit or a wrong match the title is
          // no longer what the file is called. Series: the title — a series
          // file_path is its FOLDER (the library root in a flat layout), and the
          // backend's own series re-match uses the title for the same reason.
          initialQuery={cleanFilenameForSearch(isMovie ? filePath || data.title : data.title)}
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
