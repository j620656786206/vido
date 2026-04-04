import { Link } from '@tanstack/react-router';
import { Film, FolderOpen } from 'lucide-react';

export function EmptyLibrary() {
  return (
    <div
      className="flex flex-col items-center justify-center py-24 text-center"
      data-testid="empty-library"
    >
      {/* Icons matching design — muted film + folder */}
      <div className="mb-6 flex items-center gap-3 text-[var(--text-muted)]">
        <Film className="h-10 w-10" />
        <FolderOpen className="h-10 w-10" />
      </div>

      <h2 className="mb-3 text-xl font-semibold text-[var(--text-primary)]">你的媒體庫還是空的</h2>
      <p className="mb-8 max-w-sm text-sm text-[var(--text-secondary)]">
        透過 qBittorrent 或將媒體檔案加入監控資料夾即可開始
      </p>

      {/* Two CTA buttons matching design */}
      <div className="flex items-center gap-3">
        <Link
          to="/settings/qbittorrent"
          className="rounded-lg bg-[var(--accent-primary)] px-5 py-2.5 text-sm font-medium text-white transition-colors hover:bg-[var(--accent-pressed)]"
          data-testid="connect-qbittorrent-btn"
        >
          連接 qBittorrent
        </Link>
        <Link
          to="/search"
          className="rounded-lg border border-[var(--border-subtle)] px-5 py-2.5 text-sm font-medium text-[var(--text-secondary)] transition-colors hover:border-[var(--text-muted)] hover:text-white"
          data-testid="learn-more-btn"
        >
          了解更多
        </Link>
      </div>
    </div>
  );
}
