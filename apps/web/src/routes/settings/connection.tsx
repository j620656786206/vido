import { createFileRoute } from '@tanstack/react-router';
import { QBittorrentForm } from '../../components/settings/QBittorrentForm';

export const Route = createFileRoute('/settings/connection')({
  component: ConnectionSettingsPage,
});

function ConnectionSettingsPage() {
  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="mb-6 text-2xl font-bold text-slate-100">連線設定</h1>
      <p className="mb-8 text-sm text-slate-400">
        設定 qBittorrent 連線資訊，以便從 Vido 監控下載狀態。
      </p>
      <div className="rounded-lg border border-slate-700 bg-slate-800/50 p-6">
        <QBittorrentForm />
      </div>
    </div>
  );
}
