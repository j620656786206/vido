// Implements: Component/EmptyLibrary-NoFolder (U3SGxG)
import { Link } from '@tanstack/react-router';
import { FolderOpen } from 'lucide-react';

export function EmptyNoFolder() {
  return (
    <div
      className="flex flex-col items-center justify-center py-24 text-center"
      data-testid="empty-no-folder"
    >
      <div className="mb-6 flex items-center gap-3 text-[var(--text-muted)]">
        <FolderOpen className="h-10 w-10" />
      </div>

      <h2 className="mb-3 text-xl font-semibold text-[var(--text-primary)]">
        指定一個媒體資料夾即可開始
      </h2>
      <p className="mb-8 max-w-sm text-sm text-[var(--text-secondary)]">
        Vido 會掃描資料夾中的影片並自動匹配 TMDb 資訊
      </p>

      <div className="flex items-center gap-3">
        <Link
          to="/settings/libraries"
          className="rounded-lg bg-[var(--accent-primary)] px-5 py-2.5 text-sm font-medium text-white transition-colors hover:bg-[var(--accent-pressed)]"
          data-testid="empty-no-folder-libraries-btn"
        >
          設定媒體資料夾
        </Link>
        <Link
          to="/setup"
          className="rounded-lg border border-[var(--border-subtle)] px-5 py-2.5 text-sm font-medium text-[var(--text-secondary)] transition-colors hover:border-[var(--text-muted)] hover:text-white"
          data-testid="empty-no-folder-wizard-btn"
        >
          開啟設定精靈
        </Link>
      </div>
    </div>
  );
}
