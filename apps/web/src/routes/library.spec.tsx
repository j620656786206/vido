import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  createMemoryHistory,
  RouterProvider,
  createRootRoute,
  createRoute,
  createRouter,
} from '@tanstack/react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

import { Route as LibraryRoute } from './library';

vi.mock('../services/libraryService', () => ({
  libraryService: {
    listLibrary: vi.fn(),
    getRecentlyAdded: vi.fn(),
    searchLibrary: vi.fn(),
    deleteMovie: vi.fn(),
    deleteSeries: vi.fn(),
    reparseMovie: vi.fn(),
    reparseSeries: vi.fn(),
    exportMovie: vi.fn(),
    exportSeries: vi.fn(),
  },
}));

function getMockListResponse() {
  return {
    items: [
      {
        type: 'movie' as const,
        movie: {
          id: 'movie-1',
          title: '測試電影',
          original_title: 'Test Movie',
          release_date: '2023-06-15',
          genres: ['動作'],
          vote_average: 8.5,
          poster_path: '/poster.jpg',
          tmdb_id: 123,
          parse_status: 'success',
          created_at: '2024-01-15T00:00:00Z',
          updated_at: '2024-01-15T00:00:00Z',
        },
      },
      {
        type: 'series' as const,
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
    ],
    page: 1,
    pageSize: 20,
    totalItems: 2,
    totalPages: 1,
  };
}

function getEmptyResponse() {
  return {
    items: [],
    page: 1,
    pageSize: 20,
    totalItems: 0,
    totalPages: 0,
  };
}

function createTestRouter(initialSearch: Record<string, string> = {}) {
  const rootRoute = createRootRoute();

  const libraryRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/library',
    validateSearch: LibraryRoute.options.validateSearch,
    component: LibraryRoute.options.component,
  });

  const routeTree = rootRoute.addChildren([libraryRoute]);

  const searchStr = Object.keys(initialSearch).length
    ? `?${new URLSearchParams(initialSearch).toString()}`
    : '';

  const router = createRouter({
    routeTree,
    history: createMemoryHistory({
      initialEntries: [`/library${searchStr}`],
    }),
  });

  return router;
}

function getMockSearchResponse(empty = false) {
  if (empty) {
    return { results: [], totalCount: 0 };
  }
  return {
    results: [
      {
        type: 'movie' as const,
        movie: {
          id: 'movie-1',
          title: '駭客任務',
          original_title: 'The Matrix',
          release_date: '1999-03-31',
          genres: ['動作', '科幻'],
          vote_average: 8.7,
          poster_path: '/matrix.jpg',
          tmdb_id: 603,
          parse_status: 'success',
          created_at: '2024-01-15T00:00:00Z',
          updated_at: '2024-01-15T00:00:00Z',
        },
      },
    ],
    totalCount: 1,
  };
}

async function setupMocks(overrides?: { listEmpty?: boolean; searchEmpty?: boolean }) {
  const { libraryService } = await import('../services/libraryService');
  const listResponse = overrides?.listEmpty ? getEmptyResponse() : getMockListResponse();
  vi.mocked(libraryService.listLibrary).mockResolvedValue(listResponse);
  // getRecentlyAdded returns LibraryItem[] (not full response)
  vi.mocked(libraryService.getRecentlyAdded).mockResolvedValue(
    overrides?.listEmpty ? [] : [getMockListResponse().items[0]]
  );
  vi.mocked(libraryService.searchLibrary).mockResolvedValue(
    getMockSearchResponse(overrides?.searchEmpty)
  );
}

function renderLibrary(initialSearch: Record<string, string> = {}) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });

  const router = createTestRouter(initialSearch);

  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  );
}

describe('LibraryPage', () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    localStorage.clear();
    await setupMocks();
  });

  describe('AC1: View Toggle Control', () => {
    it('[P1] renders ViewToggle component when library has items', async () => {
      renderLibrary();

      await waitFor(() => {
        expect(screen.getByTestId('view-toggle')).toBeInTheDocument();
      });
    });

    it('[P1] defaults to grid view', async () => {
      renderLibrary();

      await waitFor(() => {
        expect(screen.getByLabelText('格狀檢視')).toHaveAttribute('aria-checked', 'true');
      });
    });

    it('[P1] switches to list view when list button clicked', async () => {
      renderLibrary();

      await waitFor(() => {
        expect(screen.getByTestId('view-toggle')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByLabelText('列表檢視'));

      await waitFor(() => {
        expect(screen.getByTestId('library-table')).toBeInTheDocument();
      });
    });

    it('[P1] switches back to grid view when grid button clicked', async () => {
      renderLibrary({ view: 'list' });

      await waitFor(() => {
        expect(screen.getByTestId('library-table')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByLabelText('格狀檢視'));

      await waitFor(() => {
        expect(screen.queryByTestId('library-table')).not.toBeInTheDocument();
      });
    });

    it('[P2] does not render ViewToggle when library is empty', async () => {
      await setupMocks({ listEmpty: true });

      renderLibrary();

      await waitFor(() => {
        expect(screen.queryByTestId('view-toggle')).not.toBeInTheDocument();
      });
    });
  });

  describe('AC3: Column Sorting', () => {
    it('[P1] renders sortable column headers in list view', async () => {
      renderLibrary({ view: 'list' });

      await waitFor(() => {
        expect(screen.getByTestId('sort-title')).toBeInTheDocument();
        expect(screen.getByTestId('sort-release_date')).toBeInTheDocument();
        expect(screen.getByTestId('sort-rating')).toBeInTheDocument();
        expect(screen.getByTestId('sort-created_at')).toBeInTheDocument();
      });
    });

    it('[P1] shows sort indicator when sortBy is set via URL', async () => {
      renderLibrary({ view: 'list', sortBy: 'title', sortOrder: 'asc' });

      await waitFor(() => {
        expect(screen.getByTestId('sort-indicator-title')).toBeInTheDocument();
      });
    });
  });

  describe('AC4: View Preference Persistence', () => {
    it('[P1] persists view preference to localStorage when toggled', async () => {
      renderLibrary();

      await waitFor(() => {
        expect(screen.getByTestId('view-toggle')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByLabelText('列表檢視'));

      expect(localStorage.getItem('vido:library:view')).toBe('list');
    });

    it('[P1] reads view preference from localStorage on load', async () => {
      localStorage.setItem('vido:library:view', 'list');

      renderLibrary();

      await waitFor(() => {
        expect(screen.getByTestId('library-table')).toBeInTheDocument();
      });
    });

    it('[P2] defaults to grid when localStorage is empty', async () => {
      renderLibrary();

      await waitFor(() => {
        expect(screen.getByLabelText('格狀檢視')).toHaveAttribute('aria-checked', 'true');
      });
    });

    it('[P2] defaults to grid when localStorage has invalid value', async () => {
      localStorage.setItem('vido:library:view', 'invalid');

      renderLibrary();

      await waitFor(() => {
        expect(screen.getByLabelText('格狀檢視')).toHaveAttribute('aria-checked', 'true');
      });
    });

    it('[P2] URL view param overrides localStorage preference', async () => {
      localStorage.setItem('vido:library:view', 'list');

      renderLibrary({ view: 'grid' });

      await waitFor(() => {
        expect(screen.getByLabelText('格狀檢視')).toHaveAttribute('aria-checked', 'true');
      });
    });
  });

  describe('Type filter tabs (inside filter sidebar)', () => {
    it('[P1] renders type filter chips inside filter sidebar when opened', async () => {
      renderLibrary();

      await waitFor(() => {
        expect(screen.getByTestId('filter-toggle')).toBeInTheDocument();
      });

      // Open filter sidebar
      await userEvent.click(screen.getByTestId('filter-toggle'));

      await waitFor(() => {
        expect(screen.getByTestId('filter-type-all')).toBeInTheDocument();
        expect(screen.getByTestId('filter-type-movie')).toBeInTheDocument();
        expect(screen.getByTestId('filter-type-tv')).toBeInTheDocument();
      });
    });

    it('[P2] does not render filter toggle when library is empty', async () => {
      await setupMocks({ listEmpty: true });

      renderLibrary();

      await waitFor(() => {
        expect(screen.queryByTestId('filter-toggle')).not.toBeInTheDocument();
      });
    });
  });

  describe('Empty state', () => {
    it('[P1] renders EmptyLibrary when no items', async () => {
      await setupMocks({ listEmpty: true });

      renderLibrary();

      await waitFor(() => {
        expect(screen.getByText('你的媒體庫還是空的')).toBeInTheDocument();
      });
    });
  });

  describe('Section heading display', () => {
    it('[P2] shows 全部媒體 section heading', async () => {
      renderLibrary();

      await waitFor(() => {
        expect(screen.getByText('全部媒體')).toBeInTheDocument();
      });
    });
  });

  describe('AC6: Filter toggle visible during search', () => {
    it('[P1] shows filter toggle button during active search', async () => {
      renderLibrary({ q: '駭客' });

      await waitFor(() => {
        expect(screen.getByTestId('filter-toggle')).toBeInTheDocument();
      });
    });

    it('[P1] can open filter sidebar during active search', async () => {
      renderLibrary({ q: '駭客' });

      await waitFor(() => {
        expect(screen.getByTestId('filter-toggle')).toBeInTheDocument();
      });

      await userEvent.click(screen.getByTestId('filter-toggle'));

      await waitFor(() => {
        expect(screen.getByTestId('filter-sidebar')).toBeInTheDocument();
      });
    });

    it('[P2] hides SortSelector during active search but keeps filter toggle', async () => {
      renderLibrary({ q: '駭客' });

      await waitFor(() => {
        expect(screen.getByTestId('filter-toggle')).toBeInTheDocument();
      });

      expect(screen.queryByTestId('sort-selector-button')).not.toBeInTheDocument();
    });
  });

  describe('Search integration (Story 5-3)', () => {
    it('[P1] renders search bar when library has items', async () => {
      renderLibrary();

      await waitFor(() => {
        expect(screen.getByPlaceholderText('搜尋媒體標題...')).toBeInTheDocument();
      });
    });

    it('[P1] does not render search bar when library is empty', async () => {
      await setupMocks({ listEmpty: true });
      renderLibrary();

      await waitFor(() => {
        expect(screen.getByText('你的媒體庫還是空的')).toBeInTheDocument();
      });

      expect(screen.queryByPlaceholderText('搜尋媒體標題...')).not.toBeInTheDocument();
    });

    it('[P1] initializes search bar with q param from URL', async () => {
      renderLibrary({ q: '駭客' });

      await waitFor(() => {
        const input = screen.getByPlaceholderText('搜尋媒體標題...') as HTMLInputElement;
        expect(input.value).toBe('駭客');
      });
    });

    it('[P1] shows search results when q param ≥ 2 chars', async () => {
      renderLibrary({ q: '駭客' });

      await waitFor(() => {
        expect(screen.getByTestId('search-result-count')).toBeInTheDocument();
      });
    });

    it('[P1] shows EmptySearchResults for no-match query', async () => {
      await setupMocks({ searchEmpty: true });
      renderLibrary({ q: 'zzzzz' });

      await waitFor(() => {
        expect(screen.getByTestId('empty-search-results')).toBeInTheDocument();
        expect(screen.getByText('找不到相關結果')).toBeInTheDocument();
      });
    });

    it('[P2] hides RecentlyAdded section during active search', async () => {
      renderLibrary({ q: '駭客' });

      await waitFor(() => {
        expect(screen.getByTestId('search-result-count')).toBeInTheDocument();
      });

      // RecentlyAdded should not be visible during search
      expect(screen.queryByText('最近新增')).not.toBeInTheDocument();
    });

    it('[P2] preserves sort and view params alongside search', async () => {
      renderLibrary({ q: '駭客', view: 'list', sortBy: 'title', sortOrder: 'asc' });

      await waitFor(() => {
        // List view should be active
        expect(screen.getByTestId('library-table')).toBeInTheDocument();
      });
    });
  });
});
