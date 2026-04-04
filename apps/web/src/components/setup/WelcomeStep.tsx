import type { StepProps } from './SetupWizard';

const LANGUAGES = [
  { code: 'zh-TW', label: '繁體中文' },
  { code: 'en', label: 'English' },
  { code: 'ja', label: '日本語' },
];

export function WelcomeStep({ data, onUpdate, onNext }: StepProps) {
  return (
    <div data-testid="welcome-step">
      <h2 className="mb-2 text-lg font-semibold text-[var(--text-primary)]">歡迎使用 Vido</h2>
      <p className="mb-6 text-sm text-[var(--text-secondary)]">
        Vido 是您的 NAS 媒體管理工具。讓我們快速完成基本設定。
      </p>

      <div className="mb-6">
        <label
          htmlFor="language-select"
          className="mb-2 block text-sm font-medium text-[var(--text-secondary)]"
        >
          選擇語言
        </label>
        <select
          id="language-select"
          value={data.language || 'zh-TW'}
          onChange={(e) => onUpdate({ language: e.target.value })}
          className="w-full rounded-lg border border-[var(--border-subtle)]/50 bg-[var(--bg-secondary)]/60 px-4 py-2.5 text-sm text-[var(--text-primary)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
          data-testid="language-select"
        >
          {LANGUAGES.map((lang) => (
            <option key={lang.code} value={lang.code}>
              {lang.label}
            </option>
          ))}
        </select>
      </div>

      <button
        type="button"
        onClick={onNext}
        className="w-full rounded-lg bg-[var(--accent-primary)] px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-[var(--accent-pressed)]"
        data-testid="next-button"
      >
        下一步
      </button>
    </div>
  );
}
