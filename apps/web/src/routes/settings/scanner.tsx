import { createFileRoute } from '@tanstack/react-router';
import { ScannerSettings } from '../../components/settings/ScannerSettings';

export const Route = createFileRoute('/settings/scanner')({
  component: ScannerSettingsPage,
});

function ScannerSettingsPage() {
  return (
    <div>
      <h1 className="mb-2 text-2xl font-bold text-[var(--text-primary)]">媒體庫掃描</h1>
      <p className="mb-6 text-sm text-[var(--text-secondary)]">
        設定掃描資料夾、排程，以及手動觸發媒體庫掃描。
      </p>
      <ScannerSettings />
    </div>
  );
}
