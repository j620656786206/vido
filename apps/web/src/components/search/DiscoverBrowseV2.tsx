// Design ref: ux-design.pen Screen I1-D-v2 (fxCVk)
/**
 * ux3-3-2: the v2 Discover experience — rendered by the /discover route ONLY under
 * the v2 shell (the route branches on `useShellVersion()`; the flag stays read-once
 * in __root, F4). Restyle + refine of the already-instant discover: a persistent
 * 264px filter rail (DiscoverFilterRail, converges with the 媒體庫 LibraryFilterRail),
 * collapsible (grid reclaims width via MediaGrid auto-fill), a single live total
 * (AC #3), a demoted read/remove chip-bar summary (AC #7), four v2 states (AC #8),
 * and a v2 batch mobile sheet (AC #9). Reuses MediaTypeTabs / PresetChips /
 * FilterChipBar / FilterBottomSheet / SavePresetDialog / useFilterState /
 * useDiscoverResults — no new filter engine, no new backend.
 */
import { useCallback, useEffect, useState } from 'react';
import { getRouteApi } from '@tanstack/react-router';
import { SlidersHorizontal } from 'lucide-react';
import { MediaTypeTabs, type MediaTypeFilter } from './MediaTypeTabs';
import { DiscoverFilterRail } from './DiscoverFilterRail';
import { FilterChipBar } from './FilterChipBar';
import { PresetChips } from './PresetChips';
import { SavePresetDialog } from './SavePresetDialog';
import { FilterBottomSheet } from './FilterBottomSheet';
import {
  DiscoverGridSkeletonV2,
  DiscoverNoResultV2,
  DiscoverSectionErrorV2,
} from './DiscoverStatesV2';
import { MediaGrid, type MediaItem } from '../media/MediaGrid';
import { Pagination } from '../ui/Pagination';
import { RequestsView } from '../requests/RequestsView';
import { useFilterState } from '../../hooks/useFilterState';
import { useDiscoverResults } from '../../hooks/useDiscoverResults';
import { useOwnedMedia } from '../../hooks/useOwnedMedia';
import { useRequestProgress } from '../../hooks/useRequestProgress';
import { usePageVisibility } from '../../hooks/useDownloads';
import { activeFilterChips, countActiveFilters, hasActiveFilters } from '../../lib/discoverFilters';

const routeApi = getRouteApi('/discover');
const RAIL_STORAGE_KEY = 'vido:discover:rail-collapsed';

function getStoredRailCollapsed(): boolean {
  try {
    return localStorage.getItem(RAIL_STORAGE_KEY) === '1';
  } catch {
    return false;
  }
}

const errCode = (e: unknown): string | undefined => (e as { code?: string } | null)?.code;

export function DiscoverBrowseV2() {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const { filters, setFilters, clearAll } = useFilterState();

  const currentType: MediaTypeFilter = search.type ?? 'all';
  const currentPage = search.page ?? 1;

  const [railCollapsed, setRailCollapsedState] = useState<boolean>(() => getStoredRailCollapsed());
  const setRailCollapsed = useCallback((next: boolean) => {
    setRailCollapsedState(next);
    try {
      localStorage.setItem(RAIL_STORAGE_KEY, next ? '1' : '0');
    } catch {
      /* ignore */
    }
  }, []);

  const [sheetOpen, setSheetOpen] = useState(false);
  const [saveDialogOpen, setSaveDialogOpen] = useState(false);

  // keepPrevious=true: a chip toggle keeps the prior grid visible (no skeleton
  // flash) and the rail's single total is the SAME query — zero extra count
  // queries (AC #3/#6).
  const { moviesQuery, tvQuery, isLoading, isFetching, totalResults } = useDiscoverResults(
    filters,
    currentType,
    currentPage,
    { enabled: true, keepPrevious: true }
  );

  const activeCount = countActiveFilters(filters);
  const activeLabels = activeFilterChips(filters).map((c) => c.label);

  const handleTypeChange = (t: MediaTypeFilter) =>
    navigate({ search: (prev) => ({ ...prev, type: t, page: 1 }) });
  const handlePageChange = (p: number) => navigate({ search: (prev) => ({ ...prev, page: p }) });

  // Per-section fail-soft (AC #8): each TMDB query is its own section.
  const wantMovies = currentType === 'all' || currentType === 'movie';
  const wantTV = currentType === 'all' || currentType === 'tv';
  const moviesErr = wantMovies && moviesQuery.isError && !moviesQuery.data;
  const tvErr = wantTV && tvQuery.isError && !tvQuery.data;
  const allErr = (!wantMovies || moviesErr) && (!wantTV || tvErr);

  const movieResults = wantMovies ? (moviesQuery.data?.results ?? []) : [];
  const tvResults = wantTV ? (tvQuery.data?.results ?? []) : [];
  const items: MediaItem[] = [
    ...movieResults.map((item): MediaItem => ({ item, mediaType: 'movie' })),
    ...tvResults.map((item): MediaItem => ({ item, mediaType: 'tv' })),
  ].sort((a, b) => b.item.voteCount - a.item.voteCount);
  const hasResults = items.length > 0;

  // Story 13-1b: ownership + requested state drive the per-card 想要 affordance
  // (owned → 已入庫, active request → 已請求·處理中, else the ＋想要 button).
  const ownership = useOwnedMedia(items.map((i) => i.item.id));

  // Story 13-1b AC #5 — the lit PH3-R2 entry opens the Discover-hosted 想要清單.
  const showRequests = search.view === 'requests';
  const toggleRequests = () =>
    navigate({
      search: (prev) => ({ ...prev, view: showRequests ? undefined : 'requests' }),
    });

  // Story 13-3b AC #3 — lazy request_progress SSE. Mirrors DownloadsBrowseV2:109-114
  // (never a bare mount effect, §8) with an ADDED view gate: connect only while the
  // 想要清單 view is active AND the page is visible; leaving the view or hiding the
  // tab closes the EventSource (no idle connection — Playwright networkidle stays safe).
  const { startTracking, stopTracking } = useRequestProgress();
  const isVisible = usePageVisibility();
  useEffect(() => {
    if (showRequests && isVisible) startTracking();
    else stopTracking();
  }, [showRequests, isVisible, startTracking, stopTracking]);

  const totalPages =
    currentType === 'movie'
      ? (moviesQuery.data?.totalPages ?? 1)
      : currentType === 'tv'
        ? (tvQuery.data?.totalPages ?? 1)
        : Math.max(moviesQuery.data?.totalPages ?? 1, tvQuery.data?.totalPages ?? 1);

  const sectionError = (() => {
    if (!moviesErr && !tvErr) return null;
    if (allErr) {
      return {
        message: 'TMDB 服務暫時無法連線，請稍後再試',
        code: errCode(moviesQuery.error ?? tvQuery.error),
        onRetry: () => {
          if (wantMovies) moviesQuery.refetch();
          if (wantTV) tvQuery.refetch();
        },
      };
    }
    const failedLabel = moviesErr ? '電影' : '節目';
    return {
      message: `${failedLabel}結果暫時無法載入，其他結果不受影響`,
      code: errCode(moviesErr ? moviesQuery.error : tvQuery.error),
      onRetry: () => (moviesErr ? moviesQuery.refetch() : tvQuery.refetch()),
    };
  })();

  const triggerClass = (active: boolean) =>
    `flex min-h-[44px] items-center gap-2 rounded-[var(--radius-md)] px-3 text-sm font-medium transition-colors ${
      active
        ? 'bg-[var(--accent-subtle)] text-[var(--accent-text)]'
        : 'bg-[var(--bg-secondary)] text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]'
    }`;

  return (
    <div className="px-4 py-6 sm:px-6">
      <h1 className="mb-4 text-2xl font-bold text-[var(--text-primary)]">探索</h1>

      <div className="lg:flex lg:gap-6">
        {/* Desktop filter rail (lg+); hidden when collapsed. <lg uses the sheet. */}
        {!railCollapsed && (
          <div className="hidden lg:block">
            <DiscoverFilterRail
              filters={filters}
              activeCount={activeCount}
              totalResults={totalResults}
              // AC #3: with keepPreviousData the prior total stays on-screen during a
              // refetch, so reflect isFetching (not isLoading, which only flips on a
              // cold first load) to show 計算中… while the new total is computing.
              isCounting={isFetching}
              onChange={setFilters}
              onClearAll={clearAll}
              onCollapse={() => setRailCollapsed(true)}
            />
          </div>
        )}

        <div className="min-w-0 lg:flex-1">
          {/* Toolbar: type tabs + filter triggers + inert Requests entry (PH3-R2) */}
          <div className="mb-4 flex flex-wrap items-center gap-2">
            <MediaTypeTabs
              activeType={currentType}
              onTypeChange={handleTypeChange}
              movieCount={moviesQuery.data?.totalResults}
              tvCount={tvQuery.data?.totalResults}
            />
            {/* Mobile (<lg): open the bottom sheet */}
            <button
              type="button"
              onClick={() => setSheetOpen(true)}
              data-testid="open-filter-sheet"
              aria-label="開啟篩選"
              className={`${triggerClass(activeCount > 0)} lg:hidden`}
            >
              <SlidersHorizontal className="h-4 w-4" aria-hidden="true" />
              篩選
            </button>
            {/* Desktop (lg+): re-open the rail when collapsed (grid reclaims width) */}
            {railCollapsed && (
              <button
                type="button"
                onClick={() => setRailCollapsed(false)}
                data-testid="discover-rail-expand"
                className={`${triggerClass(activeCount > 0)} hidden lg:flex`}
              >
                <SlidersHorizontal className="h-4 w-4" aria-hidden="true" />
                篩選
                {activeCount > 0 && (
                  <span className="rounded-full bg-[var(--accent-primary)] px-1.5 font-mono text-[11px] tabular-nums text-[var(--text-on-accent)]">
                    {activeCount}
                  </span>
                )}
              </button>
            )}
            {/* Epic-13 Requests entry — the PH3-R2 reserved slot, LIVE since
                Story 13-1b: toggles the Discover-hosted 想要清單 (?view=requests). */}
            <button
              type="button"
              data-testid="discover-requests-entry"
              aria-pressed={showRequests}
              onClick={toggleRequests}
              className={`ml-auto min-h-[44px] rounded-[var(--radius-md)] px-3 text-sm font-medium transition-colors ${
                showRequests
                  ? 'bg-[var(--accent-subtle)] text-[var(--accent-text)]'
                  : 'border border-[var(--border-subtle)] text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]'
              }`}
            >
              想要清單
            </button>
          </div>

          {/* Saved preset quick-access row */}
          <PresetChips onApplyPreset={setFilters} className="mb-3" />

          {/* Demoted read/remove chip-bar summary (AC #7) */}
          <FilterChipBar
            filters={filters}
            onChange={setFilters}
            onClearAll={clearAll}
            onSavePreset={hasActiveFilters(filters) ? () => setSaveDialogOpen(true) : undefined}
            summary
            className="mb-4"
          />

          {/* Story 13-1b: the 想要清單 view replaces the results area only —
              rail/toolbar/chips keep rendering (design L1 keeps the chrome). */}
          {showRequests ? (
            <RequestsView onExplore={toggleRequests} />
          ) : (
            <>
              {/* Per-section fail-soft banner (AC #8) — page never hard-fails */}
              {sectionError && (
                <DiscoverSectionErrorV2
                  message={sectionError.message}
                  code={sectionError.code}
                  onRetry={sectionError.onRetry}
                />
              )}

              {/* States */}
              {isLoading ? (
                <DiscoverGridSkeletonV2 />
              ) : allErr ? null : hasResults ? (
                <>
                  <MediaGrid items={items} ownership={ownership} />
                  {totalPages > 1 && (
                    <Pagination
                      currentPage={currentPage}
                      totalPages={totalPages}
                      onPageChange={handlePageChange}
                      className="mt-8"
                    />
                  )}
                </>
              ) : (
                <DiscoverNoResultV2 activeLabels={activeLabels} onClearFilters={clearAll} />
              )}
            </>
          )}
        </div>
      </div>

      {/* Mobile bottom sheet — batch, v2 restyle (AC #9) */}
      <FilterBottomSheet
        isOpen={sheetOpen}
        onClose={() => setSheetOpen(false)}
        filters={filters}
        onApply={setFilters}
        mediaType={currentType}
        variant="v2"
      />

      {saveDialogOpen && (
        <SavePresetDialog filters={filters} onClose={() => setSaveDialogOpen(false)} />
      )}
    </div>
  );
}
