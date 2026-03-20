import { createFileRoute } from '@tanstack/react-router';
import { BackupManagement } from '../../components/settings/BackupManagement';

export const Route = createFileRoute('/settings/backup')({
  component: BackupSettingsPage,
});

function BackupSettingsPage() {
  return <BackupManagement />;
}
