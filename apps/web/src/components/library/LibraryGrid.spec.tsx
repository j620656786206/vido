import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { LibraryGrid } from './LibraryGrid';
import type { LibraryItem } from '../../types/library';

vi.mock('@tanstack/react-router', () => ({
  Link: ({
    children,
    to,
    params,
    ...props
  }: {
    children: React.ReactNode;
    to: string;
    params?: Record<string, string>;
    [key: string]: unknown;
  }) => {
    let href = to;
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        href = href.replace(`$${key}`, value);
      });
    }
    return (
      <a href={href} {...props}>
        {children}
      </a>
    );
  },
}));

const mockItems: LibraryItem[] = [
  {
    type: 'movie',
    movie: {
      id: 'movie-1',
      title: 'Test Movie',
      original_title: 'Test Movie EN',
      release_date: '2023-01-15',
      genres: ['Action'],
      vote_average: 8.5,
      poster_path: '/poster.jpg',
      tmdb_id: 123,
      parse_status: 'success',
      created_at: '2023-01-01',
      updated_at: '2023-01-01',
    },
  },
  {
    type: 'series',
    series: {
      id: 'series-1',
      title: 'Test Series',
      first_air_date: '2023-06-01',
      genres: ['Drama'],
      vote_average: 9.0,
      poster_path: '/poster2.jpg',
      tmdb_id: 456,
      parse_status: 'success',
      created_at: '2023-01-01',
      updated_at: '2023-01-01',
    },
  },
];

describe('LibraryGrid', () => {
  it('renders loading skeletons when isLoading is true', () => {
    render(<LibraryGrid items={[]} isLoading={true} />);
    expect(screen.getByTestId('library-grid-loading')).toBeInTheDocument();
  });

  it('renders nothing when items are empty and not loading', () => {
    const { container } = render(<LibraryGrid items={[]} />);
    expect(container.querySelector('[data-testid="library-grid"]')).not.toBeInTheDocument();
  });

  it('renders poster cards for items', () => {
    render(<LibraryGrid items={mockItems} />);
    expect(screen.getByTestId('library-grid')).toBeInTheDocument();
    expect(screen.getByText('Test Movie')).toBeInTheDocument();
    expect(screen.getByText('Test Series')).toBeInTheDocument();
  });

  it('renders correct number of items', () => {
    render(<LibraryGrid items={mockItems} />);
    const cards = screen.getAllByTestId('poster-card');
    expect(cards).toHaveLength(2);
  });

  it('applies density settings', () => {
    const { container } = render(<LibraryGrid items={mockItems} density="large" />);
    const grid = container.querySelector('[data-testid="library-grid"]');
    expect(grid).toBeInTheDocument();
  });
});
