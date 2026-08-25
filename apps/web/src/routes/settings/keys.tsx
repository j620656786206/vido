import { createFileRoute } from '@tanstack/react-router';
import { ApiKeysForm } from '../../components/settings/ApiKeysForm';

export const Route = createFileRoute('/settings/keys')({
  component: KeysSettingsPage,
});

function KeysSettingsPage() {
  return (
    <div>
      {/* Page header spans the layout column (see ApiKeysForm's J7-D note). */}
      <h1 className="mb-2 text-2xl font-bold text-[var(--text-primary)]">金鑰設定</h1>
      <p className="mb-6 text-sm text-[var(--text-secondary)]">
        設定 Vido 使用的第三方服務 API 金鑰。金鑰會加密後儲存於 NAS，並優先於環境變數。
      </p>
      <ApiKeysForm />
    </div>
  );
}
