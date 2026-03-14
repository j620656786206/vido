import { useState } from 'react';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useLibraryList } from '../hooks/useLibrary';
import { LibraryGrid } from '../components/library/LibraryGrid';
import { RecentlyAdded } from '../components/library/RecentlyAdded';
import { EmptyLibrary } from '../components/library/EmptyLibrary';
import {
  SettingsGearDropdown,
  getStoredPreferences,
} from '../components/library/SettingsGearDropdown';
import { Pagination } from '../components/ui/Pagination';
import type { LibraryMediaType } from '../types/library';
import type { PosterDensity, TitleLanguage } from '../components/library/SettingsGearDropdown';

interface LibrarySearchParams {
  page?: number;
  pageSize?: number;
  type?: LibraryMediaType;
  sortBy?: string;
  sortOrder?: string;
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
  }),
  component: LibraryPage,
});

function LibraryPage() {
  const { page, pageSize, type, sortBy, sortOrder } = Route.useSearch();
  const navigate = useNavigate({ from: Route.fullPath });

  const [preferences, setPreferences] = useState(() => getStoredPreferences());

  const currentPage = page || 1;
  const currentPageSize = pageSize || 20;
  const currentType = type || 'all';

  // Search params override preferences for sort (e.g., from "查看全部" link)
  const effectiveSortBy = sortBy || preferences.defaultSort;
  const effectiveSortOrder = sortOrder as 'asc' | 'desc' | undefined;

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
    navigate({ search: { page: newPage, pageSize: currentPageSize, type: currentType } });
  };

  const handleTypeChange = (newType: LibraryMediaType) => {
    navigate({ search: { page: 1, pageSize: currentPageSize, type: newType } });
  };

  const totalItems = data?.totalItems ?? 0;
  const totalPages = data?.totalPages ?? 0;
  const items = data?.items ?? [];
  const isEmpty = !isLoading && items.length === 0;

  return (
    <div>
      <div className="container mx-auto px-4 py-8">
        <div className="mb-6 flex items-center justify-between">
          <div className="flex items-center gap-3">
            {!isEmpty && (
              <span className="text-sm text-slate-400">
                顯示 {(currentPage - 1) * currentPageSize + 1}-
                {Math.min(currentPage * currentPageSize, totalItems)} / {totalItems} 項
              </span>
            )}
          </div>
          <SettingsGearDropdown preferences={preferences} onPreferencesChange={setPreferences} />
        </div>

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
            <LibraryGrid
              items={items}
              isLoading={isLoading}
              totalItems={totalItems}
              density={preferences.density}
            />
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
