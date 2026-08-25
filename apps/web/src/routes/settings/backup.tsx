import { createFileRoute } from '@tanstack/react-router';
import { BackupManagement } from '../../components/settings/BackupManagement';

export const Route = createFileRoute('/settings/backup')({
  component: BackupSettingsPage,
});

function BackupSettingsPage() {
  return (
    <div>
      <h1 className="mb-2 text-2xl font-bold text-[var(--text-primary)]">備份與還原</h1>
      <p className="mb-6 text-sm text-[var(--text-secondary)]">
        建立與管理 Vido 資料庫備份，確保資料安全。
      </p>
      <BackupManagement />
    </div>
  );
}
