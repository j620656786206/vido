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

  // On setup page, render without AppShell
  if (location.pathname === '/setup') {
    return (
      <div className="text-slate-100">
        <Outlet />
      </div>
    );
  }

  return (
    <div className="text-slate-100">
      <AppShell>
        <Outlet />
      </AppShell>
      <ScanProgress />
    </div>
  );
}
