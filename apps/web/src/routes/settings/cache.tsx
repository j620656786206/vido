import { createFileRoute } from '@tanstack/react-router';
import { CacheManagement } from '../../components/settings/CacheManagement';

export const Route = createFileRoute('/settings/cache')({
  component: CacheSettingsPage,
});

function CacheSettingsPage() {
  return (
    <div>
      <h1 className="mb-2 text-2xl font-bold text-[var(--text-primary)]">快取管理</h1>
      <p className="mb-6 text-sm text-[var(--text-secondary)]">
        檢視外部資料快取的佔用，並依類型或時間清除。
      </p>
      <CacheManagement />
    </div>
  );
}
