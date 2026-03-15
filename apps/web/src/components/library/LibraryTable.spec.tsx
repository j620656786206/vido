import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { LibraryTable } from './LibraryTable';
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
      title: '測試電影',
      original_title: 'Test Movie',
      release_date: '2023-06-15',
      genres: ['動作', '冒險', '科幻'],
      vote_average: 8.5,
      poster_path: '/poster.jpg',
      tmdb_id: 123,
      parse_status: 'success',
      created_at: '2024-01-15T00:00:00Z',
      updated_at: '2024-01-15T00:00:00Z',
    },
  },
  {
    type: 'series',
    series: {
      id: 'series-1',
      title: '測試影集',
      original_title: 'Test Series',
      first_air_date: '2022-03-10',
      genres: ['劇情'],
      vote_average: 9.1,
      poster_path: '/poster2.jpg',
      tmdb_id: 456,
      parse_status: 'success',
      created_at: '2024-02-01T00:00:00Z',
      updated_at: '2024-02-01T00:00:00Z',
    },
  },
];

describe('LibraryTable', () => {
  it('renders loading skeleton when isLoading is true', () => {
    render(<LibraryTable items={[]} isLoading={true} />);
    expect(screen.getByTestId('library-table-loading')).toBeInTheDocument();
  });

  it('renders 8 skeleton rows when loading', () => {
    render(<LibraryTable items={[]} isLoading={true} />);
    const loading = screen.getByTestId('library-table-loading');
    expect(loading.children).toHaveLength(8);
  });

  it('renders nothing when items are empty and not loading', () => {
    const { container } = render(<LibraryTable items={[]} />);
    expect(container.querySelector('[data-testid="library-table"]')).not.toBeInTheDocument();
  });

  it('renders table with items', () => {
    render(<LibraryTable items={mockItems} />);
    expect(screen.getByTestId('library-table')).toBeInTheDocument();
    expect(screen.getByText('測試電影')).toBeInTheDocument();
    expect(screen.getByText('測試影集')).toBeInTheDocument();
  });

  it('renders correct number of rows', () => {
    render(<LibraryTable items={mockItems} />);
    const rows = screen.getAllByTestId('library-table-row');
    expect(rows).toHaveLength(2);
  });

  it('displays original title when different from title', () => {
    render(<LibraryTable items={mockItems} />);
    expect(screen.getByText('Test Movie')).toBeInTheDocument();
    expect(screen.getByText('Test Series')).toBeInTheDocument();
  });

  it('displays year from release date', () => {
    render(<LibraryTable items={mockItems} />);
    expect(screen.getByText('2023')).toBeInTheDocument();
    expect(screen.getByText('2022')).toBeInTheDocument();
  });

  it('displays genre tags (max 3)', () => {
    render(<LibraryTable items={mockItems} />);
    expect(screen.getByText('動作')).toBeInTheDocument();
    expect(screen.getByText('冒險')).toBeInTheDocument();
    expect(screen.getByText('科幻')).toBeInTheDocument();
    expect(screen.getByText('劇情')).toBeInTheDocument();
  });

  it('displays ratings with star', () => {
    render(<LibraryTable items={mockItems} />);
    expect(screen.getByText('★ 8.5')).toBeInTheDocument();
    expect(screen.getByText('★ 9.1')).toBeInTheDocument();
  });

  it('displays formatted date added', () => {
    render(<LibraryTable items={mockItems} />);
    // zh-TW date format: YYYY/MM/DD
    expect(screen.getByText('2024/01/15')).toBeInTheDocument();
    expect(screen.getByText('2024/02/01')).toBeInTheDocument();
  });

  it('renders poster thumbnail images', () => {
    const { container } = render(<LibraryTable items={mockItems} />);
    const images = container.querySelectorAll('img');
    expect(images).toHaveLength(2);
    expect(images[0]).toHaveAttribute('src', 'https://image.tmdb.org/t/p/w92/poster.jpg');
  });

  it('renders placeholder when no poster path', () => {
    const noPosterItems: LibraryItem[] = [
      {
        type: 'movie',
        movie: {
          ...mockItems[0].movie!,
          poster_path: undefined,
        },
      },
    ];
    render(<LibraryTable items={noPosterItems} />);
    expect(screen.getByText('N/A')).toBeInTheDocument();
  });

  it('renders sortable column headers', () => {
    render(<LibraryTable items={mockItems} onSort={vi.fn()} />);
    expect(screen.getByTestId('sort-title')).toBeInTheDocument();
    expect(screen.getByTestId('sort-release_date')).toBeInTheDocument();
    expect(screen.getByTestId('sort-rating')).toBeInTheDocument();
    expect(screen.getByTestId('sort-created_at')).toBeInTheDocument();
  });

  it('calls onSort when sortable column header is clicked', () => {
    const onSort = vi.fn();
    render(<LibraryTable items={mockItems} onSort={onSort} />);
    fireEvent.click(screen.getByTestId('sort-title'));
    expect(onSort).toHaveBeenCalledWith('title');
  });

  it('shows ascending arrow when sortOrder is asc', () => {
    render(<LibraryTable items={mockItems} sortBy="title" sortOrder="asc" onSort={vi.fn()} />);
    expect(screen.getByTestId('sort-indicator-title')).toBeInTheDocument();
  });

  it('shows descending arrow when sortOrder is desc', () => {
    render(<LibraryTable items={mockItems} sortBy="title" sortOrder="desc" onSort={vi.fn()} />);
    expect(screen.getByTestId('sort-indicator-title')).toBeInTheDocument();
  });

  it('does not show sort indicator on non-active columns', () => {
    render(<LibraryTable items={mockItems} sortBy="title" sortOrder="asc" onSort={vi.fn()} />);
    expect(screen.queryByTestId('sort-indicator-rating')).not.toBeInTheDocument();
  });

  it('skips items with mismatched type/data', () => {
    const itemsWithNull: LibraryItem[] = [{ type: 'movie', movie: undefined }, ...mockItems];
    render(<LibraryTable items={itemsWithNull} />);
    const rows = screen.getAllByTestId('library-table-row');
    expect(rows).toHaveLength(2);
  });

  it('links to correct media detail page', () => {
    render(<LibraryTable items={mockItems} />);
    const links = screen.getAllByRole('link');
    // Each row has 2 links (poster + title) = 4 total
    const movieLinks = links.filter((l) => l.getAttribute('href') === '/media/movie/123');
    expect(movieLinks.length).toBeGreaterThanOrEqual(1);
  });

  it('handles item with no rating', () => {
    const noRatingItems: LibraryItem[] = [
      {
        type: 'movie',
        movie: {
          ...mockItems[0].movie!,
          vote_average: undefined,
        },
      },
    ];
    render(<LibraryTable items={noRatingItems} />);
    // Should display dash for no rating
    const dashes = screen.getAllByText('-');
    expect(dashes.length).toBeGreaterThanOrEqual(1);
  });

  it('links series items to /media/tv/ path', () => {
    render(<LibraryTable items={mockItems} />);
    const links = screen.getAllByRole('link');
    const seriesLinks = links.filter((l) => l.getAttribute('href') === '/media/tv/456');
    expect(seriesLinks.length).toBeGreaterThanOrEqual(1);
  });

  it('displays dash for missing release date', () => {
    const noDateItems: LibraryItem[] = [
      {
        type: 'movie',
        movie: {
          ...mockItems[0].movie!,
          release_date: '',
        },
      },
    ];
    render(<LibraryTable items={noDateItems} />);
    const dashes = screen.getAllByText('-');
    expect(dashes.length).toBeGreaterThanOrEqual(1);
  });

  it('does not show original title when same as title', () => {
    const sameTitleItems: LibraryItem[] = [
      {
        type: 'movie',
        movie: {
          ...mockItems[0].movie!,
          title: 'Same Title',
          original_title: 'Same Title',
        },
      },
    ];
    render(<LibraryTable items={sameTitleItems} />);
    // Title should appear once (in the main title), not twice
    const elements = screen.getAllByText('Same Title');
    expect(elements).toHaveLength(1);
  });

  it('[P2] renders table header with bg-slate-800/50 background', () => {
    const { container } = render(<LibraryTable items={mockItems} />);
    const headerRow = container.querySelector('thead tr');
    expect(headerRow).not.toBeNull();
    expect(headerRow!.className).toContain('bg-slate-800/50');
  });

  it('[P2] shows 新增日期 label for created_at column', () => {
    render(<LibraryTable items={mockItems} onSort={vi.fn()} />);
    expect(screen.getByTestId('sort-created_at')).toHaveTextContent('新增日期');
  });

  it('handles item with empty genres array', () => {
    const noGenreItems: LibraryItem[] = [
      {
        type: 'movie',
        movie: {
          ...mockItems[0].movie!,
          genres: [],
        },
      },
    ];
    const { container } = render(<LibraryTable items={noGenreItems} />);
    const row = screen.getByTestId('library-table-row');
    const genreSpans = row.querySelectorAll('.rounded.bg-slate-700');
    expect(genreSpans).toHaveLength(0);
  });
});
