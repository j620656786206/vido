// Implements: Component/Detail-Movie-v2 (uRGu2)
/**
 * v2 detail for TMDb-numeric items (UX Redesign Phase 2 — UX2-3, AC #8). These
 * are discover/homepage items with no local DB row — so no tech-info, Douban,
 * metadata editor, or subtitle actions (those are LocalDetailV2-only). Same v2
 * backdrop hero + reused Epic-12 sections (trailer / providers / credits / recs),
 * each failing soft. An owned item shows the 已入庫 status badge.
 */
import { useNavigate } from '@tanstack/react-router';
import {
  useMovieDetails,
  useTVShowDetails,
  useMovieCredits,
  useTVShowCredits,
  useRecommendations,
  useWatchProviders,
} from '../../hooks/useMediaDetails';
import { useOwnedMedia } from '../../hooks/useOwnedMedia';
import { CreditsSection } from './CreditsSection';
import { RelatedContent } from './RelatedContent';
import { StreamingAvailability } from './StreamingAvailability';
import { TrailerSection } from './TrailerSection';
import { DualRatingDisplay } from './DualRatingDisplay';
import { DetailHeroV2 } from './DetailHeroV2';
import { DetailSkeletonV2, DetailNotFoundV2 } from './DetailStatesV2';

const WATCH_REGION = 'TW';

export function TMDbDetailV2({ type, tmdbId }: { type: 'movie' | 'tv'; tmdbId: number }) {
  const navigate = useNavigate();
  const isMovie = type === 'movie';

  const movieDetails = useMovieDetails(isMovie ? tmdbId : 0);
  const tvDetails = useTVShowDetails(!isMovie ? tmdbId : 0);
  const movieCredits = useMovieCredits(isMovie ? tmdbId : 0);
  const tvCredits = useTVShowCredits(!isMovie ? tmdbId : 0);
  const recs = useRecommendations(tmdbId, isMovie ? 'movie' : 'tv', tmdbId > 0);
  const watch = useWatchProviders(tmdbId, isMovie ? 'movie' : 'tv', tmdbId > 0, WATCH_REGION);
  const ownership = useOwnedMedia([tmdbId]);

  const detailsQuery = isMovie ? movieDetails : tvDetails;
  const creditsQuery = isMovie ? movieCredits : tvCredits;
  const data = isMovie ? movieDetails.data : tvDetails.data;

  const onBack = () => navigate({ to: '/library' });

  if (detailsQuery.isLoading) return <DetailSkeletonV2 />;
  if (detailsQuery.isError || !data) return <DetailNotFoundV2 onBack={onBack} />;

  const title = isMovie ? movieDetails.data!.title : tvDetails.data!.name;
  const originalTitle = isMovie ? movieDetails.data!.originalTitle : tvDetails.data!.originalName;
  const releaseDate = isMovie ? movieDetails.data!.releaseDate : tvDetails.data!.firstAirDate;
  const genreNames = data.genres?.map((g) => g.name).filter(Boolean) ?? [];
  const director = isMovie ? creditsQuery.data?.crew?.find((c) => c.job === 'Director') : undefined;
  const owned = !ownership.isLoading && ownership.isOwned(tmdbId);

  const meta = (
    <>
      {releaseDate && <span className="font-mono">{releaseDate.slice(0, 4)}</span>}
      {genreNames.length > 0 && <span>{genreNames.join(' / ')}</span>}
      <DualRatingDisplay tmdbRating={data.voteAverage} tmdbVoteCount={data.voteCount} />
    </>
  );

  return (
    <div className="min-h-screen bg-[var(--bg-primary)]" data-testid="tmdb-detail-v2">
      <DetailHeroV2
        backdropPath={data.backdropPath}
        posterPath={data.posterPath}
        title={title}
        originalTitle={originalTitle}
        badges={
          owned
            ? [{ label: '已入庫', className: 'bg-[var(--success-tint)] text-[var(--success)]' }]
            : []
        }
        meta={meta}
        onBack={onBack}
      />

      <div className="mx-auto max-w-5xl space-y-8 px-4 pb-16 pt-8 sm:px-8">
        {data.overview && (
          <section data-testid="detail-overview">
            <h2 className="mb-2 text-lg font-semibold text-[var(--text-primary)]">簡介</h2>
            <p className="leading-relaxed text-[var(--text-secondary)]">{data.overview}</p>
          </section>
        )}

        {tmdbId > 0 && <TrailerSection tmdbId={tmdbId} type={type} title={title} />}

        {tmdbId > 0 && (
          <StreamingAvailability
            region={watch.data?.results?.[WATCH_REGION]}
            isLoading={watch.isLoading}
            isError={watch.isError}
            onRetry={() => watch.refetch()}
          />
        )}

        {creditsQuery.data && (
          <CreditsSection director={director} cast={creditsQuery.data.cast?.slice(0, 8)} />
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
    </div>
  );
}
