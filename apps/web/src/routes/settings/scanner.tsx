// Design ref: ux-design.pen Screen E1-D (KvZSc) · E1-M (uABWl)
import { createFileRoute } from '@tanstack/react-router';
import { SettingsPageHeader } from '../../components/settings/SettingsPageHeader';
import { ScannerSettings } from '../../components/settings/ScannerSettings';

export const Route = createFileRoute('/settings/scanner')({
  component: ScannerSettingsPage,
});

function ScannerSettingsPage() {
  return (
    <div>
      <SettingsPageHeader
        title="媒體庫掃描"
        description="設定掃描資料夾、排程，以及手動觸發媒體庫掃描。"
      />
      <ScannerSettings />
    </div>
  );
}
