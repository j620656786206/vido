import { createFileRoute } from '@tanstack/react-router';
import { HardDrive } from 'lucide-react';
import { SettingsPlaceholder } from '../../components/settings/SettingsPlaceholder';

export const Route = createFileRoute('/settings/backup')({
  component: BackupSettingsPage,
});

function BackupSettingsPage() {
  return (
    <SettingsPlaceholder icon={HardDrive} title="備份與還原" description="備份與還原資料庫及設定" />
  );
}
