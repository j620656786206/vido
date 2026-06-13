import { createRootRoute, Outlet, useNavigate, useLocation } from '@tanstack/react-router';
import { useEffect } from 'react';
import { AppShell, AppShellV2 } from '../components/shell';
import { ScanProgress } from '../components/scanner/ScanProgress';
import { useSetupStatus } from '../hooks/useSetupStatus';
import { useNewShellEnabled } from '../hooks/useNewShellEnabled';

export const Route = createRootRoute({
  component: RootComponent,
});

function RootComponent() {
  const navigate = useNavigate();
  const location = useLocation();
  const { data: setupStatus, isLoading } = useSetupStatus();
  // F4 single chokepoint — the ONLY place the v2-shell flag is read.
  const newShellEnabled = useNewShellEnabled();

  useEffect(() => {
    if (isLoading) return;

    const isSetupRoute = location.pathname === '/setup';

    if (setupStatus?.needsSetup && !isSetupRoute) {
      navigate({ to: '/setup' });
    }
  }, [setupStatus, isLoading, location.pathname, navigate]);

  const isSetupPage = location.pathname === '/setup';

  // On setup page, render without any shell or ScanProgress.
  if (isSetupPage) {
    return (
      <div className="text-[var(--text-primary)]">
        <Outlet />
      </div>
    );
  }

  // Flag ON → v2 shell (owns ScanProgress); flag OFF → legacy shell, unchanged.
  if (newShellEnabled) {
    return (
      <div className="text-[var(--text-primary)]">
        <AppShellV2>
          <Outlet />
        </AppShellV2>
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
