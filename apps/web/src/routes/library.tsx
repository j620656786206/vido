import { useState, useMemo, useCallback, useEffect, useRef } from 'react';
import {
  createFileRoute,
  useNavigate,
  useMatchRoute,
  Outlet,
  redirect,
} from '@tanstack/react-router';
import { Filter, CheckSquare } from 'lucide-react';
import {
  useLibraryList,
  useLibrarySearch,
  useLibraryStats,
  useMovieStats,
  useSeriesStats,
  useBatchDelete,
  useBatchReparse,
  useBatchExport,
} from '../hooks/useLibrary';
import { LibraryGrid } from '../components/library/LibraryGrid';
import { LibraryTable } from '../components/library/LibraryTable';
import { SortSelector } from '../components/library/SortSelector';
import { LibrarySearchBar } from '../components/library/LibrarySearchBar';
import { FilterPanel } from '../components/library/FilterPanel';
import type { FilterValues } from '../components/library/FilterPanel';
import { FilterChips } from '../components/library/FilterChips';
import { EmptySearchResults } from '../components/library/EmptySearchResults';
import { RecentlyAdded } from '../components/library/RecentlyAdded';
import { EmptyNoQBT } from '../components/library/EmptyNoQBT';
import { EmptyNoFolder } from '../components/library/EmptyNoFolder';
import { EmptyReadyForScan } from '../components/library/EmptyReadyForScan';
import { useQBittorrentConfig } from '../hooks/useQBittorrent';
import { useMediaLibraries } from '../hooks/useMediaLibrary';
import { classifyEmptyState } from '../utils/emptyLibraryState';
import { ViewToggle } from '../components/library/ViewToggle';
import type { ViewMode } from '../components/library/ViewToggle';
import { SelectionToolbar } from '../components/library/SelectionToolbar';
import { BatchConfirmDialog } from '../components/library/BatchConfirmDialog';
import { BatchProgress } from '../components/library/BatchProgress';
import { GenerationBatchDialogV2 } from '../components/subtitle/GenerationBatchDialogV2';
import { Pagination } from '../components/ui/Pagination';
import type { LibraryMediaType, LibraryItem, SortField, SortOrder } from '../types/library';
import { VALID_SORT_FIELDS } from '../types/library';
import { useShellVersion } from '../components/shell/shellVersion';
import { LibraryBrowseV2 } from '../components/library/LibraryBrowseV2';

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
  unmatched?: boolean;
  /**
   * Forward-compatible deep-link target for the batch-subtitle "查看未找到項目"
   * link (Story 8-11 AC #6). Preserved here so the URL is valid; backend list
   * filtering by subtitle_status is a tracked follow-up (not yet wired).
   */
  subtitleStatus?: string;
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
    unmatched: search.unmatched === true ? true : undefined,
    subtitleStatus: typeof search.subtitleStatus === 'string' ? search.subtitleStatus : undefined,
  }),
  // UX2-2: migrated route — full-bleed under the v2 shell (LegacyContentContainer
  // opt-out). Content is gated by the shell version (NOT a second flag read, F4).
  staticData: { shell: 'v2' },
  // ux3-0-5: old ?type= deep links → clean type routes (D2). Route-level redirect
  // (never a component redirect, F1); 'all'/absent stays at /library (merged view).
  beforeLoad: ({ search }) => {
    if (search.type === 'movie' || search.type === 'tv') {
      throw redirect({
        to: search.type === 'movie' ? '/library/movies' : '/library/tv',
        search: { ...search, type: undefined },
      });
    }
  },
  component: LibraryRoute,
});

/**
 * Shell-version switch (UX2-2): under the v2 shell render the redesigned Browse;
 * under the legacy shell render the existing library page pixel-unchanged (P3
 * strangler discipline — flag OFF leaves the current library exactly as-is).
 */
function LibraryRoute() {
  const shell = useShellVersion();
  const matchRoute = useMatchRoute();
  // ux3-0-5 / F5: the Browse UI is mounted ONCE here in the layout, so movies↔tv
  // preserves filter + scroll state. The active type is derived from the matched
  // clean child; children are path markers (they render null via the Outlet).
  const type: LibraryMediaType = matchRoute({ to: '/library/movies' })
    ? 'movie'
    : matchRoute({ to: '/library/tv' })
      ? 'tv'
      : 'all';
  return (
    <>
      {shell === 'v2' ? <LibraryBrowseV2 type={type} /> : <LibraryPage />}
      <Outlet />
    </>
  );
}

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
    unmatched: unmatchedParam,
  } = Route.useSearch();
  const navigate = useNavigate({ from: Route.fullPath });

  const [currentView, setCurrentView] = useState<ViewMode>(
    () => (viewParam as ViewMode) || getStoredView()
  );
  const [searchQuery, setSearchQuery] = useState(q || '');
  const [isFilterOpen, setIsFilterOpen] = useState(false);

  // Selection mode state (Task 4)
  const [isSelectionMode, setIsSelectionMode] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [selectedType, setSelectedType] = useState<'movie' | 'series'>('movie');
  const [confirmAction, setConfirmAction] = useState<'delete' | 'reparse' | 'export' | null>(null);
  const [batchProgress, setBatchProgress] = useState<{
    isOpen: boolean;
    current: number;
    total: number;
    action: string;
    isComplete: boolean;
    errors?: { id: string; message: string }[];
  }>({ isOpen: false, current: 0, total: 0, action: '', isComplete: false });
  // Batch subtitle GENERATION dialog (ux3-subtitle-v2-batch AC 5 — re-pointed
  // from the Story 8-11 fetch dialog; the selection now actually flows in).
  const [isBatchSubtitleOpen, setIsBatchSubtitleOpen] = useState(false);
  const [generationBatchSelection, setGenerationBatchSelection] = useState<{
    movieIds: string[];
    excludedCount: number;
  }>({ movieIds: [], excludedCount: 0 });

  const lastSelectedIndexRef = useRef<number>(-1);
  // id → type map accumulated as items are selected — selections can span
  // pages, so the current `items` page alone cannot classify every id.
  const selectionTypesRef = useRef(new Map<string, 'movie' | 'series'>());

  const batchDeleteMutation = useBatchDelete();
  const batchReparseMutation = useBatchReparse();
  const batchExportMutation = useBatchExport();

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
      unmatched: unmatchedParam,
    }),
    [genresParam, yearMinParam, yearMaxParam, unmatchedParam]
  );

  const hasActiveFilters =
    currentFilters.genres.length > 0 ||
    currentFilters.yearMin !== undefined ||
    currentFilters.yearMax !== undefined ||
    unmatchedParam === true;

  // URL params > localStorage > default
  const storedSort = useMemo(() => getStoredSort(), []);
  const effectiveSortBy = (sortBy || storedSort.sortBy) as SortField;
  const effectiveSortOrder: SortOrder = (sortOrder as SortOrder) || storedSort.sortOrder;

  // Show recently added only in clean browse mode (no custom sort/filter/search)
  const isCleanBrowse = !sortBy && !sortOrder && !isSearchActive && !hasActiveFilters;

  const { data: libraryStats } = useLibraryStats();
  const { data: movieStats } = useMovieStats();
  const { data: seriesStats } = useSeriesStats();
  // bugfix-10-5: state-classifier inputs for the 3-state empty-library branch
  const qbtConfigQuery = useQBittorrentConfig();
  const mediaLibrariesQuery = useMediaLibraries();
  const totalUnmatchedCount =
    (movieStats?.unmatchedCount ?? 0) + (seriesStats?.unmatchedCount ?? 0);

  const listQuery = useLibraryList({
    page: currentPage,
    pageSize: currentPageSize,
    type: currentType,
    sortBy: effectiveSortBy,
    sortOrder: effectiveSortOrder,
    genres: genresParam || undefined,
    yearMin: yearMinParam,
    yearMax: yearMaxParam,
    unmatched: unmatchedParam || undefined,
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
      unmatched:
        overrides.unmatched !== undefined ? overrides.unmatched : unmatchedParam || undefined,
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
      unmatchedParam,
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
          unmatched: filters.unmatched || undefined,
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
        unmatched: undefined,
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

  const handleRemoveUnmatched = useCallback(() => {
    navigate({ search: buildSearchParams({ page: 1, unmatched: undefined }) });
  }, [navigate, buildSearchParams]);

  const activeFilterCount =
    currentFilters.genres.length +
    (currentFilters.yearMin !== undefined ? 1 : 0) +
    (currentFilters.yearMax !== undefined ? 1 : 0) +
    (unmatchedParam === true ? 1 : 0);

  // Selection handlers
  const enterSelectionMode = useCallback(() => {
    setIsSelectionMode(true);
    setSelectedIds(new Set());
    selectionTypesRef.current.clear();
  }, []);

  const exitSelectionMode = useCallback(() => {
    setIsSelectionMode(false);
    setSelectedIds(new Set());
    setSelectedType('movie');
    selectionTypesRef.current.clear();
  }, []);

  const executeBatchAction = useCallback(
    async (action: 'delete' | 'reparse' | 'export') => {
      const ids = Array.from(selectedIds);
      const total = ids.length;

      setBatchProgress({
        isOpen: true,
        current: 0,
        total,
        action:
          action === 'delete' ? '刪除中...' : action === 'reparse' ? '重新解析中...' : '匯出中...',
        isComplete: false,
      });

      try {
        if (action === 'delete') {
          const result = await batchDeleteMutation.mutateAsync({ ids, type: selectedType });
          setBatchProgress((prev) => ({
            ...prev,
            current: total,
            isComplete: true,
            errors: result.errors,
          }));
        } else if (action === 'reparse') {
          const result = await batchReparseMutation.mutateAsync({ ids, type: selectedType });
          setBatchProgress((prev) => ({
            ...prev,
            current: total,
            isComplete: true,
            errors: result.errors,
          }));
        } else {
          const result = await batchExportMutation.mutateAsync({ ids, type: selectedType });
          // Trigger download of the JSON
          const blob = new Blob([JSON.stringify(result, null, 2)], { type: 'application/json' });
          const url = URL.createObjectURL(blob);
          const a = document.createElement('a');
          a.href = url;
          a.download = `vido-export-${selectedType}-${new Date().toISOString().slice(0, 10)}.json`;
          a.click();
          URL.revokeObjectURL(url);
          setBatchProgress((prev) => ({ ...prev, current: total, isComplete: true }));
        }
      } catch {
        setBatchProgress((prev) => ({
          ...prev,
          current: total,
          isComplete: true,
          errors: [{ id: 'batch', message: '操作失敗' }],
        }));
      }
    },
    [selectedIds, selectedType, batchDeleteMutation, batchReparseMutation, batchExportMutation]
  );

  const handleBatchConfirm = useCallback(() => {
    if (confirmAction) {
      executeBatchAction(confirmAction);
      setConfirmAction(null);
    }
  }, [confirmAction, executeBatchAction]);

  const closeBatchProgress = useCallback(() => {
    setBatchProgress({ isOpen: false, current: 0, total: 0, action: '', isComplete: false });
    exitSelectionMode();
  }, [exitSelectionMode]);

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

  const getAllItemIds = useCallback(
    (): string[] =>
      items.map((item) => item.movie?.id || item.series?.id).filter((id): id is string => !!id),
    [items]
  );

  // Record the current page's id → type mapping so a cross-page selection can
  // still be classified when the generation-batch dialog opens (AC 5).
  const recordSelectionTypes = useCallback(() => {
    for (const item of items) {
      const id = item.movie?.id || item.series?.id;
      if (id) selectionTypesRef.current.set(id, item.movie ? 'movie' : 'series');
    }
  }, [items]);

  const handleSelect = useCallback(
    (id: string, e: React.MouseEvent) => {
      recordSelectionTypes();
      const allIds = getAllItemIds();
      const currentIndex = allIds.indexOf(id);

      if (e.shiftKey && lastSelectedIndexRef.current >= 0) {
        // Shift+Click: range select
        const start = Math.min(lastSelectedIndexRef.current, currentIndex);
        const end = Math.max(lastSelectedIndexRef.current, currentIndex);
        const rangeIds = allIds.slice(start, end + 1);
        setSelectedIds((prev) => {
          const next = new Set(prev);
          for (const rangeId of rangeIds) {
            next.add(rangeId);
          }
          return next;
        });
      } else {
        // Single click or Ctrl/Cmd+Click: toggle individual item
        setSelectedIds((prev) => {
          const next = new Set(prev);
          if (next.has(id)) {
            next.delete(id);
          } else {
            next.add(id);
          }
          return next;
        });
        lastSelectedIndexRef.current = currentIndex;
      }

      // Determine item type from the clicked item
      const found = items.find((item) => (item.movie?.id || item.series?.id) === id);
      if (found) {
        setSelectedType(found.type === 'movie' ? 'movie' : 'series');
      }
    },
    [items, getAllItemIds, recordSelectionTypes]
  );

  const handleSelectAll = useCallback(() => {
    recordSelectionTypes();
    const allIds = items
      .map((item) => item.movie?.id || item.series?.id)
      .filter((id): id is string => !!id);
    setSelectedIds(new Set(allIds));
  }, [items, recordSelectionTypes]);

  // Open the generation-batch dialog with the selection ACTUALLY flowing in
  // (AC 5): movie ids are UUID STRINGS ([@contract-v2], 9R-18) and pass through
  // unconverted; series (or unclassifiable) ids are excluded client-side — the
  // backend REJECTS the whole request with 400 if ANY id is not a movie with a
  // file (9R-16 AC 8); the dialog shows the note.
  const handleOpenGenerationBatch = useCallback(() => {
    const movieIds: string[] = [];
    let excludedCount = 0;
    for (const id of selectedIds) {
      if (selectionTypesRef.current.get(id) === 'movie') {
        movieIds.push(id);
      } else {
        excludedCount++;
      }
    }
    setGenerationBatchSelection({ movieIds, excludedCount });
    setIsBatchSubtitleOpen(true);
  }, [selectedIds]);

  // Keyboard shortcuts for selection mode (Escape, Ctrl+A)
  useEffect(() => {
    if (!isSelectionMode) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        exitSelectionMode();
      } else if ((e.ctrlKey || e.metaKey) && e.key === 'a') {
        e.preventDefault();
        handleSelectAll();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isSelectionMode, exitSelectionMode, handleSelectAll]);

  return (
    <div>
      <div className="mx-auto max-w-7xl w-full px-4 py-8">
        {/* Selection toolbar */}
        {isSelectionMode && (
          <div className="mb-4">
            <SelectionToolbar
              selectedCount={selectedIds.size}
              onDelete={() => setConfirmAction('delete')}
              onReparse={() => setConfirmAction('reparse')}
              onExport={() => setConfirmAction('export')}
              onBatchSubtitle={handleOpenGenerationBatch}
              onCancel={exitSelectionMode}
              isProcessing={
                batchDeleteMutation.isPending ||
                batchReparseMutation.isPending ||
                batchExportMutation.isPending
              }
            />
            {items.length > 0 && (
              <div className="mt-2 flex items-center gap-2">
                <button
                  onClick={handleSelectAll}
                  data-testid="select-all-btn"
                  className="text-sm text-[var(--accent-primary)] hover:text-blue-300"
                >
                  全選 ({items.length})
                </button>
                {selectedIds.size > 0 && (
                  <button
                    onClick={() => setSelectedIds(new Set())}
                    data-testid="deselect-all-btn"
                    className="text-sm text-[var(--text-secondary)] hover:text-[var(--text-secondary)]"
                  >
                    取消全選
                  </button>
                )}
              </div>
            )}
          </div>
        )}

        {/* Search bar */}
        {!isLibraryEmpty && !isSelectionMode && (
          <div className="mb-6">
            <LibrarySearchBar
              onSearch={handleSearch}
              initialQuery={searchQuery}
              resultCount={isSearchActive ? totalItems : undefined}
            />
          </div>
        )}

        {/* Controls row: heading left, controls right */}
        {!isEmpty && !isSelectionMode && (
          <div className="mb-6 flex items-center justify-between">
            <div className="flex items-baseline gap-3">
              <h2 className="text-xl font-semibold text-white">全部媒體</h2>
              {hasActiveFilters && !isSearchActive && (
                <span className="text-sm text-[var(--text-secondary)]" data-testid="filter-count">
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
                onClick={enterSelectionMode}
                data-testid="enter-selection-btn"
                className="flex items-center gap-2 rounded-lg bg-[var(--bg-secondary)] px-3 py-2 text-sm font-medium text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-white"
              >
                <CheckSquare size={16} />
                選取
              </button>
              <button
                onClick={() => setIsFilterOpen(!isFilterOpen)}
                data-testid="filter-toggle"
                className={`flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
                  isFilterOpen || hasActiveFilters
                    ? 'bg-[var(--accent-primary)] text-white'
                    : 'bg-[var(--bg-secondary)] text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] hover:text-white'
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
              onRemoveUnmatched={handleRemoveUnmatched}
              onClearAll={handleFilterClear}
            />
          </div>
        )}

        {isSearchEmpty ? (
          <EmptySearchResults query={searchQuery} onClear={() => handleSearch('')} />
        ) : isLibraryEmpty ? (
          (() => {
            // bugfix-10-5 AC #1 [@contract-v1]: 3-state classifier branch
            const emptyState = classifyEmptyState({
              qbtConfigured: qbtConfigQuery.data?.configured,
              mediaLibrariesCount: mediaLibrariesQuery.data?.libraries?.length ?? 0,
              itemsCount: items.length,
              isLoading: qbtConfigQuery.isLoading || mediaLibrariesQuery.isLoading,
            });
            if (emptyState === 'loading') return null;
            if (emptyState === 'no-qbt') return <EmptyNoQBT />;
            if (emptyState === 'no-folder') return <EmptyNoFolder />;
            return <EmptyReadyForScan />;
          })()
        ) : (
          <div className="flex gap-0">
            {/* Filter sidebar */}
            {isFilterOpen && (
              <aside
                data-testid="filter-sidebar"
                className="sticky top-16 mr-6 h-[calc(100vh-8rem)] w-[200px] flex-shrink-0 overflow-y-auto border-r border-[var(--border-subtle)] pr-4"
              >
                <FilterPanel
                  filters={currentFilters}
                  mediaType={currentType}
                  unmatchedCount={totalUnmatchedCount}
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
                  selectionMode={isSelectionMode}
                  selectedIds={selectedIds}
                  onSelect={handleSelect}
                />
              ) : (
                <LibraryTable
                  items={items}
                  isLoading={isLoading}
                  sortBy={effectiveSortBy}
                  sortOrder={effectiveSortOrder}
                  onSort={handleColumnSort}
                  highlightQuery={isSearchActive ? searchQuery : undefined}
                  selectionMode={isSelectionMode}
                  selectedIds={selectedIds}
                  onSelect={handleSelect}
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

      {/* Batch confirmation dialog */}
      <BatchConfirmDialog
        isOpen={confirmAction !== null}
        itemCount={selectedIds.size}
        action={confirmAction || 'delete'}
        onConfirm={handleBatchConfirm}
        onCancel={() => setConfirmAction(null)}
      />

      {/* Batch progress */}
      <BatchProgress
        isOpen={batchProgress.isOpen}
        current={batchProgress.current}
        total={batchProgress.total}
        action={batchProgress.action}
        errors={batchProgress.errors}
        isComplete={batchProgress.isComplete}
        onClose={closeBatchProgress}
      />

      {/* Batch subtitle GENERATION (ux3-subtitle-v2-batch AC 5 — supersedes the
          Story 8-11 fetch dialog here; selection flows in, series pre-excluded) */}
      <GenerationBatchDialogV2
        open={isBatchSubtitleOpen}
        onOpenChange={setIsBatchSubtitleOpen}
        selectedMovieIds={generationBatchSelection.movieIds}
        excludedSeriesCount={generationBatchSelection.excludedCount}
      />
    </div>
  );
}
