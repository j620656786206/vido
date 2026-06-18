// Design ref: ux-design.pen Screen I5-D (YEqii)
/**
 * ux3-0-7: the desktop (lg+) library filter rail. A second-level sidebar inside the
 * content area (bg-primary + right hairline — distinct from the bg-secondary nav shell)
 * that hosts the shared FilterPanel in INSTANT mode (apply-on-change + URL sync, no
 * 套用/重置). Collapsible; the genre list scrolls inside the rail so the 清除全部 footer
 * stays pinned even with many genres (UX review watch-out #1). Mobile (<lg) keeps the
 * existing LibraryFilterSheetV2 bottom sheet — this rail is never rendered there.
 */
import { PanelLeftClose, RotateCcw } from 'lucide-react';
import { FilterPanel } from './FilterPanel';
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
    <aside
      data-testid="library-filter-rail"
      className="sticky top-16 flex h-[calc(100vh-5rem)] w-[264px] flex-shrink-0 flex-col border-r border-[var(--border-subtle)]"
    >
      {/* Rail header */}
      <div className="flex items-center justify-between px-5 pb-3 pt-5">
        <div className="flex items-center gap-2">
          <h3 className="text-[15px] font-bold text-[var(--text-primary)]">篩選</h3>
          {activeCount > 0 && (
            <span
              data-testid="library-rail-active-count"
              className="inline-flex items-center justify-center rounded-full bg-[var(--accent-primary)] px-1.5 py-0.5 font-mono text-[11px] font-medium tabular-nums text-[var(--text-on-accent)]"
            >
              {activeCount}
            </span>
          )}
        </div>
        <button
          type="button"
          onClick={onCollapse}
          data-testid="library-rail-collapse"
          aria-label="收合篩選"
          className="flex h-11 w-11 items-center justify-center rounded-[var(--radius-md)] text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]"
        >
          <PanelLeftClose className="h-[18px] w-[18px]" aria-hidden="true" />
        </button>
      </div>

      {/* Scrollable filter body — keeps the clear-all footer pinned for long genre lists */}
      <div className="min-h-0 flex-1 overflow-y-auto px-5">
        <FilterPanel
          instant
          filters={filters}
          mediaType={mediaType}
          unmatchedCount={unmatchedCount}
          onApply={onApply}
          onClear={onClear}
          onTypeChange={onTypeChange}
        />
      </div>

      {/* Pinned clear-all footer */}
      {activeCount > 0 && (
        <div className="border-t border-[var(--border-subtle)] px-5 py-3">
          <button
            type="button"
            onClick={onClear}
            data-testid="library-rail-clear-all"
            className="inline-flex min-h-[44px] items-center gap-1.5 text-sm text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)]"
          >
            <RotateCcw className="h-3.5 w-3.5" aria-hidden="true" />
            清除全部篩選
          </button>
        </div>
      )}
    </aside>
  );
}
