// Design ref: ux-design.pen Screen C12-D (K28SdR) · C12-M (dOEbF) · C17-D (Gw61P) · C17-M (J186P)
import { createFileRoute } from '@tanstack/react-router';
import { SettingsPageHeader } from '../../components/settings/SettingsPageHeader';
import { LogsViewer } from '../../components/settings/LogsViewer';

export const Route = createFileRoute('/settings/logs')({
  component: LogsSettingsPage,
});

function LogsSettingsPage() {
  return (
    <div>
      <SettingsPageHeader
        title="系統日誌"
        description="檢視 Vido 的執行記錄，依等級與關鍵字篩選。"
      />
      <LogsViewer />
    </div>
  );
}
