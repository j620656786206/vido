// Design ref: ux-design.pen Screen C11-D (TrU8k) · C11-M (aYEWP) · C18-D (dfwSb)
import { createFileRoute } from '@tanstack/react-router';
import { SettingsPageHeader } from '../../components/settings/SettingsPageHeader';
import { CacheManagement } from '../../components/settings/CacheManagement';

export const Route = createFileRoute('/settings/cache')({
  component: CacheSettingsPage,
});

function CacheSettingsPage() {
  return (
    <div>
      <SettingsPageHeader
        title="快取管理"
        description="檢視外部資料快取的佔用，並依類型或時間清除。"
      />
      <CacheManagement />
    </div>
  );
}
