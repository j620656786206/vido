import { cn } from '../../lib/utils';

export type MediaTypeFilter = 'all' | 'movie' | 'tv';

interface MediaTypeTabsProps {
  activeType: MediaTypeFilter;
  onTypeChange: (type: MediaTypeFilter) => void;
  movieCount?: number;
  tvCount?: number;
  className?: string;
}

interface TabConfig {
  type: MediaTypeFilter;
  label: string;
  count?: number;
}

export function MediaTypeTabs({
  activeType,
  onTypeChange,
  movieCount,
  tvCount,
  className,
}: MediaTypeTabsProps) {
  const totalCount =
    (movieCount !== undefined ? movieCount : 0) + (tvCount !== undefined ? tvCount : 0);

  const tabs: TabConfig[] = [
    { type: 'all', label: '全部', count: totalCount || undefined },
    { type: 'movie', label: '電影', count: movieCount },
    { type: 'tv', label: '影集', count: tvCount },
  ];

  return (
    <div className={cn('flex space-x-2', className)} role="tablist" aria-label="媒體類型篩選">
      {tabs.map((tab) => {
        const isActive = activeType === tab.type;
        return (
          <button
            key={tab.type}
            onClick={() => onTypeChange(tab.type)}
            role="tab"
            aria-selected={isActive}
            aria-controls={`tabpanel-${tab.type}`}
            className={cn(
              'px-4 py-2 rounded-lg font-medium transition-colors',
              'focus:outline-none focus:ring-2 focus:ring-blue-500',
              isActive
                ? 'bg-blue-600 text-white'
                : 'bg-slate-800 text-slate-400 hover:bg-slate-700 hover:text-white'
            )}
          >
            {tab.label}
            {tab.count !== undefined && (
              <span
                className={cn(
                  'ml-2 px-2 py-0.5 text-xs rounded-full',
                  isActive ? 'bg-blue-500 text-white' : 'bg-slate-700 text-slate-300'
                )}
              >
                {tab.count}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}
