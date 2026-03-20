import type { StepProps } from './SetupWizard';

export function QBittorrentStep({ data, onUpdate, onNext, onBack, onSkip }: StepProps) {
  return (
    <div data-testid="qbittorrent-step">
      <h2 className="mb-2 text-lg font-semibold text-slate-100">qBittorrent 連線</h2>
      <p className="mb-6 text-sm text-slate-400">
        連接 qBittorrent 以監控下載進度。如果你尚未安裝，可以跳過此步驟。
      </p>

      <div className="mb-4">
        <label htmlFor="qbt-url" className="mb-2 block text-sm font-medium text-slate-300">
          WebUI 網址
        </label>
        <input
          id="qbt-url"
          type="text"
          value={data.qbtUrl || ''}
          onChange={(e) => onUpdate({ qbtUrl: e.target.value })}
          placeholder="http://localhost:8080"
          className="w-full rounded-lg border border-slate-600/50 bg-slate-800/60 px-4 py-2.5 text-sm text-slate-200 placeholder-slate-500 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          data-testid="qbt-url-input"
        />
      </div>

      <div className="mb-4">
        <label htmlFor="qbt-username" className="mb-2 block text-sm font-medium text-slate-300">
          使用者名稱
        </label>
        <input
          id="qbt-username"
          type="text"
          value={data.qbtUsername || ''}
          onChange={(e) => onUpdate({ qbtUsername: e.target.value })}
          placeholder="admin"
          className="w-full rounded-lg border border-slate-600/50 bg-slate-800/60 px-4 py-2.5 text-sm text-slate-200 placeholder-slate-500 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          data-testid="qbt-username-input"
        />
      </div>

      <div className="mb-6">
        <label htmlFor="qbt-password" className="mb-2 block text-sm font-medium text-slate-300">
          密碼
        </label>
        <input
          id="qbt-password"
          type="password"
          value={data.qbtPassword || ''}
          onChange={(e) => onUpdate({ qbtPassword: e.target.value })}
          className="w-full rounded-lg border border-slate-600/50 bg-slate-800/60 px-4 py-2.5 text-sm text-slate-200 placeholder-slate-500 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          data-testid="qbt-password-input"
        />
      </div>

      <div className="flex gap-3">
        <button
          type="button"
          onClick={onBack}
          className="rounded-lg border border-slate-600/50 px-4 py-2.5 text-sm font-medium text-slate-300 transition-colors hover:bg-slate-800"
          data-testid="back-button"
        >
          上一步
        </button>
        <button
          type="button"
          onClick={onNext}
          className="flex-1 rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-blue-700"
          data-testid="next-button"
        >
          下一步
        </button>
        {onSkip && (
          <button
            type="button"
            onClick={onSkip}
            className="rounded-lg px-4 py-2.5 text-sm font-medium text-slate-400 transition-colors hover:text-slate-200"
            data-testid="skip-button"
          >
            跳過
          </button>
        )}
      </div>
    </div>
  );
}
