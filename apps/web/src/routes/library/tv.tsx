import { createFileRoute } from '@tanstack/react-router';

/**
 * Clean type route (ux3-0-5 / D2). Path marker — see library/movies.tsx. The
 * persistent Browse UI lives in the `/library` layout (F5); the layout derives the
 * active type from this matched child.
 */
export const Route = createFileRoute('/library/tv')({
  component: () => null,
});
