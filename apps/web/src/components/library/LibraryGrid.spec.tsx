import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
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

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

function renderWithQuery(ui: React.ReactElement) {
  const Wrapper = createWrapper();
  return render(ui, { wrapper: Wrapper });
}

const mockItems: LibraryItem[] = [
  {
    type: 'movie',
    movie: {
      id: 'movie-1',
      title: 'Test Movie',
      originalTitle: 'Test Movie EN',
      releaseDate: '2023-01-15',
      genres: ['Action'],
      voteAverage: 8.5,
      posterPath: '/poster.jpg',
      tmdbId: 123,
      parseStatus: 'success',
      createdAt: '2023-01-01',
      updatedAt: '2023-01-01',
    },
  },
  {
    type: 'series',
    series: {
      id: 'series-1',
      title: 'Test Series',
      firstAirDate: '2023-06-01',
      genres: ['Drama'],
      voteAverage: 9.0,
      posterPath: '/poster2.jpg',
      tmdbId: 456,
      parseStatus: 'success',
      createdAt: '2023-01-01',
      updatedAt: '2023-01-01',
    },
  },
];

describe('LibraryGrid', () => {
  it('renders loading skeletons when isLoading is true', () => {
    renderWithQuery(<LibraryGrid items={[]} isLoading={true} />);
    expect(screen.getByTestId('library-grid-loading')).toBeInTheDocument();
  });

  it('renders nothing when items are empty and not loading', () => {
    const { container } = renderWithQuery(<LibraryGrid items={[]} />);
    expect(container.querySelector('[data-testid="library-grid"]')).not.toBeInTheDocument();
  });

  it('renders poster cards for items', () => {
    renderWithQuery(<LibraryGrid items={mockItems} />);
    expect(screen.getByTestId('library-grid')).toBeInTheDocument();
    expect(screen.getByText('Test Movie')).toBeInTheDocument();
    expect(screen.getByText('Test Series')).toBeInTheDocument();
  });

  it('renders correct number of items', () => {
    renderWithQuery(<LibraryGrid items={mockItems} />);
    const cards = screen.getAllByTestId('poster-card');
    expect(cards).toHaveLength(2);
  });

  it('applies density settings', () => {
    const { container } = renderWithQuery(<LibraryGrid items={mockItems} density="large" />);
    const grid = container.querySelector('[data-testid="library-grid"]');
    expect(grid).toBeInTheDocument();
  });

  it('renders correct skeleton count for small density', () => {
    renderWithQuery(<LibraryGrid items={[]} isLoading={true} density="small" />);
    const loading = screen.getByTestId('library-grid-loading');
    // small density = 18 skeletons
    expect(loading.children).toHaveLength(18);
  });

  it('renders correct skeleton count for medium density', () => {
    renderWithQuery(<LibraryGrid items={[]} isLoading={true} density="medium" />);
    const loading = screen.getByTestId('library-grid-loading');
    // medium density = 12 skeletons
    expect(loading.children).toHaveLength(12);
  });

  it('renders correct skeleton count for large density', () => {
    renderWithQuery(<LibraryGrid items={[]} isLoading={true} density="large" />);
    const loading = screen.getByTestId('library-grid-loading');
    // large density = 8 skeletons
    expect(loading.children).toHaveLength(8);
  });

  it('skips items with mismatched type/data', () => {
    const itemsWithNull: LibraryItem[] = [{ type: 'movie', movie: undefined }, ...mockItems];
    renderWithQuery(<LibraryGrid items={itemsWithNull} />);
    const cards = screen.getAllByTestId('poster-card');
    // Only the 2 valid items should render
    expect(cards).toHaveLength(2);
  });

  it('uses normal grid when totalItems <= 1000', () => {
    renderWithQuery(<LibraryGrid items={mockItems} totalItems={500} />);
    expect(screen.getByTestId('library-grid')).toBeInTheDocument();
  });

  it('maps series firstAirDate to releaseDate', () => {
    const seriesOnly: LibraryItem[] = [
      {
        type: 'series',
        series: {
          id: 's-1',
          title: 'Series Title',
          firstAirDate: '2024-06-15',
          genres: [],
          voteAverage: 7.5,
          posterPath: '/poster.jpg',
          tmdbId: 789,
          parseStatus: 'success',
          createdAt: '2024-01-01',
          updatedAt: '2024-01-01',
        },
      },
    ];
    renderWithQuery(<LibraryGrid items={seriesOnly} />);
    expect(screen.getByText('2024')).toBeInTheDocument();
  });

  it('maps movie internal id to PosterCard id', () => {
    renderWithQuery(<LibraryGrid items={[mockItems[0]]} />);
    const link = screen.getByRole('link');
    expect(link).toHaveAttribute('href', '/media/movie/movie-1');
  });

  it('renders menu button on each poster card', () => {
    renderWithQuery(<LibraryGrid items={mockItems} />);
    const menuButtons = screen.getAllByTestId('poster-menu-button');
    expect(menuButtons).toHaveLength(2);
  });

  it('opens context menu when menu button is clicked', () => {
    renderWithQuery(<LibraryGrid items={[mockItems[0]]} />);
    const menuButton = screen.getByTestId('poster-menu-button');
    fireEvent.click(menuButton);
    expect(screen.getByTestId('poster-card-menu')).toBeInTheDocument();
  });

  describe('Selection Mode (Story 5-7)', () => {
    it('[P0] hides menu buttons when selectionMode is true', () => {
      renderWithQuery(<LibraryGrid items={mockItems} selectionMode={true} />);
      expect(screen.queryAllByTestId('poster-menu-button')).toHaveLength(0);
    });

    it('[P0] renders selection checkboxes when selectionMode is true', () => {
      renderWithQuery(<LibraryGrid items={mockItems} selectionMode={true} />);
      expect(screen.getAllByTestId('selection-checkbox')).toHaveLength(2);
    });

    it('[P0] marks items as selected based on selectedIds', () => {
      const selectedIds = new Set(['movie-1']);
      renderWithQuery(
        <LibraryGrid items={mockItems} selectionMode={true} selectedIds={selectedIds} />
      );
      const checkboxes = screen.getAllByTestId('selection-checkbox');
      // First item (movie-1) should have check icon, second (series-1) should not
      expect(checkboxes[0].querySelector('svg')).toBeInTheDocument();
      expect(checkboxes[1].querySelector('svg')).not.toBeInTheDocument();
    });

    it('[P0] calls onSelect with itemId when card is clicked in selection mode', () => {
      const onSelect = vi.fn();
      renderWithQuery(
        <LibraryGrid items={[mockItems[0]]} selectionMode={true} onSelect={onSelect} />
      );

      fireEvent.click(screen.getByTestId('poster-card'));
      expect(onSelect).toHaveBeenCalledWith('movie-1', expect.any(Object));
    });

    it('[P1] does not render selection checkboxes when selectionMode is false', () => {
      renderWithQuery(<LibraryGrid items={mockItems} selectionMode={false} />);
      expect(screen.queryAllByTestId('selection-checkbox')).toHaveLength(0);
    });

    it('[P1] does not render PosterCardMenu when selectionMode is active', () => {
      renderWithQuery(<LibraryGrid items={mockItems} selectionMode={true} />);
      // No menu buttons to click, so no menu should appear
      expect(screen.queryByTestId('poster-card-menu')).not.toBeInTheDocument();
    });
  });
});
