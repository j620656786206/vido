import { createFileRoute } from '@tanstack/react-router';
import { Gauge } from 'lucide-react';
import { SettingsPlaceholder } from '../../components/settings/SettingsPlaceholder';

export const Route = createFileRoute('/settings/performance')({
  component: PerformanceSettingsPage,
});

function PerformanceSettingsPage() {
  return (
    <SettingsPlaceholder
      icon={Gauge}
      title="效能監控"
      description="查看系統效能指標與趨勢"
    />
  );
}
