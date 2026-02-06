import { render, screen, waitFor } from '@testing-library/react';
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

// Mock the tmdb service
vi.mock('../services/tmdb', () => ({
  tmdbService: {
    searchMovies: vi.fn().mockResolvedValue({
      page: 1,
      results: [],
      total_pages: 0,
      total_results: 0,
    }),
    searchTVShows: vi.fn().mockResolvedValue({
      page: 1,
      results: [],
      total_pages: 0,
      total_results: 0,
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

  it('should render search page with title', async () => {
    renderWithProviders();

    await waitFor(() => {
      expect(screen.getByText('搜尋媒體')).toBeInTheDocument();
    });
  });

  it('should render search input with placeholder', async () => {
    renderWithProviders();

    await waitFor(() => {
      expect(screen.getByPlaceholderText('搜尋電影或影集...')).toBeInTheDocument();
    });
  });

  it('should display search query from URL parameters', async () => {
    renderWithProviders({ q: '鬼滅之刃' });

    await waitFor(() => {
      const input = screen.getByPlaceholderText('搜尋電影或影集...') as HTMLInputElement;
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
});
