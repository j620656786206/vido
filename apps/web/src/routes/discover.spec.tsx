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

// Stub the v2 experience so the gate test asserts which composition the route
// picks without rendering the full v2 tree (covered by DiscoverBrowseV2.spec).
vi.mock('../components/search/DiscoverBrowseV2', () => ({
  DiscoverBrowseV2: () => React.createElement('div', { 'data-testid': 'stub-discover-v2' }),
}));
// Keep the legacy branch render network-free.
vi.mock('../hooks/useDiscoverResults', () => ({
  useDiscoverResults: () => ({
    moviesQuery: { data: undefined },
    tvQuery: { data: undefined },
    isLoading: false,
    totalResults: 0,
  }),
}));
vi.mock('../hooks/useFilterPresets', () => ({
  useFilterPresets: () => ({ data: [] }),
  useDeleteFilterPreset: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

import { Route as DiscoverRoute } from './discover';
import { ShellVersionProvider } from '../components/shell/shellVersion';

function renderDiscover(shell: 'legacy' | 'v2' = 'legacy') {
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
  return render(
    shell === 'v2' ? React.createElement(ShellVersionProvider, { value: 'v2' }, tree) : tree
  );
}

describe('routes/discover (shell-version gate — ux3-3-2 AC #1)', () => {
  beforeEach(() => vi.clearAllMocks());

  it('v2 shell renders the persistent-rail Discover v2', async () => {
    renderDiscover('v2');
    expect(await screen.findByTestId('stub-discover-v2')).toBeInTheDocument();
    // The legacy sidebar layout is gone under the v2 shell.
    expect(screen.queryByTestId('filter-sidebar')).toBeNull();
  });

  it('legacy shell (default) renders the byte-unchanged legacy discover, not v2', async () => {
    renderDiscover('legacy');
    expect(await screen.findByTestId('filter-sidebar')).toBeInTheDocument();
    expect(screen.queryByTestId('stub-discover-v2')).toBeNull();
  });
});
