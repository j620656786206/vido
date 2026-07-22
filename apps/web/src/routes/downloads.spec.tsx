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

// Stub the v2 experience so the route test asserts composition without rendering
// the full v2 tree (covered by the DownloadsBrowseV2 children specs).
vi.mock('../components/downloads/DownloadsBrowseV2', () => ({
  DownloadsBrowseV2: () => React.createElement('div', { 'data-testid': 'stub-downloads-v2' }),
}));

import { Route as DownloadsRoute } from './downloads';

function renderDownloads() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const rootRoute = createRootRoute({ component: () => React.createElement(Outlet) });
  const downloadsRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/downloads',
    validateSearch: DownloadsRoute.options.validateSearch,
    component: DownloadsRoute.options.component,
  });
  const router = createRouter({
    routeTree: rootRoute.addChildren([downloadsRoute]),
    history: createMemoryHistory({ initialEntries: ['/downloads'] }),
  });
  const tree = React.createElement(
    QueryClientProvider,
    { client: queryClient },
    React.createElement(RouterProvider, { router } as never)
  );
  return render(tree);
}

// ux3-cutover-3: the shell gate is gone — DownloadsBrowseV2 is the route's only render.
describe('routes/downloads (ux3-cutover-3)', () => {
  beforeEach(() => vi.clearAllMocks());

  it('renders the v2 deep page unconditionally', async () => {
    renderDownloads();
    expect(await screen.findByTestId('stub-downloads-v2')).toBeInTheDocument();
    expect(screen.queryByText('下載管理')).toBeNull();
  });
});
