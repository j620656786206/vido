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

// ux3-cutover-3: the legacy dashboard home is deleted — HomeBrowseV2 is the
// route's only render. Stub it so the route test asserts composition + loader
// behavior without rendering the full v2 tree (covered by HomeBrowseV2.spec).
vi.mock('../components/homepage/HomeBrowseV2', () => ({
  HomeBrowseV2: () => React.createElement('div', { 'data-testid': 'stub-home-v2' }, 'home-v2'),
}));

// ux3-1-8: the loader seeds the OWN-library recently-added query (the hero's
// new source) instead of TMDb trending. Mock the service so the loader issues
// no real network traffic during the render.
vi.mock('../services/libraryService', () => ({
  libraryService: {
    getRecentlyAdded: vi.fn().mockResolvedValue([]),
  },
}));

// Mock the shared queryClient singleton — prefetchQuery should be observed by
// the test but never execute.
vi.mock('../queryClient', () => ({
  queryClient: {
    prefetchQuery: vi.fn().mockResolvedValue(undefined),
    // Shape TanStack Router expects — never actually called in these tests.
    defaultOptions: { queries: {} },
  },
}));

import { Route as IndexRoute } from './index';
import { queryClient as mockedQueryClient } from '../queryClient';

function renderHome() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  // Build a minimal route tree that includes the real index route so the
  // loader runs and the component renders.
  const rootRoute = createRootRoute({
    component: () => React.createElement(React.Fragment, null, React.createElement(Outlet)),
  });
  const indexRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/',
    component: IndexRoute.options.component,
    loader: IndexRoute.options.loader,
  });
  const routeTree = rootRoute.addChildren([indexRoute]);
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: ['/'] }),
  });

  const tree = React.createElement(
    QueryClientProvider,
    { client: queryClient },
    React.createElement(RouterProvider, { router } as never)
  );

  return render(tree);
}

describe('routes/index (ux3-cutover-3)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders Home v2 unconditionally (legacy dashboard home deleted)', async () => {
    renderHome();
    expect(await screen.findByTestId('stub-home-v2')).toBeInTheDocument();
    expect(screen.queryByTestId('homepage-root')).toBeNull();
  });

  it('[P1] ux3-1-8 — the loader prefetches the OWN-library recently-added query (the hero source), not TMDb trending', async () => {
    renderHome();
    await screen.findByTestId('stub-home-v2');
    const mockPrefetch = vi.mocked(mockedQueryClient.prefetchQuery);
    expect(mockPrefetch).toHaveBeenCalled();
    const call = mockPrefetch.mock.calls[0]?.[0];
    // MUST match useRecentlyAdded(20)'s key exactly — a drifted key seeds a
    // cache entry nothing reads, and the prefetch silently buys nothing.
    expect(call?.queryKey).toEqual(['library', 'recent', 20]);
    expect(call?.queryKey).not.toEqual(['trending', 'hero', 'week']);
  });
});
