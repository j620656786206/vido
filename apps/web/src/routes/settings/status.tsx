import { createFileRoute } from '@tanstack/react-router';
import { ServiceStatusDashboard } from '../../components/settings/ServiceStatusDashboard';

export const Route = createFileRoute('/settings/status')({
  component: StatusSettingsPage,
});

function StatusSettingsPage() {
  return (
    <div>
      <h1 className="mb-2 text-2xl font-bold text-[var(--text-primary)]">服務狀態</h1>
      <p className="mb-6 text-sm text-[var(--text-secondary)]">監控外部服務連線狀態。</p>
      <ServiceStatusDashboard />
    </div>
  );
}
