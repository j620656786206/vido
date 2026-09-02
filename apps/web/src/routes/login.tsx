import { createFileRoute } from '@tanstack/react-router';
import { LoginForm } from '../components/auth/LoginForm';

interface LoginSearch {
  /** Set by the 登出 button so the gate can acknowledge the action instead of
   *  showing the same anonymous card a stranger would get. */
  loggedOut?: boolean;
}

export const Route = createFileRoute('/login')({
  validateSearch: (search: Record<string, unknown>): LoginSearch => ({
    loggedOut: search.loggedOut === true || search.loggedOut === 'true' || undefined,
  }),
  component: LoginPage,
});

function LoginPage() {
  const { loggedOut } = Route.useSearch();
  return (
    // Anchored near the top rather than dead-centre: a centred card re-centres
    // itself every time its content grows, which on this screen means the field
    // slides while the user is reaching for it. px-4 keeps the card off the
    // phone's edges — at 390px it was landing 3px from the bezel.
    <div className="flex min-h-screen items-start justify-center bg-[var(--bg-primary)] px-4 pb-12 pt-[15vh]">
      <LoginForm justLoggedOut={loggedOut === true} />
    </div>
  );
}
