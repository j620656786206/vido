// Design ref: ux-design.pen Screen N1-D (dzgq9)
import { ChevronDown } from 'lucide-react';
import type { StepProps } from './SetupWizard';
import { SETUP_LANGUAGES } from './setupLanguages';
import { StepNav } from './StepNav';

export function WelcomeStep({ data, onUpdate, onNext }: StepProps) {
  return (
    <div className="flex flex-col gap-4" data-testid="welcome-step">
      <h2 className="text-lg font-semibold text-[var(--text-primary)]">歡迎使用 Vido</h2>
      <p className="text-sm text-[var(--text-secondary)]">
        Vido 是您的 NAS 媒體管理工具。讓我們快速完成基本設定。
      </p>

      <div className="flex flex-col gap-2">
        <label
          htmlFor="language-select"
          className="text-sm font-medium text-[var(--text-secondary)]"
        >
          選擇語言
        </label>
        <div className="relative">
          <select
            id="language-select"
            value={data.language || 'zh-TW'}
            onChange={(e) => onUpdate({ language: e.target.value })}
            className="h-11 w-full appearance-none rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] pl-3 pr-10 text-sm text-[var(--text-primary)] focus:border-[var(--focus-ring)] focus:outline-none focus:ring-1 focus:ring-[var(--focus-ring)]"
            data-testid="language-select"
          >
            {SETUP_LANGUAGES.map((lang) => (
              <option key={lang.code} value={lang.code}>
                {lang.label}
              </option>
            ))}
          </select>
          <ChevronDown
            className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--text-muted)]"
            aria-hidden="true"
          />
        </div>
      </div>

      <StepNav onNext={onNext} />
    </div>
  );
}
