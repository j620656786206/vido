import { createFileRoute } from '@tanstack/react-router';
import {
  DashboardLayout,
  DownloadPanel,
  RecentMediaPanel,
  QuickSearchBar,
} from '../components/dashboard';

export const Route = createFileRoute('/')({
  component: DashboardPage,
});

function DashboardPage() {
  return (
    <div className="min-h-screen bg-slate-900">
      {/* Header */}
      <header className="border-b border-slate-800 px-4 py-4">
        <div className="mx-auto max-w-7xl">
          <h1 className="text-xl font-bold text-slate-100">Vido</h1>
        </div>
      </header>

      {/* Dashboard Content */}
      <DashboardLayout>
        <DownloadPanel />
        <RecentMediaPanel />
      </DashboardLayout>

      {/* Quick Search */}
      <div className="mx-auto max-w-7xl px-4 pb-6">
        <QuickSearchBar />
      </div>
    </div>
  );
}
