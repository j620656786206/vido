import { useState, useCallback } from 'react';
import { createFileRoute, notFound, useNavigate } from '@tanstack/react-router';
import { Pencil } from 'lucide-react';
import { SidePanel } from '../../components/ui/SidePanel';
import { MediaDetailPanel } from '../../components/media/MediaDetailPanel';
import { CreditsSection } from '../../components/media/CreditsSection';
import { TVShowInfo } from '../../components/media/TVShowInfo';
import { MetadataEditorDialog } from '../../components/metadata-editor';
import type { MediaMetadata } from '../../components/metadata-editor';
import {
  useMovieDetails,
  useTVShowDetails,
  useMovieCredits,
  useTVShowCredits,
} from '../../hooks/useMediaDetails';
import type { TVShowDetails, MovieDetails } from '../../types/tmdb';
import { cn } from '../../lib/utils';

// Task 1.2: Validate $type parameter (movie | tv)
const validMediaTypes = ['movie', 'tv'] as const;
type ValidMediaType = (typeof validMediaTypes)[number];

function isValidMediaType(type: string): type is ValidMediaType {
  return validMediaTypes.includes(type as ValidMediaType);
}

export const Route = createFileRoute('/media/$type/$id')({
  // Task 1.3: Route loader for validation and potential prefetching
  loader: async ({ params }) => {
    const { type, id } = params;

    // Task 1.4: Handle invalid routes with 404
    if (!isValidMediaType(type)) {
      throw notFound();
    }

    const numericId = parseInt(id, 10);
    if (isNaN(numericId) || numericId <= 0) {
      throw notFound();
    }

    return {
      type: type as ValidMediaType,
      id: numericId,
    };
  },
  notFoundComponent: NotFoundComponent,
  component: MediaDetailRoute,
});

function NotFoundComponent() {
  const navigate = useNavigate();

  return (
    <div className="flex min-h-screen items-center justify-center bg-slate-900">
      <div className="text-center">
        <h1 className="mb-4 text-4xl font-bold text-white">404</h1>
        <p className="mb-6 text-gray-400">找不到該媒體內容</p>
        <button
          onClick={() => navigate({ to: '/search' })}
          className="rounded-lg bg-blue-600 px-4 py-2 text-white hover:bg-blue-700"
        >
          返回搜尋
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

  // Fetch details based on type
  const movieDetails = useMovieDetails(isMovie ? id : 0);
  const tvDetails = useTVShowDetails(!isMovie ? id : 0);
  const movieCredits = useMovieCredits(isMovie ? id : 0);
  const tvCredits = useTVShowCredits(!isMovie ? id : 0);

  const details = isMovie ? movieDetails : tvDetails;
  const credits = isMovie ? movieCredits : tvCredits;

  const isLoading = details.isLoading || credits.isLoading;

  const handleClose = () => {
    navigate({ to: '/search' });
  };

  // Find director from movie credits
  const director = isMovie
    ? credits.data?.crew?.find((c) => c.job === 'Director')
    : undefined;

  // Get TV show data if available
  const tvShowData = !isMovie ? (details.data as TVShowDetails | undefined) : undefined;

  // Build metadata for editor dialog (Story 3.8 - Task 8)
  const buildMetadataForEditor = useCallback((): MediaMetadata | null => {
    if (!details.data) return null;

    const data = details.data;
    const movieData = isMovie ? (data as MovieDetails) : null;
    const tvData = !isMovie ? (data as TVShowDetails) : null;

    return {
      id: String(id),
      mediaType: isMovie ? 'movie' : 'series',
      title: isMovie ? movieData!.title : tvData!.name,
      titleEnglish: isMovie ? movieData!.original_title : tvData!.original_name,
      year: parseInt(
        (isMovie ? movieData!.release_date : tvData!.first_air_date)?.slice(0, 4) || '0',
        10
      ),
      genres: data.genres?.map((g) => g.name) || [],
      director: director?.name,
      cast: credits.data?.cast?.slice(0, 10).map((c) => c.name) || [],
      overview: data.overview,
      posterUrl: data.poster_path
        ? `https://image.tmdb.org/t/p/w500${data.poster_path}`
        : undefined,
    };
  }, [details.data, isMovie, id, director, credits.data]);

  const handleEditClick = () => {
    setIsEditorOpen(true);
  };

  const handleEditorClose = () => {
    setIsEditorOpen(false);
  };

  const handleEditorSuccess = () => {
    // Refresh the data
    details.refetch();
    credits.refetch();

    // Show success toast
    setShowSuccessToast(true);
    setTimeout(() => setShowSuccessToast(false), 3000);
  };

  const editorMetadata = buildMetadataForEditor();

  return (
    <>
      <SidePanel isOpen={true} onClose={handleClose}>
        {/* Edit Metadata Button (Story 3.8 - Task 8.1) */}
        {!isLoading && details.data && (
          <div className="absolute right-14 top-4 z-10">
            <button
              onClick={handleEditClick}
              className={cn(
                'flex items-center gap-1.5 px-3 py-1.5 rounded-lg',
                'bg-slate-800/80 text-white text-sm',
                'hover:bg-slate-700 transition-colors',
                'backdrop-blur-sm'
              )}
              aria-label="編輯媒體資訊"
              data-testid="edit-metadata-button"
            >
              <Pencil className="h-4 w-4" />
              編輯
            </button>
          </div>
        )}

        {/* Task 7: Mobile view handled by SidePanel (full-width on mobile) */}
        <MediaDetailPanel
          type={type}
          details={details.data ?? null}
          credits={credits.data}
          isLoading={isLoading}
        />

        {/* TV Show specific info */}
        {tvShowData && !isLoading && (
          <div className="px-4 pb-4">
            <TVShowInfo show={tvShowData} />
          </div>
        )}

        {/* Credits section */}
        {credits.data && !isLoading && (
          <div className="px-4 pb-6">
            <CreditsSection
              director={director}
              cast={credits.data.cast?.slice(0, 6)}
              createdBy={tvShowData?.created_by}
            />
          </div>
        )}
      </SidePanel>

      {/* Metadata Editor Dialog (Story 3.8 - Task 8.2) */}
      {editorMetadata && (
        <MetadataEditorDialog
          isOpen={isEditorOpen}
          onClose={handleEditorClose}
          mediaId={String(id)}
          mediaType={isMovie ? 'movie' : 'series'}
          initialData={editorMetadata}
          onSuccess={handleEditorSuccess}
        />
      )}

      {/* Success Toast (Story 3.8 - Task 8.4) */}
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
