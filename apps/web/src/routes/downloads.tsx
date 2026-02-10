import { createFileRoute } from '@tanstack/react-router';
import { useState } from 'react';
import { useDownloads } from '../hooks/useDownloads';
import { DownloadList } from '../components/downloads/DownloadList';
import type { SortField, SortOrder } from '../services/downloadService';

export const Route = createFileRoute('/downloads')({
  component: DownloadsPage,
});

function DownloadsPage() {
  const [sortField, setSortField] = useState<SortField>('added_on');
  const [sortOrder, setSortOrder] = useState<SortOrder>('desc');

  const { data: downloads, isLoading, error } = useDownloads(sortField, sortOrder);

  return (
    <div className="mx-auto max-w-4xl px-4 py-8">
      <h1 className="mb-2 text-2xl font-bold text-slate-100">下載管理</h1>
      <p className="mb-6 text-sm text-slate-400">
        即時監控 qBittorrent 下載狀態，每 5 秒自動更新。
      </p>

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
        <DownloadList
          downloads={downloads}
          sortField={sortField}
          sortOrder={sortOrder}
          onSortChange={setSortField}
          onOrderChange={setSortOrder}
        />
      )}
    </div>
  );
}
