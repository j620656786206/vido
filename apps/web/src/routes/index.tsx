import { useState } from 'react';
import { createFileRoute } from '@tanstack/react-router';
import {
  DashboardLayout,
  DownloadPanel,
  RecentMediaPanel,
  QuickSearchBar,
} from '../components/dashboard';
import { NewMediaNotifications } from '../components/notifications/NewMediaNotifications';
import { useNewMediaNotifications } from '../hooks/useNewMediaNotifications';
import { QBStatusIndicator, ConnectionHistoryPanel } from '../components/health';

export const Route = createFileRoute('/')({
  component: DashboardPage,
});

function DashboardPage() {
  const { notifications, dismissNotification } = useNewMediaNotifications();
  const [historyOpen, setHistoryOpen] = useState(false);

  return (
    <div className="min-h-screen bg-slate-900">
      {/* Header */}
      <header className="border-b border-slate-800 px-4 py-4">
        <div className="mx-auto flex max-w-7xl items-center justify-between">
          <h1 className="text-xl font-bold text-slate-100">Vido</h1>
          <QBStatusIndicator onClick={() => setHistoryOpen(true)} />
        </div>
      </header>

      {/* Connection History Panel (AC4) */}
      <ConnectionHistoryPanel isOpen={historyOpen} onClose={() => setHistoryOpen(false)} />

      {/* Dashboard Content */}
      <DashboardLayout>
        <DownloadPanel />
        <RecentMediaPanel />
      </DashboardLayout>

      {/* Quick Search */}
      <div className="mx-auto max-w-7xl px-4 pb-6">
        <QuickSearchBar />
      </div>

      {/* New Media Notifications (AC2) */}
      <NewMediaNotifications notifications={notifications} onDismiss={dismissNotification} />
    </div>
  );
}
