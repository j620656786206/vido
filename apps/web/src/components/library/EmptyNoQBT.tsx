// Implements: Component/EmptyLibrary-NoQBT (fSKuT)
import { Link } from '@tanstack/react-router';
import { Film, FolderOpen } from 'lucide-react';

export function EmptyNoQBT() {
  return (
    <div
      className="flex flex-col items-center justify-center py-24 text-center"
      data-testid="empty-no-qbt"
    >
      <div className="mb-6 flex items-center gap-3 text-[var(--text-muted)]">
        <Film className="h-10 w-10" />
        <FolderOpen className="h-10 w-10" />
      </div>

      <h2 className="mb-3 text-xl font-semibold text-[var(--text-primary)]">
        連接 qBittorrent 開始下載
      </h2>
      <p className="mb-8 max-w-sm text-sm text-[var(--text-secondary)]">
        Vido 會自動追蹤你的下載並建立媒體庫
      </p>

      <div className="flex items-center gap-3">
        <Link
          to="/settings/qbittorrent"
          className="rounded-lg bg-[var(--accent-primary)] px-5 py-2.5 text-sm font-medium text-white transition-colors hover:bg-[var(--accent-pressed)]"
          data-testid="empty-no-qbt-connect-btn"
        >
          連接 qBittorrent
        </Link>
        <Link
          to="/settings/libraries"
          className="rounded-lg border border-[var(--border-subtle)] px-5 py-2.5 text-sm font-medium text-[var(--text-secondary)] transition-colors hover:border-[var(--text-muted)] hover:text-white"
          data-testid="empty-no-qbt-folder-btn"
        >
          已有檔案？設定資料夾
        </Link>
      </div>
    </div>
  );
}
