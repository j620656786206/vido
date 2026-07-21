// Time-bomb-exempt: the only wall-clock read stamps the batch-export download
// FILENAME (vido-export-<type>-<date>.json) — user-facing file naming, never
// rendered UI state, so no fixture can flake on it (Murat: test-architecture call,
// same read the legacy LibraryPage ships).
// Implements: Component/Browse-Grid-v2 (LcHBs) + Component/Browse-List-v2 (b1H71g)
/**
 * v2 Browse experience (UX Redesign Phase 2 — UX2-2). Rendered by the /library
 * route ONLY under the v2 shell (the route branches on `useShellVersion()`; the
 * flag stays read-once in __root, F4). One component serves all three type views
 * (all / movie / tv) off the `?type=` param — a shared grid + shared filter/sort/
 * scroll state that survives movies↔tv switches (AC #1 intent + F5). Integrated
 * toolbar, four states (empty/loading/no-result/error), grid (PosterCardV2) and
 * list (LibraryListRowV2), continuous scroll (infinite query, AC #11), and a
 * merged mobile sort+filter sheet. Reads URL state via the /library route api.
 */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { getRouteApi } from '@tanstack/react-router';
import { CheckSquare, SlidersHorizontal } from 'lucide-react';
import { useLibraryInfinite } from '../../hooks/useLibraryInfinite';
import { useQBittorrentConfig } from '../../hooks/useQBittorrent';
import { useMediaLibraries } from '../../hooks/useMediaLibrary';
import {
  useMovieStats,
  useSeriesStats,
  useBatchDelete,
  useBatchReparse,
  useBatchExport,
} from '../../hooks/useLibrary';
import { SelectionToolbar } from './SelectionToolbar';
import { BatchConfirmDialog } from './BatchConfirmDialog';
import { BatchProgress } from './BatchProgress';
import { GenerationBatchDialogV2 } from '../subtitle/GenerationBatchDialogV2';
import { classifyEmptyState } from '../../utils/emptyLibraryState';
import { SortSelector } from './SortSelector';
import { FilterChips } from './FilterChips';
import { ViewToggle, type ViewMode } from './ViewToggle';
import { PosterCardV2 } from './PosterCardV2';
import { LibraryListRowV2 } from './LibraryListRowV2';
import { LibraryFilterSheetV2 } from './LibraryFilterSheetV2';
import { LibraryFilterRail } from './LibraryFilterRail';
import { LibraryGridSkeletonV2, LibraryNoResultV2, LibraryErrorV2 } from './LibraryStatesV2';
import { EmptyNoQBT } from './EmptyNoQBT';
import { EmptyNoFolder } from './EmptyNoFolder';
import { EmptyReadyForScan } from './EmptyReadyForScan';
import type { FilterValues } from './FilterPanel';
import type {
  LibraryItem,
  LibraryMediaType,
  SortField,
  SortOrder,
  LibraryMovie,
  LibrarySeries,
} from '../../types/library';
import { VALID_SORT_FIELDS } from '../../types/library';

const routeApi = getRouteApi('/library');
const VIEW_STORAGE_KEY = 'vido:library:view';
const SORT_STORAGE_KEY = 'vido:library:sort';
const RAIL_STORAGE_KEY = 'vido:library:rail-collapsed';
const DEFAULT_SORT = { sortBy: 'created_at' as SortField, sortOrder: 'desc' as SortOrder };

function getStoredView(): ViewMode {
  try {
    const v = localStorage.getItem(VIEW_STORAGE_KEY);
    if (v === 'grid' || v === 'list') return v;
  } catch {
    /* ignore */
  }
  return 'grid';
}
function getStoredRailCollapsed(): boolean {
  try {
    return localStorage.getItem(RAIL_STORAGE_KEY) === '1';
  } catch {
    return false;
  }
}
function getStoredSort(): { sortBy: SortField; sortOrder: SortOrder } {
  try {
    const parsed = JSON.parse(localStorage.getItem(SORT_STORAGE_KEY) || '');
    if (VALID_SORT_FIELDS.includes(parsed.sortBy) && ['asc', 'desc'].includes(parsed.sortOrder)) {
      return parsed;
    }
  } catch {
    /* ignore */
  }
  return DEFAULT_SORT;
}

interface DisplayFields {
  id: string;
  type: 'movie' | 'tv';
  title: string;
  posterPath?: string | null;
  year?: string;
  meta: string;
  listMeta: string;
  voteAverage?: number;
  media: LibraryMovie | LibrarySeries;
}

function toDisplay(item: LibraryItem): DisplayFields | null {
  const media = item.movie ?? item.series;
  if (!media) return null;
  const isMovie = item.type === 'movie';
  const date = isMovie ? item.movie?.releaseDate : item.series?.firstAirDate;
  const year = date ? date.slice(0, 4) : undefined;
  const meta = isMovie
    ? item.movie?.runtime
      ? `${item.movie.runtime} 分`
      : ''
    : item.series?.numberOfSeasons
      ? `${item.series.numberOfSeasons} 季`
      : '';
  const genre = media.genres?.[0];
  const listMeta = [year, meta, genre].filter(Boolean).join(' · ');
  return {
    id: media.id,
    type: isMovie ? 'movie' : 'tv',
    title: media.title,
    posterPath: media.posterPath,
    year,
    meta,
    listMeta,
    voteAverage: media.voteAverage,
    media,
  };
}

export function LibraryBrowseV2({ type: typeProp }: { type?: LibraryMediaType } = {}) {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();

  // ux3-0-5: type comes from the clean route (/library/{movies,tv}) via the layout;
  // falls back to the legacy ?type= for any non-migrated caller.
  const currentType = typeProp ?? ((search.type as LibraryMediaType) || 'all');
  const stored = useMemo(() => getStoredSort(), []);
  const effectiveSortBy = (search.sortBy as SortField) || stored.sortBy;
  const effectiveSortOrder = (search.sortOrder as SortOrder) || stored.sortOrder;
  const [view, setView] = useState<ViewMode>(() => (search.view as ViewMode) || getStoredView());
  const [filterSheetOpen, setFilterSheetOpen] = useState(false);
  // ux3-0-7: desktop rail collapse state (persisted); only meaningful at lg+.
  const [railCollapsed, setRailCollapsedState] = useState<boolean>(() => getStoredRailCollapsed());
  const setRailCollapsed = useCallback((next: boolean) => {
    setRailCollapsedState(next);
    try {
      localStorage.setItem(RAIL_STORAGE_KEY, next ? '1' : '0');
    } catch {
      /* ignore */
    }
  }, []);

  const filters: FilterValues = useMemo(
    () => ({
      genres: search.genres ? search.genres.split(',').filter(Boolean) : [],
      yearMin: search.yearMin,
      yearMax: search.yearMax,
      unmatched: search.unmatched,
    }),
    [search.genres, search.yearMin, search.yearMax, search.unmatched]
  );
  const hasActiveFilters =
    filters.genres.length > 0 ||
    filters.yearMin !== undefined ||
    filters.yearMax !== undefined ||
    filters.unmatched === true;
  // Constraining-facet count for the rail badge / collapsed 篩選(n) button.
  // Decade range (yearMin/yearMax) counts as ONE facet; type=全部 is not a constraint.
  const activeFilterCount =
    filters.genres.length +
    (filters.yearMin !== undefined || filters.yearMax !== undefined ? 1 : 0) +
    (filters.unmatched === true ? 1 : 0);

  const {
    items,
    totalItems,
    isLoading,
    isError,
    error,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    refetch,
  } = useLibraryInfinite({
    type: currentType,
    sortBy: effectiveSortBy,
    sortOrder: effectiveSortOrder,
    genres: search.genres || undefined,
    yearMin: search.yearMin,
    yearMax: search.yearMax,
    unmatched: search.unmatched || undefined,
  });

  // Empty-state classifier inputs (reuse the bugfix-10-5 3-state classifier).
  const qbtConfig = useQBittorrentConfig();
  const mediaLibraries = useMediaLibraries();
  const movieStats = useMovieStats();
  const seriesStats = useSeriesStats();
  const unmatchedCount =
    (movieStats.data?.unmatchedCount ?? 0) + (seriesStats.data?.unmatchedCount ?? 0);

  const patchSearch = useCallback(
    (patch: Record<string, unknown>) => {
      navigate({ search: (prev) => ({ ...prev, ...patch }) });
    },
    [navigate]
  );

  const handleSortChange = useCallback(
    (field: SortField, order: SortOrder) => {
      try {
        localStorage.setItem(SORT_STORAGE_KEY, JSON.stringify({ sortBy: field, sortOrder: order }));
      } catch {
        /* ignore */
      }
      patchSearch({ sortBy: field, sortOrder: order });
    },
    [patchSearch]
  );

  const handleViewChange = useCallback(
    (next: ViewMode) => {
      setView(next);
      try {
        localStorage.setItem(VIEW_STORAGE_KEY, next);
      } catch {
        /* ignore */
      }
      patchSearch({ view: next !== 'grid' ? next : undefined });
    },
    [patchSearch]
  );

  const applyFilters = useCallback(
    (f: FilterValues) =>
      patchSearch({
        genres: f.genres.length ? f.genres.join(',') : undefined,
        yearMin: f.yearMin,
        yearMax: f.yearMax,
        unmatched: f.unmatched || undefined,
      }),
    [patchSearch]
  );
  const clearFilters = useCallback(
    () =>
      patchSearch({
        genres: undefined,
        yearMin: undefined,
        yearMax: undefined,
        unmatched: undefined,
      }),
    [patchSearch]
  );
  const handleTypeChange = useCallback(
    (t: LibraryMediaType) =>
      navigate({
        to: t === 'movie' ? '/library/movies' : t === 'tv' ? '/library/tv' : '/library',
        search: (prev) => ({ ...prev, type: undefined }),
      }),
    [navigate]
  );

  // ── ux3-cutover-2: selection mode + batch ops (v2 port of the legacy
  // LibraryPage chain — the deletion gate for the legacy shell). Selection spans
  // the loaded (infinite-scroll) items; the id→type map accumulates as items
  // load so a long selection can still be classified for the batch calls.
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
  const [isBatchSubtitleOpen, setIsBatchSubtitleOpen] = useState(false);
  const [generationBatchSelection, setGenerationBatchSelection] = useState<{
    movieIds: string[];
    excludedCount: number;
  }>({ movieIds: [], excludedCount: 0 });
  const lastSelectedIndexRef = useRef<number>(-1);
  const selectionTypesRef = useRef(new Map<string, 'movie' | 'series'>());

  const batchDeleteMutation = useBatchDelete();
  const batchReparseMutation = useBatchReparse();
  const batchExportMutation = useBatchExport();

  // Continuous scroll — observe a sentinel near the end of the grid.
  const sentinelRef = useRef<HTMLDivElement | null>(null);
  useEffect(() => {
    const el = sentinelRef.current;
    if (!el || !hasNextPage) return;
    const io = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting && !isFetchingNextPage) fetchNextPage();
      },
      { rootMargin: '600px' }
    );
    io.observe(el);
    return () => io.disconnect();
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  const isEmpty = !isLoading && items.length === 0;
  const display = items.map(toDisplay).filter(Boolean) as DisplayFields[];

  // Selection handlers (over the loaded display list).
  const recordSelectionTypes = useCallback(() => {
    for (const d of display) {
      selectionTypesRef.current.set(d.id, d.type === 'movie' ? 'movie' : 'series');
    }
  }, [display]);

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

  const handleSelect = useCallback(
    (id: string, e: React.MouseEvent) => {
      recordSelectionTypes();
      const allIds = display.map((d) => d.id);
      const currentIndex = allIds.indexOf(id);

      if (e.shiftKey && lastSelectedIndexRef.current >= 0) {
        const start = Math.min(lastSelectedIndexRef.current, currentIndex);
        const end = Math.max(lastSelectedIndexRef.current, currentIndex);
        const rangeIds = allIds.slice(start, end + 1);
        setSelectedIds((prev) => {
          const next = new Set(prev);
          for (const rangeId of rangeIds) next.add(rangeId);
          return next;
        });
      } else {
        setSelectedIds((prev) => {
          const next = new Set(prev);
          if (next.has(id)) next.delete(id);
          else next.add(id);
          return next;
        });
        lastSelectedIndexRef.current = currentIndex;
      }

      const found = display.find((d) => d.id === id);
      if (found) setSelectedType(found.type === 'movie' ? 'movie' : 'series');
    },
    [display, recordSelectionTypes]
  );

  const handleSelectAll = useCallback(() => {
    recordSelectionTypes();
    setSelectedIds(new Set(display.map((d) => d.id)));
  }, [display, recordSelectionTypes]);

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

  // Generation-batch dialog: movie UUIDs pass through, series ids are excluded
  // client-side (the backend 400s the whole request otherwise — 9R-16 AC 8).
  const handleOpenGenerationBatch = useCallback(() => {
    const movieIds: string[] = [];
    let excludedCount = 0;
    for (const id of selectedIds) {
      if (selectionTypesRef.current.get(id) === 'movie') movieIds.push(id);
      else excludedCount++;
    }
    setGenerationBatchSelection({ movieIds, excludedCount });
    setIsBatchSubtitleOpen(true);
  }, [selectedIds]);

  // Keyboard shortcuts while selecting (Escape exits, Ctrl/Cmd+A selects loaded).
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
  // ux3-0-7: grid reflows when the desktop rail takes the left column (lg+).
  const gridColsClass = railCollapsed
    ? 'grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6'
    : 'grid-cols-2 sm:grid-cols-3 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5';

  return (
    <div className="px-4 py-6 sm:px-6">
      <div className="lg:flex lg:gap-6">
        {/* Desktop filter rail (lg+ only); hidden when collapsed. <lg uses the sheet. */}
        {!railCollapsed && (
          <div className="hidden lg:block">
            <LibraryFilterRail
              filters={filters}
              mediaType={currentType}
              unmatchedCount={unmatchedCount}
              activeCount={activeFilterCount}
              onApply={applyFilters}
              onClear={clearFilters}
              onTypeChange={handleTypeChange}
              onCollapse={() => setRailCollapsed(true)}
            />
          </div>
        )}

        <div className="min-w-0 lg:flex-1">
          {/* ux3-cutover-2: selection toolbar replaces the browse toolbar while selecting */}
          {isSelectionMode ? (
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
              {display.length > 0 && (
                <div className="mt-2 flex items-center gap-2">
                  <button
                    onClick={handleSelectAll}
                    data-testid="select-all-btn"
                    className="text-sm text-[var(--accent-text)] hover:underline"
                  >
                    全選 ({display.length})
                  </button>
                  {selectedIds.size > 0 && (
                    <button
                      onClick={() => setSelectedIds(new Set())}
                      data-testid="deselect-all-btn"
                      className="text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
                    >
                      取消全選
                    </button>
                  )}
                </div>
              )}
            </div>
          ) : (
            <div className="mb-4 flex flex-wrap items-center gap-2">
              <SortSelector
                sortBy={effectiveSortBy}
                sortOrder={effectiveSortOrder}
                onSortChange={handleSortChange}
              />
              {/* Mobile (<lg): opens the bottom sheet */}
              <button
                type="button"
                onClick={() => setFilterSheetOpen(true)}
                data-testid="library-filter-open"
                className={`flex min-h-[44px] items-center gap-2 rounded-[var(--radius-md)] px-3 text-sm font-medium transition-colors lg:hidden ${
                  hasActiveFilters
                    ? 'bg-[var(--accent-subtle)] text-[var(--accent-text)]'
                    : 'bg-[var(--bg-secondary)] text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]'
                }`}
              >
                <SlidersHorizontal className="h-4 w-4" aria-hidden="true" />
                篩選
              </button>
              {/* Desktop (lg+): re-open the rail when collapsed */}
              {railCollapsed && (
                <button
                  type="button"
                  onClick={() => setRailCollapsed(false)}
                  data-testid="library-rail-expand"
                  className={`hidden min-h-[44px] items-center gap-2 rounded-[var(--radius-md)] px-3 text-sm font-medium transition-colors lg:flex ${
                    activeFilterCount > 0
                      ? 'bg-[var(--accent-subtle)] text-[var(--accent-text)]'
                      : 'bg-[var(--bg-secondary)] text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]'
                  }`}
                >
                  <SlidersHorizontal className="h-4 w-4" aria-hidden="true" />
                  篩選
                  {activeFilterCount > 0 && (
                    <span className="rounded-full bg-[var(--accent-primary)] px-1.5 font-mono text-[11px] tabular-nums text-[var(--text-on-accent)]">
                      {activeFilterCount}
                    </span>
                  )}
                </button>
              )}

              {hasActiveFilters && (
                <div className="order-last w-full sm:order-none sm:w-auto">
                  <FilterChips
                    filters={filters}
                    onRemoveGenre={(g) =>
                      patchSearch({
                        genres: filters.genres.filter((x) => x !== g).join(',') || undefined,
                      })
                    }
                    onRemoveYearMin={() => patchSearch({ yearMin: undefined })}
                    onRemoveYearMax={() => patchSearch({ yearMax: undefined })}
                    onRemoveYears={() => patchSearch({ yearMin: undefined, yearMax: undefined })}
                    onRemoveUnmatched={() => patchSearch({ unmatched: undefined })}
                    onClearAll={clearFilters}
                  />
                </div>
              )}

              <span
                data-testid="library-result-count"
                className="ml-auto font-mono text-xs tabular-nums text-[var(--text-secondary)]"
              >
                {totalItems.toLocaleString()} 項
              </span>
              <button
                type="button"
                onClick={enterSelectionMode}
                data-testid="enter-selection-btn"
                className="flex min-h-[44px] items-center gap-2 rounded-[var(--radius-md)] bg-[var(--bg-secondary)] px-3 text-sm font-medium text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]"
              >
                <CheckSquare className="h-4 w-4" aria-hidden="true" />
                選取
              </button>
              <ViewToggle view={view} onViewChange={handleViewChange} />
            </div>
          )}

          {/* States */}
          {isError ? (
            <LibraryErrorV2 code={(error as { code?: string })?.code} onRetry={() => refetch()} />
          ) : isLoading ? (
            <LibraryGridSkeletonV2 />
          ) : isEmpty ? (
            hasActiveFilters ? (
              <LibraryNoResultV2 onClearFilters={clearFilters} />
            ) : (
              (() => {
                const state = classifyEmptyState({
                  qbtConfigured: qbtConfig.data?.configured,
                  mediaLibrariesCount: mediaLibraries.data?.libraries?.length ?? 0,
                  itemsCount: 0,
                  isLoading: qbtConfig.isLoading || mediaLibraries.isLoading,
                });
                if (state === 'loading') return <LibraryGridSkeletonV2 />;
                if (state === 'no-qbt') return <EmptyNoQBT />;
                if (state === 'no-folder') return <EmptyNoFolder />;
                return <EmptyReadyForScan />;
              })()
            )
          ) : view === 'grid' ? (
            <div data-testid="library-grid-v2" className={`grid gap-3 md:gap-4 ${gridColsClass}`}>
              {display.map((d) => (
                <PosterCardV2
                  key={`${d.type}-${d.id}`}
                  id={d.id}
                  type={d.type}
                  title={d.title}
                  posterPath={d.posterPath}
                  year={d.year}
                  meta={d.meta}
                  voteAverage={d.voteAverage}
                  media={d.media}
                  selectable={isSelectionMode}
                  selected={isSelectionMode && selectedIds.has(d.id)}
                  onSelect={isSelectionMode ? (e) => handleSelect(d.id, e) : undefined}
                />
              ))}
            </div>
          ) : (
            <div data-testid="library-list-v2" className="flex flex-col gap-0.5">
              {display.map((d) => (
                <LibraryListRowV2
                  key={`${d.type}-${d.id}`}
                  id={d.id}
                  type={d.type}
                  title={d.title}
                  posterPath={d.posterPath}
                  meta={d.listMeta}
                  voteAverage={d.voteAverage}
                  media={d.media}
                  selectable={isSelectionMode}
                  selected={isSelectionMode && selectedIds.has(d.id)}
                  onSelect={isSelectionMode ? (e) => handleSelect(d.id, e) : undefined}
                />
              ))}
            </div>
          )}

          {/* Continuous-scroll sentinel */}
          {!isEmpty && hasNextPage && (
            <div ref={sentinelRef} className="h-10" aria-hidden="true">
              {isFetchingNextPage && (
                <p className="py-3 text-center text-xs text-[var(--text-muted)]">載入更多…</p>
              )}
            </div>
          )}
        </div>
      </div>

      <LibraryFilterSheetV2
        open={filterSheetOpen}
        onOpenChange={setFilterSheetOpen}
        sortBy={effectiveSortBy}
        sortOrder={effectiveSortOrder}
        onSortChange={handleSortChange}
        filters={filters}
        mediaType={currentType}
        unmatchedCount={unmatchedCount}
        onApply={applyFilters}
        onClear={clearFilters}
        onTypeChange={handleTypeChange}
      />

      {/* ux3-cutover-2: batch dialogs */}
      <BatchConfirmDialog
        isOpen={confirmAction !== null}
        itemCount={selectedIds.size}
        action={confirmAction || 'delete'}
        onConfirm={handleBatchConfirm}
        onCancel={() => setConfirmAction(null)}
      />
      <BatchProgress
        isOpen={batchProgress.isOpen}
        current={batchProgress.current}
        total={batchProgress.total}
        action={batchProgress.action}
        errors={batchProgress.errors}
        isComplete={batchProgress.isComplete}
        onClose={closeBatchProgress}
      />
      <GenerationBatchDialogV2
        open={isBatchSubtitleOpen}
        onOpenChange={setIsBatchSubtitleOpen}
        selectedMovieIds={generationBatchSelection.movieIds}
        excludedSeriesCount={generationBatchSelection.excludedCount}
      />
    </div>
  );
}
