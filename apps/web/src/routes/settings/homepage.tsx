import { createFileRoute } from '@tanstack/react-router';
import { ExploreBlocksSettings } from '../../components/settings/ExploreBlocksSettings';

export const Route = createFileRoute('/settings/homepage')({
  component: ExploreBlocksSettings,
});
