import { useState, useCallback } from 'react';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useLibraryList } from '../hooks/useLibrary';
import { LibraryGrid } from '../components/library/LibraryGrid';
import { LibraryTable } from '../components/library/LibraryTable';
import type { SortField } from '../components/library/LibraryTable';
import { RecentlyAdded } from '../components/library/RecentlyAdded';
import { EmptyLibrary } from '../components/library/EmptyLibrary';
import { ViewToggle } from '../components/library/ViewToggle';
import type { ViewMode } from '../components/library/ViewToggle';
import { getStoredPreferences } from '../components/library/SettingsGearDropdown';
import { Pagination } from '../components/ui/Pagination';
import type { LibraryMediaType } from '../types/library';

const VIEW_STORAGE_KEY = 'vido:library:view';

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

interface LibrarySearchParams {
  page?: number;
  pageSize?: number;
  type?: LibraryMediaType;
  sortBy?: string;
  sortOrder?: string;
  view?: string;
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
  }),
  component: LibraryPage,
});

function LibraryPage() {
  const { page, pageSize, type, sortBy, sortOrder, view: viewParam } = Route.useSearch();
  const navigate = useNavigate({ from: Route.fullPath });

  const [preferences] = useState(() => getStoredPreferences());
  const [currentView, setCurrentView] = useState<ViewMode>(
    () => (viewParam as ViewMode) || getStoredView()
  );

  const currentPage = page || 1;
  const currentPageSize = pageSize || 20;
  const currentType = type || 'all';

  // Search params override preferences for sort (e.g., from "查看全部" link)
  const effectiveSortBy = sortBy || preferences.defaultSort;
  const effectiveSortOrder = (sortOrder as 'asc' | 'desc') || undefined;

  // Show recently added only in clean browse mode (no custom sort/filter)
  const isCleanBrowse = !sortBy && !sortOrder;

  const { data, isLoading } = useLibraryList({
    page: currentPage,
    pageSize: currentPageSize,
    type: currentType,
    sortBy: effectiveSortBy,
    sortOrder: effectiveSortOrder,
  });

  const handlePageChange = (newPage: number) => {
    navigate({
      search: {
        page: newPage,
        pageSize: currentPageSize,
        type: currentType,
        sortBy: sortBy || undefined,
        sortOrder: sortOrder || undefined,
        view: currentView !== 'grid' ? currentView : undefined,
      },
    });
  };

  const handleTypeChange = (newType: LibraryMediaType) => {
    navigate({
      search: {
        page: 1,
        pageSize: currentPageSize,
        type: newType,
        sortBy: sortBy || undefined,
        sortOrder: sortOrder || undefined,
        view: currentView !== 'grid' ? currentView : undefined,
      },
    });
  };

  const handleViewChange = useCallback(
    (newView: ViewMode) => {
      setCurrentView(newView);
      setStoredView(newView);
      navigate({
        search: {
          page: currentPage,
          pageSize: currentPageSize,
          type: currentType,
          sortBy: sortBy || undefined,
          sortOrder: sortOrder || undefined,
          view: newView !== 'grid' ? newView : undefined,
        },
      });
    },
    [currentPage, currentPageSize, currentType, sortBy, sortOrder, navigate]
  );

  const handleColumnSort = useCallback(
    (field: SortField) => {
      const newOrder = sortBy === field && sortOrder === 'asc' ? 'desc' : 'asc';
      navigate({
        search: {
          page: 1,
          pageSize: currentPageSize,
          type: currentType,
          sortBy: field,
          sortOrder: newOrder,
          view: currentView !== 'grid' ? currentView : undefined,
        },
      });
    },
    [sortBy, sortOrder, currentPageSize, currentType, currentView, navigate]
  );

  const totalItems = data?.totalItems ?? 0;
  const totalPages = data?.totalPages ?? 0;
  const items = data?.items ?? [];
  const isEmpty = !isLoading && items.length === 0;

  return (
    <div>
      <div className="container mx-auto px-4 py-8">
        {!isEmpty && (
          <div className="mb-6 flex items-center justify-between">
            <span className="text-sm text-slate-400">
              顯示 {(currentPage - 1) * currentPageSize + 1}-
              {Math.min(currentPage * currentPageSize, totalItems)} / {totalItems} 項
            </span>
            <ViewToggle view={currentView} onViewChange={handleViewChange} />
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

        {isEmpty ? (
          <EmptyLibrary />
        ) : (
          <>
            {isCleanBrowse && <RecentlyAdded />}
            {currentView === 'grid' ? (
              <LibraryGrid
                items={items}
                isLoading={isLoading}
                totalItems={totalItems}
                density={preferences.density}
              />
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
