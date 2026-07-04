import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  createRootRoute,
  createRoute,
  createRouter,
  createMemoryHistory,
  RouterProvider,
} from '@tanstack/react-router';
import type { DiscoverFilters } from '../../lib/discoverFilters';

const h = vi.hoisted(() => ({
  discover: {} as Record<string, unknown>,
  filterState: {} as Record<string, unknown>,
}));

vi.mock('../../hooks/useDiscoverResults', () => ({
  useDiscoverResults: () => h.discover,
}));
// The rail fetches per-facet counts (ux3-discover-facet-aggregation-fe); mock it so
// these tests stay focused on the browse layout (count behaviour is covered by
// useDiscoverFacetCounts.spec / FilterPanel.spec / the discover-filters E2E).
vi.mock('../../hooks/useDiscoverFacetCounts', () => ({
  useDiscoverFacetCounts: () => ({
    counts: undefined,
    partial: false,
    isLoading: false,
    isFetching: false,
  }),
}));
vi.mock('../../hooks/useFilterState', () => ({
  useFilterState: () => h.filterState,
}));
vi.mock('../../hooks/useFilterPresets', () => ({
  useFilterPresets: () => ({ data: [] }),
  useDeleteFilterPreset: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));
vi.mock('../media/MediaGrid', () => ({
  MediaGrid: ({ items }: { items?: unknown[] }) => (
    <div data-testid="media-grid">{items?.length ?? 0} items</div>
  ),
}));
// Story 13-1b: ownership drives the per-card 想要 affordance; stub it so these
// tests stay layout-focused (real behaviour covered by useRequestedMedia.spec +
// RequestButton.spec).
vi.mock('../../hooks/useOwnedMedia', () => ({
  useOwnedMedia: () => ({
    owned: new Set<number>(),
    isOwned: () => false,
    isRequested: () => false,
    isLoading: false,
    error: null,
  }),
}));
vi.mock('../requests/RequestsView', () => ({
  RequestsView: ({ onExplore }: { onExplore: () => void }) => (
    <div data-testid="requests-view-stub">
      <button type="button" data-testid="requests-stub-explore" onClick={onExplore} />
    </div>
  ),
}));

import { DiscoverBrowseV2 } from './DiscoverBrowseV2';

const base: DiscoverFilters = { genre: [], platform: [], sortBy: 'popularity' };

function query(over: Record<string, unknown> = {}) {
  return { data: undefined, isError: false, error: null, refetch: vi.fn(), ...over };
}
function discover(over: Record<string, unknown> = {}) {
  return {
    moviesQuery: query(),
    tvQuery: query(),
    isLoading: false,
    isFetching: false,
    totalResults: 0,
    ...over,
  };
}
function filterState(filters: DiscoverFilters = base, over: Record<string, unknown> = {}) {
  return { filters, setFilters: vi.fn(), clearAll: vi.fn(), ...over };
}

function renderBrowse(initial = '/discover') {
  const rootRoute = createRootRoute();
  const discoverRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/discover',
    validateSearch: (s: Record<string, unknown>) => s,
    component: DiscoverBrowseV2,
  });
  const router = createRouter({
    routeTree: rootRoute.addChildren([discoverRoute]),
    history: createMemoryHistory({ initialEntries: [initial] }),
  });
  return render(<RouterProvider router={router} />);
}

describe('DiscoverBrowseV2', () => {
  beforeEach(() => {
    h.discover = discover();
    h.filterState = filterState();
  });

  it('renders the persistent rail, the LIVE Requests entry (13-1b), and the grid with results', async () => {
    h.discover = discover({
      moviesQuery: query({
        data: {
          results: [
            { id: 1, voteCount: 10 },
            { id: 2, voteCount: 5 },
          ],
          totalResults: 2,
          totalPages: 1,
        },
      }),
      totalResults: 2,
    });
    renderBrowse();
    expect(await screen.findByTestId('discover-filter-rail')).toBeInTheDocument();
    // Story 13-1b lit the PH3-R2 reserved entry — enabled, no 即將推出.
    const entry = screen.getByTestId('discover-requests-entry');
    expect(entry).toBeEnabled();
    expect(entry).toHaveTextContent('想要清單');
    expect(entry).not.toHaveTextContent('即將推出');
    expect(screen.getByTestId('media-grid')).toHaveTextContent('2 items');
  });

  it('deep link ?view=requests renders the 想要清單 view in place of the results (13-1b AC #5)', async () => {
    renderBrowse('/discover?view=requests');
    expect(await screen.findByTestId('requests-view-stub')).toBeInTheDocument();
    expect(screen.queryByTestId('media-grid')).not.toBeInTheDocument();
    // The entry reflects the active view.
    expect(screen.getByTestId('discover-requests-entry')).toHaveAttribute('aria-pressed', 'true');
  });

  it('renders the chip bar as a lighter read/remove summary (AC #7)', async () => {
    h.filterState = filterState({ genre: [28], platform: [], sortBy: 'popularity' });
    renderBrowse();
    const chip = await screen.findByTestId('filter-chip-genre-28');
    // summary variant uses muted outline styling, not the accent-filled legacy chip.
    expect(chip.className).toContain('text-[var(--text-secondary)]');
    expect(chip.className).not.toContain('text-blue-300');
  });

  it('shows the v2 loading skeleton while results load (AC #8 / I6)', async () => {
    h.discover = discover({ isLoading: true });
    renderBrowse();
    expect(await screen.findByTestId('discover-grid-skeleton')).toBeInTheDocument();
    expect(screen.queryByTestId('media-grid')).toBeNull();
  });

  it('shows the no-result state with an active-filter echo (AC #8 / I7)', async () => {
    h.filterState = filterState({ genre: [28], platform: [], sortBy: 'popularity' });
    h.discover = discover({
      moviesQuery: query({ data: { results: [], totalResults: 0, totalPages: 1 } }),
      tvQuery: query({ data: { results: [], totalResults: 0, totalPages: 1 } }),
      totalResults: 0,
    });
    renderBrowse();
    expect(await screen.findByTestId('discover-no-result')).toBeInTheDocument();
    expect(screen.getByTestId('discover-no-result-echo')).toHaveTextContent('動作');
  });

  it('per-section fail-soft: one section errors inline, the other still renders (AC #8 / I8)', async () => {
    h.discover = discover({
      moviesQuery: query({ isError: true, error: { code: 'TMDB_TIMEOUT' } }),
      tvQuery: query({
        data: { results: [{ id: 9, voteCount: 1 }], totalResults: 1, totalPages: 1 },
      }),
      totalResults: 1,
    });
    renderBrowse();
    expect(await screen.findByTestId('discover-section-error')).toHaveTextContent(
      '電影結果暫時無法載入'
    );
    // The healthy section still renders its grid — page never hard-fails.
    expect(screen.getByTestId('media-grid')).toHaveTextContent('1 items');
  });

  it('all sections failing shows a fail-soft banner and no grid (page never hard-fails)', async () => {
    h.discover = discover({
      moviesQuery: query({ isError: true, error: { code: 'TMDB_TIMEOUT' } }),
      tvQuery: query({ isError: true, error: { code: 'TMDB_TIMEOUT' } }),
    });
    renderBrowse();
    expect(await screen.findByTestId('discover-section-error')).toHaveTextContent(
      'TMDB 服務暫時無法連線'
    );
    expect(screen.queryByTestId('media-grid')).toBeNull();
    expect(screen.queryByTestId('discover-no-result')).toBeNull();
    // The rail still renders — the page is intact.
    expect(screen.getByTestId('discover-filter-rail')).toBeInTheDocument();
  });

  it('rail shows 計算中… during a background refetch (isFetching) while prior results stay visible (AC #3)', async () => {
    // keepPreviousData: a re-filter keeps isLoading=false (only cold loads flip it)
    // but isFetching=true — the rail count must reflect isFetching, not isLoading,
    // so the stale total does not silently linger without a counting indicator.
    h.discover = discover({
      isLoading: false,
      isFetching: true,
      moviesQuery: query({
        data: { results: [{ id: 1, voteCount: 10 }], totalResults: 1, totalPages: 1 },
      }),
      totalResults: 1,
    });
    renderBrowse();
    expect(await screen.findByTestId('discover-rail-count')).toHaveTextContent('計算中…');
    // The prior grid stays visible — no skeleton flash on a re-filter.
    expect(screen.getByTestId('media-grid')).toHaveTextContent('1 items');
    expect(screen.queryByTestId('discover-grid-skeleton')).toBeNull();
  });
});
