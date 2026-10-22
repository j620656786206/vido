// Time-bomb-exempt: the only wall-clock read stamps the batch-export download
// FILENAME (vido-export-<type>-<date>.json) — user-facing file naming, never
// rendered UI state, so no fixture can flake on it (Murat: test-architecture call,
// same read the legacy LibraryPage ships).
// Design ref: ux-design.pen Screen A3p-D (LcHBs) · A4p-D (b1H71g) · A1p-M (BfGVZ) · A3p-M (h1v1U6) · A6p-M (Bz0YN) · E4-M (n7jVF)
/**
 * v2 Browse experience (UX Redesign Phase 2 — UX2-2). Rendered by the /library
 * route (sole render since ux3-cutover-3). One component serves all three type views
 * (all / movie / tv) off the `?type=` param — a shared grid + shared filter/sort/
 * scroll state that survives movies↔tv switches (AC #1 intent + F5). Integrated
 * toolbar, four states (empty/loading/no-result/error), grid (PosterCardV2) and
 * list (LibraryListV2), continuous scroll (infinite query, AC #11), and a
 * merged mobile sort+filter sheet. Reads URL state via the /library route api.
 */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { getRouteApi } from '@tanstack/react-router';
import { CheckSquare, SlidersHorizontal } from 'lucide-react';
import { useLibraryInfinite } from '../../hooks/useLibraryInfinite';
import { useEnrichmentRefresh } from '../../hooks/useEnrichmentRefresh';
import { useFilterRailTransition } from '../../hooks/useFilterRailTransition';
import { useQBittorrentConfig } from '../../hooks/useQBittorrent';
import { useMediaLibraries } from '../../hooks/useMediaLibrary';
import {
  useMovieStats,
  useSeriesStats,
  useBatchDelete,
  useBatchReparse,
  useBatchExport,
  useLibraryList,
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
import { LibraryListV2 } from './LibraryListV2';
import { LibraryFilterSheetV2, sheetCountParams } from './LibraryFilterSheetV2';
import { LibraryFilterRail } from './LibraryFilterRail';
import {
  LibraryGridSkeletonV2,
  LibraryListSkeletonV2,
  LibraryNoResultV2,
  LibraryErrorV2,
} from './LibraryStatesV2';
import { LIBRARY_GRID_COLS } from './libraryGridCols';
import { yearFilterLabel } from './FilterPanel';
import {
  chineseSubtitleChipLabel,
  joinChineseSubtitleCsv,
  parseChineseSubtitleCsv,
} from './chineseSubtitleFilter';
import { EmptyNoQBT } from './EmptyNoQBT';
import { EmptyNoFolder } from './EmptyNoFolder';
import { EmptyReadyForScan } from './EmptyReadyForScan';
import type { FilterValues } from './FilterPanel';
import type {
  BatchResult,
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

/**
 * A3p-D 頁首標題.
 *
 * ⚠️ NOT yet the single source: `shell/navModel.ts` says 媒體庫/電影/影集 and
 * `FilterPanel.tsx`'s type selector says 全部/電影/影集 — three maps, two answers
 * for `all`. Consolidating them is disc-2026-09-media-type-label-four-maps.
 */
const TYPE_TITLE: Record<LibraryMediaType, string> = {
  all: '媒體庫',
  movie: '電影',
  tv: '影集',
};
const DEFAULT_SORT = { sortBy: 'created_at' as SortField, sortOrder: 'desc' as SortOrder };
/** The clean route each type lives on (ux3-0-5). */
const TYPE_PATH = {
  all: '/library',
  movie: '/library/movies',
  tv: '/library/tv',
} as const satisfies Record<LibraryMediaType, string>;

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
  // dsr-1b-b: the sheet has two openers (phone header icon <640, toolbar 篩選 640–1024).
  // Focus returns to whichever is still shown; if neither (rotated past lg) the heading.
  const phoneFilterBtnRef = useRef<HTMLButtonElement>(null);
  const toolbarFilterBtnRef = useRef<HTMLButtonElement>(null);
  const headingRef = useRef<HTMLHeadingElement>(null);
  const isShown = (el: HTMLElement | null): el is HTMLElement =>
    !!el && el.getClientRects().length > 0;
  const filterSheetFinalFocus = () =>
    [phoneFilterBtnRef.current, toolbarFilterBtnRef.current].find(isShown) ??
    headingRef.current ??
    true;
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
  // disc-2026-10-filter-rail-toggle-no-motion (I11-D): clicks go through the shared hook,
  // which animates (View Transitions) and moves focus; the initial state above never does.
  const railExpandBtnRef = useRef<HTMLButtonElement>(null);
  const railCollapseBtnRef = useRef<HTMLButtonElement>(null);
  const { collapse: collapseRail, expand: expandRail } = useFilterRailTransition({
    setRailCollapsed,
    // In selection mode SelectionToolbar replaces the toolbar, so there is no 篩選
    // button to land on — fall back to the page title (AC #6, same as the sheet's).
    getCollapsedFocusTarget: useCallback(() => railExpandBtnRef.current ?? headingRef.current, []),
    getExpandedFocusTarget: useCallback(() => railCollapseBtnRef.current, []),
  });

  const filters: FilterValues = useMemo(
    () => ({
      genres: search.genres ? search.genres.split(',').filter(Boolean) : [],
      yearMin: search.yearMin,
      yearMax: search.yearMax,
      unmatched: search.unmatched,
      // csv on the URL, array in the UI; unknown values dropped here, so they
      // never reach the wire (disc-2026-10-subtitle-filter-disagrees-with-badges).
      chineseSubtitle: parseChineseSubtitleCsv(search.chineseSubtitle),
    }),
    [search.genres, search.yearMin, search.yearMax, search.unmatched, search.chineseSubtitle]
  );
  const chineseSubtitles = filters.chineseSubtitle ?? [];
  const hasActiveFilters =
    filters.genres.length > 0 ||
    filters.yearMin !== undefined ||
    filters.yearMax !== undefined ||
    filters.unmatched === true ||
    chineseSubtitles.length > 0;
  // Constraining-facet count for the rail badge / collapsed 篩選(n) button.
  // Decade range (yearMin/yearMax) counts as ONE facet; type=全部 is not a constraint.
  const activeFilterCount =
    filters.genres.length +
    (filters.yearMin !== undefined || filters.yearMax !== undefined ? 1 : 0) +
    (filters.unmatched === true ? 1 : 0) +
    chineseSubtitles.length;

  // Human labels for the same facets, in chip order — A7p-D names them in the
  // no-result line so you do not have to go looking for what excluded everything.
  // Same `filters` object as the count above AND the same `yearFilterLabel` the
  // chips use, so neither the number nor the wording can drift from the chip row
  // sitting directly above the sentence.
  const activeFilterLabels = useMemo(() => {
    const labels = [...filters.genres];
    const year = yearFilterLabel(filters);
    if (year) labels.push(year);
    if (filters.unmatched === true) labels.push('未匹配');
    for (const v of filters.chineseSubtitle ?? []) labels.push(chineseSubtitleChipLabel(v));
    return labels;
  }, [filters]);

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
    chineseSubtitle: joinChineseSubtitleCsv(chineseSubtitles),
  });

  // Empty-state classifier inputs (reuse the bugfix-10-5 3-state classifier).
  const qbtConfig = useQBittorrentConfig();
  const mediaLibraries = useMediaLibraries();
  const movieStats = useMovieStats();
  const seriesStats = useSeriesStats();
  const unmatchedCount =
    (movieStats.data?.unmatchedCount ?? 0) + (seriesStats.data?.unmatchedCount ?? 0);

  // Stay on the type's own route. `navigate` comes from the /library route api, so a
  // navigate with no `to` lands on /library itself: on 電影 or 影集 every sort, view or
  // filter change used to drop you on 全部 (with `type=all` in the URL) — found by
  // disc-2026-10-library-list-triage-b2, whose 「N 部缺中文」 counts within the type.
  const typePath = TYPE_PATH[currentType];
  const patchSearch = useCallback(
    (patch: Record<string, unknown>) => {
      navigate({ to: typePath, search: (prev) => ({ ...prev, type: undefined, ...patch }) });
    },
    [navigate, typePath]
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
        chineseSubtitle: joinChineseSubtitleCsv(f.chineseSubtitle),
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
        chineseSubtitle: undefined,
      }),
    [patchSearch]
  );
  // A4p 「· N 部缺中文」 (disc-2026-10-library-list-triage-b2 AC #3, ⚖️ Sally 5): how many
  // titles the 缺中文字幕 chip would list on top of what is filtered now. The SAME
  // 1-row query the sort+filter sheet's 「套用篩選 · N 部」 makes (sheetCountParams), so
  // the number cannot disagree with the list the link opens — no count endpoint, no
  // second copy of the rule. Not asked while a 字幕 filter is on: the link has no work
  // to do then.
  const hasSubtitleFilter = chineseSubtitles.length > 0;
  const missingZh = useLibraryList(
    sheetCountParams(
      { ...filters, chineseSubtitle: ['missing'] },
      currentType,
      effectiveSortBy,
      effectiveSortOrder
    ),
    { enabled: !hasSubtitleFilter }
  );
  const missingZhCount =
    !hasSubtitleFilter && !missingZh.isError ? (missingZh.data?.totalItems ?? 0) : 0;
  const showMissingZh = missingZhCount > 0;
  const showOnlyMissingZh = useCallback(() => {
    applyFilters({ ...filters, chineseSubtitle: ['missing'] });
    // The link unmounts with the filter it just set; keep focus on the page instead of
    // dropping it to <body>.
    headingRef.current?.focus();
  }, [applyFilters, filters]);

  // P3 跳到結果 (AC #4): skip the header, the rail's chips and the toolbar (≈40 Tabs).
  const resultsRef = useRef<HTMLElement>(null);

  const handleTypeChange = useCallback(
    (t: LibraryMediaType) =>
      navigate({ to: TYPE_PATH[t], search: (prev) => ({ ...prev, type: undefined }) }),
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
    /** Which batch this dialog is about — only 重新解析 shows the matching states. */
    kind?: 'delete' | 'reparse' | 'export';
    errors?: { id: string; message: string; title?: string }[];
  }>({ isOpen: false, current: 0, total: 0, action: '', isComplete: false });
  // disc-2026-09-batch-reparse-never-runs: after a batch 重新解析 the match
  // runs in the background; from then on this page refetches whenever a pass
  // completes, so 整理中 flips to the real result without a reload — and the
  // dialog shows the pass (C24-D, disc-2026-09-batch-reparse-progress-in-dialog).
  const [watchEnrichment, setWatchEnrichment] = useState(false);
  const [reparseRun, setReparseRun] = useState(0);
  const matching = useEnrichmentRefresh(watchEnrichment, reparseRun);
  const [isBatchSubtitleOpen, setIsBatchSubtitleOpen] = useState(false);
  // Mixed movie+episode selection since sub-4-2 D1 — no client-side filtering.
  const [generationBatchSelection, setGenerationBatchSelection] = useState<string[]>([]);
  // F17 deep-link opens force a fresh analysis (CR H2) — the scan just changed
  // the library, a ready snapshot would show a pre-scan list.
  const [generationForceAnalyze, setGenerationForceAnalyze] = useState(false);
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
      // disc-2026-10-batch-mixed-selection-wrong-type: 「全部」 mixes movies and
      // series, and the batch endpoints take ONE type per call — send each type
      // its own ids instead of tagging the whole batch with the last-clicked one.
      const groups = new Map<'movie' | 'series', string[]>();
      for (const id of ids) {
        const t = selectionTypesRef.current.get(id) ?? selectedType;
        groups.set(t, [...(groups.get(t) ?? []), id]);
      }
      const perType = Array.from(groups.entries());
      const runEach = async (
        mutate: (v: { ids: string[]; type: 'movie' | 'series' }) => Promise<BatchResult>
      ): Promise<BatchResult> => {
        const merged: BatchResult = { successCount: 0, failedCount: 0, errors: [] };
        for (const [type, typeIds] of perType) {
          const r = await mutate({ ids: typeIds, type });
          merged.successCount += r.successCount ?? 0;
          merged.failedCount += r.failedCount ?? 0;
          merged.errors = [...(merged.errors ?? []), ...(r.errors ?? [])];
        }
        if (merged.errors?.length === 0) merged.errors = undefined;
        return merged;
      };
      setBatchProgress({
        isOpen: true,
        current: 0,
        total,
        kind: action,
        action:
          action === 'delete' ? '刪除中...' : action === 'reparse' ? '重新解析中...' : '匯出中...',
        isComplete: false,
      });
      try {
        if (action === 'delete') {
          const result = await runEach(batchDeleteMutation.mutateAsync);
          setBatchProgress((prev) => ({
            ...prev,
            current: total,
            isComplete: true,
            errors: result.errors,
          }));
        } else if (action === 'reparse') {
          const result = await runEach(batchReparseMutation.mutateAsync);
          if (result.successCount > 0) {
            // A new run id restarts the watch from 已排入 even if it was already on.
            setReparseRun((n) => n + 1);
            setWatchEnrichment(true);
          }
          setBatchProgress((prev) => ({
            ...prev,
            // current = what was actually queued: the dialog says 含你勾的 N 部.
            current: result.successCount,
            isComplete: true,
            errors: result.errors,
          }));
        } else {
          const result: unknown[] = [];
          for (const [type, typeIds] of perType) {
            result.push(...(await batchExportMutation.mutateAsync({ ids: typeIds, type })));
          }
          const blob = new Blob([JSON.stringify(result, null, 2)], { type: 'application/json' });
          const url = URL.createObjectURL(blob);
          const a = document.createElement('a');
          a.href = url;
          const label = perType.length === 1 ? perType[0][0] : 'library';
          a.download = `vido-export-${label}-${new Date().toISOString().slice(0, 10)}.json`;
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

  // Generation consent flow: ALL selected ids pass through (mixed movie/episode
  // UUIDs are valid since sub-4-2 D1 — the old series filter + excluded note is
  // history). A series ROW id preselects that series' episodes, matched on
  // their seriesId (consentSelection.preselectionFor,
  // disc-2026-10-series-selection-preselects-episodes).
  const handleOpenGenerationBatch = useCallback(() => {
    setGenerationBatchSelection([...selectedIds]);
    setGenerationForceAnalyze(false);
    setIsBatchSubtitleOpen(true);
  }, [selectedIds]);

  // sub-4-3 F17 deep link: ?generate=1 (scan-complete toast) opens the consent
  // flow with no preselection, then strips the one-shot param (8-11
  // subtitleStatus deep-link precedent).
  const generateParam = search.generate === true;
  useEffect(() => {
    if (!generateParam) return;
    setGenerationBatchSelection([]);
    setGenerationForceAnalyze(true);
    setIsBatchSubtitleOpen(true);
    navigate({
      to: typePath,
      search: (prev) => ({ ...prev, type: undefined, generate: undefined }),
      replace: true,
    });
  }, [generateParam, navigate, typePath]);

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
  // dsr-1b-c: one table shared with LibraryGridSkeletonV2 — see libraryGridCols.ts.
  const gridColsClass = railCollapsed
    ? LIBRARY_GRID_COLS.railCollapsed
    : LIBRARY_GRID_COLS.railOpen;
  // b2 AC #6: the list view loads under a list-shaped skeleton, not poster tiles.
  const loadingSkeleton =
    view === 'list' ? (
      <LibraryListSkeletonV2 />
    ) : (
      <LibraryGridSkeletonV2 railCollapsed={railCollapsed} />
    );

  return (
    <div className="relative px-4 py-6 sm:px-6">
      {/* P3 跳到結果 (b2 AC #4): the first stop on the page, seen only when focused. It
          moves focus with focus() — an href="#…" jump would let the hash into the URL. */}
      <a
        href="#library-results"
        data-testid="library-skip-to-results"
        onClick={(e) => {
          e.preventDefault();
          resultsRef.current?.focus();
        }}
        className="sr-only rounded-[var(--radius-md)] bg-[var(--bg-secondary)] text-sm font-medium text-[var(--text-primary)] shadow-[var(--shadow-xl)] focus:not-sr-only focus:absolute focus:left-4 focus:top-2 focus:z-30 focus:px-3 focus:py-2 sm:focus:left-6"
      >
        跳到結果
      </a>
      {/* A3p-D 頁首 — the screen says WHERE YOU ARE and HOW MUCH IS HERE in one
            line. ⚖️ Alexyu 2026-09-16 (dsr-1 AC #7): the count belongs beside the
            title, counted in 部 — 「電影 1,284 部」 reads as a sentence, and the
            homepage readout band already counts in 部. It used to sit in the
            toolbar in 項, which (a) said it twice in the product's two voices and
            (b) VANISHED in selection mode, exactly when 已選取 N 項 makes the
            total worth keeping on screen.
            ⚠️ A3p-D draws this INSIDE the top app bar, on the omnisearch row. Here
            it is the first thing in the content column instead, so it scrolls away
            with the page rather than staying pinned. Moving it into AppShellV2
            needs a pageTitle slot on the shell — a bigger change than this story,
            tracked as disc-2026-09-library-header-not-in-shell-bar.
            The count is hidden while loading/errored, matching A2p-D and A8p-D
            (which show the title with no number) and A1p-D (which shows 「電影 0 部」). */}
      <div className="mb-4 flex items-center justify-between gap-3">
        <div className="flex min-w-0 items-baseline gap-3">
          {/* tabIndex -1: only ever focused by code — where the sort+filter sheet lands
              when the button that opened it is no longer shown (dsr-1b-b). */}
          <h1
            ref={headingRef}
            tabIndex={-1}
            data-testid="library-page-title"
            // A3p-M: the phone title is H4 18 (DESIGN.md — headings step down one size below sm).
            className="text-xl font-semibold text-[var(--text-primary)] outline-none max-sm:text-lg"
          >
            {TYPE_TITLE[currentType]}
          </h1>
          {!isLoading && !isError && (
            <span
              data-testid="library-result-count"
              className="shrink-0 whitespace-nowrap font-mono text-xs tabular-nums text-[var(--text-secondary)]"
            >
              {totalItems.toLocaleString()} 部
            </span>
          )}
          {/* b2 AC #3 — its own element, NOT inside library-result-count: that span's text
              is matched exactly (「2 部」) by the e2e. Pressing it = the 缺中文字幕 chip.
              The padding/negative margin grow the tap target without moving the line. */}
          {!isLoading && !isError && showMissingZh && (
            <button
              type="button"
              data-testid="library-missing-zh-link"
              onClick={showOnlyMissingZh}
              aria-label={`${missingZhCount.toLocaleString()} 部缺中文，只看缺中文字幕的片`}
              className="-my-2.5 min-w-0 truncate py-2.5 font-mono text-xs tabular-nums text-[var(--accent-text)] underline-offset-2 hover:underline"
            >
              <span aria-hidden="true">· </span>
              {missingZhCount.toLocaleString()} 部缺中文
            </button>
          )}
        </div>
        {/* A6p-M / A3p-M (dsr-1b-b): on a phone the sort+filter sheet opens from a 44×44
            icon button on the title row — the toolbar's 篩選 button takes over at sm. */}
        {!isSelectionMode && (
          <button
            type="button"
            ref={phoneFilterBtnRef}
            onClick={() => setFilterSheetOpen(true)}
            aria-label="篩選"
            aria-haspopup="dialog"
            aria-expanded={filterSheetOpen}
            data-testid="library-filter-open-phone"
            className={`relative flex size-11 shrink-0 items-center justify-center rounded-[var(--radius-md)] sm:hidden ${
              activeFilterCount > 0
                ? 'bg-[var(--accent-subtle)] text-[var(--accent-text)]'
                : 'bg-[var(--bg-tertiary)] text-[var(--text-secondary)]'
            }`}
          >
            <SlidersHorizontal className="size-5" aria-hidden="true" />
            {activeFilterCount > 0 && (
              <span
                data-testid="library-filter-open-phone-count"
                className="absolute -right-1 -top-1 min-w-[18px] rounded-full bg-[var(--accent-primary)] px-1 text-center font-mono text-xs leading-[18px] tabular-nums text-[var(--text-on-accent)]"
              >
                {activeFilterCount}
              </span>
            )}
          </button>
        )}
      </div>

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
              onCollapse={collapseRail}
              collapseButtonRef={railCollapseBtnRef}
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
              {/* Tablet (sm–lg): opens the bottom sheet. Below sm the header icon button
                  is the entry (dsr-1b-b); this one stays in the DOM, CSS-hidden. */}
              <button
                type="button"
                ref={toolbarFilterBtnRef}
                onClick={() => setFilterSheetOpen(true)}
                aria-haspopup="dialog"
                aria-expanded={filterSheetOpen}
                data-testid="library-filter-open"
                className={`hidden min-h-[44px] items-center gap-2 rounded-[var(--radius-md)] px-3 text-sm font-medium transition-colors sm:flex lg:hidden ${
                  hasActiveFilters
                    ? 'bg-[var(--accent-subtle)] text-[var(--accent-text)]'
                    : 'bg-[var(--bg-secondary)] text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]'
                }`}
              >
                <SlidersHorizontal className="h-4 w-4" aria-hidden="true" />
                篩選
              </button>
              {/* Desktop (lg+): re-open the rail when collapsed. aria-expanded talks about the
                  RAIL (always false here — the button only exists while it is collapsed);
                  data-rail-vt="trigger" pairs it with the rail's 「篩選 N」 for the I11-D morph. */}
              {railCollapsed && (
                <button
                  type="button"
                  ref={railExpandBtnRef}
                  onClick={expandRail}
                  aria-expanded={false}
                  data-rail-vt="trigger"
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
                    <span className="rounded-full bg-[var(--accent-primary)] px-1.5 font-mono text-xs tabular-nums text-[var(--text-on-accent)]">
                      {activeFilterCount}
                    </span>
                  )}
                </button>
              )}

              {hasActiveFilters && (
                <div className="order-last w-full sm:order-none sm:w-auto">
                  <FilterChips
                    // Phone (dsr-1b-b, D1-M pattern): ONE row that scrolls sideways, bled to
                    // the screen edges; -my-1 py-1 keeps the 4px focus ring from being
                    // clipped by the scroller; scroll-px-4 keeps a focused chip off the edges.
                    className="[scrollbar-width:none] max-sm:-mx-4 max-sm:-my-1 max-sm:flex-nowrap max-sm:overflow-x-auto max-sm:overscroll-x-contain max-sm:scroll-px-4 max-sm:px-4 max-sm:py-1 [&::-webkit-scrollbar]:hidden"
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
                    onRemoveChineseSubtitle={(v) =>
                      patchSearch({
                        chineseSubtitle: joinChineseSubtitleCsv(
                          chineseSubtitles.filter((x) => x !== v)
                        ),
                      })
                    }
                    onClearAll={clearFilters}
                  />
                </div>
              )}

              <button
                type="button"
                onClick={enterSelectionMode}
                // The count used to live here with ml-auto; it moved to the page
                // header (dsr-1 AC #7), so this button now carries the right edge.
                data-testid="enter-selection-btn"
                className="ml-auto flex min-h-[44px] items-center gap-2 rounded-[var(--radius-md)] bg-[var(--bg-secondary)] px-3 text-sm font-medium text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]"
              >
                <CheckSquare className="h-4 w-4" aria-hidden="true" />
                選取
              </button>
              <ViewToggle view={view} onViewChange={handleViewChange} />
            </div>
          )}

          {/* States — the 跳到結果 target: one container whatever is showing, so the next
              Tab after the jump is the first card / row (or the state's own button). */}
          <section
            ref={resultsRef}
            id="library-results"
            tabIndex={-1}
            aria-label="媒體庫結果"
            data-testid="library-results"
            className="outline-none"
          >
            {isError ? (
              <LibraryErrorV2 code={(error as { code?: string })?.code} onRetry={() => refetch()} />
            ) : isLoading ? (
              loadingSkeleton
            ) : isEmpty ? (
              hasActiveFilters ? (
                <LibraryNoResultV2
                  onClearFilters={clearFilters}
                  mediaType={currentType}
                  activeFilters={activeFilterLabels}
                />
              ) : (
                (() => {
                  const state = classifyEmptyState({
                    qbtConfigured: qbtConfig.data?.configured,
                    mediaLibrariesCount: mediaLibraries.data?.libraries?.length ?? 0,
                    itemsCount: 0,
                    isLoading: qbtConfig.isLoading || mediaLibraries.isLoading,
                  });
                  if (state === 'loading') return loadingSkeleton;
                  if (state === 'no-qbt') return <EmptyNoQBT />;
                  if (state === 'no-folder') return <EmptyNoFolder />;
                  return <EmptyReadyForScan />;
                })()
              )
            ) : view === 'grid' ? (
              <div
                data-testid="library-grid-v2"
                className={`grid ${LIBRARY_GRID_COLS.gap} ${gridColsClass}`}
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
                    selectable={isSelectionMode}
                    selected={isSelectionMode && selectedIds.has(d.id)}
                    onSelect={isSelectionMode ? (e) => handleSelect(d.id, e) : undefined}
                  />
                ))}
              </div>
            ) : (
              <LibraryListV2
                rows={display.map(
                  ({ id, type, title, posterPath, listMeta, voteAverage, media }) => ({
                    id,
                    type,
                    title,
                    posterPath,
                    meta: listMeta,
                    voteAverage,
                    media,
                  })
                )}
                selectable={isSelectionMode}
                selectedIds={isSelectionMode ? selectedIds : undefined}
                onSelect={isSelectionMode ? handleSelect : undefined}
              />
            )}

            {/* Continuous-scroll sentinel */}
            {!isEmpty && hasNextPage && (
              <div ref={sentinelRef} className="h-10" aria-hidden="true">
                {isFetchingNextPage && (
                  <p className="py-3 text-center text-xs text-[var(--text-muted)]">載入更多…</p>
                )}
              </div>
            )}
          </section>
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
        finalFocus={filterSheetFinalFocus}
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
        matching={batchProgress.kind === 'reparse' && batchProgress.current > 0 ? matching : null}
        onClose={closeBatchProgress}
      />
      <GenerationBatchDialogV2
        open={isBatchSubtitleOpen}
        onOpenChange={setIsBatchSubtitleOpen}
        selectedMediaIds={generationBatchSelection}
        forceAnalyze={generationForceAnalyze}
        onGoToKeySettings={() => {
          setIsBatchSubtitleOpen(false);
          void navigate({ to: '/settings/keys' });
        }}
      />
    </div>
  );
}
