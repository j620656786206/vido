// Design ref: ux-design.pen Screen D1-D-v2 (cK1KF)
import { useState } from 'react';
import { getRouteApi } from '@tanstack/react-router';
import { cn } from '../../lib/utils';
import { useDownloads, useDownloadCounts } from '../../hooks/useDownloads';
import { useQBittorrentConfig } from '../../hooks/useQBittorrent';
import type { FilterStatus, SortField, SortOrder } from '../../services/downloadService';
import { Pagination } from '../ui/Pagination';
import { DownloadCardV2 } from './DownloadCardV2';
import { DownloadsSkeletonV2, DownloadsEmptyV2, DownloadsQbtErrorV2 } from './DownloadsStatesV2';

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

/**
 * DownloadsBrowseV2 — the v2 deep page (ux3-4-3 AC2/AC6). Restyles the EXISTING read-only list:
 * reuses `useDownloads` / `useDownloadCounts` (the 5s poll stays for 4-3a; the lazy-SSE swap is
 * 4-3b) and the route's filter/page/pageSize search params. Renders the status-filter toolbar
 * (6 values, Mono counts), a single-column DownloadCard-v2 list, v2 pagination, and the four states.
 * Sort control, List|Table (D7) view toggle, and select/batch mode land in ux3-4-3b (GATE B).
 */
export function DownloadsBrowseV2() {
  const { filter: urlFilter, page: urlPage, pageSize: urlPageSize } = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const activeFilter: FilterStatus = urlFilter || 'all';
  const currentPageSize = urlPageSize || 100;

  // Sort is fixed to the newest-first default for 4-3a (the sort control is a 4-3b addition).
  const [sortField] = useState<SortField>('added_on');
  const [sortOrder] = useState<SortOrder>('desc');

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

  // Mirror the legacy search normalization (default filter/pageSize → undefined) so the URL and the
  // route's validateSearch stay in sync and the legacy/v2 URLs are interchangeable.
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

  // qBT unreachable (poll errored) OR resolved-but-not-configured → the same fail-soft section (AC6).
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
      <div className="flex flex-wrap gap-2" role="tablist" aria-label="下載狀態篩選">
        {FILTERS.map((f) => {
          const count = counts?.[f.value] ?? 0;
          const isActive = activeFilter === f.value;
          // Hide the error pill when there are no errors and it isn't the active filter (legacy parity).
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

      <div id="downloads-list-v2" role="tabpanel">
        {showQbtError ? (
          <DownloadsQbtErrorV2 onRetry={() => void refetch()} message={error?.message} />
        ) : !data || isLoading ? (
          <DownloadsSkeletonV2 />
        ) : data.items.length === 0 ? (
          <DownloadsEmptyV2 filter={activeFilter} />
        ) : (
          <div className="flex flex-col gap-3">
            {data.items.map((d) => (
              <DownloadCardV2 key={d.hash} download={d} />
            ))}
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
