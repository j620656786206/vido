import { createRootRoute, Outlet, useNavigate, useLocation } from '@tanstack/react-router';
import { useEffect } from 'react';
import { AppShell } from '../components/shell';
import { ScanProgress } from '../components/scanner/ScanProgress';
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

  // On setup page, render without AppShell or ScanProgress
  if (isSetupPage) {
    return (
      <div className="text-[var(--text-primary)]">
        <Outlet />
      </div>
    );
  }

  return (
    <div className="text-[var(--text-primary)]">
      <AppShell>
        <Outlet />
      </AppShell>
      <ScanProgress />
    </div>
  );
}
