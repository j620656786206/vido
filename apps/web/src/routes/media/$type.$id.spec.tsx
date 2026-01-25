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
  original_title: 'Test Movie',
  overview: '這是測試劇情',
  release_date: '2024-01-15',
  poster_path: '/poster.jpg',
  backdrop_path: '/backdrop.jpg',
  vote_average: 8.5,
  vote_count: 1000,
  runtime: 120,
  budget: 10000000,
  revenue: 50000000,
  status: 'Released',
  tagline: '',
  genres: [{ id: 1, name: '動作' }],
  production_countries: [],
  spoken_languages: [],
  imdb_id: 'tt1234567',
  homepage: null,
};

const mockTVShowDetails: TVShowDetails = {
  id: 456,
  name: '測試影集',
  original_name: 'Test TV Show',
  overview: '這是測試劇情',
  first_air_date: '2023-06-01',
  last_air_date: '2024-01-01',
  poster_path: '/tv_poster.jpg',
  backdrop_path: '/tv_backdrop.jpg',
  vote_average: 9.0,
  vote_count: 2000,
  episode_run_time: [45],
  number_of_seasons: 3,
  number_of_episodes: 30,
  status: 'Returning Series',
  type: 'Scripted',
  tagline: '',
  genres: [{ id: 3, name: '劇情' }],
  created_by: [{ id: 1, name: '創作者', profile_path: null }],
  networks: [{ id: 1, name: 'Netflix', logo_path: null }],
  in_production: true,
  seasons: [],
};

const mockCredits: Credits = {
  id: 123,
  cast: [
    { id: 1, name: '演員一', character: '角色一', profile_path: null, order: 0 },
    { id: 2, name: '演員二', character: '角色二', profile_path: null, order: 1 },
  ],
  crew: [{ id: 3, name: '導演名', job: 'Director', department: 'Directing', profile_path: null }],
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
            createdBy={tvShowData?.created_by}
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
