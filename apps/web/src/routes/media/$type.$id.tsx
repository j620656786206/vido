import { useState, useCallback } from 'react';
import { createFileRoute, notFound, useNavigate, Link } from '@tanstack/react-router';
import { Pencil, ArrowLeft, Film, Search, Loader2, FileText, HardDrive, Clock } from 'lucide-react';
import { CreditsSection } from '../../components/media/CreditsSection';
import { MetadataEditorDialog } from '../../components/metadata-editor';
import type { MediaMetadata } from '../../components/metadata-editor';
import {
  useLocalMovieDetails,
  useLocalSeriesDetails,
  useMovieCredits,
  useTVShowCredits,
} from '../../hooks/useMediaDetails';
import { getImageUrl } from '../../lib/image';
import { cn } from '../../lib/utils';

const validMediaTypes = ['movie', 'tv'] as const;
type ValidMediaType = (typeof validMediaTypes)[number];

function isValidMediaType(type: string): type is ValidMediaType {
  return validMediaTypes.includes(type as ValidMediaType);
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
        <p className="mb-6 text-gray-400">找不到該媒體內容</p>
        <button
          onClick={() => navigate({ to: '/library' })}
          className="rounded-lg bg-blue-600 px-4 py-2 text-white hover:bg-blue-700"
        >
          返回媒體庫
        </button>
      </div>
    </div>
  );
}

function MediaDetailRoute() {
  const { type, id } = Route.useLoaderData();
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
        <Loader2 className="h-8 w-8 animate-spin text-blue-500" />
      </div>
    );
  }

  // Error state
  if (isError || !localData) {
    return <NotFoundComponent />;
  }

  return (
    <>
      <div className="relative min-h-screen bg-gray-900">
        {/* Backdrop image */}
        {localData.backdropPath && (
          <div className="absolute inset-x-0 top-0 h-[400px] overflow-hidden">
            <img
              src={getImageUrl(localData.backdropPath, 'w780') ?? ''}
              alt=""
              className="h-full w-full object-cover opacity-30"
            />
            <div className="absolute inset-0 bg-gradient-to-b from-transparent to-gray-900" />
          </div>
        )}

        {/* Content */}
        <div className="relative mx-auto max-w-5xl px-4 py-6">
          {/* Back button */}
          <button
            onClick={handleBack}
            className="mb-6 flex items-center gap-2 text-gray-400 hover:text-white transition-colors"
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
                  <div className="flex aspect-[2/3] w-full items-center justify-center rounded-lg bg-gray-800">
                    <Film className="h-16 w-16 text-gray-600" />
                  </div>
                )}
              </div>

              {/* Info */}
              <div className="flex-1">
                <div className="flex items-start justify-between">
                  <div>
                    <h1 className="text-3xl font-bold text-white">{localData.title}</h1>
                    {localData.originalTitle && localData.originalTitle !== localData.title && (
                      <p className="mt-1 text-lg text-gray-400">{localData.originalTitle}</p>
                    )}
                  </div>
                  <button
                    onClick={() => setIsEditorOpen(true)}
                    className={cn(
                      'flex items-center gap-1.5 px-3 py-1.5 rounded-lg',
                      'bg-slate-800/80 text-white text-sm',
                      'hover:bg-slate-700 transition-colors'
                    )}
                    aria-label="編輯媒體資訊"
                    data-testid="edit-metadata-button"
                  >
                    <Pencil className="h-4 w-4" />
                    編輯
                  </button>
                </div>

                {/* Meta line */}
                <div className="mt-3 flex flex-wrap items-center gap-3 text-sm text-gray-400">
                  {(isMovie ? localData.releaseDate : (localData as typeof localSeries.data)?.firstAirDate) && (
                    <span>
                      {(isMovie
                        ? localData.releaseDate
                        : (localData as typeof localSeries.data)?.firstAirDate
                      )?.slice(0, 4)}
                    </span>
                  )}
                  {localData.voteAverage != null && localData.voteAverage > 0 && (
                    <span className="text-yellow-400">⭐ {localData.voteAverage.toFixed(1)}</span>
                  )}
                  {localData.genres?.length > 0 && <span>{localData.genres.join(' / ')}</span>}
                  {localData.metadataSource && (
                    <span className="rounded bg-blue-600/30 px-2 py-0.5 text-xs text-blue-300">
                      {localData.metadataSource}
                    </span>
                  )}
                </div>

                {/* Overview */}
                {localData.overview && (
                  <p className="mt-4 leading-relaxed text-gray-300">{localData.overview}</p>
                )}

                {/* Credits */}
                {credits.data && (
                  <div className="mt-6">
                    <CreditsSection director={director} cast={credits.data.cast?.slice(0, 6)} />
                  </div>
                )}
              </div>
            </div>
          ) : (
            /* Fallback UI — no TMDB metadata */
            <div className="rounded-xl bg-gray-800/50 p-8">
              <div className="flex flex-col items-center gap-6 md:flex-row md:items-start">
                {/* Placeholder poster */}
                <div className="flex aspect-[2/3] w-[200px] flex-shrink-0 items-center justify-center rounded-lg bg-gray-700">
                  <Film className="h-16 w-16 text-gray-500" />
                </div>

                <div className="flex-1">
                  <h1 className="text-2xl font-bold text-white">{localData.title}</h1>

                  {localData.parseStatus === 'pending' && (
                    <div className="mt-3 flex items-center gap-2 rounded-lg bg-blue-900/30 px-4 py-2 text-blue-300">
                      <Loader2 className="h-4 w-4 animate-spin" />
                      正在取得媒體資訊...
                    </div>
                  )}

                  {(localData.parseStatus === 'failed' || localData.parseStatus === '') && (
                    <div className="mt-3 rounded-lg bg-yellow-900/20 px-4 py-2 text-yellow-300">
                      尚未取得媒體資訊
                    </div>
                  )}

                  {/* File info */}
                  <div className="mt-6 space-y-3">
                    <h2 className="text-sm font-semibold uppercase tracking-wider text-gray-400">
                      檔案資訊
                    </h2>
                    {localData.filePath && (
                      <div className="flex items-center gap-2 text-sm text-gray-300">
                        <FileText className="h-4 w-4 text-gray-500" />
                        <span className="break-all">{localData.filePath}</span>
                      </div>
                    )}
                    {localData.fileSize != null && localData.fileSize > 0 && (
                      <div className="flex items-center gap-2 text-sm text-gray-300">
                        <HardDrive className="h-4 w-4 text-gray-500" />
                        <span>{(localData.fileSize / (1024 * 1024 * 1024)).toFixed(2)} GB</span>
                      </div>
                    )}
                    <div className="flex items-center gap-2 text-sm text-gray-300">
                      <Clock className="h-4 w-4 text-gray-500" />
                      <span>新增於 {new Date(localData.createdAt).toLocaleString('zh-TW')}</span>
                    </div>
                  </div>

                  {/* Actions */}
                  <div className="mt-6 flex gap-3">
                    <Link
                      to="/search"
                      search={{ q: localData.title.replace(/\.\w{2,4}$/, '') }}
                      className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-700 transition-colors"
                    >
                      <Search className="h-4 w-4" />
                      搜尋元資料
                    </Link>
                    <button
                      onClick={() => setIsEditorOpen(true)}
                      className="flex items-center gap-2 rounded-lg bg-gray-700 px-4 py-2 text-sm text-white hover:bg-gray-600 transition-colors"
                    >
                      <Pencil className="h-4 w-4" />
                      手動編輯
                    </button>
                  </div>
                </div>
              </div>
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
            'px-4 py-2 rounded-lg bg-green-600 text-white',
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
