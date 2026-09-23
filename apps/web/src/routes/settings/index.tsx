// Design ref: ux-design.pen — no current screen frame; redirect only (to /settings/connection, C4-D)
import { createFileRoute, redirect } from '@tanstack/react-router';

export const Route = createFileRoute('/settings/')({
  beforeLoad: () => {
    throw redirect({ to: '/settings/connection' });
  },
});
