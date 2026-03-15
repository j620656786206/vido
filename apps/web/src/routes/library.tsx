import { useState, useMemo, useCallback } from 'react';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { Filter } from 'lucide-react';
import { useLibraryList, useLibrarySearch, useLibraryStats } from '../hooks/useLibrary';
import { LibraryGrid } from '../components/library/LibraryGrid';
import { LibraryTable } from '../components/library/LibraryTable';
import { SortSelector } from '../components/library/SortSelector';
import { LibrarySearchBar } from '../components/library/LibrarySearchBar';
import { FilterPanel } from '../components/library/FilterPanel';
import type { FilterValues } from '../components/library/FilterPanel';
import { FilterChips } from '../components/library/FilterChips';
import { EmptySearchResults } from '../components/library/EmptySearchResults';
import { RecentlyAdded } from '../components/library/RecentlyAdded';
import { EmptyLibrary } from '../components/library/EmptyLibrary';
import { ViewToggle } from '../components/library/ViewToggle';
import type { ViewMode } from '../components/library/ViewToggle';
import { Pagination } from '../components/ui/Pagination';
import type { LibraryMediaType, LibraryItem, SortField, SortOrder } from '../types/library';
import { VALID_SORT_FIELDS } from '../types/library';

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
      if (
        VALID_SORT_FIELDS.includes(parsed.sortBy) &&
        (parsed.sortOrder === 'asc' || parsed.sortOrder === 'desc')
      ) {
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
  genres?: string;
  yearMin?: number;
  yearMax?: number;
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
    genres: typeof search.genres === 'string' ? search.genres : undefined,
    yearMin: typeof search.yearMin === 'number' ? search.yearMin : undefined,
    yearMax: typeof search.yearMax === 'number' ? search.yearMax : undefined,
  }),
  component: LibraryPage,
});

function LibraryPage() {
  const {
    page,
    pageSize,
    type,
    sortBy,
    sortOrder,
    view: viewParam,
    q,
    genres: genresParam,
    yearMin: yearMinParam,
    yearMax: yearMaxParam,
  } = Route.useSearch();
  const navigate = useNavigate({ from: Route.fullPath });

  const [currentView, setCurrentView] = useState<ViewMode>(
    () => (viewParam as ViewMode) || getStoredView()
  );
  const [searchQuery, setSearchQuery] = useState(q || '');
  const [isFilterOpen, setIsFilterOpen] = useState(false);

  const currentPage = page || 1;
  const currentPageSize = pageSize || 20;
  const currentType = type || 'all';
  const isSearchActive = searchQuery.length >= 2;

  // Parse filter state from URL
  const currentFilters: FilterValues = useMemo(
    () => ({
      genres: genresParam ? genresParam.split(',').filter(Boolean) : [],
      yearMin: yearMinParam,
      yearMax: yearMaxParam,
    }),
    [genresParam, yearMinParam, yearMaxParam]
  );

  const hasActiveFilters =
    currentFilters.genres.length > 0 ||
    currentFilters.yearMin !== undefined ||
    currentFilters.yearMax !== undefined;

  // URL params > localStorage > default
  const storedSort = useMemo(() => getStoredSort(), []);
  const effectiveSortBy = (sortBy || storedSort.sortBy) as SortField;
  const effectiveSortOrder: SortOrder = (sortOrder as SortOrder) || storedSort.sortOrder;

  // Show recently added only in clean browse mode (no custom sort/filter/search)
  const isCleanBrowse = !sortBy && !sortOrder && !isSearchActive && !hasActiveFilters;

  const { data: libraryStats } = useLibraryStats();

  const listQuery = useLibraryList({
    page: currentPage,
    pageSize: currentPageSize,
    type: currentType,
    sortBy: effectiveSortBy,
    sortOrder: effectiveSortOrder,
    genres: genresParam || undefined,
    yearMin: yearMinParam,
    yearMax: yearMaxParam,
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
      genres: overrides.genres !== undefined ? overrides.genres : genresParam || undefined,
      yearMin: overrides.yearMin !== undefined ? overrides.yearMin : yearMinParam,
      yearMax: overrides.yearMax !== undefined ? overrides.yearMax : yearMaxParam,
    }),
    [
      currentPage,
      currentPageSize,
      currentType,
      sortBy,
      sortOrder,
      currentView,
      searchQuery,
      genresParam,
      yearMinParam,
      yearMaxParam,
    ]
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

  const handleTypeChange = useCallback(
    (newType: LibraryMediaType) => {
      navigate({ search: buildSearchParams({ page: 1, type: newType }) });
    },
    [navigate, buildSearchParams]
  );

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
    (field: SortField, order: SortOrder) => {
      setStoredSort({ sortBy: field, sortOrder: order });
      navigate({ search: buildSearchParams({ page: 1, sortBy: field, sortOrder: order }) });
    },
    [navigate, buildSearchParams]
  );

  const handleFilterApply = useCallback(
    (filters: FilterValues) => {
      navigate({
        search: buildSearchParams({
          page: 1,
          genres: filters.genres.length > 0 ? filters.genres.join(',') : undefined,
          yearMin: filters.yearMin,
          yearMax: filters.yearMax,
        }),
      });
    },
    [navigate, buildSearchParams]
  );

  const handleFilterClear = useCallback(() => {
    navigate({
      search: buildSearchParams({
        page: 1,
        genres: undefined,
        yearMin: undefined,
        yearMax: undefined,
      }),
    });
  }, [navigate, buildSearchParams]);

  const handleRemoveGenre = useCallback(
    (genre: string) => {
      const newGenres = currentFilters.genres.filter((g) => g !== genre);
      navigate({
        search: buildSearchParams({
          page: 1,
          genres: newGenres.length > 0 ? newGenres.join(',') : undefined,
        }),
      });
    },
    [currentFilters.genres, navigate, buildSearchParams]
  );

  const handleRemoveYearMin = useCallback(() => {
    navigate({ search: buildSearchParams({ page: 1, yearMin: undefined }) });
  }, [navigate, buildSearchParams]);

  const handleRemoveYearMax = useCallback(() => {
    navigate({ search: buildSearchParams({ page: 1, yearMax: undefined }) });
  }, [navigate, buildSearchParams]);

  const activeFilterCount =
    currentFilters.genres.length +
    (currentFilters.yearMin !== undefined ? 1 : 0) +
    (currentFilters.yearMax !== undefined ? 1 : 0);

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
        {/* Search bar */}
        {!isLibraryEmpty && (
          <div className="mb-6">
            <LibrarySearchBar
              onSearch={handleSearch}
              initialQuery={searchQuery}
              resultCount={isSearchActive ? totalItems : undefined}
            />
          </div>
        )}

        {/* Controls row: heading left, controls right */}
        {!isEmpty && (
          <div className="mb-6 flex items-center justify-between">
            <div className="flex items-baseline gap-3">
              <h2 className="text-xl font-semibold text-white">全部媒體</h2>
              {hasActiveFilters && !isSearchActive && (
                <span className="text-sm text-slate-400" data-testid="filter-count">
                  顯示 {totalItems} / {libraryStats?.totalCount ?? totalItems} 項
                </span>
              )}
            </div>
            <div className="flex items-center gap-2">
              {!isSearchActive && (
                <SortSelector
                  sortBy={effectiveSortBy}
                  sortOrder={effectiveSortOrder}
                  onSortChange={handleSortChange}
                />
              )}
              <button
                onClick={() => setIsFilterOpen(!isFilterOpen)}
                data-testid="filter-toggle"
                className={`flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
                  isFilterOpen || hasActiveFilters
                    ? 'bg-blue-600 text-white'
                    : 'bg-slate-800 text-slate-400 hover:bg-slate-700 hover:text-white'
                }`}
              >
                <Filter size={16} />
                篩選
                {activeFilterCount > 0 && (
                  <span className="rounded-full bg-white/20 px-1.5 text-xs">
                    {activeFilterCount}
                  </span>
                )}
              </button>
              <ViewToggle view={currentView} onViewChange={handleViewChange} />
            </div>
          </div>
        )}

        {/* Active filter chips */}
        {hasActiveFilters && !isEmpty && (
          <div className="mb-4">
            <FilterChips
              filters={currentFilters}
              onRemoveGenre={handleRemoveGenre}
              onRemoveYearMin={handleRemoveYearMin}
              onRemoveYearMax={handleRemoveYearMax}
              onClearAll={handleFilterClear}
            />
          </div>
        )}

        {isSearchEmpty ? (
          <EmptySearchResults query={searchQuery} onClear={() => handleSearch('')} />
        ) : isLibraryEmpty ? (
          <EmptyLibrary />
        ) : (
          <div className="flex gap-0">
            {/* Filter sidebar */}
            {isFilterOpen && (
              <aside
                data-testid="filter-sidebar"
                className="sticky top-16 mr-6 h-[calc(100vh-8rem)] w-[200px] flex-shrink-0 overflow-y-auto border-r border-slate-700 pr-4"
              >
                <FilterPanel
                  filters={currentFilters}
                  mediaType={currentType}
                  onApply={handleFilterApply}
                  onClear={handleFilterClear}
                  onTypeChange={handleTypeChange}
                />
              </aside>
            )}

            {/* Main content */}
            <div className="min-w-0 flex-1">
              {isCleanBrowse && <RecentlyAdded />}
              {currentView === 'grid' ? (
                <LibraryGrid
                  items={items}
                  isLoading={isLoading}
                  totalItems={totalItems}
                  highlightQuery={isSearchActive ? searchQuery : undefined}
                />
              ) : (
                <LibraryTable
                  items={items}
                  isLoading={isLoading}
                  sortBy={effectiveSortBy}
                  sortOrder={effectiveSortOrder}
                  onSort={handleColumnSort}
                  highlightQuery={isSearchActive ? searchQuery : undefined}
                />
              )}
              <Pagination
                currentPage={currentPage}
                totalPages={totalPages}
                onPageChange={handlePageChange}
                className="mt-8"
              />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
