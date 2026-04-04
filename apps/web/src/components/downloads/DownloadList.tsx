import { useState } from 'react';
import type { Download, SortField, SortOrder } from '../../services/downloadService';
import { DownloadItem } from './DownloadItem';
import { DownloadDetails } from './DownloadDetails';

interface DownloadListProps {
  downloads: Download[];
  totalItems?: number;
  sortField: SortField;
  sortOrder: SortOrder;
  onSortChange: (field: SortField) => void;
  onOrderChange: (order: SortOrder) => void;
}

const sortOptions: { value: SortField; label: string }[] = [
  { value: 'added_on', label: '新增時間' },
  { value: 'name', label: '名稱' },
  { value: 'progress', label: '進度' },
  { value: 'status', label: '狀態' },
];

export function DownloadList({
  downloads,
  totalItems,
  sortField,
  sortOrder,
  onSortChange,
  onOrderChange,
}: DownloadListProps) {
  const [expandedHash, setExpandedHash] = useState<string | null>(null);

  const handleToggleExpand = (hash: string) => {
    setExpandedHash((prev) => (prev === hash ? null : hash));
  };

  return (
    <div>
      {/* Sort Controls */}
      <div className="mb-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <label htmlFor="sort-select" className="text-sm text-[var(--text-secondary)]">
            排序：
          </label>
          <select
            id="sort-select"
            value={sortField}
            onChange={(e) => onSortChange(e.target.value as SortField)}
            className="rounded-md border border-[var(--border-subtle)]/50 bg-[var(--bg-secondary)] px-3 py-1.5 text-sm text-[var(--text-primary)] focus:border-[var(--accent-primary)] focus:outline-none"
          >
            {sortOptions.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
          <button
            type="button"
            onClick={() => onOrderChange(sortOrder === 'asc' ? 'desc' : 'asc')}
            className="rounded-md border border-[var(--border-subtle)]/50 bg-[var(--bg-secondary)] p-1.5 text-[var(--text-secondary)] hover:border-[var(--text-muted)]"
            title={sortOrder === 'asc' ? '升冪' : '降冪'}
          >
            {sortOrder === 'asc' ? '↑' : '↓'}
          </button>
        </div>
        <span className="text-sm text-[var(--text-secondary)]">
          {(totalItems ?? downloads.length).toLocaleString()} 個項目
        </span>
      </div>

      {/* Download Items */}
      {downloads.length === 0 ? (
        <div className="rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-secondary)]/50 py-12 text-center text-[var(--text-secondary)]">
          <p className="text-lg">目前沒有下載任務</p>
          <p className="mt-1 text-sm">在 qBittorrent 中新增種子後會自動顯示</p>
        </div>
      ) : (
        <div className="space-y-2">
          {downloads.map((download) => (
            <div key={download.hash}>
              <DownloadItem
                download={download}
                expanded={expandedHash === download.hash}
                onToggleExpand={() => handleToggleExpand(download.hash)}
              />
              {expandedHash === download.hash && (
                <div className="mt-px rounded-b-lg border border-t-0 border-[var(--border-subtle)] bg-[var(--bg-secondary)]/30 px-4 pb-4">
                  <DownloadDetails hash={download.hash} />
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
