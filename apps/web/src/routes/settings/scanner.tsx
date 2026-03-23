import { createFileRoute } from '@tanstack/react-router';
import { ScannerSettings } from '../../components/settings/ScannerSettings';

export const Route = createFileRoute('/settings/scanner')({
  component: ScannerSettingsPage,
});

function ScannerSettingsPage() {
  return <ScannerSettings />;
}
