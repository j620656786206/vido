import type { StepProps } from './SetupWizard';

export function MediaFolderStep({ data, onUpdate, onNext, onBack }: StepProps) {
  return (
    <div data-testid="media-folder-step">
      <h2 className="mb-2 text-lg font-semibold text-slate-100">媒體資料夾</h2>
      <p className="mb-6 text-sm text-slate-400">
        設定您的媒體檔案存放路徑。Vido 會掃描此資料夾來管理您的影片。
      </p>

      <div className="mb-6">
        <label
          htmlFor="media-folder-path"
          className="mb-2 block text-sm font-medium text-slate-300"
        >
          資料夾路徑
        </label>
        <input
          id="media-folder-path"
          type="text"
          value={data.mediaFolderPath || ''}
          onChange={(e) => onUpdate({ mediaFolderPath: e.target.value })}
          placeholder="/media/videos"
          className="w-full rounded-lg border border-slate-600/50 bg-slate-800/60 px-4 py-2.5 text-sm text-slate-200 placeholder-slate-500 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          data-testid="media-folder-input"
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
      </div>
    </div>
  );
}
