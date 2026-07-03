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

// Stub the v2 experience so the gate test asserts which composition the route picks
// without rendering the full v2 tree (covered by the DownloadsBrowseV2 children specs).
vi.mock('../components/downloads/DownloadsBrowseV2', () => ({
  DownloadsBrowseV2: () => React.createElement('div', { 'data-testid': 'stub-downloads-v2' }),
}));
// Keep the legacy branch network-free.
vi.mock('../hooks/useDownloads', () => ({
  useDownloads: () => ({ data: undefined, isLoading: false, error: null }),
  useDownloadCounts: () => ({ data: undefined }),
}));

import { Route as DownloadsRoute } from './downloads';
import { ShellVersionProvider } from '../components/shell/shellVersion';

function renderDownloads(shell: 'legacy' | 'v2' = 'legacy') {
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
  return render(
    shell === 'v2' ? React.createElement(ShellVersionProvider, { value: 'v2' }, tree) : tree
  );
}

describe('routes/downloads (shell-version gate — ux3-4-3 AC #1)', () => {
  beforeEach(() => vi.clearAllMocks());

  it('v2 shell renders the v2 deep page (DownloadsBrowseV2)', async () => {
    renderDownloads('v2');
    expect(await screen.findByTestId('stub-downloads-v2')).toBeInTheDocument();
    // The legacy header is gone under the v2 shell.
    expect(screen.queryByText('下載管理')).toBeNull();
  });

  it('legacy shell (default) renders the byte-unchanged legacy page, not v2', async () => {
    renderDownloads('legacy');
    expect(await screen.findByText('下載管理')).toBeInTheDocument();
    expect(screen.queryByTestId('stub-downloads-v2')).toBeNull();
  });
});
