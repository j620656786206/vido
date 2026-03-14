import { createRootRoute, Outlet } from '@tanstack/react-router';
import { AppShell } from '../components/shell';

export const Route = createRootRoute({
  component: RootComponent,
});

function RootComponent() {
  return (
    <div className="min-h-screen bg-slate-900 text-slate-100">
      <AppShell>
        <Outlet />
      </AppShell>
    </div>
  );
}
