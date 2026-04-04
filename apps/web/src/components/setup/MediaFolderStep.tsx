import type { StepProps } from './SetupWizard';

export function MediaFolderStep({ data, onUpdate, onNext, onBack }: StepProps) {
  return (
    <div data-testid="media-folder-step">
      <h2 className="mb-2 text-lg font-semibold text-[var(--text-primary)]">媒體資料夾</h2>
      <p className="mb-6 text-sm text-[var(--text-secondary)]">
        設定您的媒體檔案存放路徑。Vido 會掃描此資料夾來管理您的影片。
      </p>

      <div className="mb-6">
        <label
          htmlFor="media-folder-path"
          className="mb-2 block text-sm font-medium text-[var(--text-secondary)]"
        >
          資料夾路徑
        </label>
        <input
          id="media-folder-path"
          type="text"
          value={data.mediaFolderPath || ''}
          onChange={(e) => onUpdate({ mediaFolderPath: e.target.value })}
          placeholder="/media/videos"
          className="w-full rounded-lg border border-[var(--border-subtle)]/50 bg-[var(--bg-secondary)]/60 px-4 py-2.5 text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
          data-testid="media-folder-input"
        />
      </div>

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
      </div>
    </div>
  );
}
