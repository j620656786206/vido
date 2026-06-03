// Design ref: ux-design.pen Screen AS-1 Advanced Filter Chips Desktop (rsAxf)
import { useState } from 'react';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { SlidersHorizontal } from 'lucide-react';
import { MediaTypeTabs, type MediaTypeFilter } from '../components/search/MediaTypeTabs';
import { FilterChipBar } from '../components/search/FilterChipBar';
import { FilterPanel } from '../components/search/FilterPanel';
import { FilterBottomSheet } from '../components/search/FilterBottomSheet';
import { SearchResults } from '../components/search/SearchResults';
import { useFilterState } from '../hooks/useFilterState';
import { useDiscoverResults } from '../hooks/useDiscoverResults';
import type { DiscoverFilters, SortKey } from '../lib/discoverFilters';

interface DiscoverSearchParams {
  genre?: string;
  year_gte?: number;
  year_lte?: number;
  region?: string;
  rating_gte?: number;
  platform?: string;
  sort_by?: SortKey;
  type?: MediaTypeFilter;
  page?: number;
}

const SORT_KEYS: SortKey[] = ['popularity', 'date', 'rating'];

function toOptionalNumber(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined;
}

export const Route = createFileRoute('/discover')({
  validateSearch: (search: Record<string, unknown>): DiscoverSearchParams => ({
    genre: typeof search.genre === 'string' ? search.genre : undefined,
    year_gte: toOptionalNumber(search.year_gte),
    year_lte: toOptionalNumber(search.year_lte),
    region: typeof search.region === 'string' ? search.region : undefined,
    rating_gte: toOptionalNumber(search.rating_gte),
    platform: typeof search.platform === 'string' ? search.platform : undefined,
    sort_by: SORT_KEYS.includes(search.sort_by as SortKey)
      ? (search.sort_by as SortKey)
      : undefined,
    type: ['all', 'movie', 'tv'].includes(search.type as string)
      ? (search.type as MediaTypeFilter)
      : undefined,
    page: toOptionalNumber(search.page),
  }),
  component: DiscoverPage,
});

function DiscoverPage() {
  const { type, page } = Route.useSearch();
  const navigate = useNavigate({ from: Route.fullPath });
  const { filters, setFilters, clearAll } = useFilterState();

  const [sheetOpen, setSheetOpen] = useState(false);

  const currentType: MediaTypeFilter = type ?? 'all';
  const currentPage = page ?? 1;

  const { moviesQuery, tvQuery, isLoading, totalResults } = useDiscoverResults(
    filters,
    currentType,
    currentPage
  );

  const handleTypeChange = (newType: MediaTypeFilter) => {
    navigate({ search: (prev) => ({ ...prev, type: newType, page: 1 }) });
  };

  const handlePageChange = (newPage: number) => {
    navigate({ search: (prev) => ({ ...prev, page: newPage }) });
  };

  const handleFilterChange = (next: DiscoverFilters) => setFilters(next);

  return (
    <div className="container mx-auto px-4 py-8">
      <h1 className="mb-4 text-2xl font-bold text-white">探索</h1>

      {/* Tabs + mobile filter trigger */}
      <div className="mb-6 flex items-center justify-between gap-3">
        <MediaTypeTabs
          activeType={currentType}
          onTypeChange={handleTypeChange}
          movieCount={moviesQuery.data?.totalResults}
          tvCount={tvQuery.data?.totalResults}
        />
        <button
          onClick={() => setSheetOpen(true)}
          data-testid="open-filter-sheet"
          aria-label="開啟篩選"
          className="inline-flex items-center gap-1.5 rounded-full border border-[var(--border-subtle)] bg-[var(--bg-tertiary)] px-3 py-1.5 text-sm text-[var(--text-secondary)] hover:text-white lg:hidden"
        >
          <SlidersHorizontal className="h-4 w-4" />
          篩選
        </button>
      </div>

      <div className="flex gap-6">
        {/* Desktop sidebar — instant-apply filters (AC #5) */}
        <aside className="hidden w-64 shrink-0 lg:block" data-testid="filter-sidebar">
          <h2 className="mb-4 text-sm font-semibold text-white">進階篩選</h2>
          <FilterPanel filters={filters} onChange={handleFilterChange} />
        </aside>

        {/* Results */}
        <div className="min-w-0 flex-1">
          {/* Persistent chip bar (AC #1-3) */}
          <FilterChipBar
            filters={filters}
            onChange={handleFilterChange}
            onClearAll={clearAll}
            className="mb-4"
          />

          <SearchResults
            movies={moviesQuery.data}
            tvShows={tvQuery.data}
            isLoading={isLoading}
            type={currentType}
            currentPage={currentPage}
            onPageChange={handlePageChange}
          />
        </div>
      </div>

      {/* Mobile bottom sheet (AC #6) */}
      <FilterBottomSheet
        isOpen={sheetOpen}
        onClose={() => setSheetOpen(false)}
        filters={filters}
        onApply={handleFilterChange}
        resultCount={totalResults}
      />
    </div>
  );
}
