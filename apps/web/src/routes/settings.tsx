import { createFileRoute, Outlet } from '@tanstack/react-router';
import { SettingsLayout } from '../components/settings/SettingsLayout';

export const Route = createFileRoute('/settings')({
  component: SettingsPage,
});

function SettingsPage() {
  return (
    <SettingsLayout>
      <Outlet />
    </SettingsLayout>
  );
}
