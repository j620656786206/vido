// Implements: Component/FilterChip (jD7gF)
// Design ref: ux-design.pen Screen AS-1 Advanced Filter Chips Desktop (rsAxf)
// Source: ux-design.pen (Pencil app)
import { Plus, X } from 'lucide-react';
import { cn } from '../../lib/utils';
import { activeFilterChips, type DiscoverFilters } from '../../lib/discoverFilters';

interface FilterChipBarProps {
  filters: DiscoverFilters;
  /** Called with the next filter object when an individual chip is removed (AC #2). */
  onChange: (next: DiscoverFilters) => void;
  /** Called when "清除全部" is pressed (AC #3). */
  onClearAll: () => void;
  /**
   * Called when "儲存篩選" is pressed (Story 11-4 AC #1). When omitted, the save
   * button is not rendered. Only shown while ≥1 filter is active (i.e. chips exist).
   */
  onSavePreset?: () => void;
  /**
   * ux3-3-2 AC #7: when the v2 rail is the primary editor, render the chip bar as a
   * lighter read/remove SUMMARY (muted outline chips) so it does not read as a second
   * competing editor. Default (`false`) keeps the legacy accent-filled chips.
   */
  summary?: boolean;
  className?: string;
}

/**
 * Persistent, always-visible row of removable filter chips shown above the
 * content area (AC #1). Each chip removes its own filter (AC #2); when ≥2 filters
 * are active a "清除全部" button removes them all at once (AC #3, Task 2.5).
 */
export function FilterChipBar({
  filters,
  onChange,
  onClearAll,
  onSavePreset,
  summary = false,
  className,
}: FilterChipBarProps) {
  const chips = activeFilterChips(filters);

  if (chips.length === 0) return null;

  // AC #7: summary variant uses muted outline chips (read/remove), the default
  // keeps the legacy accent-filled chips byte-unchanged.
  const chipClass = summary
    ? 'inline-flex items-center gap-1 rounded-full border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3 py-1 text-sm text-[var(--text-secondary)]'
    : 'inline-flex items-center gap-1 rounded-full bg-[var(--accent-primary)]/20 px-3 py-1 text-sm text-blue-300';
  const removeClass = summary
    ? 'ml-0.5 rounded-full p-0.5 hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]'
    : 'ml-0.5 rounded-full p-0.5 hover:bg-[var(--accent-primary)]/30';

  return (
    <div
      className={cn('flex flex-wrap items-center gap-2', className)}
      data-testid="filter-chip-bar"
    >
      {chips.map((chip) => (
        <span key={chip.key} data-testid={`filter-chip-${chip.key}`} className={chipClass}>
          {chip.label}
          <button
            onClick={() => onChange(chip.next)}
            className={removeClass}
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

      {/* "儲存篩選" — save the active filter combination as a preset (Story 11-4 AC #1). */}
      {onSavePreset && (
        <button
          onClick={onSavePreset}
          data-testid="save-preset-button"
          aria-label="儲存篩選"
          className="inline-flex items-center gap-1 rounded-full border border-[var(--border-subtle)] px-3 py-1 text-xs text-[var(--text-muted)] hover:border-[var(--accent-primary)] hover:text-white"
        >
          <Plus className="h-3 w-3" />
          儲存篩選
        </button>
      )}
    </div>
  );
}
