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
import { SlidersHorizontal } from 'lucide-react';
import { useLibraryInfinite } from '../../hooks/useLibraryInfinite';
import { useQBittorrentConfig } from '../../hooks/useQBittorrent';
import { useMediaLibraries } from '../../hooks/useMediaLibrary';
import { useMovieStats, useSeriesStats } from '../../hooks/useLibrary';
import { classifyEmptyState } from '../../utils/emptyLibraryState';
import { SortSelector } from './SortSelector';
import { FilterChips } from './FilterChips';
import { ViewToggle, type ViewMode } from './ViewToggle';
import { PosterCardV2 } from './PosterCardV2';
import { LibraryListRowV2 } from './LibraryListRowV2';
import { LibraryFilterSheetV2 } from './LibraryFilterSheetV2';
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

export function LibraryBrowseV2() {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();

  const currentType = (search.type as LibraryMediaType) || 'all';
  const stored = useMemo(() => getStoredSort(), []);
  const effectiveSortBy = (search.sortBy as SortField) || stored.sortBy;
  const effectiveSortOrder = (search.sortOrder as SortOrder) || stored.sortOrder;
  const [view, setView] = useState<ViewMode>(() => (search.view as ViewMode) || getStoredView());
  const [filterSheetOpen, setFilterSheetOpen] = useState(false);

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

  return (
    <div className="px-4 py-6 sm:px-6">
      {/* Integrated toolbar */}
      <div className="mb-4 flex flex-wrap items-center gap-2">
        <SortSelector
          sortBy={effectiveSortBy}
          sortOrder={effectiveSortOrder}
          onSortChange={handleSortChange}
        />
        <button
          type="button"
          onClick={() => setFilterSheetOpen(true)}
          data-testid="library-filter-open"
          className={`flex min-h-[44px] items-center gap-2 rounded-[var(--radius-md)] px-3 text-sm font-medium transition-colors ${
            hasActiveFilters
              ? 'bg-[var(--accent-subtle)] text-[var(--accent-text)]'
              : 'bg-[var(--bg-secondary)] text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]'
          }`}
        >
          <SlidersHorizontal className="h-4 w-4" aria-hidden="true" />
          篩選
        </button>

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
        <ViewToggle view={view} onViewChange={handleViewChange} />
      </div>

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
        <div
          data-testid="library-grid-v2"
          className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:gap-4 lg:grid-cols-4 xl:grid-cols-6"
        >
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
        onTypeChange={(t) => patchSearch({ type: t === 'all' ? undefined : t })}
      />
    </div>
  );
}
