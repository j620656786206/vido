import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  createMemoryHistory,
  RouterProvider,
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
} from '@tanstack/react-router';
import React from 'react';

// Stub the v2 experience so the route test asserts composition without
// rendering the full v2 tree (covered by DiscoverBrowseV2.spec).
vi.mock('../components/search/DiscoverBrowseV2', () => ({
  DiscoverBrowseV2: () => React.createElement('div', { 'data-testid': 'stub-discover-v2' }),
}));

import { Route as DiscoverRoute } from './discover';

function renderDiscover() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const rootRoute = createRootRoute({
    component: () => React.createElement(Outlet),
  });
  const discoverRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/discover',
    validateSearch: DiscoverRoute.options.validateSearch,
    component: DiscoverRoute.options.component,
  });
  const router = createRouter({
    routeTree: rootRoute.addChildren([discoverRoute]),
    history: createMemoryHistory({ initialEntries: ['/discover'] }),
  });
  const tree = React.createElement(
    QueryClientProvider,
    { client: queryClient },
    React.createElement(RouterProvider, { router } as never)
  );
  return render(tree);
}

// ux3-cutover-3: the shell gate is gone — DiscoverBrowseV2 is the route's only render.
describe('routes/discover (ux3-cutover-3)', () => {
  beforeEach(() => vi.clearAllMocks());

  it('renders Discover v2 unconditionally', async () => {
    renderDiscover();
    expect(await screen.findByTestId('stub-discover-v2')).toBeInTheDocument();
    expect(screen.queryByTestId('filter-sidebar')).toBeNull();
  });
});
