import { createFileRoute } from '@tanstack/react-router';
import { ExploreBlocksSettings } from '../../components/settings/ExploreBlocksSettings';

export const Route = createFileRoute('/settings/homepage')({
  component: HomepageSettingsPage,
});

function HomepageSettingsPage() {
  return (
    <div>
      <h1 className="mb-2 text-2xl font-bold text-[var(--text-primary)]">自訂首頁</h1>
      <p className="mb-6 text-sm text-[var(--text-secondary)]">
        管理首頁上的探索區塊。每個區塊會依條件從 TMDb 拉取推薦內容。
      </p>
      <ExploreBlocksSettings />
    </div>
  );
}
