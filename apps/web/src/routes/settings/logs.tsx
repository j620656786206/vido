import { createFileRoute } from '@tanstack/react-router';
import { LogsViewer } from '../../components/settings/LogsViewer';

export const Route = createFileRoute('/settings/logs')({
  component: LogsSettingsPage,
});

function LogsSettingsPage() {
  return (
    <div>
      <h1 className="mb-2 text-2xl font-bold text-[var(--text-primary)]">系統日誌</h1>
      <p className="mb-6 text-sm text-[var(--text-secondary)]">
        檢視 Vido 的執行記錄，依等級與關鍵字篩選。
      </p>
      <LogsViewer />
    </div>
  );
}
