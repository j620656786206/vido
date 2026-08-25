import { createFileRoute } from '@tanstack/react-router';
import { QBittorrentForm } from '../../components/settings/QBittorrentForm';

export const Route = createFileRoute('/settings/connection')({
  component: ConnectionSettingsPage,
});

function ConnectionSettingsPage() {
  return (
    <div>
      {/* J7-D applies to the FORM CARD, not the page. Capping the whole page
          narrowed the h1 too, so the heading jumped 160px between settings tabs
          and a deliberate rule read as a bug. Header spans the layout's column;
          only the fields are held at 768px for scannability. */}
      <h1 className="mb-2 text-2xl font-bold text-[var(--text-primary)]">連線設定</h1>
      <p className="mb-6 text-sm text-[var(--text-secondary)]">
        設定 qBittorrent 連線資訊，以便從 Vido 監控下載狀態。
      </p>
      <div className="max-w-3xl rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-secondary)]/50 p-6">
        <QBittorrentForm />
      </div>
    </div>
  );
}
