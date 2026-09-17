import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  createMemoryHistory,
  RouterProvider,
  createRootRoute,
  createRoute,
  createRouter,
} from '@tanstack/react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

// Import the route component for testing
import { Route as SearchRoute } from './search';
import { tmdbService } from '../services/tmdb';

// Mock the tmdb service
vi.mock('../services/tmdb', () => ({
  tmdbService: {
    searchMovies: vi.fn().mockResolvedValue({
      page: 1,
      results: [],
      totalPages: 0,
      totalResults: 0,
    }),
    searchTVShows: vi.fn().mockResolvedValue({
      page: 1,
      results: [],
      totalPages: 0,
      totalResults: 0,
    }),
  },
  getImageUrl: vi.fn((path) => (path ? `https://image.tmdb.org/t/p/w342${path}` : null)),
}));

// Create a test router setup
function createTestRouter(initialSearch = {}) {
  const rootRoute = createRootRoute();

  const searchRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/search',
    validateSearch: SearchRoute.options.validateSearch,
    component: SearchRoute.options.component,
  });

  const routeTree = rootRoute.addChildren([searchRoute]);

  const router = createRouter({
    routeTree,
    history: createMemoryHistory({
      initialEntries: [
        `/search?${new URLSearchParams(initialSearch as Record<string, string>).toString()}`,
      ],
    }),
  });

  return router;
}

function renderWithProviders(initialSearch = {}) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  const router = createTestRouter(initialSearch);

  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  );
}

describe('SearchPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render search page with search input', async () => {
    renderWithProviders();

    await waitFor(() => {
      expect(screen.getByPlaceholderText('搜尋媒體庫...')).toBeInTheDocument();
    });
  });

  it('should render search input with placeholder', async () => {
    renderWithProviders();

    await waitFor(() => {
      expect(screen.getByPlaceholderText('搜尋媒體庫...')).toBeInTheDocument();
    });
  });

  it('should display search query from URL parameters', async () => {
    renderWithProviders({ q: '鬼滅之刃' });

    await waitFor(() => {
      const input = screen.getByPlaceholderText('搜尋媒體庫...') as HTMLInputElement;
      expect(input.value).toBe('鬼滅之刃');
    });
  });

  it('should show minimum character message when query is 1 character', async () => {
    renderWithProviders({ q: '鬼' });

    await waitFor(() => {
      expect(screen.getByText('請輸入至少 2 個字元進行搜尋')).toBeInTheDocument();
    });
  });

  it('should show empty state when query returns no results', async () => {
    renderWithProviders({ q: '測試搜尋' });

    await waitFor(() => {
      expect(screen.getByText(/找不到符合的結果/)).toBeInTheDocument();
    });
  });

  it('should render SearchBar component with clear button when query exists', async () => {
    renderWithProviders({ q: '鬼滅之刃' });

    await waitFor(() => {
      expect(screen.getByLabelText('清除搜尋')).toBeInTheDocument();
    });
  });

  // dsr-8 AC #5: a failed search is not an empty search. It used to read
  // 「找不到符合的結果，請嘗試使用不同的關鍵字搜尋」 — blaming the user's words for a server fault.
  describe('when the search request fails', () => {
    const fail = () => Object.assign(new Error('TMDb unreachable'), { code: 'TMDB_TIMEOUT' });
    const movieHit = {
      page: 1,
      results: [
        {
          id: 372058,
          title: '你的名字',
          originalTitle: '君の名は。',
          overview: '',
          posterPath: null,
          backdropPath: null,
          releaseDate: '2016-08-26',
          voteAverage: 8.4,
          voteCount: 100,
          genreIds: [],
        },
      ],
      totalPages: 1,
      totalResults: 1,
    };

    it('all needed queries failed → an error with the code and a retry, never 找不到', async () => {
      vi.mocked(tmdbService.searchMovies).mockRejectedValue(fail());
      vi.mocked(tmdbService.searchTVShows).mockRejectedValue(fail());
      renderWithProviders({ q: '你的名字' });

      const alert = await screen.findByTestId('search-error');
      expect(alert).toHaveAttribute('role', 'alert');
      expect(alert).toHaveTextContent('搜尋暫時無法使用，請稍後再試');
      expect(screen.getByTestId('search-error-code')).toHaveTextContent('TMDB_TIMEOUT');
      expect(screen.queryByText(/找不到符合的結果/)).toBeNull();
      expect(screen.getByTestId('search-error-retry')).toBeInTheDocument();
    });

    it('one side failed → keeps the good results, says which side failed, prints no half-count', async () => {
      vi.mocked(tmdbService.searchMovies).mockResolvedValue(movieHit as never);
      vi.mocked(tmdbService.searchTVShows).mockRejectedValue(fail());
      renderWithProviders({ q: '你的名字' });

      const banner = await screen.findByTestId('search-partial-error');
      expect(banner).toHaveAttribute('role', 'alert');
      expect(banner).toHaveTextContent('影集結果暫時無法載入，其他結果不受影響');
      expect(screen.getByText('你的名字')).toBeInTheDocument();
      expect(screen.queryByText(/找到 \d+ 個結果/)).toBeNull();
      expect(screen.queryByTestId('search-error')).toBeNull();
      // 全部 would otherwise read 1 — the movie half only.
      expect(screen.getByRole('tab', { name: '全部' })).toHaveTextContent(/^全部$/);
    });

    it('one side failed and the other found nothing → no "try other keywords"', async () => {
      vi.mocked(tmdbService.searchMovies).mockResolvedValue({
        page: 1,
        results: [],
        totalPages: 0,
        totalResults: 0,
      });
      vi.mocked(tmdbService.searchTVShows).mockRejectedValue(fail());
      renderWithProviders({ q: '你的名字' });

      await screen.findByTestId('search-partial-error');
      expect(screen.queryByText(/找不到符合的結果/)).toBeNull();
    });

    it('while the retry is in flight the page shows loading, never an error or 找不到', async () => {
      vi.mocked(tmdbService.searchMovies).mockResolvedValue(movieHit as never);
      vi.mocked(tmdbService.searchTVShows).mockRejectedValue(fail());
      renderWithProviders({ q: '你的名字' });

      const retry = await screen.findByTestId('search-partial-error-retry');
      expect(retry).toBeEnabled();
      vi.mocked(tmdbService.searchTVShows).mockReturnValue(new Promise(() => {}));
      fireEvent.click(retry);
      // query-core 5.90: refetching a query with no cached data drops it back to
      // pending, so the banner gives way to the loading grid (nothing left to double-click).
      await waitFor(() => expect(screen.queryByTestId('search-partial-error')).toBeNull());
      expect(screen.getByTestId('media-grid')).toBeInTheDocument();
      expect(screen.queryByTestId('search-error')).toBeNull();
      expect(screen.queryByText(/找不到符合的結果/)).toBeNull();
    });

    it('retry refetches only the query that failed', async () => {
      vi.mocked(tmdbService.searchMovies).mockResolvedValue(movieHit as never);
      vi.mocked(tmdbService.searchTVShows).mockRejectedValue(fail());
      renderWithProviders({ q: '你的名字' });

      await screen.findByTestId('search-partial-error');
      const moviesCalls = vi.mocked(tmdbService.searchMovies).mock.calls.length;
      const tvCalls = vi.mocked(tmdbService.searchTVShows).mock.calls.length;
      fireEvent.click(screen.getByTestId('search-partial-error-retry'));
      await waitFor(() =>
        expect(vi.mocked(tmdbService.searchTVShows).mock.calls.length).toBe(tvCalls + 1)
      );
      expect(vi.mocked(tmdbService.searchMovies).mock.calls.length).toBe(moviesCalls);
    });

    it('a failure in a query the current tab does not use is ignored', async () => {
      vi.mocked(tmdbService.searchMovies).mockResolvedValue(movieHit as never);
      vi.mocked(tmdbService.searchTVShows).mockRejectedValue(fail());
      renderWithProviders({ q: '你的名字', type: 'movie' });

      expect(await screen.findByText('你的名字')).toBeInTheDocument();
      // Both queries always run on /search, so 全部 still has no count to give.
      await waitFor(() =>
        expect(screen.getByRole('tab', { name: '全部' })).toHaveTextContent(/^全部$/)
      );
      expect(screen.queryByTestId('search-partial-error')).toBeNull();
      expect(screen.queryByTestId('search-error')).toBeNull();
    });
  });
});
