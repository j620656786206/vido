import { createFileRoute } from '@tanstack/react-router';

/**
 * Merged cross-type library view (ux3-0-5 / D2) at `/library`. Path marker — the
 * persistent Browse UI lives in the `/library` layout (F5); with no type child
 * matched, the layout renders the "all" view.
 */
export const Route = createFileRoute('/library/')({
  component: () => null,
});
