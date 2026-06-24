// Design ref: ux-design.pen Screen I5-D (YEqii)
/**
 * ux3-0-7: the desktop (lg+) library filter rail. A second-level sidebar inside the
 * content area (bg-primary + right hairline — distinct from the bg-secondary nav shell)
 * that hosts the shared FilterPanel in INSTANT mode (apply-on-change + URL sync, no
 * 套用/重置). Collapsible; the genre list scrolls inside the rail so the 清除全部 footer
 * stays pinned even with many genres (UX review watch-out #1). Mobile (<lg) keeps the
 * existing LibraryFilterSheetV2 bottom sheet — this rail is never rendered there.
 */
import { RotateCcw } from 'lucide-react';
import { FilterPanel } from './FilterPanel';
import { FilterRailShell } from '../ui/FilterRailShell';
import type { FilterValues } from './FilterPanel';
import type { LibraryMediaType } from '../../types/library';

interface LibraryFilterRailProps {
  filters: FilterValues;
  mediaType: LibraryMediaType;
  unmatchedCount?: number;
  /** Count of constraining facets (genres + decade + unmatched) — excludes type=全部. */
  activeCount: number;
  onApply: (filters: FilterValues) => void;
  onClear: () => void;
  onTypeChange: (type: LibraryMediaType) => void;
  onCollapse: () => void;
}

export function LibraryFilterRail({
  filters,
  mediaType,
  unmatchedCount,
  activeCount,
  onApply,
  onClear,
  onTypeChange,
  onCollapse,
}: LibraryFilterRailProps) {
  return (
    <FilterRailShell
      testId="library-filter-rail"
      activeCount={activeCount}
      activeCountTestId="library-rail-active-count"
      collapseTestId="library-rail-collapse"
      onCollapse={onCollapse}
      footer={
        activeCount > 0 ? (
          <button
            type="button"
            onClick={onClear}
            data-testid="library-rail-clear-all"
            className="inline-flex min-h-[44px] items-center gap-1.5 text-sm text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)]"
          >
            <RotateCcw className="h-3.5 w-3.5" aria-hidden="true" />
            清除全部篩選
          </button>
        ) : undefined
      }
    >
      {/* Instant mode (apply-on-change + URL sync, no 套用/重置); the genre list
          scrolls inside the rail so the clear-all footer stays pinned. */}
      <FilterPanel
        instant
        filters={filters}
        mediaType={mediaType}
        unmatchedCount={unmatchedCount}
        onApply={onApply}
        onClear={onClear}
        onTypeChange={onTypeChange}
      />
    </FilterRailShell>
  );
}
