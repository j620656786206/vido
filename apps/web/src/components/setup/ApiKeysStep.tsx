// Design ref: ux-design.pen Screen N4-D (D990CP)
import { AlertTriangle } from 'lucide-react';
import type { StepProps } from './SetupWizard';
import { StepNav } from './StepNav';
import { WizardTextField } from './WizardTextField';

export function ApiKeysStep({ data, onUpdate, onNext, onBack, onSkip }: StepProps) {
  const nothingEntered = !data.tmdbApiKey && !data.claudeApiKey;

  return (
    <div className="flex flex-col gap-4" data-testid="api-keys-step">
      <h2 className="text-lg font-semibold text-[var(--text-primary)]">API 金鑰</h2>
      <p className="text-sm text-[var(--text-secondary)]">
        設定 API 金鑰以啟用進階功能。可以稍後在設定頁面新增。
      </p>

      <WizardTextField
        id="tmdb-api-key"
        label="TMDb 金鑰"
        value={data.tmdbApiKey || ''}
        onChange={(tmdbApiKey) => onUpdate({ tmdbApiKey })}
        placeholder="輸入 TMDb API 金鑰..."
        hint="用於取得電影和影集的中文元資料"
        testId="tmdb-key-input"
      />

      {/* Claude only (Alexyu 2026-09-14, dsr-13). It is the one text-AI key the
          running server reads back from the secret store — Gemini is env-only.
          The wizard used to offer a provider picker and stored whatever was
          typed under a name nothing read, then reported 已設定. */}
      <WizardTextField
        id="claude-api-key"
        label="Claude 金鑰"
        type="password"
        value={data.claudeApiKey || ''}
        onChange={(claudeApiKey) => onUpdate({ claudeApiKey })}
        placeholder="輸入 Claude API 金鑰..."
        hint="用於字幕翻譯與 AI 檔名解析"
        testId="claude-key-input"
      />

      {nothingEntered && (
        // A note about what skipping WOULD do, not about the world now — so no
        // status colour (DESIGN.md 2026-09-11: 赭說的是「現在的世界」). Neutral
        // ground, primary text, the warning glyph kept.
        <div
          className="flex items-center gap-2 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] p-3"
          data-testid="skip-warning"
        >
          <AlertTriangle
            className="h-4 w-4 shrink-0 text-[var(--text-secondary)]"
            aria-hidden="true"
          />
          <p className="text-xs text-[var(--text-primary)]">
            跳過 API 金鑰設定將會限制部分功能，例如自動取得元資料和 AI 檔名解析。
          </p>
        </div>
      )}

      <StepNav onNext={onNext} onBack={onBack} onSkip={onSkip} />
    </div>
  );
}
