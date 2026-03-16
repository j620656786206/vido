import { createFileRoute } from '@tanstack/react-router';
import { Database } from 'lucide-react';
import { SettingsPlaceholder } from '../../components/settings/SettingsPlaceholder';

export const Route = createFileRoute('/settings/cache')({
  component: CacheSettingsPage,
});

function CacheSettingsPage() {
  return (
    <SettingsPlaceholder
      icon={Database}
      title="快取管理"
      description="管理快取資料，釋放儲存空間"
    />
  );
}
