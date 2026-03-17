import { CheckCircle } from 'lucide-react';
import type { StepProps } from './SetupWizard';

export function CompleteStep({ data, onNext, onBack, isSubmitting }: StepProps) {
  return (
    <div data-testid="complete-step">
      <div className="mb-6 flex flex-col items-center">
        <CheckCircle className="mb-3 h-12 w-12 text-green-400" />
        <h2 className="text-lg font-semibold text-slate-100">設定完成！</h2>
        <p className="mt-1 text-sm text-slate-400">以下是您的設定摘要。</p>
      </div>

      <div className="mb-6 space-y-3 rounded-lg border border-slate-700/50 bg-slate-800/40 p-4">
        <SummaryRow label="語言" value={data.language || 'zh-TW'} />
        <SummaryRow label="qBittorrent" value={data.qbtUrl || '未設定'} muted={!data.qbtUrl} />
        <SummaryRow
          label="媒體資料夾"
          value={data.mediaFolderPath || '未設定'}
          muted={!data.mediaFolderPath}
        />
        <SummaryRow
          label="TMDb API"
          value={data.tmdbApiKey ? '已設定' : '未設定'}
          muted={!data.tmdbApiKey}
        />
        <SummaryRow label="AI 服務" value={data.aiProvider || '未設定'} muted={!data.aiProvider} />
      </div>

      <div className="flex gap-3">
        <button
          type="button"
          onClick={onBack}
          disabled={isSubmitting}
          className="rounded-lg border border-slate-600/50 px-4 py-2.5 text-sm font-medium text-slate-300 transition-colors hover:bg-slate-800 disabled:opacity-50"
          data-testid="back-button"
        >
          上一步
        </button>
        <button
          type="button"
          onClick={onNext}
          disabled={isSubmitting}
          className="flex-1 rounded-lg bg-green-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-green-700 disabled:opacity-50"
          data-testid="finish-button"
        >
          {isSubmitting ? '儲存中...' : '完成設定'}
        </button>
      </div>
    </div>
  );
}

function SummaryRow({ label, value, muted }: { label: string; value: string; muted?: boolean }) {
  return (
    <div className="flex items-center justify-between text-sm">
      <span className="text-slate-400">{label}</span>
      <span className={muted ? 'text-slate-500' : 'text-slate-200'}>{value}</span>
    </div>
  );
}
