import { createFileRoute, notFound, useNavigate } from '@tanstack/react-router';
import { SidePanel } from '../../components/ui/SidePanel';
import { MediaDetailPanel } from '../../components/media/MediaDetailPanel';
import { CreditsSection } from '../../components/media/CreditsSection';
import { TVShowInfo } from '../../components/media/TVShowInfo';
import {
  useMovieDetails,
  useTVShowDetails,
  useMovieCredits,
  useTVShowCredits,
} from '../../hooks/useMediaDetails';
import type { TVShowDetails } from '../../types/tmdb';

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

  return (
    <SidePanel isOpen={true} onClose={handleClose}>
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
  );
}
