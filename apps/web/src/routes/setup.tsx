import { createFileRoute } from '@tanstack/react-router';
import { SetupWizard } from '../components/setup/SetupWizard';

export const Route = createFileRoute('/setup')({
  component: SetupPage,
});

function SetupPage() {
  return (
    // px-6: N3-M keeps a 24px gutter around the card on a 390px phone.
    <div className="flex min-h-screen items-center justify-center bg-[var(--bg-primary)] px-6 py-10">
      <SetupWizard />
    </div>
  );
}
