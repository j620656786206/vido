// Implements: Component/FilterChip (jD7gF)
// Design ref: ux-design.pen Screen AS-1 Advanced Filter Chips Desktop (rsAxf)
// Source: ux-design.pen (Pencil app)
import { X } from 'lucide-react';
import { cn } from '../../lib/utils';
import { activeFilterChips, type DiscoverFilters } from '../../lib/discoverFilters';

interface FilterChipBarProps {
  filters: DiscoverFilters;
  /** Called with the next filter object when an individual chip is removed (AC #2). */
  onChange: (next: DiscoverFilters) => void;
  /** Called when "清除全部" is pressed (AC #3). */
  onClearAll: () => void;
  className?: string;
}

/**
 * Persistent, always-visible row of removable filter chips shown above the
 * content area (AC #1). Each chip removes its own filter (AC #2); when ≥2 filters
 * are active a "清除全部" button removes them all at once (AC #3, Task 2.5).
 */
export function FilterChipBar({ filters, onChange, onClearAll, className }: FilterChipBarProps) {
  const chips = activeFilterChips(filters);

  if (chips.length === 0) return null;

  return (
    <div
      className={cn('flex flex-wrap items-center gap-2', className)}
      data-testid="filter-chip-bar"
    >
      {chips.map((chip) => (
        <span
          key={chip.key}
          data-testid={`filter-chip-${chip.key}`}
          className="inline-flex items-center gap-1 rounded-full bg-[var(--accent-primary)]/20 px-3 py-1 text-sm text-blue-300"
        >
          {chip.label}
          <button
            onClick={() => onChange(chip.next)}
            className="ml-0.5 rounded-full p-0.5 hover:bg-[var(--accent-primary)]/30"
            aria-label={`移除${chip.label}篩選`}
          >
            <X className="h-3 w-3" />
          </button>
        </span>
      ))}

      {chips.length >= 2 && (
        <button
          onClick={onClearAll}
          data-testid="clear-all-filters"
          className="text-sm text-[var(--text-secondary)] underline-offset-2 hover:text-white hover:underline"
        >
          清除全部
        </button>
      )}
    </div>
  );
}
