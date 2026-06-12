import { useState, useCallback } from 'react';
import { createFileRoute, notFound, useNavigate } from '@tanstack/react-router';
import { Pencil, ArrowLeft, Film, Loader2 } from 'lucide-react';
import { ColorPlaceholder } from '../../components/media/ColorPlaceholder';
import { FallbackPending } from '../../components/media/FallbackPending';
import { FallbackFailed } from '../../components/media/FallbackFailed';
import { CreditsSection } from '../../components/media/CreditsSection';
import { TechBadgeGroup } from '../../components/media/TechBadgeGroup';
import { DualRatingDisplay } from '../../components/media/DualRatingDisplay';
import { MetadataEditorDialog } from '../../components/metadata-editor';
import type { MediaMetadata } from '../../components/metadata-editor';
import {
  useLocalMovieDetails,
  useLocalSeriesDetails,
  useMovieDetails,
  useTVShowDetails,
  useMovieCredits,
  useTVShowCredits,
  useSeriesSeasons,
  useRecommendations,
  useWatchProviders,
} from '../../hooks/useMediaDetails';
import { SeasonAccordion } from '../../components/media/SeasonAccordion';
import { RelatedContent } from '../../components/media/RelatedContent';
import { StreamingAvailability } from '../../components/media/StreamingAvailability';
import { TrailerSection } from '../../components/media/TrailerSection';
import { DoubanSection } from '../../components/media/DoubanSection';
import { useOwnedMedia } from '../../hooks/useOwnedMedia';
import { useDoubanRating } from '../../hooks/useDoubanRating';
import { useDoubanReviewSummary } from '../../hooks/useDoubanReviewSummary';
import type { MovieDetails, TVShowDetails } from '../../types/tmdb';
import { getImageUrl } from '../../lib/image';
import { cn } from '../../lib/utils';

const validMediaTypes = ['movie', 'tv'] as const;
type ValidMediaType = (typeof validMediaTypes)[number];

// Story 12-4 — single source of truth for the watch-providers region (TW-only for
// Epic 12). Used for the hook arg AND the results-map lookup so they never drift.
const WATCH_REGION = 'TW';

function isValidMediaType(type: string): type is ValidMediaType {
  return validMediaTypes.includes(type as ValidMediaType);
}

export type IdKind = 'local-uuid' | 'tmdb-numeric';

// bugfix-10-1 [@contract-v1] AC #2 — A pure positive-integer string is a TMDb
// numeric ID; everything else (UUIDs, mixed strings) routes through the local
// DB path. Widens bugfix-1 [@contract-v0] (UUID-only) to cover homepage TMDb
// items surfaced by Story 10-3 ExploreBlock.
export function classifyId(id: string): IdKind {
  if (/^\d+$/.test(id) && parseInt(id, 10) > 0) return 'tmdb-numeric';
  return 'local-uuid';
}

export const Route = createFileRoute('/media/$type/$id')({
  loader: async ({ params }) => {
    const { type, id } = params;

    if (!isValidMediaType(type)) {
      throw notFound();
    }

    if (!id || id.trim() === '') {
      throw notFound();
    }

    return {
      type: type as ValidMediaType,
      id,
      idKind: classifyId(id),
    };
  },
  notFoundComponent: NotFoundComponent,
  component: MediaDetailRoute,
});

function NotFoundComponent() {
  const navigate = useNavigate();

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center">
        <h1 className="mb-4 text-4xl font-bold text-white">404</h1>
        <p className="mb-6 text-[var(--text-secondary)]">找不到該媒體內容</p>
        <button
          onClick={() => navigate({ to: '/library' })}
          className="rounded-lg bg-[var(--accent-primary)] px-4 py-2 text-white hover:bg-[var(--accent-pressed)]"
        >
          返回媒體庫
        </button>
      </div>
    </div>
  );
}

function MediaDetailRoute() {
  const { type, id, idKind } = Route.useLoaderData();

  // bugfix-10-1 — Homepage / search PosterCards emit raw TMDb numeric IDs
  // (Story 10-3 ExploreBlock + Story 2-3 search MediaGrid). Those never resolve
  // against /api/v1/movies/:id (UUID-keyed). Branch off to the TMDb-backed
  // detail render and skip the local-DB hooks entirely.
  if (idKind === 'tmdb-numeric') {
    return <TMDbDetailView type={type} tmdbId={parseInt(id, 10)} />;
  }

  return <LocalDetailView type={type} id={id} />;
}

function LocalDetailView({ type, id }: { type: ValidMediaType; id: string }) {
  const navigate = useNavigate();
  const [isEditorOpen, setIsEditorOpen] = useState(false);
  const [showSuccessToast, setShowSuccessToast] = useState(false);

  const isMovie = type === 'movie';

  // Primary: fetch from local DB API
  const localMovie = useLocalMovieDetails(isMovie ? id : '');
  const localSeries = useLocalSeriesDetails(!isMovie ? id : '');
  const localData = isMovie ? localMovie.data : localSeries.data;
  const isLoading = isMovie ? localMovie.isLoading : localSeries.isLoading;
  const isError = isMovie ? localMovie.isError : localSeries.isError;

  // Progressive enhancement: fetch TMDB credits when tmdbId is available
  const tmdbId = localData?.tmdbId ?? 0;
  const movieCredits = useMovieCredits(isMovie && tmdbId > 0 ? tmdbId : 0);
  const tvCredits = useTVShowCredits(!isMovie && tmdbId > 0 ? tmdbId : 0);
  const credits = isMovie ? movieCredits : tvCredits;

  // Story 12-1 — lazily enrich with the Douban rating. Gated on tmdbId > 0:
  // an unmatched record cannot be reliably matched on Douban either.
  const doubanQuery = useDoubanRating(id, isMovie ? 'movie' : 'series', tmdbId > 0);

  // Story 12-6 — Douban review summary (短評). Gated on a RESOLVED doubanId (not
  // merely tmdbId > 0): the summary only renders once the rating query has resolved
  // a doubanId, so fetching before then would issue a Douban search scrape whose
  // result is discarded for every unmatched title (CR M1). Gating here matches the
  // render condition below and avoids that wasted, rate-limited scrape (Rule 27 ①).
  // The direct link is built from doubanQuery's doubanId; this query supplies the comments.
  const doubanReviewQuery = useDoubanReviewSummary(
    id,
    isMovie ? 'movie' : 'series',
    Boolean(doubanQuery.data?.doubanId)
  );

  // Story 12-2 — season summaries for the TV accordion (cached SeasonsJSON,
  // no TMDB call). Gated on TV + tmdbId > 0 (AC #1).
  const seasonsQuery = useSeriesSeasons(id, !isMovie && tmdbId > 0);

  // Story 12-3 — related-content recommendations, keyed by the TMDB id. Loads
  // with the page but as its own query so a slow/failed fetch never blocks the
  // core metadata render (per-section isolation, AC #6/#9).
  const recsQuery = useRecommendations(tmdbId, isMovie ? 'movie' : 'tv', tmdbId > 0);

  // Story 12-4 — streaming-platform availability (TMDB watch providers, region TW).
  // Its own query so a slow/failed fetch never blocks the page (per-section isolation).
  const watchProvidersQuery = useWatchProviders(
    tmdbId,
    isMovie ? 'movie' : 'tv',
    tmdbId > 0,
    WATCH_REGION
  );

  const hasMetadata = !!localData?.tmdbId && localData.tmdbId > 0;
  const posterUrl = getImageUrl(localData?.posterPath ?? null, 'w500');
  const director = isMovie ? credits.data?.crew?.find((c) => c.job === 'Director') : undefined;

  const handleBack = () => {
    navigate({ to: '/library' });
  };

  const buildMetadataForEditor = useCallback((): MediaMetadata | null => {
    if (!localData) return null;

    return {
      id,
      mediaType: isMovie ? 'movie' : 'series',
      title: localData.title,
      titleEnglish: localData.originalTitle,
      year: parseInt(
        (isMovie
          ? (localData as typeof localMovie.data)?.releaseDate
          : (localData as typeof localSeries.data)?.firstAirDate
        )?.slice(0, 4) || '0',
        10
      ),
      genres: localData.genres || [],
      director: director?.name,
      cast: credits.data?.cast?.slice(0, 10).map((c) => c.name) || [],
      overview: localData.overview,
      posterUrl: posterUrl ?? undefined,
    };
  }, [
    localData,
    isMovie,
    id,
    director,
    credits.data,
    posterUrl,
    localMovie.data,
    localSeries.data,
  ]);

  const handleEditorSuccess = () => {
    if (isMovie) localMovie.refetch();
    else localSeries.refetch();
    setShowSuccessToast(true);
    setTimeout(() => setShowSuccessToast(false), 3000);
  };

  const editorMetadata = buildMetadataForEditor();

  // Loading state
  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-[var(--accent-primary)]" />
      </div>
    );
  }

  // Error state
  if (isError || !localData) {
    return <NotFoundComponent />;
  }

  return (
    <>
      <div className="relative min-h-screen bg-[var(--bg-primary)]">
        {/* Backdrop image */}
        {localData.backdropPath && (
          <div className="absolute inset-x-0 top-0 h-[400px] overflow-hidden">
            <img
              src={getImageUrl(localData.backdropPath, 'w780') ?? ''}
              alt=""
              className="h-full w-full object-cover opacity-30"
            />
            <div className="absolute inset-0 bg-gradient-to-b from-transparent to-[var(--bg-primary)]" />
          </div>
        )}

        {/* Content */}
        <div className="relative mx-auto max-w-5xl px-4 py-6">
          {/* Back button */}
          <button
            onClick={handleBack}
            className="mb-6 flex items-center gap-2 text-[var(--text-secondary)] hover:text-white transition-colors"
          >
            <ArrowLeft className="h-4 w-4" />
            返回媒體庫
          </button>

          {hasMetadata ? (
            /* Full metadata view */
            <div className="flex flex-col gap-8 md:flex-row">
              {/* Poster */}
              <div className="w-full flex-shrink-0 md:w-[300px]">
                {posterUrl ? (
                  <img
                    src={posterUrl}
                    alt={localData.title}
                    className="w-full rounded-lg shadow-2xl"
                  />
                ) : (
                  <div className="flex aspect-[2/3] w-full items-center justify-center rounded-lg bg-[var(--bg-secondary)]">
                    <Film className="h-16 w-16 text-[var(--text-muted)]" />
                  </div>
                )}
              </div>

              {/* Info */}
              <div className="flex-1">
                <div className="flex items-start justify-between">
                  <div>
                    <h1 className="text-3xl font-bold text-white">{localData.title}</h1>
                    {localData.originalTitle && localData.originalTitle !== localData.title && (
                      <p className="mt-1 text-lg text-[var(--text-secondary)]">
                        {localData.originalTitle}
                      </p>
                    )}
                  </div>
                  <button
                    onClick={() => setIsEditorOpen(true)}
                    className={cn(
                      'flex items-center gap-1.5 px-3 py-1.5 rounded-lg',
                      'bg-[var(--bg-secondary)]/80 text-white text-sm',
                      'hover:bg-[var(--bg-tertiary)] transition-colors'
                    )}
                    aria-label="編輯媒體資訊"
                    data-testid="edit-metadata-button"
                  >
                    <Pencil className="h-4 w-4" />
                    編輯
                  </button>
                </div>

                {/* Meta line */}
                <div className="mt-3 flex flex-wrap items-center gap-3 text-sm text-[var(--text-secondary)]">
                  {(isMovie
                    ? localData.releaseDate
                    : (localData as typeof localSeries.data)?.firstAirDate) && (
                    <span>
                      {(isMovie
                        ? localData.releaseDate
                        : (localData as typeof localSeries.data)?.firstAirDate
                      )?.slice(0, 4)}
                    </span>
                  )}
                  <DualRatingDisplay
                    tmdbRating={localData.voteAverage}
                    tmdbVoteCount={localData.voteCount}
                    doubanRating={doubanQuery.data?.doubanRating}
                    doubanVoteCount={doubanQuery.data?.doubanVoteCount}
                    doubanLoading={doubanQuery.isLoading}
                  />
                  {localData.genres?.length > 0 && <span>{localData.genres.join(' / ')}</span>}
                  {localData.metadataSource && (
                    <span className="rounded bg-[var(--accent-primary)]/30 px-2 py-0.5 text-xs text-blue-300">
                      {localData.metadataSource}
                    </span>
                  )}
                </div>

                {/* Tech Badges */}
                <TechBadgeGroup
                  videoCodec={localData.videoCodec}
                  videoResolution={localData.videoResolution}
                  audioCodec={localData.audioCodec}
                  audioChannels={localData.audioChannels}
                  hdrFormat={localData.hdrFormat}
                  subtitleTracks={localData.subtitleTracks}
                  className="mt-3"
                />

                {/* Overview */}
                {localData.overview && (
                  <p className="mt-4 leading-relaxed text-[var(--text-secondary)]">
                    {localData.overview}
                  </p>
                )}

                {/* Streaming-platform availability (Story 12-4) — below overview, above credits */}
                {tmdbId > 0 && (
                  <div className="mt-6">
                    <StreamingAvailability
                      region={watchProvidersQuery.data?.results?.[WATCH_REGION]}
                      isLoading={watchProvidersQuery.isLoading}
                      isError={watchProvidersQuery.isError}
                      onRetry={() => watchProvidersQuery.refetch()}
                    />
                  </div>
                )}

                {/* Trailer (Story 12-5) — below streaming, above credits. Self-fetches
                    its TMDB videos; renders nothing when there's no embeddable trailer. */}
                {tmdbId > 0 && (
                  <div className="mt-6">
                    <TrailerSection tmdbId={tmdbId} type={type} title={localData.title} />
                  </div>
                )}

                {/* Douban review summary (Story 12-6) — below trailer, above credits.
                    Direct link + short comments, keyed off the doubanId the rating
                    query already resolved. Gated on a known doubanId so there's no
                    empty spacer when there's no Douban match (AC #4). */}
                {doubanQuery.data?.doubanId && (
                  <div className="mt-6">
                    <DoubanSection
                      doubanId={doubanQuery.data.doubanId}
                      summary={doubanReviewQuery.data}
                      isLoading={doubanReviewQuery.isLoading}
                      isError={doubanReviewQuery.isError}
                    />
                  </div>
                )}

                {/* Credits */}
                {credits.data && (
                  <div className="mt-6">
                    <CreditsSection director={director} cast={credits.data.cast?.slice(0, 6)} />
                  </div>
                )}

                {/* Related content recommendations (Story 12-3) */}
                {tmdbId > 0 && (
                  <div className="mt-8">
                    <RelatedContent
                      items={recsQuery.data?.results ?? []}
                      isLoading={recsQuery.isLoading}
                      isError={recsQuery.isError}
                      onRetry={() => recsQuery.refetch()}
                    />
                  </div>
                )}

                {/* Season/Episode accordion (Story 12-2, TV only) */}
                {!isMovie && (
                  <div className="mt-8">
                    <SeasonAccordion
                      seasons={seasonsQuery.data ?? []}
                      seriesId={id}
                      tmdbId={tmdbId}
                      isLoading={seasonsQuery.isLoading}
                      isError={seasonsQuery.isError}
                      onRetry={() => seasonsQuery.refetch()}
                    />
                  </div>
                )}
              </div>
            </div>
          ) : (
            /* Fallback UI — no TMDB metadata (Story 5-11) */
            <div className="overflow-hidden rounded-xl bg-[var(--bg-secondary)]/50">
              {/* Color placeholder poster as backdrop */}
              <ColorPlaceholder
                filename={localData.filePath ?? localData.title}
                initial={localData.title.charAt(0) || '?'}
                className="h-[200px] w-full rounded-none md:h-[240px]"
              />

              {/* Conditional content based on parseStatus */}
              {localData.parseStatus === 'pending' ? (
                <FallbackPending
                  filename={localData.filePath?.split('/').pop() ?? localData.title}
                />
              ) : (
                <FallbackFailed
                  title={localData.title}
                  mediaType={type}
                  filePath={localData.filePath}
                  fileSize={localData.fileSize}
                  createdAt={localData.createdAt}
                  parseStatus={localData.parseStatus}
                  onEditClick={() => setIsEditorOpen(true)}
                />
              )}
            </div>
          )}
        </div>
      </div>

      {/* Metadata Editor Dialog */}
      {editorMetadata && (
        <MetadataEditorDialog
          isOpen={isEditorOpen}
          onClose={() => setIsEditorOpen(false)}
          mediaId={id}
          mediaType={isMovie ? 'movie' : 'series'}
          initialData={editorMetadata}
          onSuccess={handleEditorSuccess}
        />
      )}

      {/* Success Toast */}
      {showSuccessToast && (
        <div
          className={cn(
            'fixed bottom-6 left-1/2 -translate-x-1/2 z-50',
            'px-4 py-2 rounded-lg bg-[var(--success)] text-white',
            'shadow-lg animate-in fade-in slide-in-from-bottom-4'
          )}
          role="status"
          aria-live="polite"
          data-testid="success-toast"
        >
          媒體資訊已更新
        </div>
      )}
    </>
  );
}

// bugfix-10-1 Task 2 — TMDb-backed detail render for posters whose `id` is a
// raw TMDb numeric (homepage trending/discover, search results). No local DB
// row exists, so we skip the editor and tech-badge UI and rely entirely on
// the TMDb endpoints already wired by Story 10-1.
// Exported for co-located unit tests; not consumed elsewhere in the app.
export function TMDbDetailView({ type, tmdbId }: { type: ValidMediaType; tmdbId: number }) {
  const navigate = useNavigate();
  const isMovie = type === 'movie';

  const movieDetails = useMovieDetails(isMovie ? tmdbId : 0);
  const tvDetails = useTVShowDetails(!isMovie ? tmdbId : 0);
  const movieCredits = useMovieCredits(isMovie ? tmdbId : 0);
  const tvCredits = useTVShowCredits(!isMovie ? tmdbId : 0);

  // Story 12-3 — related-content recommendations for this TMDB title.
  const recsQuery = useRecommendations(tmdbId, isMovie ? 'movie' : 'tv', tmdbId > 0);

  // Story 12-4 — streaming-platform availability (TMDB watch providers, region TW).
  const watchProvidersQuery = useWatchProviders(
    tmdbId,
    isMovie ? 'movie' : 'tv',
    tmdbId > 0,
    WATCH_REGION
  );

  // bugfix-10-1 CR L3 — narrow per branch via the per-type query result so we
  // avoid the `data as MovieDetails` / `data as TVShowDetails` pair below.
  const movie: MovieDetails | undefined = isMovie ? movieDetails.data : undefined;
  const show: TVShowDetails | undefined = !isMovie ? tvDetails.data : undefined;
  const detailsQuery = isMovie ? movieDetails : tvDetails;
  const creditsQuery = isMovie ? movieCredits : tvCredits;

  // Story 10-4 ownership read-through; bi-directional redirect deferred —
  // needs GET /api/v1/movies/by-tmdb/:tmdbId (out of scope for bugfix-10-1).
  const ownership = useOwnedMedia([tmdbId]);

  const handleBack = () => {
    navigate({ to: '/library' });
  };

  if (detailsQuery.isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-[var(--accent-primary)]" />
      </div>
    );
  }

  // AC #5 — any TMDb fetch failure surfaces the same NotFound UX as a missing
  // local row. Avoids a half-rendered state when the upstream is down.
  const data = movie ?? show;
  if (detailsQuery.isError || !data) {
    return <NotFoundComponent />;
  }

  const title = movie ? movie.title : show!.name;
  const originalTitle = movie ? movie.originalTitle : show!.originalName;
  const releaseDate = movie ? movie.releaseDate : show!.firstAirDate;
  const genreNames = data.genres?.map((g) => g.name).filter(Boolean) ?? [];
  const posterUrl = getImageUrl(data.posterPath ?? null, 'w500');
  const backdropPath = data.backdropPath ?? null;
  const director = isMovie ? creditsQuery.data?.crew?.find((c) => c.job === 'Director') : undefined;
  const showOwnedBadge = !ownership.isLoading && ownership.isOwned(tmdbId);

  return (
    <div data-testid="tmdb-detail-view" className="relative min-h-screen bg-[var(--bg-primary)]">
      {backdropPath && (
        <div className="absolute inset-x-0 top-0 h-[400px] overflow-hidden">
          <img
            src={getImageUrl(backdropPath, 'w780') ?? ''}
            alt=""
            className="h-full w-full object-cover opacity-30"
          />
          <div className="absolute inset-0 bg-gradient-to-b from-transparent to-[var(--bg-primary)]" />
        </div>
      )}

      <div className="relative mx-auto max-w-5xl px-4 py-6">
        <button
          onClick={handleBack}
          className="mb-6 flex items-center gap-2 text-[var(--text-secondary)] hover:text-white transition-colors"
        >
          <ArrowLeft className="h-4 w-4" />
          返回媒體庫
        </button>

        <div className="flex flex-col gap-8 md:flex-row">
          <div className="w-full flex-shrink-0 md:w-[300px]">
            {posterUrl ? (
              <img src={posterUrl} alt={title} className="w-full rounded-lg shadow-2xl" />
            ) : (
              <div className="flex aspect-[2/3] w-full items-center justify-center rounded-lg bg-[var(--bg-secondary)]">
                <Film className="h-16 w-16 text-[var(--text-muted)]" />
              </div>
            )}
          </div>

          <div className="flex-1">
            <div className="flex items-start justify-between gap-3">
              <div>
                <h1 className="text-3xl font-bold text-white">{title}</h1>
                {originalTitle && originalTitle !== title && (
                  <p className="mt-1 text-lg text-[var(--text-secondary)]">{originalTitle}</p>
                )}
              </div>
              {showOwnedBadge && (
                <span
                  data-testid="tmdb-detail-owned-badge"
                  className="inline-flex shrink-0 items-center gap-1 rounded-full bg-emerald-900/30 px-3 py-1 text-sm text-emerald-400"
                >
                  📁 已在媒體庫
                </span>
              )}
            </div>

            <div className="mt-3 flex flex-wrap items-center gap-3 text-sm text-[var(--text-secondary)]">
              {releaseDate && <span>{releaseDate.slice(0, 4)}</span>}
              {/* TMDb-numeric items are not in the local library, so no Douban
                  enrichment is possible (the rating AND the Story 12-6 review
                  summary endpoints are both keyed by local UUID). DoubanSection is
                  therefore LocalDetailView-only. */}
              <DualRatingDisplay tmdbRating={data.voteAverage} tmdbVoteCount={data.voteCount} />
              {genreNames.length > 0 && <span>{genreNames.join(' / ')}</span>}
            </div>

            {data.overview && (
              <p className="mt-4 leading-relaxed text-[var(--text-secondary)]">{data.overview}</p>
            )}

            {/* Streaming-platform availability (Story 12-4) — below overview, above credits */}
            {tmdbId > 0 && (
              <div className="mt-6">
                <StreamingAvailability
                  region={watchProvidersQuery.data?.results?.[WATCH_REGION]}
                  isLoading={watchProvidersQuery.isLoading}
                  isError={watchProvidersQuery.isError}
                  onRetry={() => watchProvidersQuery.refetch()}
                />
              </div>
            )}

            {/* Trailer (Story 12-5) — below streaming, above credits. Self-fetches
                its TMDB videos; renders nothing when there's no embeddable trailer. */}
            {tmdbId > 0 && (
              <div className="mt-6">
                <TrailerSection tmdbId={tmdbId} type={type} title={title} />
              </div>
            )}

            {creditsQuery.data && (
              <div className="mt-6">
                <CreditsSection director={director} cast={creditsQuery.data.cast?.slice(0, 6)} />
              </div>
            )}
          </div>
        </div>

        {/* Related content recommendations (Story 12-3) — final full-width section */}
        {tmdbId > 0 && (
          <div className="mt-10">
            <RelatedContent
              items={recsQuery.data?.results ?? []}
              isLoading={recsQuery.isLoading}
              isError={recsQuery.isError}
              onRetry={() => recsQuery.refetch()}
            />
          </div>
        )}
      </div>
    </div>
  );
}
