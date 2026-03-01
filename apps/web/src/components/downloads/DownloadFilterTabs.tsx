import type { FilterStatus, DownloadCounts } from '../../services/downloadService';
import { cn } from '../../lib/utils';

interface FilterConfig {
  value: FilterStatus;
  label: string;
  icon: string;
}

const filters: FilterConfig[] = [
  { value: 'all', label: '全部', icon: '☰' },
  { value: 'downloading', label: '下載中', icon: '↓' },
  { value: 'paused', label: '已暫停', icon: '⏸' },
  { value: 'completed', label: '已完成', icon: '✓' },
  { value: 'seeding', label: '做種中', icon: '↑' },
  { value: 'error', label: '錯誤', icon: '✗' },
];

interface DownloadFilterTabsProps {
  activeFilter: FilterStatus;
  counts?: DownloadCounts;
  onFilterChange: (filter: FilterStatus) => void;
}

export function DownloadFilterTabs({
  activeFilter,
  counts,
  onFilterChange,
}: DownloadFilterTabsProps) {
  return (
    <div className="mb-4 flex flex-wrap gap-2" role="tablist" aria-label="下載狀態篩選">
      {filters.map((f) => {
        const count = counts?.[f.value] ?? 0;
        const isActive = activeFilter === f.value;

        // Hide error tab if no errors
        if (f.value === 'error' && count === 0 && !isActive) return null;

        return (
          <button
            key={f.value}
            type="button"
            role="tab"
            aria-selected={isActive}
            aria-controls="download-list"
            onClick={() => onFilterChange(f.value)}
            className={cn(
              'inline-flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-sm font-medium transition-colors',
              isActive
                ? 'border-blue-500 bg-blue-500/20 text-blue-300'
                : 'border-slate-600 bg-slate-700/50 text-slate-300 hover:border-slate-500 hover:text-slate-200',
              f.value === 'error' && count > 0 && !isActive && 'border-red-700 text-red-400'
            )}
          >
            <span>{f.icon}</span>
            <span>{f.label}</span>
            <span
              className={cn(
                'ml-0.5 inline-flex h-5 min-w-[20px] items-center justify-center rounded-full px-1.5 text-xs',
                isActive ? 'bg-blue-500/30 text-blue-200' : 'bg-slate-600/50 text-slate-400'
              )}
            >
              {count}
            </span>
          </button>
        );
      })}
    </div>
  );
}
