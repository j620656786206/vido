// Implements: Component/MergedSortFilterSheet (Bz0YN)
/**
 * Mobile merged sort + filter sheet (UX Redesign Phase 2 — UX2-2, AC #7, `Bz0YN`).
 * Replaces the separate mobile sort/filter sheets with one Base UI Sheet (the
 * UX2-1 Dialog-backed wrapper). Hosts the existing SortSelector + FilterPanel so
 * the controls stay a single source of truth across desktop and mobile.
 */
import { Sheet } from '../ui/Sheet';
import { SortSelector } from './SortSelector';
import { FilterPanel } from './FilterPanel';
import type { FilterValues } from './FilterPanel';
import type { SortField, SortOrder, LibraryMediaType } from '../../types/library';

interface LibraryFilterSheetV2Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  sortBy: SortField;
  sortOrder: SortOrder;
  onSortChange: (field: SortField, order: SortOrder) => void;
  filters: FilterValues;
  mediaType: LibraryMediaType;
  unmatchedCount?: number;
  onApply: (filters: FilterValues) => void;
  onClear: () => void;
  onTypeChange: (type: LibraryMediaType) => void;
}

export function LibraryFilterSheetV2({
  open,
  onOpenChange,
  sortBy,
  sortOrder,
  onSortChange,
  filters,
  mediaType,
  unmatchedCount,
  onApply,
  onClear,
  onTypeChange,
}: LibraryFilterSheetV2Props) {
  return (
    <Sheet open={open} onOpenChange={onOpenChange} title="排序與篩選">
      <div className="space-y-5">
        <section>
          <h4 className="mb-2 text-xs font-semibold uppercase tracking-wider text-[var(--text-muted)]">
            排序
          </h4>
          <SortSelector sortBy={sortBy} sortOrder={sortOrder} onSortChange={onSortChange} />
        </section>
        <section>
          <h4 className="mb-2 text-xs font-semibold uppercase tracking-wider text-[var(--text-muted)]">
            篩選
          </h4>
          <FilterPanel
            filters={filters}
            mediaType={mediaType}
            unmatchedCount={unmatchedCount}
            onApply={(f) => {
              onApply(f);
              onOpenChange(false);
            }}
            onClear={onClear}
            onTypeChange={onTypeChange}
          />
        </section>
      </div>
    </Sheet>
  );
}
