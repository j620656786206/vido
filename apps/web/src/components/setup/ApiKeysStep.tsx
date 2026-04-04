import type { StepProps } from './SetupWizard';

const AI_PROVIDERS = [
  { id: '', label: '不使用 AI' },
  { id: 'gemini', label: 'Google Gemini' },
  { id: 'claude', label: 'Anthropic Claude' },
];

export function ApiKeysStep({ data, onUpdate, onNext, onBack, onSkip }: StepProps) {
  return (
    <div data-testid="api-keys-step">
      <h2 className="mb-2 text-lg font-semibold text-[var(--text-primary)]">API 金鑰</h2>
      <p className="mb-6 text-sm text-[var(--text-secondary)]">
        設定 API 金鑰以啟用進階功能。可以稍後在設定頁面新增。
      </p>

      <div className="mb-4">
        <label
          htmlFor="tmdb-api-key"
          className="mb-2 block text-sm font-medium text-[var(--text-secondary)]"
        >
          TMDb API 金鑰
        </label>
        <input
          id="tmdb-api-key"
          type="text"
          value={data.tmdbApiKey || ''}
          onChange={(e) => onUpdate({ tmdbApiKey: e.target.value })}
          placeholder="輸入 TMDb API 金鑰..."
          className="w-full rounded-lg border border-[var(--border-subtle)]/50 bg-[var(--bg-secondary)]/60 px-4 py-2.5 text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
          data-testid="tmdb-key-input"
        />
        <p className="mt-1 text-xs text-[var(--text-muted)]">用於取得電影和影集的中文元資料</p>
      </div>

      <div className="mb-4">
        <label
          htmlFor="ai-provider"
          className="mb-2 block text-sm font-medium text-[var(--text-secondary)]"
        >
          AI 提供者
        </label>
        <select
          id="ai-provider"
          value={data.aiProvider || ''}
          onChange={(e) => onUpdate({ aiProvider: e.target.value })}
          className="w-full rounded-lg border border-[var(--border-subtle)]/50 bg-[var(--bg-secondary)]/60 px-4 py-2.5 text-sm text-[var(--text-primary)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
          data-testid="ai-provider-select"
        >
          {AI_PROVIDERS.map((provider) => (
            <option key={provider.id} value={provider.id}>
              {provider.label}
            </option>
          ))}
        </select>
      </div>

      {data.aiProvider && (
        <div className="mb-6">
          <label
            htmlFor="ai-api-key"
            className="mb-2 block text-sm font-medium text-[var(--text-secondary)]"
          >
            AI API 金鑰
          </label>
          <input
            id="ai-api-key"
            type="password"
            value={data.aiApiKey || ''}
            onChange={(e) => onUpdate({ aiApiKey: e.target.value })}
            placeholder="輸入 AI API 金鑰..."
            className="w-full rounded-lg border border-[var(--border-subtle)]/50 bg-[var(--bg-secondary)]/60 px-4 py-2.5 text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
            data-testid="ai-key-input"
          />
        </div>
      )}

      {!data.tmdbApiKey && !data.aiProvider && (
        <div
          className="mb-6 rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-400"
          data-testid="skip-warning"
        >
          跳過 API 金鑰設定將會限制部分功能，例如自動取得元資料和 AI 檔名解析。
        </div>
      )}

      <div className="flex gap-3">
        <button
          type="button"
          onClick={onBack}
          className="rounded-lg border border-[var(--border-subtle)]/50 px-4 py-2.5 text-sm font-medium text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-secondary)]"
          data-testid="back-button"
        >
          上一步
        </button>
        <button
          type="button"
          onClick={onNext}
          className="flex-1 rounded-lg bg-[var(--accent-primary)] px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-[var(--accent-pressed)]"
          data-testid="next-button"
        >
          下一步
        </button>
        {onSkip && (
          <button
            type="button"
            onClick={onSkip}
            className="rounded-lg px-4 py-2.5 text-sm font-medium text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)]"
            data-testid="skip-button"
          >
            跳過
          </button>
        )}
      </div>
    </div>
  );
}
