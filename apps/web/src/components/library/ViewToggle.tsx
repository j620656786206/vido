import { LayoutGrid, List } from 'lucide-react';
import { cn } from '../../lib/utils';

export type ViewMode = 'grid' | 'list';

interface ViewToggleProps {
  view: ViewMode;
  onViewChange: (view: ViewMode) => void;
}

export function ViewToggle({ view, onViewChange }: ViewToggleProps) {
  return (
    <div
      role="radiogroup"
      aria-label="切換檢視模式"
      className="flex gap-1"
      data-testid="view-toggle"
    >
      <button
        role="radio"
        aria-checked={view === 'grid'}
        aria-label="格狀檢視"
        onClick={() => onViewChange('grid')}
        className={cn(
          'rounded p-2 transition-colors',
          view === 'grid'
            ? 'bg-blue-600 text-white'
            : 'text-slate-400 hover:bg-slate-800 hover:text-white'
        )}
      >
        <LayoutGrid size={18} />
      </button>
      <button
        role="radio"
        aria-checked={view === 'list'}
        aria-label="列表檢視"
        onClick={() => onViewChange('list')}
        className={cn(
          'rounded p-2 transition-colors',
          view === 'list'
            ? 'bg-blue-600 text-white'
            : 'text-slate-400 hover:bg-slate-800 hover:text-white'
        )}
      >
        <List size={18} />
      </button>
    </div>
  );
}
