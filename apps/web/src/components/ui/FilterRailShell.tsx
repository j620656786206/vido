// Design ref: ux-design.pen Screen I1-D-v2 (fxCVk)
// ux3-3-2 (AC #11): the shared persistent-filter-rail chrome. Extracted so the
// 探索 DiscoverFilterRail and the 媒體庫 LibraryFilterRail CONVERGE on one rail
// shell (264px, $bg-primary, right hairline, 篩選 header + Mono active-count badge
// + collapse chevron, scrollable body, pinned footer) instead of forking a copy.
// The body (a FilterPanel) and the footer content stay rail-specific via slots.
import type { ReactNode } from 'react';
import { PanelLeftClose } from 'lucide-react';

interface FilterRailShellProps {
  /** testid for the <aside> root (e.g. `discover-filter-rail`). */
  testId: string;
  /** Count of active constraining facets — drives the Mono badge. */
  activeCount: number;
  /** testid for the active-count badge (e.g. `discover-rail-active-count`). */
  activeCountTestId: string;
  /** testid for the collapse button (e.g. `discover-rail-collapse`). */
  collapseTestId: string;
  onCollapse: () => void;
  /** Scrollable filter body — the FilterPanel. */
  children: ReactNode;
  /**
   * Pinned footer content (rail-specific: a live total and/or clear-all). When
   * omitted no footer (and no top border) renders, matching the library rail's
   * "footer only while filters are active" behavior.
   */
  footer?: ReactNode;
}

export function FilterRailShell({
  testId,
  activeCount,
  activeCountTestId,
  collapseTestId,
  onCollapse,
  children,
  footer,
}: FilterRailShellProps) {
  return (
    <aside
      data-testid={testId}
      className="sticky top-16 flex h-[calc(100vh-4rem)] w-[264px] flex-shrink-0 flex-col border-r border-[var(--border-subtle)]"
    >
      {/* Rail header */}
      <div className="flex items-center justify-between px-5 pb-3 pt-5">
        <div className="flex items-center gap-2">
          <h3 className="text-[15px] font-bold text-[var(--text-primary)]">篩選</h3>
          {activeCount > 0 && (
            <span
              data-testid={activeCountTestId}
              className="inline-flex items-center justify-center rounded-full bg-[var(--accent-primary)] px-1.5 py-0.5 font-mono text-[11px] font-medium tabular-nums text-[var(--text-on-accent)]"
            >
              {activeCount}
            </span>
          )}
        </div>
        <button
          type="button"
          onClick={onCollapse}
          data-testid={collapseTestId}
          aria-label="收合篩選"
          className="flex h-11 w-11 items-center justify-center rounded-[var(--radius-md)] text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]"
        >
          <PanelLeftClose className="h-[18px] w-[18px]" aria-hidden="true" />
        </button>
      </div>

      {/* Scrollable filter body — keeps the footer pinned for long control lists */}
      <div className="min-h-0 flex-1 overflow-y-auto px-5">{children}</div>

      {/* Pinned footer */}
      {footer && <div className="border-t border-[var(--border-subtle)] px-5 py-3">{footer}</div>}
    </aside>
  );
}
