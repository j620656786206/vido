// Design ref: ux-design.pen Screen D1-D-v2 (cK1KF)
import { useEffect, useMemo, useState, useSyncExternalStore } from 'react';
import { getRouteApi } from '@tanstack/react-router';
import { cn } from '../../lib/utils';
import { useDownloads, useDownloadCounts, usePageVisibility } from '../../hooks/useDownloads';
import { useDownloadActions } from '../../hooks/useDownloadActions';
import { useDownloadProgress } from '../../hooks/useDownloadProgress';
import { useDownloadsView } from '../../hooks/useDownloadsView';
import { useQBittorrentConfig } from '../../hooks/useQBittorrent';
import type { FilterStatus, SortField, SortOrder } from '../../services/downloadService';
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
import { DownloadCardV2 } from './DownloadCardV2';
import { DownloadsTableV2 } from './DownloadsTableV2';
import {
  DownloadsSkeletonV2,
  DownloadsTableSkeletonV2,
  DownloadsEmptyV2,
  DownloadsQbtErrorV2,
} from './DownloadsStatesV2';

const routeApi = getRouteApi('/downloads');

const PAGE_SIZE_OPTIONS = [50, 100, 200, 500] as const;

const FILTERS: { value: FilterStatus; label: string }[] = [
  { value: 'all', label: '全部' },
  { value: 'downloading', label: '下載中' },
  { value: 'paused', label: '已暫停' },
  { value: 'completed', label: '已完成' },
  { value: 'seeding', label: '做種中' },
  { value: 'error', label: '錯誤' },
];

const SORT_FIELDS: { value: SortField; label: string }[] = [
  { value: 'added_on', label: '加入時間' },
  { value: 'name', label: '名稱' },
  { value: 'progress', label: '進度' },
  { value: 'status', label: '狀態' },
];

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
  const actions = useDownloadActions();

  // AC4: lazy SSE — connect only while the page is visible (never a bare mount effect, §8).
  const { startTracking, stopTracking } = useDownloadProgress();
  const isVisible = usePageVisibility();
  useEffect(() => {
    if (isVisible) startTracking();
    else stopTracking();
  }, [isVisible, startTracking, stopTracking]);

  const items = useMemo(() => data?.items ?? [], [data]);

  const handleFilterChange = (f: FilterStatus) =>
    navigate({
      search: {
        filter: f === 'all' ? undefined : f,
        pageSize: currentPageSize !== 100 ? currentPageSize : undefined,
      },
      replace: true,
    });
  const handlePageChange = (p: number) =>
    navigate({
      search: {
        filter: urlFilter,
        page: p > 1 ? p : undefined,
        pageSize: currentPageSize !== 100 ? currentPageSize : undefined,
      },
      replace: true,
    });
  const handlePageSizeChange = (s: number) =>
    navigate({
      search: { filter: urlFilter, pageSize: s !== 100 ? s : undefined },
      replace: true,
    });

  // Shared sort — the List sort control and the Table column headers both call this.
  const handleSort = (field: SortField) => {
    if (field === sortField) setSortOrder((o) => (o === 'desc' ? 'asc' : 'desc'));
    else {
      setSortField(field);
      setSortOrder('desc');
    }
  };

  // --- actions (AC3) ---
  const onPause = (hash: string) => actions.pause.mutate([hash]);
  const onResume = (hash: string) => actions.resume.mutate([hash]);
  const onRemove = (hash: string, deleteFiles: boolean) =>
    actions.remove.mutate({ hashes: [hash], deleteFiles });

  // --- selection + batch (AC5; shared by list select-mode + table persistent checkboxes) ---
  const toggleSelect = (hash: string, next: boolean) =>
    setSelected((prev) => {
      const s = new Set(prev);
      if (next) s.add(hash);
      else s.delete(hash);
      return s;
    });
  const selectAll = () => setSelected(new Set(items.map((d) => d.hash)));
  const clearSelection = () => setSelected(new Set());
  const exitSelection = () => {
    setSelectMode(false);
    clearSelection();
  };
  const selectedHashes = useMemo(() => [...selected], [selected]);
  const batchPause = () => selectedHashes.length && actions.pause.mutate(selectedHashes);
  const batchResume = () => selectedHashes.length && actions.resume.mutate(selectedHashes);
  const batchRemove = (deleteFiles: boolean) => {
    if (selectedHashes.length) actions.remove.mutate({ hashes: selectedHashes, deleteFiles });
    clearSelection();
  };

  // Table checkboxes are persistent → the batch bar follows the selection; List follows select-mode.
  const showBatchBar = showTable ? selected.size > 0 : selectMode;
  const showQbtError = Boolean(error) || (configResolved && !isConfigured);

  return (
    <div
      data-testid="downloads-browse-v2"
      className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-8 sm:px-6"
    >
      <header className="flex flex-col gap-1">
        <h1 className="text-2xl font-bold text-[var(--text-primary)]">下載</h1>
        <p className="text-sm text-[var(--text-secondary)]">即時監控 qBittorrent 下載狀態</p>
      </header>

      {/* Status-filter toolbar — 6 live values, counts in Mono */}
      <div className="flex flex-wrap items-center gap-2">
        <div className="flex flex-wrap gap-2" role="tablist" aria-label="下載狀態篩選">
          {FILTERS.map((f) => {
            const count = counts?.[f.value] ?? 0;
            const isActive = activeFilter === f.value;
            if (f.value === 'error' && count === 0 && !isActive) return null;
            return (
              <button
                key={f.value}
                type="button"
                role="tab"
                aria-selected={isActive}
                aria-controls="downloads-list-v2"
                onClick={() => handleFilterChange(f.value)}
                className={cn(
                  'inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-sm font-medium transition-colors',
                  isActive
                    ? 'border-[var(--accent-primary)] bg-[var(--accent-tint)] text-[var(--accent-text)]'
                    : 'border-[var(--border-subtle)] bg-[var(--bg-secondary)] text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
                )}
              >
                <span>{f.label}</span>
                <span className="font-mono text-xs tabular-nums text-[var(--text-muted)]">
                  {count}
                </span>
              </button>
            );
          })}
        </div>

        {/* right-aligned list toolbar: sort + List|Table toggle + select toggle */}
        <div className="ml-auto flex items-center gap-2">
          <label className="flex items-center gap-1.5 text-sm text-[var(--text-secondary)]">
            <span className="sr-only sm:not-sr-only">排序</span>
            <select
              value={sortField}
              onChange={(e) => setSortField(e.target.value as SortField)}
              aria-label="排序欄位"
              className="rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-tertiary)] px-2 py-1 text-sm text-[var(--text-primary)] focus:border-[var(--accent-primary)] focus:outline-none"
            >
              {SORT_FIELDS.map((s) => (
                <option key={s.value} value={s.value}>
                  {s.label}
                </option>
              ))}
            </select>
          </label>
          <Button
            size="sm"
            variant="outline"
            aria-label={`排序方向：${sortOrder === 'desc' ? '遞減' : '遞增'}`}
            onClick={() => setSortOrder((o) => (o === 'desc' ? 'asc' : 'desc'))}
          >
            {sortOrder === 'desc' ? '↓' : '↑'}
          </Button>

          {/* List | Table view toggle — desktop only (AC1) */}
          <div
            className="hidden items-center overflow-hidden rounded-[var(--radius-md)] border border-[var(--border-subtle)] lg:flex"
            role="group"
            aria-label="檢視方式"
          >
            <button
              type="button"
              aria-pressed={view === 'list'}
              onClick={() => setView('list')}
              className={cn(
                'px-3 py-1 text-sm',
                view === 'list'
                  ? 'bg-[var(--accent-tint)] text-[var(--accent-text)]'
                  : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
              )}
            >
              清單
            </button>
            <button
              type="button"
              aria-pressed={view === 'table'}
              onClick={() => setView('table')}
              className={cn(
                'px-3 py-1 text-sm',
                view === 'table'
                  ? 'bg-[var(--accent-tint)] text-[var(--accent-text)]'
                  : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
              )}
            >
              表格
            </button>
          </div>

          {/* select-mode toggle — List view only (Table has a persistent checkbox column) */}
          {!showTable && (
            <Button
              size="sm"
              variant={selectMode ? 'secondary' : 'outline'}
              aria-pressed={selectMode}
              onClick={() => (selectMode ? exitSelection() : setSelectMode(true))}
            >
              選取
            </Button>
          )}
        </div>
      </div>

      {/* batch action bar (AC5) */}
      {showBatchBar && (
        <div
          data-testid="downloads-batch-bar"
          className="flex flex-wrap items-center gap-2 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-3"
        >
          <span className="text-sm text-[var(--text-secondary)]">
            已選 <span className="font-mono tabular-nums">{selected.size}</span> 項
          </span>
          <Button size="sm" variant="outline" onClick={selectAll}>
            全選
          </Button>
          <Button size="sm" variant="outline" disabled={!selected.size} onClick={batchPause}>
            批次暫停
          </Button>
          <Button size="sm" variant="outline" disabled={!selected.size} onClick={batchResume}>
            批次繼續
          </Button>
          <Dialog>
            <DialogTrigger asChild>
              <Button size="sm" variant="destructive" disabled={!selected.size}>
                批次移除
              </Button>
            </DialogTrigger>
            <DialogContent aria-describedby={undefined}>
              <DialogHeader>
                <DialogTitle>批次移除 {selected.size} 項下載</DialogTitle>
                <DialogDescription>
                  保留檔案只從 qBittorrent 移除任務；連同檔案刪除會一併刪除已下載的檔案，無法復原。
                </DialogDescription>
              </DialogHeader>
              <DialogFooter>
                <DialogClose asChild>
                  <Button variant="outline" onClick={() => batchRemove(false)}>
                    移除（保留檔案）
                  </Button>
                </DialogClose>
                <DialogClose asChild>
                  <Button variant="destructive" onClick={() => batchRemove(true)}>
                    移除（連同檔案刪除）
                  </Button>
                </DialogClose>
              </DialogFooter>
            </DialogContent>
          </Dialog>
          <Button size="sm" variant="ghost" className="ml-auto" onClick={exitSelection}>
            取消
          </Button>
        </div>
      )}

      <div id="downloads-list-v2" role="tabpanel">
        {showQbtError ? (
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
          <div className="flex flex-col gap-3">
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
                  selected={selected.has(d.hash)}
                  onSelectChange={toggleSelect}
                  onPause={onPause}
                  onResume={onResume}
                  onRemove={onRemove}
                />
              ))
            )}
            {data.totalPages > 1 && (
              <div className="mt-3 flex flex-wrap items-center justify-between gap-3">
                <label className="flex items-center gap-2 text-sm text-[var(--text-secondary)]">
                  <span>每頁</span>
                  <select
                    value={currentPageSize}
                    onChange={(e) => handlePageSizeChange(Number(e.target.value))}
                    aria-label="每頁筆數"
                    className="rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-tertiary)] px-2 py-1 font-mono text-sm text-[var(--text-primary)] focus:border-[var(--accent-primary)] focus:outline-none"
                  >
                    {PAGE_SIZE_OPTIONS.map((s) => (
                      <option key={s} value={s}>
                        {s}
                      </option>
                    ))}
                  </select>
                  <span>筆</span>
                </label>
                <Pagination
                  currentPage={data.page}
                  totalPages={data.totalPages}
                  onPageChange={handlePageChange}
                />
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
