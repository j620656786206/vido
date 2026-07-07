import { createFileRoute } from '@tanstack/react-router';
import { ActivityHub } from '../components/activity/ActivityHub';

/** ux3-ai-2 — the 活動-hosted AI generation workspace (F11). A lit view inside the
 * Activity destination, NOT a nav slot (destination-map: G ≠ destination; requests
 * `?view=requests` precedent, nav-ADR:630). */
interface ActivitySearch {
  view?: 'generation';
}

// ux3-2-3: net-new v2 destination (ADR D4-1). Marked for the v2 shell so AppShellV2
// renders it full-bleed (LegacyContentContainer opt-out, F4 — the flag stays read-once
// in __root). Activity has no legacy version, so there is no shell-version branch here:
// the hub renders directly under both shells (only the v2 shell links to it).
export const Route = createFileRoute('/activity')({
  staticData: { shell: 'v2' },
  // Rule 26: literal-only coercion — a lone `?view=generation` deep link must not be
  // JSON-parsed into something else; anything but the exact literal drops to undefined.
  validateSearch: (search: Record<string, unknown>): ActivitySearch => ({
    view: search.view === 'generation' ? 'generation' : undefined,
  }),
  component: ActivityHub,
});
