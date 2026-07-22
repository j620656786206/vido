import { createRootRoute, Outlet, useNavigate, useLocation } from '@tanstack/react-router';
import { useEffect } from 'react';
import { AppShellV2 } from '../components/shell';
import { useSetupStatus } from '../hooks/useSetupStatus';

export const Route = createRootRoute({
  component: RootComponent,
});

function RootComponent() {
  const navigate = useNavigate();
  const location = useLocation();
  const { data: setupStatus, isLoading } = useSetupStatus();

  useEffect(() => {
    if (isLoading) return;

    const isSetupRoute = location.pathname === '/setup';

    if (setupStatus?.needsSetup && !isSetupRoute) {
      navigate({ to: '/setup' });
    }
  }, [setupStatus, isLoading, location.pathname, navigate]);

  const isSetupPage = location.pathname === '/setup';

  // On setup page, render without the shell (AppShellV2 owns ScanProgress).
  if (isSetupPage) {
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
