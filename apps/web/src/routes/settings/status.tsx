import { createFileRoute } from '@tanstack/react-router';
import { Activity } from 'lucide-react';
import { SettingsPlaceholder } from '../../components/settings/SettingsPlaceholder';

export const Route = createFileRoute('/settings/status')({
  component: StatusSettingsPage,
});

function StatusSettingsPage() {
  return (
    <SettingsPlaceholder
      icon={Activity}
      title="服務狀態"
      description="監控外部服務連線狀態"
    />
  );
}
