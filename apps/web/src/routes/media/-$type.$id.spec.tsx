// bugfix-10-1 Task 5 — mock @tanstack/react-router AND useOwnedMedia at the
// top of the file (vi.mock is hoisted) so importing the route module below
// does not register the live route, and so TMDbDetailView's useNavigate call
// resolves to a no-op stub. Existing tests in this file render presentational
// components that do not touch router APIs, so the mock is safe.
vi.mock('@tanstack/react-router', () => ({
  createFileRoute: () => (opts: Record<string, unknown>) => opts,
  notFound: () => new Error('notFound'),
  useNavigate: () => vi.fn(),
}));
vi.mock('../../hooks/useOwnedMedia', () => ({
  useOwnedMedia: vi.fn(),
}));

import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { SidePanel } from '../../components/ui/SidePanel';
import { MediaDetailPanel } from '../../components/media/MediaDetailPanel';
import { TVShowInfo } from '../../components/media/TVShowInfo';
import { CreditsSection } from '../../components/media/CreditsSection';
import {
  useMovieDetails,
  useTVShowDetails,
  useMovieCredits,
  useTVShowCredits,
} from '../../hooks/useMediaDetails';
import { tmdbService } from '../../services/tmdb';
import type { MovieDetails, TVShowDetails, Credits } from '../../types/tmdb';
import { classifyId, TMDbDetailView } from './$type.$id';
import { useOwnedMedia, type OwnedMediaState } from '../../hooks/useOwnedMedia';

// Mock the tmdb service
vi.mock('../../services/tmdb', () => ({
  tmdbService: {
    getMovieDetails: vi.fn(),
    getTVShowDetails: vi.fn(),
    getMovieCredits: vi.fn(),
    getTVShowCredits: vi.fn(),
  },
}));

const mockMovieDetails: MovieDetails = {
  id: 123,
  title: '測試電影',
  originalTitle: 'Test Movie',
  overview: '這是測試劇情',
  releaseDate: '2024-01-15',
  posterPath: '/poster.jpg',
  backdropPath: '/backdrop.jpg',
  voteAverage: 8.5,
  voteCount: 1000,
  runtime: 120,
  budget: 10000000,
  revenue: 50000000,
  status: 'Released',
  tagline: '',
  genres: [{ id: 1, name: '動作' }],
  productionCountries: [],
  spokenLanguages: [],
  imdbId: 'tt1234567',
  homepage: null,
};

const mockTVShowDetails: TVShowDetails = {
  id: 456,
  name: '測試影集',
  originalName: 'Test TV Show',
  overview: '這是測試劇情',
  firstAirDate: '2023-06-01',
  lastAirDate: '2024-01-01',
  posterPath: '/tv_poster.jpg',
  backdropPath: '/tv_backdrop.jpg',
  voteAverage: 9.0,
  voteCount: 2000,
  episodeRunTime: [45],
  numberOfSeasons: 3,
  numberOfEpisodes: 30,
  status: 'Returning Series',
  type: 'Scripted',
  tagline: '',
  genres: [{ id: 3, name: '劇情' }],
  createdBy: [{ id: 1, name: '創作者', profilePath: null }],
  networks: [{ id: 1, name: 'Netflix', logoPath: null }],
  inProduction: true,
  seasons: [],
};

const mockCredits: Credits = {
  id: 123,
  cast: [
    { id: 1, name: '演員一', character: '角色一', profilePath: null, order: 0 },
    { id: 2, name: '演員二', character: '角色二', profilePath: null, order: 1 },
  ],
  crew: [{ id: 3, name: '導演名', job: 'Director', department: 'Directing', profilePath: null }],
};

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
}

/**
 * Test component that simulates the media detail route behavior
 * This mirrors the actual route component structure
 */
function TestMediaDetailRoute({
  type,
  id,
  onClose,
}: {
  type: 'movie' | 'tv';
  id: number;
  onClose: () => void;
}) {
  const isMovie = type === 'movie';

  // Fetch details based on type (same logic as actual route)
  const movieDetails = useMovieDetails(isMovie ? id : 0);
  const tvDetails = useTVShowDetails(!isMovie ? id : 0);
  const movieCredits = useMovieCredits(isMovie ? id : 0);
  const tvCredits = useTVShowCredits(!isMovie ? id : 0);

  const details = isMovie ? movieDetails : tvDetails;
  const credits = isMovie ? movieCredits : tvCredits;
  const isLoading = details.isLoading || credits.isLoading;

  // Find director from movie credits
  const director = isMovie ? credits.data?.crew?.find((c) => c.job === 'Director') : undefined;

  // Get TV show data if available
  const tvShowData = !isMovie ? (details.data as TVShowDetails | undefined) : undefined;

  return (
    <SidePanel isOpen={true} onClose={onClose}>
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
            createdBy={tvShowData?.createdBy}
          />
        </div>
      )}
    </SidePanel>
  );
}

function renderWithQuery(ui: React.ReactElement) {
  const queryClient = createQueryClient();
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

// bugfix-10-1 Task 5 — fixture for the ownership hook (Story 10-4 read-through).
const ownedFalse: OwnedMediaState = {
  owned: new Set<number>(),
  isOwned: () => false,
  isRequested: () => false,
  isLoading: false,
  error: null,
};
const ownedTrue: OwnedMediaState = {
  owned: new Set<number>([12345]),
  isOwned: (id) => id === 12345,
  isRequested: () => false,
  isLoading: false,
  error: null,
};
const ownedLoading: OwnedMediaState = {
  owned: new Set<number>(),
  isOwned: () => false,
  isRequested: () => false,
  isLoading: true,
  error: null,
};

function renderTMDbView(
  type: 'movie' | 'tv',
  tmdbId: number,
  ownership: OwnedMediaState = ownedFalse
) {
  vi.mocked(useOwnedMedia).mockReturnValue(ownership);
  const queryClient = createQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <TMDbDetailView type={type} tmdbId={tmdbId} />
    </QueryClientProvider>
  );
}

describe('classifyId (bugfix-10-1 [@contract-v1] AC #2)', () => {
  it.each([
    // [input, expected, rationale]
    ['movie-uuid-abc', 'local-uuid', 'arbitrary string with letters'],
    ['83533', 'tmdb-numeric', 'pure positive integer (TMDb movie ID)'],
    ['76479', 'tmdb-numeric', 'pure positive integer'],
    ['687163', 'tmdb-numeric', 'pure positive integer (long form)'],
    ['1', 'tmdb-numeric', 'positive integer boundary'],
    ['0', 'local-uuid', 'numeric but non-positive — falls through to local handler'],
    ['abc-123', 'local-uuid', 'alphanumeric mixed'],
    ['550e8400-e29b-41d4-a716-446655440000', 'local-uuid', 'canonical UUID v4'],
    ['12.5', 'local-uuid', 'decimal — not a pure integer'],
    ['-5', 'local-uuid', 'negative — fails ^\\d+$ guard'],
  ] as const)('classifyId(%j) = %j (%s)', (input, expected) => {
    expect(classifyId(input)).toBe(expected);
  });
});

describe('Media Detail Route', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(tmdbService.getMovieDetails).mockResolvedValue(mockMovieDetails);
    vi.mocked(tmdbService.getTVShowDetails).mockResolvedValue(mockTVShowDetails);
    vi.mocked(tmdbService.getMovieCredits).mockResolvedValue(mockCredits);
    vi.mocked(tmdbService.getTVShowCredits).mockResolvedValue(mockCredits);
  });

  describe('API Integration', () => {
    it('should call movie API endpoints for movie type', async () => {
      const onClose = vi.fn();
      renderWithQuery(<TestMediaDetailRoute type="movie" id={123} onClose={onClose} />);

      await waitFor(() => {
        expect(tmdbService.getMovieDetails).toHaveBeenCalledWith(123);
        expect(tmdbService.getMovieCredits).toHaveBeenCalledWith(123);
      });

      // Should NOT call TV endpoints
      expect(tmdbService.getTVShowDetails).not.toHaveBeenCalled();
      expect(tmdbService.getTVShowCredits).not.toHaveBeenCalled();
    });

    it('should call TV API endpoints for tv type', async () => {
      const onClose = vi.fn();
      renderWithQuery(<TestMediaDetailRoute type="tv" id={456} onClose={onClose} />);

      await waitFor(() => {
        expect(tmdbService.getTVShowDetails).toHaveBeenCalledWith(456);
        expect(tmdbService.getTVShowCredits).toHaveBeenCalledWith(456);
      });

      // Should NOT call movie endpoints
      expect(tmdbService.getMovieDetails).not.toHaveBeenCalled();
      expect(tmdbService.getMovieCredits).not.toHaveBeenCalled();
    });
  });

  describe('SidePanel Integration', () => {
    it('should render inside SidePanel component', async () => {
      const onClose = vi.fn();
      renderWithQuery(<TestMediaDetailRoute type="movie" id={123} onClose={onClose} />);

      expect(screen.getByTestId('side-panel')).toBeInTheDocument();
    });

    it('should show loading skeleton while fetching', () => {
      // Don't resolve the mock immediately
      vi.mocked(tmdbService.getMovieDetails).mockImplementation(
        () => new Promise(() => {}) // Never resolves
      );

      const onClose = vi.fn();
      renderWithQuery(<TestMediaDetailRoute type="movie" id={123} onClose={onClose} />);

      expect(screen.getByTestId('media-detail-skeleton')).toBeInTheDocument();
    });

    it('should call onClose when close button is clicked', async () => {
      const onClose = vi.fn();
      renderWithQuery(<TestMediaDetailRoute type="movie" id={123} onClose={onClose} />);

      fireEvent.click(screen.getByTestId('side-panel-close'));
      expect(onClose).toHaveBeenCalledTimes(1);
    });

    it('should call onClose when Escape key is pressed', async () => {
      const onClose = vi.fn();
      renderWithQuery(<TestMediaDetailRoute type="movie" id={123} onClose={onClose} />);

      fireEvent.keyDown(document, { key: 'Escape' });
      expect(onClose).toHaveBeenCalledTimes(1);
    });
  });

  describe('Responsive Behavior (Task 9.5)', () => {
    it('should have responsive width classes on SidePanel', () => {
      const onClose = vi.fn();
      renderWithQuery(<TestMediaDetailRoute type="movie" id={123} onClose={onClose} />);

      const panel = screen.getByTestId('side-panel');
      // Desktop: 450px, Mobile: full width
      expect(panel).toHaveClass('w-full');
      expect(panel).toHaveClass('sm:w-[450px]');
    });

    it('should render full-width on mobile viewport (w-full class)', () => {
      const onClose = vi.fn();
      renderWithQuery(<TestMediaDetailRoute type="movie" id={123} onClose={onClose} />);

      const panel = screen.getByTestId('side-panel');
      // Mobile-first: w-full is the base class
      expect(panel.className).toContain('w-full');
    });

    it('should render fixed width on desktop viewport (sm:w-[450px] class)', () => {
      const onClose = vi.fn();
      renderWithQuery(<TestMediaDetailRoute type="movie" id={123} onClose={onClose} />);

      const panel = screen.getByTestId('side-panel');
      // Desktop breakpoint class
      expect(panel.className).toContain('sm:w-[450px]');
    });
  });

  describe('Movie Content Display', () => {
    it('should display movie title after loading', async () => {
      const onClose = vi.fn();
      renderWithQuery(<TestMediaDetailRoute type="movie" id={123} onClose={onClose} />);

      await waitFor(() => {
        expect(screen.getByTestId('detail-title')).toHaveTextContent('測試電影');
      });
    });

    it('should display movie rating', async () => {
      const onClose = vi.fn();
      renderWithQuery(<TestMediaDetailRoute type="movie" id={123} onClose={onClose} />);

      await waitFor(() => {
        expect(screen.getByTestId('detail-rating')).toHaveTextContent('8.5');
      });
    });

    it('should display credits section for movie', async () => {
      const onClose = vi.fn();
      renderWithQuery(<TestMediaDetailRoute type="movie" id={123} onClose={onClose} />);

      await waitFor(() => {
        expect(screen.getByTestId('credits-section')).toBeInTheDocument();
      });
    });

    it('should display cast members', async () => {
      const onClose = vi.fn();
      renderWithQuery(<TestMediaDetailRoute type="movie" id={123} onClose={onClose} />);

      await waitFor(() => {
        expect(screen.getByText('演員一')).toBeInTheDocument();
        expect(screen.getByText('演員二')).toBeInTheDocument();
      });
    });
  });

  describe('TV Show Content Display', () => {
    it('should display TV show title after loading', async () => {
      const onClose = vi.fn();
      renderWithQuery(<TestMediaDetailRoute type="tv" id={456} onClose={onClose} />);

      await waitFor(() => {
        expect(screen.getByTestId('detail-title')).toHaveTextContent('測試影集');
      });
    });

    it('should display TV show specific info section', async () => {
      const onClose = vi.fn();
      renderWithQuery(<TestMediaDetailRoute type="tv" id={456} onClose={onClose} />);

      await waitFor(() => {
        expect(screen.getByTestId('tv-show-info')).toBeInTheDocument();
      });
    });

    it('should display number of seasons', async () => {
      const onClose = vi.fn();
      renderWithQuery(<TestMediaDetailRoute type="tv" id={456} onClose={onClose} />);

      await waitFor(() => {
        expect(screen.getByTestId('seasons-count')).toHaveTextContent('3 季');
      });
    });

    it('should display created by for TV show', async () => {
      const onClose = vi.fn();
      renderWithQuery(<TestMediaDetailRoute type="tv" id={456} onClose={onClose} />);

      await waitFor(() => {
        // Check for credits section which contains creator info
        expect(screen.getByTestId('credits-section')).toBeInTheDocument();
      });
    });

    it('should display networks', async () => {
      const onClose = vi.fn();
      renderWithQuery(<TestMediaDetailRoute type="tv" id={456} onClose={onClose} />);

      await waitFor(() => {
        expect(screen.getByTestId('networks')).toHaveTextContent('Netflix');
      });
    });
  });
});

// =============================================================================
// bugfix-10-1 Task 2/3/5 — TMDb-numeric branch coverage.
// Verifies that a poster click from the homepage (TMDb numeric ID) renders a
// TMDb-backed detail view, NOT a 404.
// =============================================================================
describe('TMDbDetailView (bugfix-10-1 AC #3, #4, #5, #6)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(tmdbService.getMovieDetails).mockResolvedValue(mockMovieDetails);
    vi.mocked(tmdbService.getTVShowDetails).mockResolvedValue(mockTVShowDetails);
    vi.mocked(tmdbService.getMovieCredits).mockResolvedValue(mockCredits);
    vi.mocked(tmdbService.getTVShowCredits).mockResolvedValue(mockCredits);
  });

  describe('AC #3 — fetches via TMDb endpoints', () => {
    it('renders movie title from TMDb mock for tmdb-numeric URL', async () => {
      renderTMDbView('movie', 123);

      await waitFor(() => {
        expect(screen.getByTestId('tmdb-detail-view')).toBeInTheDocument();
      });
      expect(screen.getByRole('heading', { name: '測試電影' })).toBeInTheDocument();
      expect(tmdbService.getMovieDetails).toHaveBeenCalledWith(123);
    });

    it('renders tv show name from TMDb mock for tmdb-numeric URL', async () => {
      renderTMDbView('tv', 456);

      await waitFor(() => {
        expect(screen.getByTestId('tmdb-detail-view')).toBeInTheDocument();
      });
      expect(screen.getByRole('heading', { name: '測試影集' })).toBeInTheDocument();
      expect(tmdbService.getTVShowDetails).toHaveBeenCalledWith(456);
    });

    it('renders director and cast from TMDb credits', async () => {
      renderTMDbView('movie', 123);

      await waitFor(() => {
        expect(screen.getByText('演員一')).toBeInTheDocument();
      });
      expect(screen.getByText('演員二')).toBeInTheDocument();
    });
  });

  describe('AC #4 — TMDb data shape rendered correctly', () => {
    it('renders release year, vote, and genres from TMDb data', async () => {
      renderTMDbView('movie', 123);

      await waitFor(() => {
        expect(screen.getByTestId('tmdb-detail-view')).toBeInTheDocument();
      });
      expect(screen.getByText('2024')).toBeInTheDocument();
      expect(screen.getByText(/8\.5/)).toBeInTheDocument();
      expect(screen.getByText('動作')).toBeInTheDocument();
    });

    it('renders overview from TMDb data', async () => {
      renderTMDbView('movie', 123);

      await waitFor(() => {
        expect(screen.getByText('這是測試劇情')).toBeInTheDocument();
      });
    });
  });

  describe('AC #5 — error path falls through to NotFoundComponent', () => {
    it('renders 404 NotFound when TMDb fetch errors (movie)', async () => {
      vi.mocked(tmdbService.getMovieDetails).mockRejectedValueOnce(new Error('TMDB_TIMEOUT'));
      renderTMDbView('movie', 999);

      await waitFor(() => {
        expect(screen.getByText('404')).toBeInTheDocument();
      });
      expect(screen.getByText('找不到該媒體內容')).toBeInTheDocument();
      expect(screen.queryByTestId('tmdb-detail-view')).not.toBeInTheDocument();
    });

    it('renders 404 NotFound when TMDb fetch errors (tv)', async () => {
      vi.mocked(tmdbService.getTVShowDetails).mockRejectedValueOnce(new Error('TMDB_TIMEOUT'));
      renderTMDbView('tv', 999);

      await waitFor(() => {
        expect(screen.getByText('404')).toBeInTheDocument();
      });
    });
  });

  describe('AC #6 — owned indicator (Story 10-4 read-through)', () => {
    it('renders the 已在媒體庫 badge when isOwned returns true', async () => {
      renderTMDbView('movie', 12345, ownedTrue);

      await waitFor(() => {
        expect(screen.getByTestId('tmdb-detail-owned-badge')).toBeInTheDocument();
      });
      expect(screen.getByTestId('tmdb-detail-owned-badge')).toHaveTextContent('已在媒體庫');
    });

    it('hides the owned badge when isOwned returns false', async () => {
      renderTMDbView('movie', 123, ownedFalse);

      await waitFor(() => {
        expect(screen.getByTestId('tmdb-detail-view')).toBeInTheDocument();
      });
      expect(screen.queryByTestId('tmdb-detail-owned-badge')).not.toBeInTheDocument();
    });

    it('hides the owned badge while ownership query is loading', async () => {
      renderTMDbView('movie', 12345, ownedLoading);

      await waitFor(() => {
        expect(screen.getByTestId('tmdb-detail-view')).toBeInTheDocument();
      });
      expect(screen.queryByTestId('tmdb-detail-owned-badge')).not.toBeInTheDocument();
    });
  });

  describe('AC #4 / Task 2.6 — editor button is hidden (no local row)', () => {
    it('does NOT render the 編輯 button in the TMDb branch', async () => {
      renderTMDbView('movie', 123);

      await waitFor(() => {
        expect(screen.getByTestId('tmdb-detail-view')).toBeInTheDocument();
      });
      expect(screen.queryByRole('button', { name: /編輯|edit/i })).not.toBeInTheDocument();
      expect(screen.queryByTestId('edit-metadata-button')).not.toBeInTheDocument();
    });
  });
});
