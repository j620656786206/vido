import { createRootRoute, Outlet, useNavigate, useLocation } from '@tanstack/react-router';
import { useEffect } from 'react';
import { AppShellV2 } from '../components/shell';
import { useSetupStatus } from '../hooks/useSetupStatus';
import { useAuthStatus } from '../hooks/useAuthStatus';

export const Route = createRootRoute({
  component: RootComponent,
});

function RootComponent() {
  const navigate = useNavigate();
  const location = useLocation();
  const { data: authStatus, isLoading: authLoading } = useAuthStatus();
  const { data: setupStatus, isLoading: setupLoading } = useSetupStatus();

  const needsLogin = authStatus?.authEnabled === true && authStatus.authenticated === false;

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
  // ScanProgress and fires app data queries that would 401 before login. Once
  // auth status resolves as "needs login", render bare so the shell unmounts
  // before the redirect lands. (During the brief authLoading window the shell
  // may still mount and fire queries that 401 harmlessly — the server gates
  // them and there is no global 401 handler to surface an error.)
  const isBareRoute =
    location.pathname === '/setup' || location.pathname === '/login' || needsLogin;

  if (isBareRoute) {
    return (
      <div className="text-[var(--text-primary)]">
        <Outlet />
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
