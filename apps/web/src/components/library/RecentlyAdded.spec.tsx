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
            tmdb_id: 123,
            title: 'Test Movie',
            poster_path: '/test.jpg',
            release_date: '2026-03-10',
            created_at: new Date().toISOString(),
            parse_status: 'complete',
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
            tmdb_id: 123,
            title: 'Test Movie',
            poster_path: '/test.jpg',
            release_date: '2026-03-10',
            created_at: new Date().toISOString(),
            parse_status: 'complete',
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
            tmdb_id: 100,
            title: 'Movie One',
            poster_path: '/m1.jpg',
            release_date: '2026-03-10',
            created_at: new Date().toISOString(),
            parse_status: 'complete',
            genres: [],
          },
        },
        {
          type: 'series',
          series: {
            id: 's1',
            tmdb_id: 200,
            title: 'Series One',
            poster_path: '/s1.jpg',
            first_air_date: '2026-03-08',
            created_at: new Date().toISOString(),
            parse_status: 'complete',
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
});
