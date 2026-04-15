import { useState } from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { DashboardLayout, DownloadPanel, RecentMediaPanel } from '../components/dashboard';
import { NewMediaNotifications } from '../components/notifications/NewMediaNotifications';
import { useNewMediaNotifications } from '../hooks/useNewMediaNotifications';
import { QBStatusIndicator, ConnectionHistoryPanel } from '../components/health';
import { HeroBanner } from '../components/homepage/HeroBanner';

export const Route = createFileRoute('/')({
  component: DashboardPage,
});

function DashboardPage() {
  const { notifications, dismissNotification } = useNewMediaNotifications();
  const [historyOpen, setHistoryOpen] = useState(false);

  return (
    <div>
      {/* Hero Banner — Story 10-2 (P2-001) */}
      <HeroBanner />

      {/* Connection Health Status */}
      <div className="mx-auto max-w-7xl px-4 pt-4 sm:px-6">
        <div className="flex justify-end">
          <QBStatusIndicator onClick={() => setHistoryOpen(true)} />
        </div>
      </div>

      {/* Connection History Panel (AC4) */}
      <ConnectionHistoryPanel isOpen={historyOpen} onClose={() => setHistoryOpen(false)} />

      {/* Dashboard Content */}
      <DashboardLayout>
        <DownloadPanel />
        <RecentMediaPanel />
      </DashboardLayout>

      {/* New Media Notifications (AC2) */}
      <NewMediaNotifications notifications={notifications} onDismiss={dismissNotification} />
    </div>
  );
}
