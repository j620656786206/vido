// Design ref: ux-design.pen Screen C4-D (6UCtX) · C4-M (2H4OM) · C23-D (Qva0y) · C23-M (p37q9)
// (C23 is the Sonarr / Radarr cards further down the same page)
import { createFileRoute } from '@tanstack/react-router';
import { SettingsPageHeader } from '../../components/settings/SettingsPageHeader';
import { QBittorrentForm } from '../../components/settings/QBittorrentForm';
import { ArrConnectionForm } from '../../components/settings/ArrConnectionForm';

export const Route = createFileRoute('/settings/connection')({
  component: ConnectionSettingsPage,
});

function ConnectionSettingsPage() {
  return (
    <div>
      {/* J7-D applies to the FORM CARDS, not the page. Capping the whole page
          narrowed the h1 too, so the heading jumped 160px between settings tabs
          and a deliberate rule read as a bug. Header spans the layout's column;
          only the cards are held at 768px for scannability. */}
      <SettingsPageHeader
        title="連線設定"
        description="設定 Vido 連到 qBittorrent、Sonarr 與 Radarr 的方式。"
      />
      <div className="flex flex-col gap-6">
        <section
          aria-labelledby="qbittorrent-card-title"
          className="max-w-3xl rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-4 md:p-8"
        >
          <header className="mb-6">
            <h2
              id="qbittorrent-card-title"
              className="text-base font-semibold text-[var(--text-primary)] md:text-lg"
            >
              qBittorrent
            </h2>
            <p className="text-xs text-[var(--text-muted)]">下載器</p>
          </header>
          <QBittorrentForm />
        </section>
        <ArrConnectionForm plugin="sonarr" />
        <ArrConnectionForm plugin="radarr" />
      </div>
    </div>
  );
}
