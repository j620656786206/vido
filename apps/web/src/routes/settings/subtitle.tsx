import { createFileRoute } from '@tanstack/react-router';
import { LocalizationLevelForm } from '../../components/settings/LocalizationLevelForm';

export const Route = createFileRoute('/settings/subtitle')({
  component: SubtitleSettingsPage,
});

function SubtitleSettingsPage() {
  return (
    <div>
      {/* Page header spans the layout column (J7-D — see settings-page-header-width.spec.ts). */}
      <h1 className="mb-2 text-2xl font-bold text-[var(--text-primary)]">字幕設定</h1>
      <p className="mb-6 text-sm text-[var(--text-secondary)]">
        AI 翻譯字幕的風格與口味。金鑰請到「金鑰設定」，媒體庫的自動字幕請到「媒體庫掃描」。
      </p>
      <LocalizationLevelForm />
    </div>
  );
}
