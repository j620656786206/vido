import { createFileRoute } from '@tanstack/react-router';
import { ActivityHub } from '../components/activity/ActivityHub';

// ux3-2-3: net-new v2 destination (ADR D4-1). Marked for the v2 shell so AppShellV2
// renders it full-bleed (LegacyContentContainer opt-out, F4 — the flag stays read-once
// in __root). Activity has no legacy version, so there is no shell-version branch here:
// the hub renders directly under both shells (only the v2 shell links to it).
export const Route = createFileRoute('/activity')({
  staticData: { shell: 'v2' },
  component: ActivityHub,
});
