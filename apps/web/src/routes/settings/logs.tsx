import { createFileRoute } from '@tanstack/react-router';
import { LogsViewer } from '../../components/settings/LogsViewer';

export const Route = createFileRoute('/settings/logs')({
  component: LogsSettingsPage,
});

function LogsSettingsPage() {
  return <LogsViewer />;
}
