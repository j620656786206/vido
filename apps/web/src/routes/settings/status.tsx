// Design ref: ux-design.pen Screen C8-D (wqcqY) · C8-M (qx8Ma) · C15-D (XwdOH) · C15-M (wkUNt) · C16-D (uYGBU)
import { createFileRoute } from '@tanstack/react-router';
import { SettingsPageHeader } from '../../components/settings/SettingsPageHeader';
import { ServiceStatusDashboard } from '../../components/settings/ServiceStatusDashboard';

export const Route = createFileRoute('/settings/status')({
  component: StatusSettingsPage,
});

function StatusSettingsPage() {
  return (
    <div>
      <SettingsPageHeader title="服務狀態" description="監控外部服務連線狀態。" />
      <ServiceStatusDashboard />
    </div>
  );
}
