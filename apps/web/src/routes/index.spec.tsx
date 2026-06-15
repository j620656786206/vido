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
// ux3-1-2: stub Home v2 so the gate test asserts which composition the route picks
// without rendering the full v2 tree (covered by HomeBrowseV2.spec).
vi.mock('../components/homepage/HomeBrowseV2', () => ({
  HomeBrowseV2: () => React.createElement('div', { 'data-testid': 'stub-home-v2' }, 'home-v2'),
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
import { ShellVersionProvider } from '../components/shell/shellVersion';

function renderHome(shell: 'legacy' | 'v2' = 'legacy') {
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

  // The shell version is provided by AppShellV2 in production (read-once, F4); here we
  // supply it directly so HomeRoute's useShellVersion() gate can be exercised. No
  // provider → defaults to 'legacy' (so the existing legacy assertions are unaffected).
  const tree = React.createElement(
    QueryClientProvider,
    { client: queryClient },
    React.createElement(RouterProvider, { router } as any)
  );

  return render(
    shell === 'v2' ? React.createElement(ShellVersionProvider, { value: 'v2' }, tree) : tree
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

describe('routes/index (shell-version gate — ux3-1-2)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('[P1] legacy shell (default) renders the dashboard home, not Home v2', async () => {
    renderHome('legacy');
    expect(await screen.findByTestId('homepage-root')).toBeInTheDocument();
    expect(screen.queryByTestId('stub-home-v2')).toBeNull();
  });

  it('[P1] v2 shell renders Home v2 and drops the legacy dashboard composition', async () => {
    renderHome('v2');
    expect(await screen.findByTestId('stub-home-v2')).toBeInTheDocument();
    // Legacy dashboard root + its Epic-4 remnants are gone under the v2 shell (ux3-1-4).
    expect(screen.queryByTestId('homepage-root')).toBeNull();
    expect(screen.queryByTestId('stub-downloads')).toBeNull();
    expect(screen.queryByTestId('stub-qb')).toBeNull();
  });
});
