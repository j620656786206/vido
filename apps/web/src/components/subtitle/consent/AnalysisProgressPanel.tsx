// Design ref: ux-design.pen Screen F14-D-v2 (nBT3M)
/**
 * F14 產生字幕．分析中 (sub-4-3 AC #1) — the pre-analysis state of the consent
 * flow. The step is FREE (local ffprobe) and the copy says so explicitly.
 * Counts are SSE-supplied (`generation_candidates_progress`, 250ms-throttled);
 * zero wall-clock reads (Rule 23).
 */
import { Loader2 } from 'lucide-react';

export interface AnalysisProgressPanelProps {
  analyzed: number;
  total: number;
  cancelling?: boolean;
  onCancel: () => void;
}

export function AnalysisProgressPanel({
  analyzed,
  total,
  cancelling = false,
  onCancel,
}: AnalysisProgressPanelProps) {
  const pct = total > 0 ? (analyzed / total) * 100 : 0;
  return (
    <div
      data-testid="consent-analysis-panel"
      className="flex flex-1 flex-col items-center justify-center gap-5 px-6 py-12"
    >
      <div className="flex w-full max-w-md flex-col gap-3">
        <div
          role="progressbar"
          aria-valuemin={0}
          aria-valuemax={total}
          aria-valuenow={analyzed}
          aria-label="字幕軌分析進度"
          className="h-1.5 overflow-hidden rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)]"
        >
          <div
            data-testid="consent-analysis-bar"
            className="h-full rounded-[var(--radius-sm)] bg-[var(--accent-primary)] transition-all duration-300"
            style={{ width: `${pct}%` }}
          />
        </div>
        <p
          aria-live="polite"
          className="flex items-center justify-center gap-[3px] text-sm text-[var(--text-primary)]"
        >
          分析字幕軌
          <span data-testid="consent-analysis-counter" className="font-mono tabular-nums">
            {analyzed.toLocaleString()} / {total.toLocaleString()}
          </span>
        </p>
        <p className="text-center text-[13px] text-[var(--text-secondary)]">
          正在檢查每個檔案是否有可直接抽取的內嵌字幕。這個步驟在本機執行，不會產生費用。
        </p>
      </div>
      <button
        type="button"
        onClick={onCancel}
        disabled={cancelling}
        data-testid="consent-analysis-cancel"
        className="flex min-h-[44px] items-center gap-2 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-5 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)] disabled:cursor-not-allowed disabled:opacity-50"
      >
        {cancelling && (
          <Loader2 className="h-4 w-4 animate-spin motion-reduce:animate-none" aria-hidden="true" />
        )}
        取消
      </button>
    </div>
  );
}
