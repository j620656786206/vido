// Design ref: ux-design.pen Screen C9-D (NR3zK) · C9-M (AWYm0)
import { createFileRoute } from '@tanstack/react-router';
import { SettingsPageHeader } from '../../components/settings/SettingsPageHeader';
import { LocalizationLevelForm } from '../../components/settings/LocalizationLevelForm';

export const Route = createFileRoute('/settings/subtitle')({
  component: SubtitleSettingsPage,
});

function SubtitleSettingsPage() {
  return (
    <div>
      {/* Page header spans the layout column (J7-D — see settings-page-header-width.spec.ts). */}
      <SettingsPageHeader
        title="字幕設定"
        description="AI 翻譯字幕的風格與口味。金鑰請到「金鑰設定」，媒體庫的自動字幕請到「媒體庫掃描」。"
      />
      <LocalizationLevelForm />
    </div>
  );
}
