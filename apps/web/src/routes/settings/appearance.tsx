// Design ref: ux-design.pen Screen C6-D (B3qPq) · C6-M (XpUjm)
import { createFileRoute } from '@tanstack/react-router';
import { AppearanceSettings } from '../../components/settings/AppearanceSettings';

export const Route = createFileRoute('/settings/appearance')({
  component: AppearanceSettings,
});
