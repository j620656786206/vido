import { createFileRoute } from '@tanstack/react-router';
import { SetupWizard } from '../components/setup/SetupWizard';

export const Route = createFileRoute('/setup')({
  component: SetupPage,
});

function SetupPage() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-slate-950">
      <SetupWizard />
    </div>
  );
}
