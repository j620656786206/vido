// Design ref: ux-design.pen Screen I1-D-v2 (fxCVk)
/**
 * ux3-3-2 (AC #2/#3): the desktop (lg+) Discover filter rail. A second-level
 * sidebar inside the content area (right hairline) that hosts the shared
 * search `FilterPanel` in instant mode — converges with the 媒體庫
 * `LibraryFilterRail` chrome (264px, 篩選 header + Mono active-count badge +
 * collapse chevron, pinned footer). The footer shows a SINGLE live total result
 * count (`符合 N 部`) reused from the page's own discover query — zero extra
 * count queries (AC #3; per-facet counts deferred to a backend facet-aggregation
 * endpoint — see sprint-status `ux3-discover-facet-aggregation-be`). Mobile (<lg)
 * keeps the batch `FilterBottomSheet`; this rail is never rendered there.
 */
import { PanelLeftClose, RotateCcw } from 'lucide-react';
import { FilterPanel } from './FilterPanel';
import type { DiscoverFilters } from '../../lib/discoverFilters';

interface DiscoverFilterRailProps {
  filters: DiscoverFilters;
  /** Count of active constraining facets (genre/year/region/rating/platform). */
  activeCount: number;
  /** Single live total for the current selection (AC #3). */
  totalResults: number;
  /** True while that total is still (re)computing. */
  isCounting: boolean;
  /** Instant-apply change handler (numeric inputs debounced inside FilterPanel). */
  onChange: (next: DiscoverFilters) => void;
  onClearAll: () => void;
  onCollapse: () => void;
}

export function DiscoverFilterRail({
  filters,
  activeCount,
  totalResults,
  isCounting,
  onChange,
  onClearAll,
  onCollapse,
}: DiscoverFilterRailProps) {
  return (
    <aside
      data-testid="discover-filter-rail"
      className="sticky top-16 flex h-[calc(100vh-4rem)] w-[264px] flex-shrink-0 flex-col border-r border-[var(--border-subtle)]"
    >
      {/* Rail header — converges with LibraryFilterRail */}
      <div className="flex items-center justify-between px-5 pb-3 pt-5">
        <div className="flex items-center gap-2">
          <h3 className="text-[15px] font-bold text-[var(--text-primary)]">篩選</h3>
          {activeCount > 0 && (
            <span
              data-testid="discover-rail-active-count"
              className="inline-flex items-center justify-center rounded-full bg-[var(--accent-primary)] px-1.5 py-0.5 font-mono text-[11px] font-medium tabular-nums text-[var(--text-on-accent)]"
            >
              {activeCount}
            </span>
          )}
        </div>
        <button
          type="button"
          onClick={onCollapse}
          data-testid="discover-rail-collapse"
          aria-label="收合篩選"
          className="flex h-11 w-11 items-center justify-center rounded-[var(--radius-md)] text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]"
        >
          <PanelLeftClose className="h-[18px] w-[18px]" aria-hidden="true" />
        </button>
      </div>

      {/* Scrollable filter body — instant-apply, numeric inputs debounced */}
      <div className="min-h-0 flex-1 overflow-y-auto px-5">
        <FilterPanel filters={filters} onChange={onChange} debounceMs={350} />
      </div>

      {/* Pinned footer — single live total (AC #3) + clear-all */}
      <div className="border-t border-[var(--border-subtle)] px-5 py-3">
        <p
          data-testid="discover-rail-count"
          className="mb-2 font-mono text-xs tabular-nums text-[var(--text-secondary)]"
          aria-live="polite"
        >
          {isCounting ? '計算中…' : `符合 ${totalResults.toLocaleString()} 部`}
        </p>
        {activeCount > 0 && (
          <button
            type="button"
            onClick={onClearAll}
            data-testid="discover-rail-clear-all"
            className="inline-flex min-h-[44px] items-center gap-1.5 text-sm text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)]"
          >
            <RotateCcw className="h-3.5 w-3.5" aria-hidden="true" />
            清除全部篩選
          </button>
        )}
      </div>
    </aside>
  );
}
