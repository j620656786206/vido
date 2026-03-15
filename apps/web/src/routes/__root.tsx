import { createRootRoute, Outlet } from '@tanstack/react-router';
import { AppShell } from '../components/shell';

export const Route = createRootRoute({
  component: RootComponent,
});

function RootComponent() {
  return (
    <div className="text-slate-100">
      <AppShell>
        <Outlet />
      </AppShell>
    </div>
  );
}
