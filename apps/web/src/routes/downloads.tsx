import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useState } from 'react';
import { useDownloads, useDownloadCounts } from '../hooks/useDownloads';
import { DownloadList } from '../components/downloads/DownloadList';
import { DownloadFilterTabs } from '../components/downloads/DownloadFilterTabs';
import type { FilterStatus, SortField, SortOrder } from '../services/downloadService';

interface DownloadsSearch {
  filter?: FilterStatus;
}

export const Route = createFileRoute('/downloads')({
  validateSearch: (search: Record<string, unknown>): DownloadsSearch => {
    const validFilters = ['all', 'downloading', 'paused', 'completed', 'seeding', 'error'];
    const filter = validFilters.includes(search.filter as string)
      ? (search.filter as FilterStatus)
      : undefined;
    return { filter };
  },
  component: DownloadsPage,
});

function DownloadsPage() {
  const { filter: urlFilter } = Route.useSearch();
  const navigate = useNavigate();
  const activeFilter: FilterStatus = urlFilter || 'all';

  const [sortField, setSortField] = useState<SortField>('added_on');
  const [sortOrder, setSortOrder] = useState<SortOrder>('desc');

  const { data: downloads, isLoading, error } = useDownloads(activeFilter, sortField, sortOrder);
  const { data: counts } = useDownloadCounts();

  const handleFilterChange = (newFilter: FilterStatus) => {
    navigate({
      search: { filter: newFilter === 'all' ? undefined : newFilter },
      replace: true,
    });
  };

  return (
    <div className="mx-auto max-w-4xl px-4 py-8">
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

      {!isLoading && !error && downloads && (
        <div id="download-list" role="tabpanel">
          {downloads.length === 0 ? (
            <EmptyFilterState filter={activeFilter} />
          ) : (
            <DownloadList
              downloads={downloads}
              sortField={sortField}
              sortOrder={sortOrder}
              onSortChange={setSortField}
              onOrderChange={setSortOrder}
            />
          )}
        </div>
      )}
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
