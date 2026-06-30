// Design ref: ux-design.pen Screen I1-D-v2 (fxCVk)
/**
 * ux3-3-2 (AC #2/#3): the desktop (lg+) Discover filter rail. A second-level
 * sidebar inside the content area (right hairline) that hosts the shared
 * search `FilterPanel` in instant mode. Shares its chrome with the 媒體庫
 * `LibraryFilterRail` via the common `FilterRailShell` (264px, 篩選 header + Mono
 * active-count badge + collapse chevron, pinned footer) — convergence, not a fork
 * (AC #11). The footer shows a SINGLE live total result count (`符合 N 部`) reused
 * from the page's own discover query — zero extra count queries (AC #3; per-facet
 * counts deferred to a backend facet-aggregation endpoint — see sprint-status
 * `ux3-discover-facet-aggregation-be`). Mobile (<lg) keeps the batch
 * `FilterBottomSheet`; this rail is never rendered there.
 */
import { RotateCcw } from 'lucide-react';
import { FilterPanel } from './FilterPanel';
import { FilterRailShell } from '../ui/FilterRailShell';
import { useDiscoverFacetCounts } from '../../hooks/useDiscoverFacetCounts';
import type { DiscoverFilters } from '../../lib/discoverFilters';

interface DiscoverFilterRailProps {
  filters: DiscoverFilters;
  /** Count of active constraining facets (genre/year/region/rating/platform). */
  activeCount: number;
  /** Single live total for the current selection (AC #3). */
  totalResults: number;
  /** True while that total is still (re)computing (any query fetching). */
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
  // Contextual per-facet counts (desktop-rail only, AC7). `counts` is undefined
  // until the first response AND on hard failure — in both cases FilterPanel
  // renders the chips count-less and this rail degrades to its single-total
  // footer below (AC6). Once counts arrive, chips fill progressively; unresolved
  // facets show the computing "–" placeholder (AC1/AC5).
  const { counts } = useDiscoverFacetCounts(filters, { enabled: true });
  return (
    <FilterRailShell
      testId="discover-filter-rail"
      activeCount={activeCount}
      activeCountTestId="discover-rail-active-count"
      collapseTestId="discover-rail-collapse"
      onCollapse={onCollapse}
      footer={
        <>
          {/* Single live total (AC #3) */}
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
        </>
      }
    >
      {/* Instant-apply, numeric inputs debounced; per-facet counts are additive
          decoration (toggling a chip still applies instantly via onChange, AC8). */}
      <FilterPanel filters={filters} onChange={onChange} debounceMs={350} facetCounts={counts} />
    </FilterRailShell>
  );
}
