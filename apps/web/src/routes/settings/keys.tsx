import { createFileRoute } from '@tanstack/react-router';
import { ApiKeysForm } from '../../components/settings/ApiKeysForm';

export const Route = createFileRoute('/settings/keys')({
  component: KeysSettingsPage,
});

function KeysSettingsPage() {
  return <ApiKeysForm />;
}
