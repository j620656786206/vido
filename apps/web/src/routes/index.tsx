import { createFileRoute } from '@tanstack/react-router';
import { DashboardLayout, DownloadPanel, RecentMediaPanel } from '../components/dashboard';
import { NewMediaNotifications } from '../components/notifications/NewMediaNotifications';
import { useNewMediaNotifications } from '../hooks/useNewMediaNotifications';

export const Route = createFileRoute('/')({
  component: DashboardPage,
});

function DashboardPage() {
  const { notifications, dismissNotification } = useNewMediaNotifications();

  return (
    <div>
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
