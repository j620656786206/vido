import { createFileRoute } from '@tanstack/react-router';
import { ServiceStatusDashboard } from '../../components/settings/ServiceStatusDashboard';

export const Route = createFileRoute('/settings/status')({
  component: StatusSettingsPage,
});

function StatusSettingsPage() {
  return <ServiceStatusDashboard />;
}
