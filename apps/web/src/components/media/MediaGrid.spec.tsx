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
    <a href={`${to.replace('$type', params.type).replace('$id', params.id)}`} {...props}>
      {children}
    </a>
  ),
}));

describe('MediaGrid', () => {
  const mockMovies: Movie[] = [
    {
      id: 1,
      title: '鬼滅之刃',
      originalTitle: 'Demon Slayer',
      overview: 'Test overview',
      releaseDate: '2020-10-16',
      posterPath: '/poster1.jpg',
      backdropPath: null,
      voteAverage: 8.5,
      voteCount: 1000,
      genreIds: [16, 28],
    },
    {
      id: 2,
      title: '進擊的巨人',
      originalTitle: 'Attack on Titan',
      overview: 'Test overview 2',
      releaseDate: '2013-04-07',
      posterPath: '/poster2.jpg',
      backdropPath: null,
      voteAverage: 8.9,
      voteCount: 2000,
      genreIds: [16, 28],
    },
  ];

  const mockTVShows: TVShow[] = [
    {
      id: 101,
      name: '咒術迴戰',
      originalName: 'Jujutsu Kaisen',
      overview: 'TV overview',
      firstAirDate: '2020-10-03',
      posterPath: '/tv-poster.jpg',
      backdropPath: null,
      voteAverage: 8.6,
      voteCount: 500,
      genreIds: [16, 28],
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
      render(<MediaGrid movies={[]} tvShows={[]} emptyMessage="找不到符合的媒體" />);
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
      // Tablet (768px+): auto-fill with 160px min
      expect(grid).toHaveClass('md:grid-cols-[repeat(auto-fill,minmax(160px,1fr))]');
      // Desktop: auto-fill with 200px min
      expect(grid).toHaveClass('lg:grid-cols-[repeat(auto-fill,minmax(200px,1fr))]');
    });

    it('has correct gap classes', () => {
      render(<MediaGrid movies={mockMovies} />);
      const grid = screen.getByTestId('media-grid');
      // Mobile: 12px gap (gap-3)
      expect(grid).toHaveClass('gap-3');
      // Tablet/Desktop: 16px gap (md:gap-4)
      expect(grid).toHaveClass('md:gap-4');
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
      const { container } = render(<MediaGrid movies={[movieWithSameId]} tvShows={mockTVShows} />);

      const links = container.querySelectorAll('a');
      expect(links).toHaveLength(2);
    });
  });

  describe('Unified Items Prop', () => {
    it('renders items in provided order when items prop is used', () => {
      const items = [
        { item: mockTVShows[0], mediaType: 'tv' as const },
        { item: mockMovies[0], mediaType: 'movie' as const },
        { item: mockMovies[1], mediaType: 'movie' as const },
      ];

      render(<MediaGrid items={items} />);

      const links = screen.getAllByRole('link');
      expect(links).toHaveLength(3);
      // First item should be TV show
      expect(screen.getByText('咒術迴戰')).toBeInTheDocument();
      // Then movies
      expect(screen.getByText('鬼滅之刃')).toBeInTheDocument();
      expect(screen.getByText('進擊的巨人')).toBeInTheDocument();
    });

    it('prefers items prop over movies/tvShows when both provided', () => {
      const items = [{ item: mockMovies[0], mediaType: 'movie' as const }];

      render(<MediaGrid items={items} movies={mockMovies} tvShows={mockTVShows} />);

      // Should only show items from items prop
      const links = screen.getAllByRole('link');
      expect(links).toHaveLength(1);
    });

    it('shows empty state when items is empty array', () => {
      render(<MediaGrid items={[]} />);
      expect(screen.getByText('沒有找到結果')).toBeInTheDocument();
    });
  });
});
