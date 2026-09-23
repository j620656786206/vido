// Design ref: ux-design.pen Screen C5-D (uhAKd) · C5-M (gEQX4) · C19-D (G8BYO) · C19-M (gPZU6) · C20-D (v2C4xr)
import { createFileRoute } from '@tanstack/react-router';
import { SettingsPageHeader } from '../../components/settings/SettingsPageHeader';
import { BackupManagement } from '../../components/settings/BackupManagement';

export const Route = createFileRoute('/settings/backup')({
  component: BackupSettingsPage,
});

function BackupSettingsPage() {
  return (
    <div>
      <SettingsPageHeader
        title="備份與還原"
        description="建立與管理 Vido 資料庫備份，確保資料安全。"
      />
      <BackupManagement />
    </div>
  );
}
