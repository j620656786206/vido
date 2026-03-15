import { useState, useCallback } from 'react';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useLibraryList, useLibrarySearch } from '../hooks/useLibrary';
import { LibraryGrid } from '../components/library/LibraryGrid';
import { LibraryTable } from '../components/library/LibraryTable';
import type { SortField } from '../components/library/LibraryTable';
import { SortSelector } from '../components/library/SortSelector';
import type { SortField as SortSelectorField, SortOrder } from '../components/library/SortSelector';
import { LibrarySearchBar } from '../components/library/LibrarySearchBar';
import { EmptySearchResults } from '../components/library/EmptySearchResults';
import { RecentlyAdded } from '../components/library/RecentlyAdded';
import { EmptyLibrary } from '../components/library/EmptyLibrary';
import { ViewToggle } from '../components/library/ViewToggle';
import type { ViewMode } from '../components/library/ViewToggle';
import { Pagination } from '../components/ui/Pagination';
import type { LibraryMediaType, LibraryItem } from '../types/library';

const VIEW_STORAGE_KEY = 'vido:library:view';
const SORT_STORAGE_KEY = 'vido:library:sort';

interface SortPreference {
  sortBy: string;
  sortOrder: 'asc' | 'desc';
}

const DEFAULT_SORT: SortPreference = { sortBy: 'created_at', sortOrder: 'desc' };

function getStoredView(): ViewMode {
  try {
    const stored = localStorage.getItem(VIEW_STORAGE_KEY);
    if (stored === 'grid' || stored === 'list') return stored;
  } catch {
    // ignore
  }
  return 'grid';
}

function setStoredView(view: ViewMode) {
  localStorage.setItem(VIEW_STORAGE_KEY, view);
}

function getStoredSort(): SortPreference {
  try {
    const stored = localStorage.getItem(SORT_STORAGE_KEY);
    if (stored) {
      const parsed = JSON.parse(stored);
      if (parsed.sortBy && (parsed.sortOrder === 'asc' || parsed.sortOrder === 'desc')) {
        return parsed;
      }
    }
  } catch {
    // ignore
  }
  return DEFAULT_SORT;
}

function setStoredSort(pref: SortPreference) {
  localStorage.setItem(SORT_STORAGE_KEY, JSON.stringify(pref));
}

interface LibrarySearchParams {
  page?: number;
  pageSize?: number;
  type?: LibraryMediaType;
  sortBy?: string;
  sortOrder?: string;
  view?: string;
  q?: string;
}

export const Route = createFileRoute('/library')({
  validateSearch: (search: Record<string, unknown>): LibrarySearchParams => ({
    page: typeof search.page === 'number' ? search.page : 1,
    pageSize: typeof search.pageSize === 'number' ? search.pageSize : 20,
    type: ['all', 'movie', 'tv'].includes(search.type as string)
      ? (search.type as LibraryMediaType)
      : 'all',
    sortBy: typeof search.sortBy === 'string' ? search.sortBy : undefined,
    sortOrder: ['asc', 'desc'].includes(search.sortOrder as string)
      ? (search.sortOrder as string)
      : undefined,
    view: ['grid', 'list'].includes(search.view as string) ? (search.view as string) : undefined,
    q: typeof search.q === 'string' ? search.q : undefined,
  }),
  component: LibraryPage,
});

function LibraryPage() {
  const { page, pageSize, type, sortBy, sortOrder, view: viewParam, q } = Route.useSearch();
  const navigate = useNavigate({ from: Route.fullPath });

  const [currentView, setCurrentView] = useState<ViewMode>(
    () => (viewParam as ViewMode) || getStoredView()
  );
  const [searchQuery, setSearchQuery] = useState(q || '');

  const currentPage = page || 1;
  const currentPageSize = pageSize || 20;
  const currentType = type || 'all';
  const isSearchActive = searchQuery.length >= 2;

  // URL params > localStorage > default
  const storedSort = useState(() => getStoredSort())[0];
  const effectiveSortBy = sortBy || storedSort.sortBy;
  const effectiveSortOrder: 'asc' | 'desc' = (sortOrder as 'asc' | 'desc') || storedSort.sortOrder;

  // Show recently added only in clean browse mode (no custom sort/filter/search)
  const isCleanBrowse = !sortBy && !sortOrder && !isSearchActive;

  const listQuery = useLibraryList({
    page: currentPage,
    pageSize: currentPageSize,
    type: currentType,
    sortBy: effectiveSortBy,
    sortOrder: effectiveSortOrder,
  });

  const searchResult = useLibrarySearch(searchQuery, {
    page: currentPage,
    pageSize: currentPageSize,
    type: currentType,
  });

  const buildSearchParams = useCallback(
    (overrides: Partial<LibrarySearchParams> = {}): LibrarySearchParams => ({
      page: overrides.page ?? currentPage,
      pageSize: overrides.pageSize ?? currentPageSize,
      type: overrides.type ?? currentType,
      sortBy: overrides.sortBy !== undefined ? overrides.sortBy : sortBy || undefined,
      sortOrder: overrides.sortOrder !== undefined ? overrides.sortOrder : sortOrder || undefined,
      view: currentView !== 'grid' ? currentView : undefined,
      q: overrides.q !== undefined ? overrides.q : searchQuery || undefined,
    }),
    [currentPage, currentPageSize, currentType, sortBy, sortOrder, currentView, searchQuery]
  );

  const handleSearch = useCallback(
    (query: string) => {
      setSearchQuery(query);
      navigate({
        search: buildSearchParams({ page: 1, q: query || undefined }),
      });
    },
    [navigate, buildSearchParams]
  );

  const handlePageChange = (newPage: number) => {
    navigate({ search: buildSearchParams({ page: newPage }) });
  };

  const handleTypeChange = (newType: LibraryMediaType) => {
    navigate({ search: buildSearchParams({ page: 1, type: newType }) });
  };

  const handleViewChange = useCallback(
    (newView: ViewMode) => {
      setCurrentView(newView);
      setStoredView(newView);
      navigate({
        search: { ...buildSearchParams(), view: newView !== 'grid' ? newView : undefined },
      });
    },
    [navigate, buildSearchParams]
  );

  const handleColumnSort = useCallback(
    (field: SortField) => {
      const newOrder = sortBy === field && sortOrder === 'asc' ? 'desc' : 'asc';
      setStoredSort({ sortBy: field, sortOrder: newOrder });
      navigate({ search: buildSearchParams({ page: 1, sortBy: field, sortOrder: newOrder }) });
    },
    [sortBy, sortOrder, navigate, buildSearchParams]
  );

  const handleSortChange = useCallback(
    (field: SortSelectorField, order: SortOrder) => {
      setStoredSort({ sortBy: field, sortOrder: order });
      navigate({ search: buildSearchParams({ page: 1, sortBy: field, sortOrder: order }) });
    },
    [navigate, buildSearchParams]
  );

  // Derive display data based on search mode
  let items: LibraryItem[] = [];
  let totalItems = 0;
  let totalPages = 0;
  let isLoading = false;

  if (isSearchActive) {
    isLoading = searchResult.isLoading;
    const results = searchResult.data?.results ?? [];
    totalItems = searchResult.data?.totalCount ?? 0;
    totalPages = totalItems > 0 ? Math.ceil(totalItems / currentPageSize) : 0;
    // Convert SearchResult[] to LibraryItem[] for grid/table components
    items = results.map((r) => ({
      type: r.type,
      movie: r.movie,
      series: r.series,
    }));
  } else {
    isLoading = listQuery.isLoading;
    totalItems = listQuery.data?.totalItems ?? 0;
    totalPages = listQuery.data?.totalPages ?? 0;
    items = listQuery.data?.items ?? [];
  }

  const isEmpty = !isLoading && items.length === 0;
  const isSearchEmpty = isSearchActive && isEmpty;
  const isLibraryEmpty = !isSearchActive && isEmpty;

  return (
    <div>
      <div className="container mx-auto px-4 py-8">
        {/* Search bar - always visible unless library is empty with no search */}
        {!isLibraryEmpty && (
          <div className="mb-6">
            <LibrarySearchBar
              onSearch={handleSearch}
              initialQuery={searchQuery}
              resultCount={isSearchActive ? totalItems : undefined}
            />
          </div>
        )}

        {!isEmpty && (
          <div className="mb-6 flex items-center justify-between">
            <span className="text-sm text-slate-400">
              顯示 {(currentPage - 1) * currentPageSize + 1}-
              {Math.min(currentPage * currentPageSize, totalItems)} / {totalItems} 項
            </span>
            <div className="flex items-center gap-2">
              <SortSelector
                sortBy={effectiveSortBy as SortSelectorField}
                sortOrder={effectiveSortOrder}
                onSortChange={handleSortChange}
              />
              <ViewToggle view={currentView} onViewChange={handleViewChange} />
            </div>
          </div>
        )}

        {/* Type filter tabs */}
        {!isEmpty && (
          <div className="mb-6 flex gap-2">
            {(['all', 'movie', 'tv'] as const).map((t) => (
              <button
                key={t}
                onClick={() => handleTypeChange(t)}
                className={`rounded-lg px-4 py-2 text-sm font-medium transition-colors ${
                  currentType === t
                    ? 'bg-blue-600 text-white'
                    : 'bg-slate-800 text-slate-400 hover:bg-slate-700 hover:text-white'
                }`}
              >
                {t === 'all' ? '全部' : t === 'movie' ? '電影' : '影集'}
              </button>
            ))}
          </div>
        )}

        {isSearchEmpty ? (
          <EmptySearchResults query={searchQuery} onClear={() => handleSearch('')} />
        ) : isLibraryEmpty ? (
          <EmptyLibrary />
        ) : (
          <>
            {isCleanBrowse && <RecentlyAdded />}
            {currentView === 'grid' ? (
              <LibraryGrid items={items} isLoading={isLoading} totalItems={totalItems} />
            ) : (
              <LibraryTable
                items={items}
                isLoading={isLoading}
                sortBy={effectiveSortBy}
                sortOrder={effectiveSortOrder}
                onSort={handleColumnSort}
              />
            )}
            <Pagination
              currentPage={currentPage}
              totalPages={totalPages}
              onPageChange={handlePageChange}
              className="mt-8"
            />
          </>
        )}
      </div>
    </div>
  );
}
