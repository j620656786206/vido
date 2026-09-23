// Design ref: ux-design.pen Screen C7-D (PWvEX) · C7-M (f8Fda) · C21-D (AVUg2) · C22-D (t6FA4)
import { createFileRoute } from '@tanstack/react-router';
import { SettingsPageHeader } from '../../components/settings/SettingsPageHeader';
import { ApiKeysForm } from '../../components/settings/ApiKeysForm';

export const Route = createFileRoute('/settings/keys')({
  component: KeysSettingsPage,
});

function KeysSettingsPage() {
  return (
    <div>
      {/* Page header spans the layout column (see ApiKeysForm's J7-D note). */}
      <SettingsPageHeader
        title="金鑰設定"
        description="設定 Vido 使用的第三方服務 API 金鑰。金鑰會加密後儲存於 NAS，並優先於環境變數。"
      />
      <ApiKeysForm />
    </div>
  );
}
