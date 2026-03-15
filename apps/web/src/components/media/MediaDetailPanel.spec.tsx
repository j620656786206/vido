import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MediaDetailPanel } from './MediaDetailPanel';
import type { MovieDetails, TVShowDetails, Credits } from '../../types/tmdb';

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

const mockMovieDetails: MovieDetails = {
  id: 123,
  title: '測試電影',
  original_title: 'Test Movie',
  overview: '這是一部測試電影的劇情簡介。',
  release_date: '2024-01-15',
  poster_path: '/poster.jpg',
  backdrop_path: '/backdrop.jpg',
  vote_average: 8.5,
  vote_count: 1000,
  runtime: 120,
  budget: 10000000,
  revenue: 50000000,
  status: 'Released',
  tagline: 'Test tagline',
  genres: [
    { id: 1, name: '動作' },
    { id: 2, name: '冒險' },
  ],
  production_countries: [],
  spoken_languages: [],
  imdb_id: 'tt1234567',
  homepage: null,
  popularity: 61.4,
  genre_ids: [],
  original_language: 'en',
  adult: false,
  video: false,
};

const mockTVShowDetails: TVShowDetails = {
  id: 456,
  name: '測試影集',
  original_name: 'Test TV Show',
  overview: '這是一部測試影集的劇情簡介。',
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
  genres: [
    { id: 3, name: '劇情' },
    { id: 4, name: '懸疑' },
  ],
  created_by: [{ id: 1, name: '創作者一', profile_path: null, credit_id: 'c1', gender: 0 }],
  homepage: null,
  in_production: true,
  languages: ['en'],
  production_countries: [{ iso_3166_1: 'US', name: 'United States' }],
  seasons: [
    {
      id: 1,
      name: 'Season 1',
      overview: '',
      poster_path: null,
      season_number: 1,
      episode_count: 10,
      air_date: '2023-06-01',
    },
    {
      id: 2,
      name: 'Season 2',
      overview: '',
      poster_path: null,
      season_number: 2,
      episode_count: 12,
      air_date: '2024-01-01',
    },
  ],
  popularity: 100,
  genre_ids: [],
  original_language: 'en',
  origin_country: ['US'],
};

const mockCredits: Credits = {
  id: 123,
  cast: [
    { id: 1, name: '演員一', character: '角色一', profile_path: '/actor1.jpg', order: 0 },
    { id: 2, name: '演員二', character: '角色二', profile_path: null, order: 1 },
  ],
  crew: [
    {
      id: 3,
      name: '導演名',
      job: 'Director',
      department: 'Directing',
      profile_path: '/director.jpg',
    },
  ],
};

describe('MediaDetailPanel', () => {
  describe('Loading state', () => {
    it('should render skeleton when loading', () => {
      render(<MediaDetailPanel type="movie" details={null} isLoading={true} />, {
        wrapper: createWrapper(),
      });
      expect(screen.getByTestId('media-detail-skeleton')).toBeInTheDocument();
    });

    it('should render skeleton when details is null', () => {
      render(<MediaDetailPanel type="movie" details={null} isLoading={false} />, {
        wrapper: createWrapper(),
      });
      expect(screen.getByTestId('media-detail-skeleton')).toBeInTheDocument();
    });
  });

  describe('Movie details', () => {
    it('should render movie title', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} />, {
        wrapper: createWrapper(),
      });
      expect(screen.getByTestId('detail-title')).toHaveTextContent('測試電影');
    });

    it('should render original title when different', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} />, {
        wrapper: createWrapper(),
      });
      expect(screen.getByTestId('detail-original-title')).toHaveTextContent('Test Movie');
    });

    it('should not render original title when same as title', () => {
      const movieWithSameTitle = { ...mockMovieDetails, original_title: '測試電影' };
      render(<MediaDetailPanel type="movie" details={movieWithSameTitle} />, {
        wrapper: createWrapper(),
      });
      expect(screen.queryByTestId('detail-original-title')).not.toBeInTheDocument();
    });

    it('should render release year', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} />, {
        wrapper: createWrapper(),
      });
      expect(screen.getByTestId('detail-year')).toHaveTextContent('2024');
    });

    it('should render runtime in minutes', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} />, {
        wrapper: createWrapper(),
      });
      expect(screen.getByTestId('detail-runtime')).toHaveTextContent('120 分鐘');
    });

    it('should render rating', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} />, {
        wrapper: createWrapper(),
      });
      expect(screen.getByTestId('detail-rating')).toHaveTextContent('8.5');
    });

    it('should render genres as chips', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} />, {
        wrapper: createWrapper(),
      });
      expect(screen.getByTestId('detail-genres')).toHaveTextContent('動作');
      expect(screen.getByTestId('detail-genres')).toHaveTextContent('冒險');
    });

    it('should render overview', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} />, {
        wrapper: createWrapper(),
      });
      expect(screen.getByTestId('detail-overview')).toHaveTextContent(
        '這是一部測試電影的劇情簡介。'
      );
    });

    it('should render fallback text when overview is empty', () => {
      const movieNoOverview = { ...mockMovieDetails, overview: '' };
      render(<MediaDetailPanel type="movie" details={movieNoOverview} />, {
        wrapper: createWrapper(),
      });
      expect(screen.getByTestId('detail-overview')).toHaveTextContent('暫無簡介');
    });

    it('should render director when credits provided', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} credits={mockCredits} />, {
        wrapper: createWrapper(),
      });
      expect(screen.getByText('導演名')).toBeInTheDocument();
    });
  });

  describe('TV Show details', () => {
    it('should render TV show name', () => {
      render(<MediaDetailPanel type="tv" details={mockTVShowDetails} />, {
        wrapper: createWrapper(),
      });
      expect(screen.getByTestId('detail-title')).toHaveTextContent('測試影集');
    });

    it('should render first air date year', () => {
      render(<MediaDetailPanel type="tv" details={mockTVShowDetails} />, {
        wrapper: createWrapper(),
      });
      expect(screen.getByTestId('detail-year')).toHaveTextContent('2023');
    });

    it('should render episode runtime', () => {
      render(<MediaDetailPanel type="tv" details={mockTVShowDetails} />, {
        wrapper: createWrapper(),
      });
      expect(screen.getByTestId('detail-runtime')).toHaveTextContent('45 分鐘');
    });

    it('should render created by for TV shows', () => {
      render(<MediaDetailPanel type="tv" details={mockTVShowDetails} />, {
        wrapper: createWrapper(),
      });
      expect(screen.getByText('創作者一')).toBeInTheDocument();
    });

    it('should render season info for TV shows (AC5)', () => {
      render(<MediaDetailPanel type="tv" details={mockTVShowDetails} />, {
        wrapper: createWrapper(),
      });
      expect(screen.getByTestId('tv-seasons')).toBeInTheDocument();
      expect(screen.getByText('Season 1')).toBeInTheDocument();
      expect(screen.getByText('10 集')).toBeInTheDocument();
      expect(screen.getByText('Season 2')).toBeInTheDocument();
      expect(screen.getByText('12 集')).toBeInTheDocument();
    });
  });

  describe('Images', () => {
    it('should render poster with correct URL', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} />, {
        wrapper: createWrapper(),
      });
      const poster = screen.getByTestId('detail-poster') as HTMLImageElement;
      expect(poster.src).toContain('/w500/poster.jpg');
    });

    it('should not render poster when path is null', () => {
      const movieNoPoster = { ...mockMovieDetails, poster_path: null };
      render(<MediaDetailPanel type="movie" details={movieNoPoster} />, {
        wrapper: createWrapper(),
      });
      expect(screen.queryByTestId('detail-poster')).not.toBeInTheDocument();
    });
  });

  describe('Enhanced features (Story 5.6)', () => {
    it('renders metadata source badge (AC3)', () => {
      render(
        <MediaDetailPanel
          type="movie"
          details={mockMovieDetails}
          metadataSource="tmdb"
          createdAt="2026-01-10T00:00:00Z"
        />,
        { wrapper: createWrapper() }
      );
      expect(screen.getByTestId('metadata-source-badge')).toHaveTextContent('TMDb');
    });

    it('renders file info section (AC4)', () => {
      render(
        <MediaDetailPanel
          type="movie"
          details={mockMovieDetails}
          filePath="/media/Movie.1080p.mkv"
          fileSize={4831838208}
        />,
        { wrapper: createWrapper() }
      );
      expect(screen.getByTestId('file-info')).toBeInTheDocument();
      expect(screen.getByTestId('file-size')).toHaveTextContent('4.5 GB');
    });

    it('renders date added', () => {
      render(
        <MediaDetailPanel
          type="movie"
          details={mockMovieDetails}
          createdAt="2026-01-10T00:00:00Z"
        />,
        { wrapper: createWrapper() }
      );
      expect(screen.getByTestId('detail-date-added')).toHaveTextContent('加入日期');
    });

    it('renders trailer button when libraryId provided (AC2)', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} libraryId="movie-123" />, {
        wrapper: createWrapper(),
      });
      expect(screen.getByTestId('load-trailers-button')).toBeInTheDocument();
    });

    it('renders cast members when credits provided (AC2)', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} credits={mockCredits} />, {
        wrapper: createWrapper(),
      });
      expect(screen.getByTestId('detail-cast')).toBeInTheDocument();
      expect(screen.getByTestId('detail-cast')).toHaveTextContent('演員一');
      expect(screen.getByTestId('detail-cast')).toHaveTextContent('演員二');
    });

    it('does not render cast when credits has no cast', () => {
      const creditsNoCast: Credits = { id: 123, cast: [], crew: mockCredits.crew };
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} credits={creditsNoCast} />, {
        wrapper: createWrapper(),
      });
      expect(screen.queryByTestId('detail-cast')).not.toBeInTheDocument();
    });

    it('[P2] limits cast display to top 5 members', () => {
      const creditsManyCast: Credits = {
        id: 123,
        cast: Array.from({ length: 8 }, (_, i) => ({
          id: i + 1,
          name: `演員${i + 1}`,
          character: `角色${i + 1}`,
          profile_path: null,
          order: i,
        })),
        crew: [],
      };
      render(
        <MediaDetailPanel type="movie" details={mockMovieDetails} credits={creditsManyCast} />,
        { wrapper: createWrapper() }
      );
      const castElement = screen.getByTestId('detail-cast');
      expect(castElement).toHaveTextContent('演員1');
      expect(castElement).toHaveTextContent('演員5');
      expect(castElement).not.toHaveTextContent('演員6');
    });

    it('[P2] separates cast names with Chinese separator', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} credits={mockCredits} />, {
        wrapper: createWrapper(),
      });
      expect(screen.getByTestId('detail-cast')).toHaveTextContent('演員一、演員二');
    });

    it('renders play and add-to-list CTA buttons (AC1)', () => {
      const onPlay = vi.fn();
      const onAddToList = vi.fn();
      render(
        <MediaDetailPanel
          type="movie"
          details={mockMovieDetails}
          onPlay={onPlay}
          onAddToList={onAddToList}
        />,
        { wrapper: createWrapper() }
      );
      expect(screen.getByTestId('detail-play-button')).toHaveTextContent('播放');
      expect(screen.getByTestId('detail-add-to-list-button')).toHaveTextContent('加入清單');
    });

    it('does not render CTA buttons when callbacks not provided', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} />, {
        wrapper: createWrapper(),
      });
      expect(screen.queryByTestId('detail-cta-buttons')).not.toBeInTheDocument();
    });

    it('[P1] calls onPlay when play button is clicked', () => {
      const onPlay = vi.fn();
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} onPlay={onPlay} />, {
        wrapper: createWrapper(),
      });
      fireEvent.click(screen.getByTestId('detail-play-button'));
      expect(onPlay).toHaveBeenCalledOnce();
    });

    it('[P1] calls onAddToList when add-to-list button is clicked', () => {
      const onAddToList = vi.fn();
      render(
        <MediaDetailPanel type="movie" details={mockMovieDetails} onAddToList={onAddToList} />,
        { wrapper: createWrapper() }
      );
      fireEvent.click(screen.getByTestId('detail-add-to-list-button'));
      expect(onAddToList).toHaveBeenCalledOnce();
    });

    it('[P2] renders only play button when onAddToList not provided', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} onPlay={vi.fn()} />, {
        wrapper: createWrapper(),
      });
      expect(screen.getByTestId('detail-play-button')).toBeInTheDocument();
      expect(screen.queryByTestId('detail-add-to-list-button')).not.toBeInTheDocument();
    });

    it('[P2] renders only add-to-list button when onPlay not provided', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} onAddToList={vi.fn()} />, {
        wrapper: createWrapper(),
      });
      expect(screen.queryByTestId('detail-play-button')).not.toBeInTheDocument();
      expect(screen.getByTestId('detail-add-to-list-button')).toBeInTheDocument();
    });

    it('renders context menu when all callbacks provided (AC6)', () => {
      render(
        <MediaDetailPanel
          type="movie"
          details={mockMovieDetails}
          onReparse={vi.fn()}
          onExport={vi.fn()}
          onDelete={vi.fn()}
        />,
        { wrapper: createWrapper() }
      );
      expect(screen.getByTestId('detail-menu-trigger')).toBeInTheDocument();
    });

    it('does not render context menu without callbacks', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} />, {
        wrapper: createWrapper(),
      });
      expect(screen.queryByTestId('detail-menu-trigger')).not.toBeInTheDocument();
    });

    it('[P1] does not render metadata badge when metadataSource not provided', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} />, {
        wrapper: createWrapper(),
      });
      expect(screen.queryByTestId('metadata-source-badge')).not.toBeInTheDocument();
    });

    it('[P1] does not render file info when neither filePath nor fileSize provided', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} />, {
        wrapper: createWrapper(),
      });
      expect(screen.queryByTestId('file-info')).not.toBeInTheDocument();
    });

    it('[P1] does not render date added when createdAt not provided', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} />, {
        wrapper: createWrapper(),
      });
      expect(screen.queryByTestId('detail-date-added')).not.toBeInTheDocument();
    });

    it('[P1] does not render trailer button when libraryId not provided', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} />, {
        wrapper: createWrapper(),
      });
      expect(screen.queryByTestId('load-trailers-button')).not.toBeInTheDocument();
    });
  });

  describe('Edge cases', () => {
    it('[P1] hides rating when vote_average is 0', () => {
      const movieNoRating = { ...mockMovieDetails, vote_average: 0, vote_count: 0 };
      render(<MediaDetailPanel type="movie" details={movieNoRating} />, {
        wrapper: createWrapper(),
      });
      expect(screen.queryByTestId('detail-rating')).not.toBeInTheDocument();
    });

    it('[P1] hides runtime when runtime is 0', () => {
      const movieNoRuntime = { ...mockMovieDetails, runtime: 0 };
      render(<MediaDetailPanel type="movie" details={movieNoRuntime} />, {
        wrapper: createWrapper(),
      });
      expect(screen.queryByTestId('detail-runtime')).not.toBeInTheDocument();
    });

    it('[P1] does not render poster when poster_path is null', () => {
      const movieNoPoster = { ...mockMovieDetails, poster_path: null };
      render(<MediaDetailPanel type="movie" details={movieNoPoster} />, {
        wrapper: createWrapper(),
      });
      expect(screen.queryByTestId('detail-poster')).not.toBeInTheDocument();
    });

    it('[P1] renders TV show without seasons gracefully', () => {
      const tvNoSeasons = { ...mockTVShowDetails, seasons: [] };
      render(<MediaDetailPanel type="tv" details={tvNoSeasons} />, {
        wrapper: createWrapper(),
      });
      expect(screen.queryByTestId('tv-seasons')).not.toBeInTheDocument();
    });

    it('[P1] renders TV show production countries in season section', () => {
      render(<MediaDetailPanel type="tv" details={mockTVShowDetails} />, {
        wrapper: createWrapper(),
      });
      expect(screen.getByText(/United States/)).toBeInTheDocument();
    });

    it('[P1] does not render context menu when only some callbacks provided', () => {
      render(
        <MediaDetailPanel
          type="movie"
          details={mockMovieDetails}
          onReparse={vi.fn()}
          onExport={vi.fn()}
        />,
        { wrapper: createWrapper() }
      );
      expect(screen.queryByTestId('detail-menu-trigger')).not.toBeInTheDocument();
    });
  });
});
