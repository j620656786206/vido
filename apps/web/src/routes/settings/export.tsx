import { createFileRoute } from '@tanstack/react-router';
import { ArrowUpDown } from 'lucide-react';
import { SettingsPlaceholder } from '../../components/settings/SettingsPlaceholder';

export const Route = createFileRoute('/settings/export')({
  component: ExportSettingsPage,
});

function ExportSettingsPage() {
  return (
    <SettingsPlaceholder
      icon={ArrowUpDown}
      title="匯出/匯入"
      description="匯出或匯入媒體庫元資料"
    />
  );
}
