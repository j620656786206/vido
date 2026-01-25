import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { MediaDetailPanel } from './MediaDetailPanel';
import type { MovieDetails, TVShowDetails, Credits } from '../../types/tmdb';

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
  created_by: [{ id: 1, name: '創作者一', profile_path: null }],
  networks: [{ id: 1, name: 'Netflix', logo_path: null }],
  in_production: true,
  seasons: [],
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
      render(<MediaDetailPanel type="movie" details={null} isLoading={true} />);
      expect(screen.getByTestId('media-detail-skeleton')).toBeInTheDocument();
    });

    it('should render skeleton when details is null', () => {
      render(<MediaDetailPanel type="movie" details={null} isLoading={false} />);
      expect(screen.getByTestId('media-detail-skeleton')).toBeInTheDocument();
    });
  });

  describe('Movie details', () => {
    it('should render movie title', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} />);
      expect(screen.getByTestId('detail-title')).toHaveTextContent('測試電影');
    });

    it('should render original title when different', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} />);
      expect(screen.getByTestId('detail-original-title')).toHaveTextContent('Test Movie');
    });

    it('should not render original title when same as title', () => {
      const movieWithSameTitle = { ...mockMovieDetails, original_title: '測試電影' };
      render(<MediaDetailPanel type="movie" details={movieWithSameTitle} />);
      expect(screen.queryByTestId('detail-original-title')).not.toBeInTheDocument();
    });

    it('should render release year', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} />);
      expect(screen.getByTestId('detail-year')).toHaveTextContent('2024');
    });

    it('should render runtime in minutes', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} />);
      expect(screen.getByTestId('detail-runtime')).toHaveTextContent('120 分鐘');
    });

    it('should render rating', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} />);
      expect(screen.getByTestId('detail-rating')).toHaveTextContent('8.5');
    });

    it('should render genres as chips', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} />);
      expect(screen.getByTestId('detail-genres')).toHaveTextContent('動作');
      expect(screen.getByTestId('detail-genres')).toHaveTextContent('冒險');
    });

    it('should render overview', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} />);
      expect(screen.getByTestId('detail-overview')).toHaveTextContent(
        '這是一部測試電影的劇情簡介。'
      );
    });

    it('should render fallback text when overview is empty', () => {
      const movieNoOverview = { ...mockMovieDetails, overview: '' };
      render(<MediaDetailPanel type="movie" details={movieNoOverview} />);
      expect(screen.getByTestId('detail-overview')).toHaveTextContent('暫無簡介');
    });

    it('should render director when credits provided', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} credits={mockCredits} />);
      expect(screen.getByText('導演名')).toBeInTheDocument();
    });
  });

  describe('TV Show details', () => {
    it('should render TV show name', () => {
      render(<MediaDetailPanel type="tv" details={mockTVShowDetails} />);
      expect(screen.getByTestId('detail-title')).toHaveTextContent('測試影集');
    });

    it('should render first air date year', () => {
      render(<MediaDetailPanel type="tv" details={mockTVShowDetails} />);
      expect(screen.getByTestId('detail-year')).toHaveTextContent('2023');
    });

    it('should render episode runtime', () => {
      render(<MediaDetailPanel type="tv" details={mockTVShowDetails} />);
      expect(screen.getByTestId('detail-runtime')).toHaveTextContent('45 分鐘');
    });

    it('should render created by for TV shows', () => {
      render(<MediaDetailPanel type="tv" details={mockTVShowDetails} />);
      expect(screen.getByText('創作者一')).toBeInTheDocument();
    });
  });

  describe('Images', () => {
    it('should render poster with correct URL', () => {
      render(<MediaDetailPanel type="movie" details={mockMovieDetails} />);
      const poster = screen.getByTestId('detail-poster') as HTMLImageElement;
      expect(poster.src).toContain('/w500/poster.jpg');
    });

    it('should not render poster when path is null', () => {
      const movieNoPoster = { ...mockMovieDetails, poster_path: null };
      render(<MediaDetailPanel type="movie" details={movieNoPoster} />);
      expect(screen.queryByTestId('detail-poster')).not.toBeInTheDocument();
    });
  });
});
