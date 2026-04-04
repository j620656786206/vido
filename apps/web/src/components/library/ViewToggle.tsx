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
            ? 'bg-[var(--accent-primary)] text-white'
            : 'text-[var(--text-secondary)] hover:bg-[var(--bg-secondary)] hover:text-white'
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
            ? 'bg-[var(--accent-primary)] text-white'
            : 'text-[var(--text-secondary)] hover:bg-[var(--bg-secondary)] hover:text-white'
        )}
      >
        <List size={18} />
      </button>
    </div>
  );
}
