import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { RecentlyAdded } from './RecentlyAdded';

// Mock TanStack Router
vi.mock('@tanstack/react-router', () => ({
  Link: ({
    children,
    to,
    search,
    ...props
  }: {
    children: React.ReactNode;
    to: string;
    search?: Record<string, string>;
  }) => {
    const qs = search ? '?' + new URLSearchParams(search).toString() : '';
    return (
      <a href={`${to}${qs}`} {...props}>
        {children}
      </a>
    );
  },
}));

// Mock the useRecentlyAdded hook
const mockUseRecentlyAdded = vi.fn();
vi.mock('../../hooks/useLibrary', () => ({
  useRecentlyAdded: (...args: unknown[]) => mockUseRecentlyAdded(...args),
}));

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

describe('RecentlyAdded', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders section with title "最近新增"', () => {
    mockUseRecentlyAdded.mockReturnValue({
      data: [
        {
          type: 'movie',
          movie: {
            id: 'm1',
            tmdbId: 123,
            title: 'Test Movie',
            posterPath: '/test.jpg',
            releaseDate: '2026-03-10',
            createdAt: new Date().toISOString(),
            parseStatus: 'complete',
            genres: [],
          },
        },
      ],
      isLoading: false,
    });

    render(<RecentlyAdded />, { wrapper: createWrapper() });
    expect(screen.getByText('最近新增')).toBeInTheDocument();
  });

  it('renders "查看全部" link with correct URL', () => {
    mockUseRecentlyAdded.mockReturnValue({
      data: [
        {
          type: 'movie',
          movie: {
            id: 'm1',
            tmdbId: 123,
            title: 'Test Movie',
            posterPath: '/test.jpg',
            releaseDate: '2026-03-10',
            createdAt: new Date().toISOString(),
            parseStatus: 'complete',
            genres: [],
          },
        },
      ],
      isLoading: false,
    });

    render(<RecentlyAdded />, { wrapper: createWrapper() });
    const link = screen.getByText(/查看全部/);
    expect(link).toBeInTheDocument();
    expect(link.closest('a')).toHaveAttribute('href', expect.stringContaining('/library'));
  });

  it('renders poster cards for each item', () => {
    mockUseRecentlyAdded.mockReturnValue({
      data: [
        {
          type: 'movie',
          movie: {
            id: 'm1',
            tmdbId: 100,
            title: 'Movie One',
            posterPath: '/m1.jpg',
            releaseDate: '2026-03-10',
            createdAt: new Date().toISOString(),
            parseStatus: 'complete',
            genres: [],
          },
        },
        {
          type: 'series',
          series: {
            id: 's1',
            tmdbId: 200,
            title: 'Series One',
            posterPath: '/s1.jpg',
            firstAirDate: '2026-03-08',
            createdAt: new Date().toISOString(),
            parseStatus: 'complete',
            genres: [],
          },
        },
      ],
      isLoading: false,
    });

    render(<RecentlyAdded />, { wrapper: createWrapper() });
    expect(screen.getByText('Movie One')).toBeInTheDocument();
    expect(screen.getByText('Series One')).toBeInTheDocument();
  });

  it('shows skeleton loading state', () => {
    mockUseRecentlyAdded.mockReturnValue({
      data: undefined,
      isLoading: true,
    });

    render(<RecentlyAdded />, { wrapper: createWrapper() });
    expect(screen.getByTestId('recently-added-section')).toBeInTheDocument();
    // Should show 8 skeletons
    const skeletons = screen
      .getByTestId('recently-added-section')
      .querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBe(8);
  });

  it('returns null when no data and not loading', () => {
    mockUseRecentlyAdded.mockReturnValue({
      data: [],
      isLoading: false,
    });

    const { container } = render(<RecentlyAdded />, { wrapper: createWrapper() });
    expect(container.firstChild).toBeNull();
  });

  it('calls useRecentlyAdded with limit 20', () => {
    mockUseRecentlyAdded.mockReturnValue({
      data: [],
      isLoading: false,
    });

    render(<RecentlyAdded />, { wrapper: createWrapper() });
    expect(mockUseRecentlyAdded).toHaveBeenCalledWith(20);
  });

  it('passes isNew=true for items created within 7 days', () => {
    const recentDate = new Date(Date.now() - 3 * 24 * 60 * 60 * 1000).toISOString(); // 3 days ago
    mockUseRecentlyAdded.mockReturnValue({
      data: [
        {
          type: 'movie',
          movie: {
            id: 'new1',
            tmdbId: 500,
            title: 'New Movie',
            posterPath: '/new.jpg',
            releaseDate: '2026-03-10',
            createdAt: recentDate,
            parseStatus: 'complete',
            genres: [],
          },
        },
      ],
      isLoading: false,
    });

    render(<RecentlyAdded />, { wrapper: createWrapper() });
    expect(screen.getByText('New Movie')).toBeInTheDocument();
    expect(screen.getByTestId('new-badge')).toBeInTheDocument();
  });

  it('passes isNew=false for items created more than 7 days ago', () => {
    const oldDate = new Date(Date.now() - 8 * 24 * 60 * 60 * 1000).toISOString(); // 8 days ago
    mockUseRecentlyAdded.mockReturnValue({
      data: [
        {
          type: 'movie',
          movie: {
            id: 'old1',
            tmdbId: 501,
            title: 'Old Movie',
            posterPath: '/old.jpg',
            releaseDate: '2026-01-01',
            createdAt: oldDate,
            parseStatus: 'complete',
            genres: [],
          },
        },
      ],
      isLoading: false,
    });

    render(<RecentlyAdded />, { wrapper: createWrapper() });
    expect(screen.getByText('Old Movie')).toBeInTheDocument();
    expect(screen.queryByTestId('new-badge')).not.toBeInTheDocument();
  });

  it('maps series items correctly with isNew badge', () => {
    const recentDate = new Date(Date.now() - 1 * 24 * 60 * 60 * 1000).toISOString(); // 1 day ago
    mockUseRecentlyAdded.mockReturnValue({
      data: [
        {
          type: 'series',
          series: {
            id: 's1',
            tmdbId: 600,
            title: 'New Series',
            posterPath: '/series.jpg',
            firstAirDate: '2026-03-01',
            createdAt: recentDate,
            parseStatus: 'complete',
            genres: [],
          },
        },
      ],
      isLoading: false,
    });

    render(<RecentlyAdded />, { wrapper: createWrapper() });
    expect(screen.getByText('New Series')).toBeInTheDocument();
    expect(screen.getByTestId('new-badge')).toBeInTheDocument();
  });

  it('handles mixed new and old items correctly', () => {
    const recentDate = new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString();
    const oldDate = new Date(Date.now() - 10 * 24 * 60 * 60 * 1000).toISOString();
    mockUseRecentlyAdded.mockReturnValue({
      data: [
        {
          type: 'movie',
          movie: {
            id: 'new1',
            tmdbId: 700,
            title: 'Fresh Movie',
            posterPath: '/fresh.jpg',
            releaseDate: '2026-03-10',
            createdAt: recentDate,
            parseStatus: 'complete',
            genres: [],
          },
        },
        {
          type: 'movie',
          movie: {
            id: 'old1',
            tmdbId: 701,
            title: 'Stale Movie',
            posterPath: '/stale.jpg',
            releaseDate: '2025-12-01',
            createdAt: oldDate,
            parseStatus: 'complete',
            genres: [],
          },
        },
      ],
      isLoading: false,
    });

    render(<RecentlyAdded />, { wrapper: createWrapper() });
    expect(screen.getByText('Fresh Movie')).toBeInTheDocument();
    expect(screen.getByText('Stale Movie')).toBeInTheDocument();
    // Only 1 badge for the fresh item
    const badges = screen.getAllByTestId('new-badge');
    expect(badges).toHaveLength(1);
  });

  it('renders "查看全部" link with sortBy and sortOrder params', () => {
    mockUseRecentlyAdded.mockReturnValue({
      data: [
        {
          type: 'movie',
          movie: {
            id: 'm1',
            tmdbId: 800,
            title: 'Test',
            posterPath: '/t.jpg',
            releaseDate: '2026-03-10',
            createdAt: new Date().toISOString(),
            parseStatus: 'complete',
            genres: [],
          },
        },
      ],
      isLoading: false,
    });

    render(<RecentlyAdded />, { wrapper: createWrapper() });
    const link = screen.getByText(/查看全部/).closest('a');
    expect(link).toHaveAttribute('href', expect.stringContaining('sortBy=created_at'));
    expect(link).toHaveAttribute('href', expect.stringContaining('sortOrder=desc'));
  });
});
