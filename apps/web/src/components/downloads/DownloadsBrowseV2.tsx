// Design ref: ux-design.pen Screen D1-D-v2 (cK1KF) · D1-M-v2 (uMDjw) · D8-M-v2 (jDgxJ) · D9-M-v2 (DrYXb)
// (also renders D2-D-v2 batch select (tx6U1) + D7-D-v2 table view (w3ipb))
import { useEffect, useMemo, useRef, useState, useSyncExternalStore } from 'react';
import { getRouteApi } from '@tanstack/react-router';
import {
  ArrowDownUp,
  CheckCheck,
  ChevronDown,
  ListChecks,
  Pause,
  Play,
  Rows3,
  Table,
  Trash2,
  X,
  type LucideIcon,
} from 'lucide-react';
import { cn } from '../../lib/utils';
import { useDownloads, useDownloadCounts, usePageVisibility } from '../../hooks/useDownloads';
import { useDownloadActions } from '../../hooks/useDownloadActions';
import { useDownloadProgress } from '../../hooks/useDownloadProgress';
import { useDownloadsView, type DownloadsView } from '../../hooks/useDownloadsView';
import { useQBittorrentConfig } from '../../hooks/useQBittorrent';
import { useIsPhone } from '../../hooks/useIsPhone';
import type { Download, FilterStatus, SortField, SortOrder } from '../../services/downloadService';
import { Button } from '../ui/Button';
import { Pagination } from '../ui/Pagination';
import {
  Dialog,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogFooter,
  DialogTitle,
  DialogDescription,
  DialogClose,
} from '../ui/Dialog';
import { DeleteWithFilesDialog } from './DeleteWithFilesDialog';
import { DownloadActionsSheet } from './DownloadActionsSheet';
import { DownloadCardV2 } from './DownloadCardV2';
import { DownloadDetailSheet } from './DownloadDetailSheet';
import { DownloadSortSheet } from './DownloadSortSheet';
import { DownloadsTableV2 } from './DownloadsTableV2';
import {
  DownloadsSkeletonV2,
  DownloadsTableSkeletonV2,
  DownloadsEmptyV2,
  DownloadsQbtErrorV2,
  DownloadsQbtNotConfiguredV2,
} from './DownloadsStatesV2';

const routeApi = getRouteApi('/downloads');

const PAGE_SIZE_OPTIONS = [50, 100, 200, 500] as const;

const FILTERS: { value: FilterStatus; label: string }[] = [
  { value: 'all', label: '全部' },
  { value: 'downloading', label: '下載中' },
  { value: 'paused', label: '已暫停' },
  { value: 'completed', label: '已完成' },
  // Same word as the status pill, so the chip and the card name one state one way.
  { value: 'seeding', label: '做種' },
  { value: 'error', label: '錯誤' },
];

// One control for field + direction (D1-D-v2 sortDropdown). The table's column headers drive the
// same state, so every field/order pair they can produce has an option here.
export const SORT_OPTIONS: { field: SortField; order: SortOrder; label: string }[] = [
  { field: 'added_on', order: 'desc', label: '加入時間（新到舊）' },
  { field: 'added_on', order: 'asc', label: '加入時間（舊到新）' },
  { field: 'name', order: 'asc', label: '名稱（A–Z）' },
  { field: 'name', order: 'desc', label: '名稱（Z–A）' },
  { field: 'progress', order: 'desc', label: '進度（多到少）' },
  { field: 'progress', order: 'asc', label: '進度（少到多）' },
  { field: 'status', order: 'asc', label: '狀態' },
  { field: 'status', order: 'desc', label: '狀態（反向）' },
];

const VIEWS: { value: DownloadsView; label: string; icon: LucideIcon }[] = [
  { value: 'list', label: '清單檢視', icon: Rows3 },
  { value: 'table', label: '表格檢視', icon: Table },
];

const OUTLINE_BUTTON =
  'inline-flex h-11 shrink-0 items-center gap-2 rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-4 text-sm font-medium text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)] disabled:cursor-not-allowed disabled:opacity-50 [&_svg]:size-[18px]';

const BATCH_BUTTON =
  'inline-flex h-11 items-center gap-2 rounded-[var(--radius-md)] px-4 text-sm font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-50 [&_svg]:size-[18px]';
const BATCH_NEUTRAL =
  'bg-[var(--bg-tertiary)] text-[var(--text-primary)] hover:bg-[var(--border-subtle)] [&_svg]:text-[var(--text-secondary)]';
const BATCH_DANGER =
  'bg-[var(--error-tint)] text-[var(--error-text)] hover:bg-[var(--error-tint)]/70 [&_svg]:text-[var(--error-text)]';

/** Rendered and laid out — `display:none` (e.g. `sm:hidden`) fails; jsdom has no layout, so it passes. */
function isShown(el: HTMLElement | null): el is HTMLElement {
  return (
    el !== null &&
    el.isConnected &&
    (typeof el.checkVisibility !== 'function' || el.checkVisibility())
  );
}

// Desktop breakpoint (Tailwind lg = 1024px). Table view is desktop-only (AC1) — mobile always renders
// the card List even if a stale desktop preference says 'table'. Guarded so a missing matchMedia
// (non-browser test env) defaults to desktop; exercised for real in the E2E's 1280px viewport.
const DESKTOP_MQ = '(min-width: 1024px)';
function useIsDesktop(): boolean {
  return useSyncExternalStore(
    (cb) => {
      if (typeof window.matchMedia !== 'function') return () => {};
      const mql = window.matchMedia(DESKTOP_MQ);
      mql.addEventListener('change', cb);
      return () => mql.removeEventListener('change', cb);
    },
    () => (typeof window.matchMedia === 'function' ? window.matchMedia(DESKTOP_MQ).matches : true),
    () => true
  );
}

/**
 * DownloadsBrowseV2 — the v2 deep page (ux3-4-3 List + ux3-4-4 Table). One toolbar/state drives two
 * renderings of the SAME page data: the card List (default) and the D7 dense Table (desktop-only,
 * localStorage-persisted view). The sort control and the Table's column headers are two controls over
 * one sortField/sortOrder; card select-mode and the Table's persistent checkbox column share one
 * selection Set + batch bar. Actions (useDownloadActions) + live SSE (useDownloadProgress) are reused
 * by both — no second EventSource, no second poll.
 */
export function DownloadsBrowseV2() {
  const { filter: urlFilter, page: urlPage, pageSize: urlPageSize } = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const activeFilter: FilterStatus = urlFilter || 'all';
  const currentPageSize = urlPageSize || 100;

  const [sortField, setSortField] = useState<SortField>('added_on');
  const [sortOrder, setSortOrder] = useState<SortOrder>('desc');
  const [sortSheetOpen, setSortSheetOpen] = useState(false);
  const isPhone = useIsPhone();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const sortBtnRef = useRef<HTMLButtonElement>(null);
  const sortSelectRef = useRef<HTMLSelectElement>(null);
  const chipRowRef = useRef<HTMLDivElement>(null);

  const [view, setView] = useDownloadsView();
  const isDesktop = useIsDesktop();
  const showTable = view === 'table' && isDesktop;

  // Select mode (list only; the Table has a persistent checkbox column)
  const [selectMode, setSelectMode] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(new Set());

  const { data: qbtConfig } = useQBittorrentConfig();
  const configResolved = qbtConfig !== undefined;
  const isConfigured = qbtConfig?.configured === true;

  const { data, isLoading, error, refetch } = useDownloads(
    activeFilter,
    sortField,
    sortOrder,
    urlPage || 1,
    currentPageSize
  );
  const { data: counts } = useDownloadCounts();

  // One chip row that scrolls on a phone can start with the active chip off-screen
  // (`?filter=seeding` at 390px) — bring it inside the row's 16px gutters. On a wrapping
  // (desktop) row nothing overflows, so scrollLeft stays 0.
  useEffect(() => {
    const row = chipRowRef.current;
    const chip = row?.querySelector<HTMLElement>('[role="tab"][aria-selected="true"]');
    if (!row || !chip) return;
    const r = row.getBoundingClientRect();
    const c = chip.getBoundingClientRect();
    const gutter = 16;
    if (c.right > r.right - gutter) row.scrollLeft += c.right - (r.right - gutter);
    else if (c.left < r.left + gutter) row.scrollLeft -= r.left + gutter - c.left;
  }, [activeFilter, counts]);

  const actions = useDownloadActions();

  // AC4: lazy SSE — connect only while the page is visible (never a bare mount effect, §8).
  const { startTracking, stopTracking } = useDownloadProgress();
  const isVisible = usePageVisibility();
  useEffect(() => {
    if (isVisible) startTracking();
    else stopTracking();
  }, [isVisible, startTracking, stopTracking]);

  const items = useMemo(() => data?.items ?? [], [data]);

  // Anything that swaps the rows out drops the selection outright; rows that leave on their own
  // are filtered out by `selectedHashes` below.
  const clearSelection = () => setSelected(new Set());

  const handleFilterChange = (f: FilterStatus) => {
    clearSelection();
    navigate({
      search: {
        filter: f === 'all' ? undefined : f,
        pageSize: currentPageSize !== 100 ? currentPageSize : undefined,
      },
      replace: true,
    });
  };
  const handlePageChange = (p: number) => {
    clearSelection();
    navigate({
      search: {
        filter: urlFilter,
        page: p > 1 ? p : undefined,
        pageSize: currentPageSize !== 100 ? currentPageSize : undefined,
      },
      replace: true,
    });
  };
  const handlePageSizeChange = (s: number) => {
    clearSelection();
    navigate({
      search: { filter: urlFilter, pageSize: s !== 100 ? s : undefined },
      replace: true,
    });
  };
  const handleViewChange = (next: DownloadsView) => {
    if (next === view) return;
    setSelectMode(false);
    clearSelection();
    setView(next);
  };

  // Shared sort — the List sort control and the Table column headers both call this.
  const handleSort = (field: SortField) => {
    clearSelection();
    if (field === sortField) setSortOrder((o) => (o === 'desc' ? 'asc' : 'desc'));
    else {
      setSortField(field);
      setSortOrder('desc');
    }
  };
  const handleSortOption = (value: string) => {
    const option = SORT_OPTIONS.find((o) => `${o.field}:${o.order}` === value);
    if (!option) return;
    clearSelection();
    setSortField(option.field);
    setSortOrder(option.order);
  };

  // --- actions (AC3) ---
  const onPause = (hash: string) => actions.pause.mutate([hash]);
  const onResume = (hash: string) => actions.resume.mutate([hash]);
  const onRemove = (hash: string, deleteFiles: boolean) =>
    actions.remove.mutate({ hashes: [hash], deleteFiles });

  // --- phone card sheets (dsr-4b-2 D8-M / D9-M) — the page owns them; a card only reports ⋯ ---
  // One union-typed slot: "one overlay at a time" is a matter of shape, not discipline.
  const [sheet, setSheet] = useState<{
    kind: 'actions' | 'detail';
    hash: string;
    /** What the sheet showed last — keeps it rendered through its exit once the item is gone. */
    snapshot: Download;
    /** false = closing. The slot empties only once the exit has finished (onOpenChangeComplete):
     *  an abrupt unmount would run Base UI's return-focus while the card's ⋯ is still in the
     *  DOM, focus a button that is removed a moment later, and leave focus on <body>. */
    open: boolean;
  } | null>(null);
  // A snapshot, not a hash: removal is optimistic, so the row is gone before the dialog closes.
  const [confirmTarget, setConfirmTarget] = useState<Download | null>(null);
  const triggerRef = useRef<HTMLElement | null>(null);
  // True while one overlay is closing so that another can open (actions → detail, detail →
  // actions, actions → confirm). Read — and cleared — by the sheets' finalFocus below; also
  // cleared once the next overlay has fully opened, so an aborted exit cannot leave it stuck.
  const handoffRef = useRef(false);
  // What opens once the current sheet's exit has FINISHED: 先關再開, never both at once. A
  // sheet that opened in the same commit would cross-fade with the old one; a confirm dialog
  // (z-50) would animate in UNDER the exiting popup (z-71) and its scrim.
  const pendingRef = useRef<'actions' | 'detail' | 'confirm' | null>(null);
  // Always the CURRENT list item: the sheet follows every refetch. When the item leaves the list
  // (removed, paged, filtered) the sheet closes.
  const liveDownload = sheet ? items.find((d) => d.hash === sheet.hash) : undefined;
  const sheetDownload = liveDownload ?? sheet?.snapshot;
  if (sheet?.open && !liveDownload) setSheet({ ...sheet, open: false });
  const closeSheet = () => setSheet((s) => (s && s.open ? { ...s, open: false } : s));
  /** Once a sheet's exit has finished: open what was waiting on it, or drop the slot — unless a
   *  newer sheet has taken the slot in the meantime. */
  const sheetExited = (kind: 'actions' | 'detail') => {
    const next = pendingRef.current;
    pendingRef.current = null;
    if (next === 'confirm') setConfirmTarget(sheet?.snapshot ?? null);
    setSheet((s) => {
      if (!s || s.kind !== kind || s.open) return s;
      return next === 'actions' || next === 'detail' ? { ...s, kind: next, open: true } : null;
    });
  };
  /** Where focus lands after a card sheet or the confirm dialog: the ⋯ that opened it, or the
   *  page heading once that card has been removed — never <body>. */
  const cardFocusTarget = () => {
    const t = triggerRef.current;
    return t?.isConnected ? t : headingRef.current;
  };
  // Base UI reads this once, synchronously, as the popup unmounts. `false` = leave focus alone:
  // the next overlay's own initial focus decides, so two focus managers never fight over a tick.
  const cardSheetFinalFocus = () => {
    const handoff = handoffRef.current;
    handoffRef.current = false;
    return handoff ? false : (cardFocusTarget() ?? true);
  };
  const openActionsFromCard = (hash: string, trigger: HTMLElement) => {
    const snapshot = items.find((d) => d.hash === hash);
    if (!snapshot) return;
    triggerRef.current = trigger;
    handoffRef.current = false;
    pendingRef.current = null;
    setSheet({ kind: 'actions', hash, snapshot, open: true });
  };
  /** Close the current sheet; `next` opens when its exit has finished. */
  const handoffTo = (next: 'actions' | 'detail' | 'confirm') => {
    handoffRef.current = true;
    pendingRef.current = next;
    // Refresh the snapshot first: the confirm shows the name/size the sheet showed.
    setSheet((s) => (s && sheetDownload ? { ...s, snapshot: sheetDownload, open: false } : s));
  };

  // --- selection + batch (AC5; shared by list select-mode + table persistent checkboxes) ---
  const toggleSelect = (hash: string, next: boolean) =>
    setSelected((prev) => {
      const s = new Set(prev);
      if (next) s.add(hash);
      else s.delete(hash);
      return s;
    });
  const selectAll = () => setSelected(new Set(items.map((d) => d.hash)));
  const exitSelection = () => {
    setSelectMode(false);
    clearSelection();
  };
  // Only rows on screen count. A selected torrent can leave the page with no click here — it
  // finishes under the 下載中 filter, another client removes it — and a batch delete must never
  // reach a row the user can no longer see.
  const selectedHashes = useMemo(
    () => items.filter((d) => selected.has(d.hash)).map((d) => d.hash),
    [items, selected]
  );
  const batchPause = () => selectedHashes.length && actions.pause.mutate(selectedHashes);
  const batchResume = () => selectedHashes.length && actions.resume.mutate(selectedHashes);
  const batchRemove = (deleteFiles: boolean) => {
    if (selectedHashes.length) actions.remove.mutate({ hashes: selectedHashes, deleteFiles });
    clearSelection();
  };

  // Never set up ≠ can't reach it: the first gets its own card with no 重試 (nothing to retry).
  const notConfigured = configResolved && !isConfigured;
  const qbtUnavailable = Boolean(error) || notConfigured;
  const listSelecting = selectMode && !showTable && !qbtUnavailable;
  // Table checkboxes are persistent → the batch bar follows the selection; List follows select-mode
  // and takes the toolbar's place (D2-D-v2).
  const showBatchBar = !qbtUnavailable && (showTable ? selectedHashes.length > 0 : selectMode);
  const showToolbar = !qbtUnavailable && !listSelecting;

  // Phone sort sheet (D10-M-v2): the header's 排序 button opens it; the desktop select stays.
  // The sheet only exists where its button does, so it closes — during render, before a frame
  // with an open sheet and no button can paint — when the toolbar goes (select mode, qBT down)
  // and when the window grows past `sm` (a phone rotated to landscape).
  if (!showToolbar && sortSheetOpen) setSortSheetOpen(false);
  const [sheetOnPhone, setSheetOnPhone] = useState(isPhone);
  if (sheetOnPhone !== isPhone) {
    setSheetOnPhone(isPhone);
    if (!isPhone) {
      setSortSheetOpen(false);
      closeSheet();
    }
  }
  // Focus after closing: the 排序 button; if it is hidden now (rotated past `sm`) the select that
  // replaced it; if both are gone (toolbar hidden) the page heading — never <body>.
  const sortSheetFinalFocus = () =>
    [sortBtnRef.current, sortSelectRef.current].find(isShown) ?? headingRef.current ?? true;

  const rangeStart = data ? (data.page - 1) * data.pageSize + 1 : 0;
  const rangeEnd = data ? Math.min(data.page * data.pageSize, data.totalItems) : 0;

  return (
    <div
      data-testid="downloads-browse-v2"
      className="mx-auto flex w-full max-w-6xl flex-col gap-5 px-4 py-8 sm:px-8"
    >
      <header className="flex items-start justify-between gap-4">
        <div className="flex flex-col gap-1">
          {/* tabIndex -1: only ever focused by code — the landing spot when a sheet closes and
              the control that opened it is gone. */}
          <h1
            ref={headingRef}
            tabIndex={-1}
            className="text-2xl font-bold text-[var(--text-primary)] outline-none"
          >
            下載
          </h1>
          <p className="text-sm text-[var(--text-secondary)]">
            {listSelecting ? '批次選取模式' : '管理所有下載任務'}
          </p>
        </div>
        {listSelecting && (
          <button type="button" onClick={exitSelection} className={OUTLINE_BUTTON}>
            <X className="!size-4" aria-hidden="true" />
            取消
          </button>
        )}
        {showToolbar && (
          <button
            type="button"
            ref={sortBtnRef}
            onClick={() => setSortSheetOpen(true)}
            aria-label="排序"
            aria-haspopup="dialog"
            aria-expanded={sortSheetOpen}
            data-testid="downloads-sort-btn"
            className="flex size-11 shrink-0 items-center justify-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] text-[var(--text-secondary)] sm:hidden"
          >
            <ArrowDownUp className="size-5" aria-hidden="true" />
          </button>
        )}
      </header>

      <DownloadSortSheet
        open={sortSheetOpen}
        onOpenChange={setSortSheetOpen}
        finalFocus={sortSheetFinalFocus}
        options={SORT_OPTIONS}
        value={`${sortField}:${sortOrder}`}
        onChange={handleSortOption}
      />

      {sheetDownload && (
        <DownloadActionsSheet
          download={sheetDownload}
          open={sheet?.kind === 'actions' && sheet.open}
          onOpenChange={(open) => {
            if (!open) closeSheet();
          }}
          onOpenChangeComplete={(open) => {
            if (open) handoffRef.current = false;
            else sheetExited('actions');
          }}
          finalFocus={cardSheetFinalFocus}
          onPause={onPause}
          onResume={onResume}
          onRemove={onRemove}
          onShowDetails={() => handoffTo('detail')}
          onRequestDeleteWithFiles={() => handoffTo('confirm')}
        />
      )}
      {sheetDownload && (
        <DownloadDetailSheet
          download={sheetDownload}
          open={sheet?.kind === 'detail' && sheet.open}
          onOpenChange={(open) => {
            if (!open) closeSheet();
          }}
          onOpenChangeComplete={(open) => {
            if (open) handoffRef.current = false;
            else sheetExited('detail');
          }}
          finalFocus={cardSheetFinalFocus}
          onPause={onPause}
          onResume={onResume}
          onOpenActions={() => handoffTo('actions')}
        />
      )}
      <DeleteWithFilesDialog
        download={confirmTarget}
        open={confirmTarget !== null}
        onOpenChange={(open) => {
          if (!open) setConfirmTarget(null);
        }}
        onConfirm={(hash) => onRemove(hash, true)}
        onCloseAutoFocus={(e) => {
          e.preventDefault();
          cardFocusTarget()?.focus();
        }}
      />

      {/* Status-filter chips — 6 live values, counts in Mono. One row that scrolls sideways on a
          phone (D1-M-v2), bled to the screen edges (-mx-4 px-4). -my-1 py-1 keeps the 4px focus
          ring from being clipped top and bottom by the scroller; scroll-px-4 keeps a focused
          chip off the edges; overscroll-x-contain stops a swipe at the end turning into "back". */}
      <div
        ref={chipRowRef}
        className="flex flex-wrap gap-2 [scrollbar-width:none] max-sm:-mx-4 max-sm:-my-1 max-sm:flex-nowrap max-sm:overflow-x-auto max-sm:overscroll-x-contain max-sm:scroll-px-4 max-sm:px-4 max-sm:py-1 [&::-webkit-scrollbar]:hidden"
        role="tablist"
        aria-label="下載狀態篩選"
      >
        {FILTERS.map((f) => {
          // No counts (not set up, or still loading) reads「—」, not a confident 0.
          const count = counts?.[f.value];
          const isActive = activeFilter === f.value;
          if (f.value === 'error' && !count && !isActive) return null;
          return (
            <button
              key={f.value}
              type="button"
              role="tab"
              aria-selected={isActive}
              aria-controls="downloads-list-v2"
              onClick={() => handleFilterChange(f.value)}
              className={cn(
                'inline-flex h-11 items-center gap-2 rounded-full px-4 text-sm transition-colors max-sm:shrink-0',
                isActive
                  ? 'bg-[var(--accent-subtle)] font-semibold text-[var(--accent-text)]'
                  : 'bg-[var(--bg-tertiary)] font-medium text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
              )}
            >
              <span>{f.label}</span>
              <span className="font-mono text-xs font-normal tabular-nums">{count ?? '—'}</span>
            </button>
          );
        })}
      </div>

      {showToolbar && (
        <div className="flex flex-wrap items-center justify-between gap-3">
          {showTable ? (
            <p className="text-sm text-[var(--text-secondary)]">
              {data && (
                <>
                  共{' '}
                  <span className="font-mono font-semibold tabular-nums text-[var(--text-primary)]">
                    {data.totalItems}
                  </span>{' '}
                  筆任務
                </>
              )}
            </p>
          ) : (
            <button
              type="button"
              onClick={() => setSelectMode(true)}
              disabled={items.length === 0}
              className={OUTLINE_BUTTON}
            >
              <ListChecks aria-hidden="true" />
              選取
            </button>
          )}

          <div className="flex items-center gap-2">
            <label className="relative flex h-11 items-center rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] pl-3 text-sm text-[var(--text-secondary)] focus-within:border-[var(--accent-primary)] max-sm:hidden">
              <span aria-hidden="true">排序：</span>
              <select
                ref={sortSelectRef}
                value={`${sortField}:${sortOrder}`}
                onChange={(e) => handleSortOption(e.target.value)}
                aria-label="排序方式"
                className="h-full cursor-pointer appearance-none bg-transparent pr-9 text-sm text-[var(--text-secondary)] outline-none"
              >
                {SORT_OPTIONS.map((o) => (
                  <option
                    key={`${o.field}:${o.order}`}
                    value={`${o.field}:${o.order}`}
                    className="bg-[var(--bg-secondary)] text-[var(--text-primary)]"
                  >
                    {o.label}
                  </option>
                ))}
              </select>
              <ChevronDown
                className="pointer-events-none absolute right-3 size-3.5 text-[var(--text-muted)]"
                aria-hidden="true"
              />
            </label>

            {/* List | Table view toggle — desktop only (AC1) */}
            <div
              role="group"
              aria-label="檢視方式"
              className="hidden items-center gap-0.5 rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-1 lg:flex"
            >
              {VIEWS.map((v) => {
                const isActive = view === v.value;
                return (
                  <button
                    key={v.value}
                    type="button"
                    aria-label={v.label}
                    title={v.label}
                    aria-pressed={isActive}
                    onClick={() => handleViewChange(v.value)}
                    className={cn(
                      'flex h-[38px] w-10 items-center justify-center rounded-[var(--radius-md)] transition-colors',
                      isActive
                        ? 'bg-[var(--accent-subtle)] text-[var(--accent-text)]'
                        : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
                    )}
                  >
                    <v.icon className="size-4" aria-hidden="true" />
                  </button>
                );
              })}
            </div>
          </div>
        </div>
      )}

      {/* batch action bar (AC5) */}
      {showBatchBar && (
        <div
          data-testid="downloads-batch-bar"
          className="flex flex-wrap items-center justify-between gap-3 rounded-[var(--radius-lg)] border border-[var(--accent-primary)] bg-[var(--accent-tint)] px-4 py-2.5"
        >
          <p className="flex items-baseline gap-1 text-sm font-medium text-[var(--text-primary)]">
            已選
            <span className="font-mono text-base font-semibold tabular-nums text-[var(--accent-text)]">
              {selectedHashes.length}
            </span>
            項
          </p>
          <div className="flex flex-wrap items-center gap-2">
            <button type="button" className={cn(BATCH_BUTTON, BATCH_NEUTRAL)} onClick={selectAll}>
              <CheckCheck aria-hidden="true" />
              全選
            </button>
            <button
              type="button"
              className={cn(BATCH_BUTTON, BATCH_NEUTRAL)}
              disabled={!selectedHashes.length}
              onClick={batchPause}
            >
              <Pause aria-hidden="true" />
              批次暫停
            </button>
            <button
              type="button"
              className={cn(BATCH_BUTTON, BATCH_NEUTRAL)}
              disabled={!selectedHashes.length}
              onClick={batchResume}
            >
              <Play aria-hidden="true" />
              批次繼續
            </button>
            <Dialog>
              <DialogTrigger asChild>
                <button
                  type="button"
                  className={cn(BATCH_BUTTON, BATCH_DANGER)}
                  disabled={!selectedHashes.length}
                >
                  <Trash2 aria-hidden="true" />
                  批次移除
                </button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>批次移除 {selectedHashes.length} 項下載</DialogTitle>
                  <DialogDescription>
                    保留檔案只從 qBittorrent
                    移除任務；連同檔案刪除會一併刪除已下載的檔案，無法復原。
                  </DialogDescription>
                </DialogHeader>
                <DialogFooter>
                  <DialogClose asChild>
                    <Button
                      variant="outline"
                      className="h-11 font-semibold"
                      onClick={() => batchRemove(false)}
                    >
                      移除（保留檔案）
                    </Button>
                  </DialogClose>
                  <DialogClose asChild>
                    <Button
                      variant="destructive"
                      className="h-11 font-semibold"
                      onClick={() => batchRemove(true)}
                    >
                      移除（連同檔案刪除）
                    </Button>
                  </DialogClose>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>
        </div>
      )}

      <div id="downloads-list-v2" role="tabpanel">
        {notConfigured ? (
          <DownloadsQbtNotConfiguredV2 />
        ) : qbtUnavailable ? (
          <DownloadsQbtErrorV2 onRetry={() => void refetch()} message={error?.message} />
        ) : !data || isLoading ? (
          showTable ? (
            <DownloadsTableSkeletonV2 />
          ) : (
            <DownloadsSkeletonV2 />
          )
        ) : items.length === 0 ? (
          <DownloadsEmptyV2 filter={activeFilter} />
        ) : (
          <div className="flex flex-col gap-4">
            {showTable ? (
              <DownloadsTableV2
                items={items}
                sortField={sortField}
                sortOrder={sortOrder}
                onSort={handleSort}
                selected={selected}
                onSelectChange={toggleSelect}
                onSelectAll={selectAll}
                onClearAll={clearSelection}
                onPause={onPause}
                onResume={onResume}
                onRemove={onRemove}
              />
            ) : (
              items.map((d) => (
                <DownloadCardV2
                  key={d.hash}
                  download={d}
                  selectable={selectMode}
                  selected={selectMode && selected.has(d.hash)}
                  onSelectChange={toggleSelect}
                  onPause={onPause}
                  onResume={onResume}
                  onRemove={onRemove}
                  onOpenActions={openActionsFromCard}
                />
              ))
            )}
            <div className="mt-1 flex flex-wrap items-center justify-between gap-3">
              <div className="flex items-center gap-4 text-xs text-[var(--text-secondary)]">
                <span data-testid="downloads-page-summary" className="font-mono tabular-nums">
                  {rangeStart}–{rangeEnd} / {data.totalItems}
                </span>
                {data.totalItems > PAGE_SIZE_OPTIONS[0] && (
                  <label className="flex items-center gap-2">
                    <span>每頁</span>
                    <select
                      value={currentPageSize}
                      onChange={(e) => handlePageSizeChange(Number(e.target.value))}
                      aria-label="每頁筆數"
                      className="h-11 rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-2 font-mono text-xs text-[var(--text-primary)] focus:border-[var(--accent-primary)] focus:outline-none"
                    >
                      {PAGE_SIZE_OPTIONS.map((s) => (
                        <option key={s} value={s}>
                          {s}
                        </option>
                      ))}
                    </select>
                    <span>筆</span>
                  </label>
                )}
              </div>
              <Pagination
                currentPage={data.page}
                totalPages={data.totalPages}
                onPageChange={handlePageChange}
              />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
