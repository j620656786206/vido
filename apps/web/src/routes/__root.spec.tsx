import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  createRootRoute,
  createRoute,
  createRouter,
  createMemoryHistory,
  RouterProvider,
} from '@tanstack/react-router';

// The root component's chrome is not under test — only its setup-redirect logic.
vi.mock('../components/shell', () => ({
  AppShellV2: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="app-shell-v2">{children}</div>
  ),
}));

const useSetupStatusMock = vi.fn();
vi.mock('../hooks/useSetupStatus', () => ({
  useSetupStatus: () => useSetupStatusMock(),
}));

import { Route as RootFileRoute } from './__root';

function renderAt(path: string) {
  // Rebuild a root route from the REAL root component so the redirect effect under
  // test is the production one; children are minimal path markers.
  const rootRoute = createRootRoute({ component: RootFileRoute.options.component });
  const indexRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/',
    component: () => <div data-testid="home-page" />,
  });
  const setupRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/setup',
    component: () => <div data-testid="setup-page" />,
  });
  const router = createRouter({
    routeTree: rootRoute.addChildren([indexRoute, setupRoute]),
    history: createMemoryHistory({ initialEntries: [path] }),
  });
  render(<RouterProvider router={router} />);
  return router;
}

describe('__root setup redirect', () => {
  beforeEach(() => {
    useSetupStatusMock.mockReset();
  });

  it('redirects to /setup when setup is needed', async () => {
    useSetupStatusMock.mockReturnValue({ data: { needsSetup: true }, isLoading: false });
    const router = renderAt('/');
    await waitFor(() => expect(router.state.location.pathname).toBe('/setup'));
    expect(await screen.findByTestId('setup-page')).toBeInTheDocument();
  });

  it('bounces /setup back to the app when setup is already completed (stale tab / seeded env)', async () => {
    useSetupStatusMock.mockReturnValue({ data: { needsSetup: false }, isLoading: false });
    const router = renderAt('/setup');
    await waitFor(() => expect(router.state.location.pathname).toBe('/'));
    expect(await screen.findByTestId('home-page')).toBeInTheDocument();
  });

  it('stays on /setup while setup is still needed', async () => {
    useSetupStatusMock.mockReturnValue({ data: { needsSetup: true }, isLoading: false });
    const router = renderAt('/setup');
    expect(await screen.findByTestId('setup-page')).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/setup');
  });

  it('does not redirect anywhere while setup status is loading', async () => {
    useSetupStatusMock.mockReturnValue({ data: undefined, isLoading: true });
    const router = renderAt('/setup');
    expect(await screen.findByTestId('setup-page')).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/setup');
  });
});
