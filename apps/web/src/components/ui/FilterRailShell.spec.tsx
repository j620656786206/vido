import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { FilterRailShell } from './FilterRailShell';

function renderRail(activeCount = 0) {
  return render(
    <FilterRailShell
      testId="test-filter-rail"
      activeCountTestId="test-rail-active-count"
      collapseTestId="test-rail-collapse"
      activeCount={activeCount}
      onCollapse={vi.fn()}
      footer={null}
    >
      <div>filters</div>
    </FilterRailShell>
  );
}

describe('FilterRailShell', () => {
  // It was a bare <aside>: a complementary landmark with no accessible name, so
  // AT users got "complementary" and nothing to tell them which one.
  it('names the rail from the heading it already renders', () => {
    renderRail();
    const rail = screen.getByTestId('test-filter-rail');
    const labelledBy = rail.getAttribute('aria-labelledby');
    expect(labelledBy).toBeTruthy();
    const heading = document.getElementById(labelledBy!);
    expect(heading).not.toBeNull();
    expect(heading).toHaveTextContent('篩選');
  });

  // The badge is a READOUT, so the rationed-accent rule wants the wash rather
  // than a solid fill — and white on solid --accent-primary measured 3.68:1 at
  // 11px, under the AA floor PRODUCT.md treats as a hard gate.
  it('renders the active count on the accent wash, not a solid accent fill', () => {
    renderRail(3);
    const badge = screen.getByTestId('test-rail-active-count');
    expect(badge).toHaveTextContent('3');
    expect(badge.className).toContain('bg-[var(--accent-subtle)]');
    expect(badge.className).toContain('text-[var(--accent-text)]');
    expect(badge.className).not.toContain('bg-[var(--accent-primary)]');
  });

  it('hides the count entirely when nothing is filtered', () => {
    renderRail(0);
    expect(screen.queryByTestId('test-rail-active-count')).toBeNull();
  });

  // dsr-8 AC #6: the page's h1 sits above the rail, so the rail title is an h2 — an
  // h3 skipped a level on both /discover and /library.
  it('titles the rail with an h2 on the design size, pinned under the 56px header', () => {
    renderRail();
    const heading = screen.getByRole('heading', { level: 2, name: '篩選' });
    expect(heading.className).toContain('text-base');
    expect(heading.className).not.toContain('text-[15px]');
    const rail = screen.getByTestId('test-filter-rail');
    // AppShellV2's header is h-14 (56px); top-16 left a permanent 8px slot.
    expect(rail.className).toContain('top-14');
    expect(rail.className).toContain('h-[calc(100vh-3.5rem)]');
  });
});
