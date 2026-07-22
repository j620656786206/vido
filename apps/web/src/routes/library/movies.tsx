import { createFileRoute } from '@tanstack/react-router';

/**
 * Clean type route (ux3-0-5 / D2). A path marker: the persistent Browse UI lives in
 * the `/library` LAYOUT (mounted once → shared filter/scroll state survives type
 * switches, ADR F5). The layout derives the active type from this matched child, so
 * this component renders nothing. Search params are inherited from the layout.
 */
export const Route = createFileRoute('/library/movies')({
  component: () => null,
});
