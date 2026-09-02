import React from 'react';
import { render, screen, waitFor, act } from '@testing-library/react';
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

const useAuthStatusMock = vi.fn();
vi.mock('../hooks/useAuthStatus', () => ({
  useAuthStatus: () => useAuthStatusMock(),
  authKeys: { all: ['auth'], status: () => ['auth', 'status'] },
}));

const getStatusMock = vi.fn();
vi.mock('../services/authService', () => ({
  authService: { getStatus: () => getStatusMock() },
}));

const ensureQueryDataMock = vi.fn();
vi.mock('../queryClient', () => ({
  queryClient: { ensureQueryData: (...args: unknown[]) => ensureQueryDataMock(...args) },
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
  const loginRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/login',
    component: () => <div data-testid="login-page" />,
  });
  const router = createRouter({
    routeTree: rootRoute.addChildren([indexRoute, setupRoute, loginRoute]),
    history: createMemoryHistory({ initialEntries: [path] }),
  });
  render(<RouterProvider router={router} />);
  return router;
}

describe('__root setup redirect', () => {
  beforeEach(() => {
    useSetupStatusMock.mockReset();
    useAuthStatusMock.mockReset();
    // Default: auth disabled, so the setup-redirect tests below see the same
    // behaviour they always had (the auth gate is a no-op when disabled).
    useAuthStatusMock.mockReturnValue({
      data: { authEnabled: false, authenticated: true },
      isLoading: false,
    });
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

describe('__root auth redirect', () => {
  beforeEach(() => {
    useSetupStatusMock.mockReset();
    useAuthStatusMock.mockReset();
    // Setup is complete so nothing but the auth gate can move us around.
    useSetupStatusMock.mockReturnValue({ data: { needsSetup: false }, isLoading: false });
  });

  it('redirects to /login when auth is enabled and not authenticated', async () => {
    useAuthStatusMock.mockReturnValue({
      data: { authEnabled: true, authenticated: false },
      isLoading: false,
    });
    const router = renderAt('/');
    await waitFor(() => expect(router.state.location.pathname).toBe('/login'));
    expect(await screen.findByTestId('login-page')).toBeInTheDocument();
  });

  it('bounces /login back to the app once authenticated', async () => {
    useAuthStatusMock.mockReturnValue({
      data: { authEnabled: true, authenticated: true },
      isLoading: false,
    });
    const router = renderAt('/login');
    await waitFor(() => expect(router.state.location.pathname).toBe('/'));
    expect(await screen.findByTestId('home-page')).toBeInTheDocument();
  });

  // Waiting is not the same as deciding: the route must not move while the
  // answer is still in flight...
  it('does not redirect to /login while auth status is loading', async () => {
    useAuthStatusMock.mockReturnValue({ data: undefined, isLoading: true });
    const router = renderAt('/');
    expect(await screen.findByTestId('auth-loading')).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/');
  });

  // ...but it must not render the app either. This test used to assert the home
  // page WAS shown here, which is what let an unauthenticated visitor see the
  // whole shell — sidebar, nav, readout tiles — for the length of the
  // /auth/status round-trip before the login card replaced it. On a NAS spinning
  // up its disks that is not one frame.
  it('shows no app content at all while auth status is loading', async () => {
    useAuthStatusMock.mockReturnValue({ data: undefined, isLoading: true });
    renderAt('/');
    await screen.findByTestId('auth-loading');
    expect(screen.queryByTestId('home-page')).toBeNull();
    expect(screen.queryByTestId('app-shell-v2')).toBeNull();
  });

  // The login screen itself is exempt: it is already the bare route, so making
  // it wait would only add a frame of nothing in front of the form.
  it('renders the login screen immediately even while auth status is loading', async () => {
    useAuthStatusMock.mockReturnValue({ data: undefined, isLoading: true });
    renderAt('/login');
    expect(await screen.findByTestId('login-page')).toBeInTheDocument();
    expect(screen.queryByTestId('auth-loading')).toBeNull();
  });

  // A status endpoint that never answers must not hold the product hostage. The
  // wait surface has a hard ceiling from mount — and the ceiling must NOT be
  // restarted when `isLoading` flaps, which is what a failing endpoint on a
  // refetch interval does all day. The first version reset the timer on every
  // flap and left the wordmark up forever.
  it('renders the app anyway when auth status never resolves', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      useAuthStatusMock.mockReturnValue({ data: undefined, isLoading: true });
      renderAt('/');
      // The router mounts asynchronously, so wait for the surface rather than
      // asserting on the first synchronous frame.
      expect(await screen.findByTestId('auth-loading')).toBeInTheDocument();

      await act(async () => {
        await vi.advanceTimersByTimeAsync(2500);
      });

      expect(await screen.findByTestId('home-page')).toBeInTheDocument();
      expect(screen.queryByTestId('auth-loading')).toBeNull();
    } finally {
      vi.useRealTimers();
    }
  });

  it('takes an unauthenticated visitor to /login before the setup gate', async () => {
    // Even with setup still needed, login comes first.
    useSetupStatusMock.mockReturnValue({ data: { needsSetup: true }, isLoading: false });
    useAuthStatusMock.mockReturnValue({
      data: { authEnabled: true, authenticated: false },
      isLoading: false,
    });
    const router = renderAt('/');
    await waitFor(() => expect(router.state.location.pathname).toBe('/login'));
  });
});

// The beforeLoad gate is the one that actually stops work happening, and none of
// the render tests above can see it: they build their own root route from
// `options.component`, so the route-level guard is not in the tree they mount.
describe('__root beforeLoad gate', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const run = (pathname: string) =>
    (
      RootFileRoute.options.beforeLoad as (ctx: { location: { pathname: string } }) => Promise<void>
    )({ location: { pathname } });

  it('redirects an unauthenticated visitor before any route matches', async () => {
    ensureQueryDataMock.mockResolvedValue({ authEnabled: true, authenticated: false });
    // A thrown redirect, not a returned one — that is what aborts the match, so
    // the home route's LOADER never runs and no gated endpoint is called.
    await expect(run('/')).rejects.toMatchObject({ options: { to: '/login' } });
  });

  it('lets an authenticated visitor through', async () => {
    ensureQueryDataMock.mockResolvedValue({ authEnabled: true, authenticated: true });
    await expect(run('/library')).resolves.toBeUndefined();
  });

  it('lets everyone through when the server has no password set', async () => {
    ensureQueryDataMock.mockResolvedValue({ authEnabled: false, authenticated: true });
    await expect(run('/library')).resolves.toBeUndefined();
  });

  it('never gates the login screen itself', async () => {
    await expect(run('/login')).resolves.toBeUndefined();
    expect(ensureQueryDataMock).not.toHaveBeenCalled();
  });

  // If /auth/status is unreachable the gate must not brick the whole app —
  // the React guard picks it up a beat later.
  it('does not block the app when the status endpoint fails', async () => {
    ensureQueryDataMock.mockRejectedValue(new Error('network down'));
    await expect(run('/')).resolves.toBeUndefined();
  });

  // The router AWAITS beforeLoad, so an endpoint that never answers is a hang,
  // not a slow start — nothing renders at all. The visual-regression job proved
  // it in the worst way: it boots the Vite dev server with no API behind it and
  // every screen went blank. The wait has to be bounded.
  it('gives up and lets the app render when the status endpoint never answers', async () => {
    vi.useFakeTimers();
    try {
      ensureQueryDataMock.mockReturnValue(new Promise(() => {}));
      const settled = vi.fn();
      const pending = run('/').then(settled);

      await vi.advanceTimersByTimeAsync(1000);
      expect(settled).not.toHaveBeenCalled();

      await vi.advanceTimersByTimeAsync(1500);
      await pending;
      expect(settled).toHaveBeenCalled();
    } finally {
      vi.useRealTimers();
    }
  });
});
