import { createFileRoute } from '@tanstack/react-router';
import { CacheManagement } from '../../components/settings/CacheManagement';

export const Route = createFileRoute('/settings/cache')({
  component: CacheSettingsPage,
});

function CacheSettingsPage() {
  return <CacheManagement />;
}
