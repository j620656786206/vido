import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useState } from 'react';
import { useDownloads, useDownloadCounts } from '../hooks/useDownloads';
import { DownloadList } from '../components/downloads/DownloadList';
import { DownloadFilterTabs } from '../components/downloads/DownloadFilterTabs';
import type { FilterStatus, SortField, SortOrder } from '../services/downloadService';
import { ChevronLeft, ChevronRight } from 'lucide-react';

interface DownloadsSearch {
  filter?: FilterStatus;
  page?: number;
  pageSize?: number;
}

const PAGE_SIZE_OPTIONS = [50, 100, 200, 500] as const;

export const Route = createFileRoute('/downloads')({
  validateSearch: (search: Record<string, unknown>): DownloadsSearch => {
    const validFilters = ['all', 'downloading', 'paused', 'completed', 'seeding', 'error'];
    const filter = validFilters.includes(search.filter as string)
      ? (search.filter as FilterStatus)
      : undefined;
    const page = Number(search.page) > 0 ? Number(search.page) : undefined;
    const pageSize = PAGE_SIZE_OPTIONS.includes(
      Number(search.pageSize) as (typeof PAGE_SIZE_OPTIONS)[number]
    )
      ? Number(search.pageSize)
      : undefined;
    return { filter, page, pageSize };
  },
  component: DownloadsPage,
});

function DownloadsPage() {
  const { filter: urlFilter, page: urlPage, pageSize: urlPageSize } = Route.useSearch();
  const navigate = useNavigate();
  const activeFilter: FilterStatus = urlFilter || 'all';
  const currentPage = urlPage || 1;
  const currentPageSize = urlPageSize || 100;

  const [sortField, setSortField] = useState<SortField>('added_on');
  const [sortOrder, setSortOrder] = useState<SortOrder>('desc');

  const { data, isLoading, error } = useDownloads(
    activeFilter,
    sortField,
    sortOrder,
    currentPage,
    currentPageSize
  );
  const { data: counts } = useDownloadCounts();

  const handleFilterChange = (newFilter: FilterStatus) => {
    navigate({
      search: {
        filter: newFilter === 'all' ? undefined : newFilter,
        pageSize: currentPageSize !== 100 ? currentPageSize : undefined,
      },
      replace: true,
    });
  };

  const handlePageChange = (page: number) => {
    navigate({
      search: {
        filter: urlFilter,
        page: page > 1 ? page : undefined,
        pageSize: currentPageSize !== 100 ? currentPageSize : undefined,
      },
      replace: true,
    });
  };

  const handlePageSizeChange = (size: number) => {
    navigate({
      search: {
        filter: urlFilter,
        pageSize: size !== 100 ? size : undefined,
      },
      replace: true,
    });
  };

  return (
    <div className="mx-auto max-w-[1400px] px-6 py-8 lg:px-10">
      <h1 className="mb-2 text-2xl font-bold text-slate-100">下載管理</h1>
      <p className="mb-6 text-sm text-slate-400">
        即時監控 qBittorrent 下載狀態，每 5 秒自動更新。
      </p>

      <DownloadFilterTabs
        activeFilter={activeFilter}
        counts={counts}
        onFilterChange={handleFilterChange}
      />

      {isLoading && (
        <div className="flex items-center justify-center py-12">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-blue-500 border-t-transparent" />
          <span className="ml-3 text-slate-400">載入中...</span>
        </div>
      )}

      {error && (
        <div className="rounded-lg border border-red-800 bg-red-900/20 p-4 text-sm text-red-300">
          <p className="font-medium">無法載入下載清單</p>
          <p className="mt-1 text-red-400">{error.message}</p>
        </div>
      )}

      {!isLoading && !error && data && (
        <div id="download-list" role="tabpanel">
          {data.items.length === 0 ? (
            <EmptyFilterState filter={activeFilter} />
          ) : (
            <>
              <DownloadList
                downloads={data.items}
                totalItems={data.totalItems}
                sortField={sortField}
                sortOrder={sortOrder}
                onSortChange={setSortField}
                onOrderChange={setSortOrder}
              />
              {data.totalPages > 1 && (
                <Pagination
                  page={data.page}
                  totalPages={data.totalPages}
                  pageSize={currentPageSize}
                  onPageChange={handlePageChange}
                  onPageSizeChange={handlePageSizeChange}
                />
              )}
            </>
          )}
        </div>
      )}
    </div>
  );
}

function Pagination({
  page,
  totalPages,
  pageSize,
  onPageChange,
  onPageSizeChange,
}: {
  page: number;
  totalPages: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  onPageSizeChange: (size: number) => void;
}) {
  const getPageNumbers = () => {
    const pages: (number | '...')[] = [];
    if (totalPages <= 7) {
      for (let i = 1; i <= totalPages; i++) pages.push(i);
    } else {
      pages.push(1);
      if (page > 3) pages.push('...');
      for (let i = Math.max(2, page - 1); i <= Math.min(totalPages - 1, page + 1); i++) {
        pages.push(i);
      }
      if (page < totalPages - 2) pages.push('...');
      pages.push(totalPages);
    }
    return pages;
  };

  return (
    <div className="mt-6 flex items-center justify-between">
      <div className="flex items-center gap-2 text-sm text-slate-400">
        <span>每頁</span>
        <select
          value={pageSize}
          onChange={(e) => onPageSizeChange(Number(e.target.value))}
          className="rounded-md border border-slate-600 bg-slate-700 px-2 py-1 text-sm text-slate-200 focus:border-blue-500 focus:outline-none"
        >
          {PAGE_SIZE_OPTIONS.map((size) => (
            <option key={size} value={size}>
              {size}
            </option>
          ))}
        </select>
        <span>筆</span>
        {pageSize >= 500 && <span className="text-xs text-yellow-500">大量項目可能影響效能</span>}
      </div>

      <div className="flex items-center gap-1">
        <button
          type="button"
          onClick={() => onPageChange(page - 1)}
          disabled={page <= 1}
          className="rounded-md border border-slate-600 bg-slate-700 p-1.5 text-slate-300 hover:bg-slate-600 disabled:cursor-not-allowed disabled:opacity-40"
        >
          <ChevronLeft className="h-4 w-4" />
        </button>

        {getPageNumbers().map((p, i) =>
          p === '...' ? (
            <span key={`dots-${i}`} className="px-2 text-sm text-slate-500">
              ...
            </span>
          ) : (
            <button
              key={p}
              type="button"
              onClick={() => onPageChange(p)}
              className={`min-w-[32px] rounded-md px-2 py-1 text-sm font-medium ${
                p === page
                  ? 'bg-blue-600 text-white'
                  : 'border border-slate-600 bg-slate-700 text-slate-300 hover:bg-slate-600'
              }`}
            >
              {p}
            </button>
          )
        )}

        <button
          type="button"
          onClick={() => onPageChange(page + 1)}
          disabled={page >= totalPages}
          className="rounded-md border border-slate-600 bg-slate-700 p-1.5 text-slate-300 hover:bg-slate-600 disabled:cursor-not-allowed disabled:opacity-40"
        >
          <ChevronRight className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}

function EmptyFilterState({ filter }: { filter: FilterStatus }) {
  const messages: Record<FilterStatus, string> = {
    all: '目前沒有下載任務',
    downloading: '沒有正在下載的任務',
    paused: '沒有已暫停的任務',
    completed: '沒有已完成的任務',
    seeding: '沒有正在做種的任務',
    error: '沒有發生錯誤的任務',
  };

  return (
    <div className="rounded-lg border border-slate-700 bg-slate-800/50 py-12 text-center text-slate-400">
      <p className="text-lg">{messages[filter]}</p>
      {filter === 'all' ? (
        <p className="mt-1 text-sm">在 qBittorrent 中新增種子後會自動顯示</p>
      ) : (
        <p className="mt-1 text-sm">嘗試切換其他篩選條件</p>
      )}
    </div>
  );
}
