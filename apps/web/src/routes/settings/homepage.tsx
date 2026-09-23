// Design ref: ux-design.pen Screen C10-D (wnmGh) · C10-M (ZjsVs)
import { createFileRoute } from '@tanstack/react-router';
import { SettingsPageHeader } from '../../components/settings/SettingsPageHeader';
import { ExploreBlocksSettings } from '../../components/settings/ExploreBlocksSettings';

export const Route = createFileRoute('/settings/homepage')({
  component: HomepageSettingsPage,
});

function HomepageSettingsPage() {
  return (
    <div>
      <SettingsPageHeader
        title="自訂首頁"
        description="管理首頁上的探索區塊。每個區塊會依條件從 TMDb 拉取推薦內容。"
      />
      <ExploreBlocksSettings />
    </div>
  );
}
