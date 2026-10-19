// Design ref: ux-design.pen Screen N4-D (D990CP)
import { AlertTriangle } from 'lucide-react';
import { useKeySettings } from '../../hooks/useKeySettings';
import type { StepProps } from './SetupWizard';
import { StepNav } from './StepNav';
import { WizardTextField } from './WizardTextField';

// Where each provider hands out keys. Opened in a new tab so the wizard keeps
// what was already typed.
const CLAUDE_KEYS_URL = 'https://console.anthropic.com/settings/keys';
const GEMINI_KEYS_URL = 'https://aistudio.google.com/apikey';

export function ApiKeysStep({ data, onUpdate, onNext, onBack, onSkip }: StepProps) {
  const nothingEntered = !data.tmdbApiKey && !data.claudeApiKey && !data.geminiApiKey;
  // sub-7-7b: release images carry a TMDb key (source "bundled"); a source
  // build does not. The copy must not threaten「抓不到元資料」on an install
  // where metadata already works — nor promise it on one where it does not.
  // The request failing (offline, source build without the route) simply
  // means "not bundled", which is the honest default.
  const { data: keySettings } = useKeySettings();
  const tmdbBundled =
    keySettings?.keys.some((key) => key.name === 'tmdb' && key.source === 'bundled') ?? false;

  return (
    <div className="flex flex-col gap-4" data-testid="api-keys-step">
      <h2 className="text-lg font-semibold text-[var(--text-primary)]">API 金鑰</h2>
      <p className="text-sm text-[var(--text-secondary)]">
        Vido 不收錢。AI 字幕用你自己的金鑰，費用直接記在你的 Claude 或 Gemini 帳號，不經過
        Vido；每次按下去之前，Vido 會先告訴你大約花多少。兩把都可以先空著，之後到設定頁再填。
      </p>

      <WizardTextField
        id="tmdb-api-key"
        label="TMDb 金鑰"
        value={data.tmdbApiKey || ''}
        onChange={(tmdbApiKey) => onUpdate({ tmdbApiKey })}
        placeholder={tmdbBundled ? '可留空，使用內建金鑰' : '輸入 TMDb API 金鑰...'}
        hint={
          tmdbBundled
            ? '已內建預設金鑰，可留空；填入自己的金鑰會優先使用'
            : '用於取得電影和影集的中文元資料'
        }
        testId="tmdb-key-input"
      />

      {/* setup-wizard-byok (2026-10-19): Claude and Gemini, each stored under the
          secret name KeyResolver reads (translation_dispatcher picks Gemini when
          it is the only key). Most people arriving from the NAS groups have no
          Claude key; the Gemini free tier is how they try one episode. */}
      <WizardTextField
        id="claude-api-key"
        label="Claude 金鑰"
        type="password"
        value={data.claudeApiKey || ''}
        onChange={(claudeApiKey) => onUpdate({ claudeApiKey })}
        placeholder="輸入 Claude API 金鑰..."
        hint="預設的翻譯模型。一集影集（約一小時、有英文字幕）大約 US$0.15～0.3。"
        link={{
          href: CLAUDE_KEYS_URL,
          label: '怎麼申請 Claude 金鑰 ↗',
          testId: 'claude-key-link',
        }}
        testId="claude-key-input"
      />

      {/* Never 「免費」 on its own: it is the user's own free quota (gemini-translation-dispatch). */}
      <WizardTextField
        id="gemini-api-key"
        label="Gemini 金鑰（選填）"
        type="password"
        value={data.geminiApiKey || ''}
        onChange={(geminiApiKey) => onUpdate({ geminiApiKey })}
        placeholder="輸入 Gemini API 金鑰..."
        hint="沒有 Claude 金鑰？用你自己的 Gemini 免費額度也能翻；每分鐘 5 次，Vido 會自動放慢。"
        link={{
          href: GEMINI_KEYS_URL,
          label: '怎麼申請 Gemini 金鑰 ↗',
          testId: 'gemini-key-link',
        }}
        testId="gemini-key-input"
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
            {tmdbBundled
              ? 'Claude 和 Gemini 金鑰都沒填的話，就不能做 AI 字幕翻譯和檔名解析；電影和影集的元資料會使用內建的 TMDb 金鑰。'
              : 'Claude 和 Gemini 金鑰都沒填的話，就不能做 AI 字幕翻譯和檔名解析；也不能自動取得電影和影集的元資料。'}
          </p>
        </div>
      )}

      <StepNav onNext={onNext} onBack={onBack} onSkip={onSkip} />
    </div>
  );
}
