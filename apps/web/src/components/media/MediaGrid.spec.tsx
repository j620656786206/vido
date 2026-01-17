import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { MediaGrid } from './MediaGrid';
import type { Movie, TVShow } from '../../types/tmdb';

// Mock TanStack Router
vi.mock('@tanstack/react-router', () => ({
  Link: ({
    children,
    to,
    params,
    ...props
  }: {
    children: React.ReactNode;
    to: string;
    params: Record<string, string>;
  }) => (
    <a
      href={`${to.replace('$type', params.type).replace('$id', params.id)}`}
      {...props}
    >
      {children}
    </a>
  ),
}));

describe('MediaGrid', () => {
  const mockMovies: Movie[] = [
    {
      id: 1,
      title: '鬼滅之刃',
      original_title: 'Demon Slayer',
      overview: 'Test overview',
      release_date: '2020-10-16',
      poster_path: '/poster1.jpg',
      backdrop_path: null,
      vote_average: 8.5,
      vote_count: 1000,
      genre_ids: [16, 28],
    },
    {
      id: 2,
      title: '進擊的巨人',
      original_title: 'Attack on Titan',
      overview: 'Test overview 2',
      release_date: '2013-04-07',
      poster_path: '/poster2.jpg',
      backdrop_path: null,
      vote_average: 8.9,
      vote_count: 2000,
      genre_ids: [16, 28],
    },
  ];

  const mockTVShows: TVShow[] = [
    {
      id: 101,
      name: '咒術迴戰',
      original_name: 'Jujutsu Kaisen',
      overview: 'TV overview',
      first_air_date: '2020-10-03',
      poster_path: '/tv-poster.jpg',
      backdrop_path: null,
      vote_average: 8.6,
      vote_count: 500,
      genre_ids: [16, 28],
    },
  ];

  describe('Rendering Results', () => {
    it('renders movie cards correctly', () => {
      render(<MediaGrid movies={mockMovies} />);
      expect(screen.getByText('鬼滅之刃')).toBeInTheDocument();
      expect(screen.getByText('進擊的巨人')).toBeInTheDocument();
    });

    it('renders TV show cards correctly', () => {
      render(<MediaGrid tvShows={mockTVShows} />);
      expect(screen.getByText('咒術迴戰')).toBeInTheDocument();
    });

    it('renders both movies and TV shows together', () => {
      render(<MediaGrid movies={mockMovies} tvShows={mockTVShows} />);
      expect(screen.getByText('鬼滅之刃')).toBeInTheDocument();
      expect(screen.getByText('進擊的巨人')).toBeInTheDocument();
      expect(screen.getByText('咒術迴戰')).toBeInTheDocument();
    });

    it('renders correct number of items', () => {
      render(<MediaGrid movies={mockMovies} tvShows={mockTVShows} />);
      const links = screen.getAllByRole('link');
      expect(links).toHaveLength(3);
    });
  });

  describe('Loading State', () => {
    it('shows skeleton cards when loading', () => {
      render(<MediaGrid isLoading />);
      const skeletons = screen.getAllByTestId('poster-card-skeleton');
      expect(skeletons.length).toBeGreaterThan(0);
    });

    it('shows 12 skeleton cards by default', () => {
      render(<MediaGrid isLoading />);
      const skeletons = screen.getAllByTestId('poster-card-skeleton');
      expect(skeletons).toHaveLength(12);
    });
  });

  describe('Empty State', () => {
    it('shows empty state when no results', () => {
      render(<MediaGrid movies={[]} tvShows={[]} />);
      expect(screen.getByText('沒有找到結果')).toBeInTheDocument();
    });

    it('shows custom empty message', () => {
      render(
        <MediaGrid movies={[]} tvShows={[]} emptyMessage="找不到符合的媒體" />
      );
      expect(screen.getByText('找不到符合的媒體')).toBeInTheDocument();
    });

    it('shows empty icon', () => {
      render(<MediaGrid movies={[]} tvShows={[]} />);
      expect(screen.getByText('🔍')).toBeInTheDocument();
    });
  });

  describe('Responsive Grid Layout', () => {
    it('has correct CSS grid classes', () => {
      render(<MediaGrid movies={mockMovies} />);
      const grid = screen.getByTestId('media-grid');
      // Mobile: 2 columns
      expect(grid).toHaveClass('grid-cols-2');
      // Tablet: auto-fill with 160px min
      expect(grid).toHaveClass('sm:grid-cols-[repeat(auto-fill,minmax(160px,1fr))]');
      // Desktop: auto-fill with 200px min
      expect(grid).toHaveClass('lg:grid-cols-[repeat(auto-fill,minmax(200px,1fr))]');
    });

    it('has correct gap classes', () => {
      render(<MediaGrid movies={mockMovies} />);
      const grid = screen.getByTestId('media-grid');
      // Mobile: 12px gap (gap-3)
      expect(grid).toHaveClass('gap-3');
      // Tablet/Desktop: 16px gap (sm:gap-4)
      expect(grid).toHaveClass('sm:gap-4');
    });
  });

  describe('Unique Keys', () => {
    it('renders items with unique keys (no duplicate key warning)', () => {
      // Movies and TV shows can have the same ID, so keys must include type
      const movieWithSameId: Movie = {
        ...mockMovies[0],
        id: 101, // Same ID as TV show
      };

      // This should not cause any console warnings about duplicate keys
      const { container } = render(
        <MediaGrid movies={[movieWithSameId]} tvShows={mockTVShows} />
      );

      const links = container.querySelectorAll('a');
      expect(links).toHaveLength(2);
    });
  });
});
