import { createFileRoute } from '@tanstack/react-router';
import { FileText } from 'lucide-react';
import { SettingsPlaceholder } from '../../components/settings/SettingsPlaceholder';

export const Route = createFileRoute('/settings/logs')({
  component: LogsSettingsPage,
});

function LogsSettingsPage() {
  return (
    <SettingsPlaceholder icon={FileText} title="系統日誌" description="查看系統日誌，排除問題" />
  );
}
