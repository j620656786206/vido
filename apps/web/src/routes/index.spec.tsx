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

// Stub every child of DashboardPage so the test can focus on ordering and the
// outer flex wrapper — the individual sections are covered by their own specs.
vi.mock('../components/homepage/HeroBanner', () => ({
  HeroBanner: () => React.createElement('div', { 'data-testid': 'stub-hero' }, 'hero'),
}));
vi.mock('../components/homepage/ExploreBlocksList', () => ({
  ExploreBlocksList: () => React.createElement('div', { 'data-testid': 'stub-explore' }, 'explore'),
}));
vi.mock('../components/dashboard', () => ({
  RecentMediaPanel: ({ hideWhenEmpty }: { hideWhenEmpty?: boolean }) =>
    React.createElement(
      'div',
      { 'data-testid': 'stub-recent', 'data-hide-when-empty': String(!!hideWhenEmpty) },
      'recent'
    ),
  DownloadPanel: ({ hideWhenEmpty }: { hideWhenEmpty?: boolean }) =>
    React.createElement(
      'div',
      { 'data-testid': 'stub-downloads', 'data-hide-when-empty': String(!!hideWhenEmpty) },
      'downloads'
    ),
}));
vi.mock('../components/health', () => ({
  QBStatusIndicator: () => React.createElement('div', { 'data-testid': 'stub-qb' }),
  ConnectionHistoryPanel: () => null,
}));
vi.mock('../components/notifications/NewMediaNotifications', () => ({
  NewMediaNotifications: () => null,
}));
vi.mock('../hooks/useNewMediaNotifications', () => ({
  useNewMediaNotifications: () => ({ notifications: [], dismissNotification: vi.fn() }),
}));

// Mock tmdbService so the route loader does not issue real network traffic
// during the render.
vi.mock('../services/tmdb', () => ({
  default: {
    getTrendingMovies: vi.fn().mockResolvedValue({ results: [] }),
    getTrendingTVShows: vi.fn().mockResolvedValue({ results: [] }),
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

  return render(
    React.createElement(
      QueryClientProvider,
      { client: queryClient },
      React.createElement(RouterProvider, { router } as any)
    )
  );
}

describe('routes/index (Homepage layout)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('[P1] AC #1 — sections render in order Hero → Explore → Recent → Downloads', async () => {
    renderHome();
    const root = await screen.findByTestId('homepage-root');

    const order = Array.from(
      root.querySelectorAll<HTMLElement>(
        '[data-testid="stub-hero"], [data-testid="stub-explore"], [data-testid="stub-recent"], [data-testid="stub-downloads"]'
      )
    ).map((el) => el.getAttribute('data-testid'));

    expect(order).toEqual(['stub-hero', 'stub-explore', 'stub-recent', 'stub-downloads']);
  });

  it('[P1] Task 1.3 — outer flex wrapper uses gap-6 (mobile) and md:gap-8 (desktop)', async () => {
    renderHome();
    const root = await screen.findByTestId('homepage-root');
    const cls = root.className;
    expect(cls).toContain('flex');
    expect(cls).toContain('flex-col');
    expect(cls).toContain('gap-6');
    expect(cls).toContain('md:gap-8');
  });

  it('[P1] AC #5 — homepage passes hideWhenEmpty to Recent and Download panels', async () => {
    renderHome();
    const recent = await screen.findByTestId('stub-recent');
    const downloads = screen.getByTestId('stub-downloads');
    expect(recent.getAttribute('data-hide-when-empty')).toBe('true');
    expect(downloads.getAttribute('data-hide-when-empty')).toBe('true');
  });

  it('[P1] Task 2.4 — route loader prefetches trending hero data', async () => {
    renderHome();
    await screen.findByTestId('homepage-root');
    const mockPrefetch = vi.mocked(mockedQueryClient.prefetchQuery);
    expect(mockPrefetch).toHaveBeenCalled();
    const call = mockPrefetch.mock.calls[0]?.[0];
    expect(call?.queryKey).toEqual(['trending', 'hero', 'week']);
  });
});
