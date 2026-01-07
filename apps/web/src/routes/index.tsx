import { createFileRoute } from '@tanstack/react-router';
import NxWelcome from '../app/nx-welcome';

export const Route = createFileRoute('/')({
  component: IndexComponent,
});

function IndexComponent() {
  return <NxWelcome title="web" />;
}
