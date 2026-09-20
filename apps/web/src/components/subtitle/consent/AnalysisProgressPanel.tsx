// Design ref: ux-design.pen Screen F14-D-v2 (nBT3M)
/**
 * F14 產生字幕．分析中 (sub-4-3 AC #1) — the pre-analysis state of the consent
 * flow. The step is FREE (local ffprobe) and the copy says so explicitly.
 * Counts are SSE-supplied (`generation_candidates_progress`, 250ms-throttled);
 * zero wall-clock reads (Rule 23).
 *
 * dsr-6e-2: renders TWO siblings into the host dialog's flex column — the
 * content block and a footer holding 取消, the same place F15 and F20 keep
 * their actions (F14 srMNm).
 *
 * The counter is deliberately NOT an aria-live region. It changes four times a
 * second; as a polite live region a screen reader would read numbers for the
 * whole analysis. The progressbar's aria-valuenow carries the progress, and the
 * container's `consent-phase-live` announces the phase change once.
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
    <>
      <div
        data-testid="consent-analysis-panel"
        className="flex min-h-0 flex-1 flex-col items-center justify-center overflow-y-auto px-10 py-12"
      >
        <div className="flex w-full max-w-[480px] flex-col gap-4">
          <div className="flex flex-col gap-3">
            <div
              role="progressbar"
              aria-valuemin={0}
              aria-valuemax={total}
              aria-valuenow={analyzed}
              aria-label="字幕軌分析進度"
              className="h-1.5 overflow-hidden rounded-full bg-[var(--bg-tertiary)]"
            >
              <div
                data-testid="consent-analysis-bar"
                className="h-full rounded-full bg-[var(--accent-primary)] transition-all duration-[var(--motion-move)]"
                style={{ width: `${pct}%` }}
              />
            </div>
            <p className="flex items-center justify-center gap-[3px] text-sm font-semibold text-[var(--text-primary)]">
              分析字幕軌
              <span data-testid="consent-analysis-counter" className="font-mono tabular-nums">
                {analyzed.toLocaleString()} / {total.toLocaleString()}
              </span>
            </p>
          </div>
          <p className="text-center text-sm text-[var(--text-secondary)]">
            正在檢查每個檔案是否有可直接抽取的內嵌字幕。這個步驟在本機執行，不會產生費用。
          </p>
        </div>
      </div>
      <div
        data-testid="consent-analysis-footer"
        className="flex shrink-0 items-center justify-end border-t border-[var(--border-subtle)] px-6 py-3.5"
      >
        {/* aria-disabled, never `disabled`: a focused button that becomes
            disabled drops focus to <body>, and Esc then stops closing the
            dialog. The `aria-disabled:` variants do the dimming `disabled:`
            no longer can. */}
        <button
          type="button"
          onClick={() => {
            if (!cancelling) onCancel();
          }}
          aria-disabled={cancelling}
          data-testid="consent-analysis-cancel"
          className="flex min-h-[44px] items-center gap-2 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-5 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)] aria-disabled:cursor-not-allowed aria-disabled:opacity-50"
        >
          {cancelling && (
            <Loader2
              className="h-4 w-4 animate-spin motion-reduce:animate-none"
              aria-hidden="true"
            />
          )}
          {cancelling ? '取消中…' : '取消'}
        </button>
      </div>
    </>
  );
}
