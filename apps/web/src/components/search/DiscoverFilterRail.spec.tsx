import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { DiscoverFilterRail } from './DiscoverFilterRail';
import type { DiscoverFilters } from '../../lib/discoverFilters';

// The rail fetches per-facet counts (ux3-discover-facet-aggregation-fe); mock it so
// this spec stays focused on the rail chrome and needs no QueryClient/shell provider
// (count behaviour is covered by useDiscoverFacetCounts.spec / FilterPanel.spec / E2E).
vi.mock('../../hooks/useDiscoverFacetCounts', () => ({
  useDiscoverFacetCounts: () => ({
    counts: undefined,
    partial: false,
    isLoading: false,
    isFetching: false,
  }),
}));

const base: DiscoverFilters = { genre: [], platform: [], sortBy: 'popularity' };

function renderRail(over: Partial<React.ComponentProps<typeof DiscoverFilterRail>> = {}) {
  const props = {
    filters: base,
    activeCount: 0,
    totalResults: 0,
    isCounting: false,
    onChange: vi.fn(),
    onClearAll: vi.fn(),
    onCollapse: vi.fn(),
    ...over,
  };
  render(<DiscoverFilterRail {...props} />);
  return props;
}

describe('DiscoverFilterRail', () => {
  it('renders the rail chrome hosting the shared FilterPanel', () => {
    renderRail();
    expect(screen.getByTestId('discover-filter-rail')).toBeInTheDocument();
    expect(screen.getByTestId('filter-panel')).toBeInTheDocument();
    expect(screen.getByText('篩選')).toBeInTheDocument();
  });

  it('hides the active-count badge and clear-all when no filters are active', () => {
    renderRail({ activeCount: 0 });
    expect(screen.queryByTestId('discover-rail-active-count')).toBeNull();
    expect(screen.queryByTestId('discover-rail-clear-all')).toBeNull();
  });

  it('shows the single live total when counted (AC #3)', () => {
    renderRail({ totalResults: 412 });
    expect(screen.getByTestId('discover-rail-count')).toHaveTextContent('符合 412 部');
  });

  // dsr-8 AC #4: when the results query failed there is no number to show.
  it('says 暫時無法計算 instead of a count it could not compute', () => {
    renderRail({ totalResults: 0, countUnavailable: true });
    const count = screen.getByTestId('discover-rail-count');
    expect(count).toHaveTextContent('暫時無法計算');
    expect(count).not.toHaveTextContent('符合');
  });

  it('計算中… still wins while a retry is in flight', () => {
    renderRail({ countUnavailable: true, isCounting: true });
    expect(screen.getByTestId('discover-rail-count')).toHaveTextContent('計算中…');
  });

  it('shows a counting placeholder while the total computes', () => {
    renderRail({ isCounting: true });
    expect(screen.getByTestId('discover-rail-count')).toHaveTextContent('計算中…');
  });

  it('renders the Mono badge + wires collapse and clear-all when active', () => {
    const { onCollapse, onClearAll } = renderRail({
      activeCount: 2,
      filters: { ...base, genre: [28, 35] },
    });
    expect(screen.getByTestId('discover-rail-active-count')).toHaveTextContent('2');
    fireEvent.click(screen.getByTestId('discover-rail-collapse'));
    expect(onCollapse).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByTestId('discover-rail-clear-all'));
    expect(onClearAll).toHaveBeenCalledTimes(1);
  });

  it('keeps categorical chip toggles instant (AC #4)', () => {
    const { onChange } = renderRail();
    fireEvent.click(screen.getByTestId('filter-genre-16'));
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ genre: [16] }));
  });

  // dsr-8 AC #6: h1 探索 → h2 篩選 → h3 sections. With the rail title promoted to h2,
  // h4 sections would just move the skipped level down one.
  it('uses a gapless heading ladder: rail h2, sections h3, no h4', () => {
    renderRail();
    expect(screen.getByRole('heading', { level: 2, name: '篩選' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 3, name: '類型' })).toBeInTheDocument();
    expect(screen.queryAllByRole('heading', { level: 4 })).toHaveLength(0);
  });
});
