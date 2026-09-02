import {
  createRootRoute,
  Outlet,
  redirect,
  useNavigate,
  useLocation,
} from '@tanstack/react-router';
import { useEffect, useState } from 'react';
import { AppShellV2 } from '../components/shell';
import { queryClient } from '../queryClient';
import { useSetupStatus } from '../hooks/useSetupStatus';
import { authKeys, useAuthStatus } from '../hooks/useAuthStatus';
import { authService } from '../services/authService';

/**
 * How long the route-level gate may wait on /auth/status before giving up and
 * letting the app render. The endpoint is public, in-memory, and served by the
 * same process as the HTML — if the page loaded at all, this answers in
 * milliseconds. The ceiling exists for the case where it answers never.
 */
const GATE_MAX_WAIT_MS = 2000;

export const Route = createRootRoute({
  /**
   * The real gate. It has to live here, before route matching, because a
   * React-side guard is always one render too late: `routes/index.tsx` carries a
   * LOADER, and a loader runs when the route matches — not when its component
   * renders. So an unauthenticated visitor opening `/` still fired
   * /library/recent, /home-summary and /explore-blocks and collected three 401s
   * on the way to the login screen, no matter what the root component chose to
   * render. Throwing a redirect out of beforeLoad stops the match itself, so
   * nothing downstream ever runs.
   *
   * `ensureQueryData` shares the singleton cache with the React tree, so this
   * costs one /auth/status call, not two, and the hook below reads the result.
   */
  beforeLoad: async ({ location }) => {
    if (location.pathname === '/login') return;
    // ⚠️ The router awaits this, so nothing renders until it settles — which
    // makes an unreachable /auth/status a hard hang, not a slow start. The
    // visual-regression job proved it: it boots the Vite dev server with NO API
    // behind it, and every screen went blank because the gate never resolved.
    // Never let this wait unbounded. On timeout (or error) we fall through and
    // the React guard below takes over — the same behaviour as before this gate
    // existed, and if the status endpoint is unreachable then every gated call
    // is failing anyway, so there is nothing left to protect.
    const status = await Promise.race([
      queryClient
        .ensureQueryData({
          queryKey: authKeys.status(),
          queryFn: () => authService.getStatus(),
          retry: false,
        })
        .catch(() => null),
      new Promise<null>((resolve) => {
        setTimeout(resolve, GATE_MAX_WAIT_MS, null);
      }),
    ]);
    // Thrown, not returned — that is what aborts the match, so the home route's
    // loader never runs and no gated endpoint is called before login.
    if (status && status.authEnabled && !status.authenticated) {
      throw redirect({ to: '/login' });
    }
  },
  component: RootComponent,
});

function RootComponent() {
  const navigate = useNavigate();
  const location = useLocation();
  const { data: authStatus, isLoading: authStatusLoading } = useAuthStatus();

  // The wait surface is allowed to hold the screen, but not to keep it. A stuck
  // status check must degrade to "unknown", never to a blank product.
  //
  // ⚠️ ONE-SHOT, from mount, with no dependencies. The first version restarted
  // the timer whenever `isLoading` went false — which sounds right and is
  // exactly wrong: against a failing endpoint the query retries on an interval,
  // so `isLoading` flaps true/false forever, the timer was reset on every flap,
  // and the wordmark stayed on screen indefinitely. Verified against a Vite dev
  // server with no API behind it, which is the shape the visual job runs.
  const [waitedOut, setWaitedOut] = useState(false);
  useEffect(() => {
    const id = window.setTimeout(() => setWaitedOut(true), GATE_MAX_WAIT_MS);
    return () => window.clearTimeout(id);
  }, []);
  const authLoading = authStatusLoading && !waitedOut;

  const needsLogin = authStatus?.authEnabled === true && authStatus.authenticated === false;

  // Setup status lives behind the auth gate, so it must not be asked for until
  // the gate has cleared — otherwise every visit to the login screen fires a
  // 401 that no one surfaces and no one can act on.
  const { data: setupStatus, isLoading: setupLoading } = useSetupStatus({
    enabled: !authLoading && !needsLogin,
  });

  useEffect(() => {
    if (authLoading) return;

    const path = location.pathname;
    const isLoginRoute = path === '/login';

    // Auth gate runs FIRST. An unauthenticated visitor sees only the login
    // screen — every other route's data calls would 401 anyway (the API is
    // gated server-side), so there is nothing to show until they log in.
    if (needsLogin) {
      if (!isLoginRoute) navigate({ to: '/login' });
      return;
    }
    // Authenticated (or auth disabled): the login page has no purpose.
    if (isLoginRoute) {
      navigate({ to: '/' });
      return;
    }

    // Setup gate only matters once past auth.
    if (setupLoading) return;
    const isSetupRoute = path === '/setup';

    if (setupStatus?.needsSetup && !isSetupRoute) {
      navigate({ to: '/setup' });
    }
    // Reverse guard: landing on /setup with setup already completed (stale tab,
    // bookmark, seeded test env) must bounce to the app — the wizard can't even
    // complete against an already-configured backend. Mid-wizard this never
    // fires: needsSetup stays true until the final step's completeSetup, and
    // that handler navigates home itself.
    if (setupStatus && !setupStatus.needsSetup && isSetupRoute) {
      navigate({ to: '/' });
    }
  }, [needsLogin, authLoading, setupStatus, setupLoading, location.pathname, navigate]);

  // Login and setup render WITHOUT the shell: AppShellV2 owns the nav and
  // ScanProgress and fires app data queries that would 401 before login.
  //
  // `authLoading` belongs in this list. Without it, an unauthenticated visitor
  // opening `/` got the COMPLETE shell for the duration of the /auth/status
  // round-trip — sidebar, nav, storage strip, readout tiles, all of it — before
  // it was swapped for the login card. Measured at 60ms on a warm dev box; on a
  // NAS spinning its disks up it is a great deal longer, and what it showed was
  // the skeleton of a library the visitor had not yet proved they owned.
  const isBareRoute =
    location.pathname === '/setup' || location.pathname === '/login' || needsLogin || authLoading;

  if (isBareRoute) {
    return (
      <div className="text-[var(--text-primary)]">
        {/* Deliberately not a spinner: nothing here has measurable progress, and
            a spinner claims otherwise. The wordmark holds the space so the login
            card lands ON it rather than replacing a different picture.
            `needsLogin` is in the condition as well as `authLoading`: once the
            answer is "no", the pathname is still `/` for the tick before the
            redirect lands, and rendering the Outlet there mounts the home route
            — which fired /home-summary, /library/recent and /explore-blocks at a
            visitor who is not logged in, three 401s nobody could act on. */}
        {(authLoading || needsLogin) && location.pathname !== '/login' ? (
          <div
            data-testid="auth-loading"
            className="flex min-h-screen items-start justify-center bg-[var(--bg-primary)] px-4 pt-[15vh]"
          >
            <span className="text-2xl font-bold leading-none text-[var(--accent-text)]">vido</span>
          </div>
        ) : (
          <Outlet />
        )}
      </div>
    );
  }

  return (
    <div className="text-[var(--text-primary)]">
      <AppShellV2>
        <Outlet />
      </AppShellV2>
    </div>
  );
}
