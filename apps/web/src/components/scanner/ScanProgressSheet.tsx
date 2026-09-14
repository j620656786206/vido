// Design ref: ux-design.pen Screen E2-M (yezIo) · E3-M (ZjoEI)
/**
 * Mobile bottom sheet scan progress (Story 7.4, Task 4)
 * Peek state: 64px, full width. Expanded: half screen with drag handle.
 */

import { useState, useEffect, useRef, useCallback } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { Loader, AlertTriangle, CheckCircle, XCircle, X } from 'lucide-react';
import type { ScanProgressState } from '../../hooks/useScanProgress';
import {
  SCAN_PROBLEMS_DESTINATION,
  ScanStats,
  scanHadProblems,
  scanSummaryParts,
} from './ScanProgressCard';

const AUTO_DISMISS_MS = 10000;

export interface ScanProgressSheetProps {
  state: ScanProgressState;
  onCancel: () => void;
  onDismiss: () => void;
  isCancelling?: boolean;
  /** sub-4-3 F17 mobile twin — see ScanProgressCard. */
  missingSubtitleCount?: number;
}

export function ScanProgressSheet({
  state,
  onCancel,
  onDismiss,
  isCancelling = false,
  missingSubtitleCount,
}: ScanProgressSheetProps) {
  const navigate = useNavigate();
  const [expanded, setExpanded] = useState(false);
  const [showCancelConfirm, setShowCancelConfirm] = useState(false);
  const autoDismissTimerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const dragStartY = useRef<number | null>(null);

  const clearAutoDismiss = useCallback(() => {
    if (autoDismissTimerRef.current) clearTimeout(autoDismissTimerRef.current);
  }, []);

  // Auto-dismiss on completion
  useEffect(() => {
    if (state.isComplete || state.isCancelled) {
      autoDismissTimerRef.current = setTimeout(onDismiss, AUTO_DISMISS_MS);
    } else {
      clearAutoDismiss();
    }
    return clearAutoDismiss;
  }, [state.isComplete, state.isCancelled, onDismiss, clearAutoDismiss]);

  const handleTouchStart = (e: React.TouchEvent) => {
    dragStartY.current = e.touches[0].clientY;
  };

  const handleTouchEnd = (e: React.TouchEvent) => {
    if (dragStartY.current === null) return;
    const delta = e.changedTouches[0].clientY - dragStartY.current;
    dragStartY.current = null;

    if (expanded && delta > 50) {
      // Swipe down → collapse
      setExpanded(false);
    } else if (!expanded && delta < -30) {
      // Swipe up → expand
      setExpanded(true);
    }
  };

  const handleCancelConfirm = () => {
    setShowCancelConfirm(false);
    onCancel();
  };

  // Completion/cancelled toast — E3-M: a floating card inset from the edges,
  // just above the tab bar (the wrapper in ScanProgress lifts it clear; pb keeps
  // it off the bar below 640px and off the screen edge above it).
  if (state.isComplete || state.isCancelled) {
    const hadProblems = scanHadProblems(state);
    const showMissing =
      !state.isCancelled && missingSubtitleCount !== undefined && missingSubtitleCount > 0;
    const parts = scanSummaryParts(state);
    // Two lines on a phone: what was found and written / what went wrong.
    const splitAt = parts.findIndex((part) => part.startsWith('無法匯入'));
    const lines = splitAt > 0 ? [parts.slice(0, splitAt), parts.slice(splitAt)] : [parts];
    return (
      <div className="px-4 pb-2 sm:pb-4">
        <div
          className="flex w-full flex-col gap-2.5 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-3.5 shadow-[var(--shadow-lg)]"
          data-testid="scan-progress-sheet"
          role="status"
        >
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              {state.isCancelled ? (
                <XCircle className="h-4 w-4 text-[var(--text-secondary)]" />
              ) : hadProblems ? (
                <AlertTriangle className="h-4 w-4 text-[var(--warning-text)]" />
              ) : (
                <CheckCircle className="h-4 w-4 text-[var(--success-text)]" />
              )}
              <span className="text-sm font-semibold text-[var(--text-primary)]">
                {state.isCancelled ? '掃描已取消' : '掃描完成'}
              </span>
            </div>
            <button
              type="button"
              onClick={onDismiss}
              className="-m-2 flex size-11 items-center justify-center text-[var(--text-muted)] hover:text-[var(--text-primary)]"
              aria-label="關閉"
              data-testid="sheet-dismiss-btn"
            >
              <X className="h-3.5 w-3.5" />
            </button>
          </div>

          <div
            className="space-y-1 text-xs tabular-nums text-[var(--text-secondary)]"
            data-testid="sheet-scan-summary"
          >
            {lines.map((line) => (
              <p key={line[0]}>{line.join(' · ')}</p>
            ))}
          </div>

          {showMissing && (
            <p
              data-testid="scan-missing-subtitle-line"
              className="flex items-center gap-[3px] text-xs text-[var(--text-secondary)]"
            >
              <span className="font-mono tabular-nums">
                {missingSubtitleCount.toLocaleString()}
              </span>
              部影片缺繁中字幕
            </p>
          )}

          {/* Same destination as the desktop card (dsr-5). The phone toast used
              to offer 產生字幕 only, so what the summary had just counted as
              failed had no way in. Links are 44px tall: a phone has to hit them. */}
          {(hadProblems || showMissing) && (
            <div className="-my-2 flex flex-wrap gap-x-4">
              {hadProblems && (
                <button
                  type="button"
                  onClick={() => {
                    onDismiss();
                    navigate(SCAN_PROBLEMS_DESTINATION);
                  }}
                  className="min-h-11 text-xs font-medium text-[var(--accent-text)] underline-offset-2 hover:underline"
                  data-testid="sheet-view-scan-problems-link"
                >
                  查看無法匯入與錯誤
                </button>
              )}
              {showMissing && (
                <button
                  type="button"
                  onClick={() => {
                    onDismiss();
                    navigate({ to: '/library', search: { generate: true } });
                  }}
                  className="min-h-11 text-xs font-medium text-[var(--accent-text)] underline-offset-2 hover:underline"
                  data-testid="generate-subtitles-link"
                >
                  產生字幕 →
                </button>
              )}
            </div>
          )}

          {/* Auto-dismiss countdown.
              ⚖️ Alexyu 2026-08-27: the bar, but deliberately NO pause-on-touch.
              「當我按下掃描媒體庫之後，我不希望畫面一直停留在那個地方不動」— on a
              phone the sheet must leave on its own, so the bar's job is to make
              the leaving PREDICTABLE, not preventable. That is the opposite of
              the desktop card, where hover pauses it: a phone has no hover, and
              a touch-to-pause would trade「擋住你」for「留住門」when neither
              should be given up. The door is kept instead by giving 產生字幕 a
              permanent home on the homepage readout band.
              No isAutoDismissing state needed: this whole return branch IS the
              auto-dismissing state, so the bar mounts exactly when the timer
              starts. Duration comes from the same constant for the same reason
              as the desktop card — they must not drift. */}
          <div className="h-0.5 w-full overflow-hidden rounded-full bg-[var(--bg-tertiary)]">
            <div
              className="h-full origin-left animate-countdown bg-[var(--text-muted)] motion-reduce:animate-none"
              style={{ animationDuration: `${AUTO_DISMISS_MS}ms` }}
              data-testid="sheet-auto-dismiss-bar"
            />
          </div>
        </div>
      </div>
    );
  }

  // Peek state (collapsed)
  if (!expanded) {
    return (
      <button
        type="button"
        onClick={() => setExpanded(true)}
        onTouchStart={handleTouchStart}
        onTouchEnd={handleTouchEnd}
        className="flex h-16 w-full items-center gap-3 rounded-t-[var(--radius-xl)] bg-[var(--bg-primary)] px-4 shadow-[var(--shadow-xl)]"
        data-testid="scan-progress-sheet"
        aria-label="展開掃描進度"
      >
        <Loader className="h-4 w-4 animate-spin text-[var(--accent-text)]" />
        <span className="text-sm font-medium text-[var(--text-primary)]">
          掃描中 {state.percentDone}%
        </span>
        <span className="text-xs text-[var(--text-secondary)]">
          {state.filesFound.toLocaleString()} 檔案
        </span>
      </button>
    );
  }

  // Expanded state — E2-M
  return (
    <div
      className="w-full rounded-t-[var(--radius-xl)] bg-[var(--bg-primary)] shadow-[var(--shadow-xl)]"
      data-testid="scan-progress-sheet"
      onTouchStart={handleTouchStart}
      onTouchEnd={handleTouchEnd}
      role="status"
    >
      {/* Drag handle */}
      <div className="flex justify-center pt-3">
        <div
          className="h-1 w-10 rounded-full bg-[var(--text-muted)]"
          data-testid="sheet-drag-handle"
        />
      </div>

      <div className="flex flex-col gap-4 px-5 pb-6 pt-4">
        <p className="text-base font-semibold text-[var(--text-primary)]">媒體庫掃描中</p>

        {/* Progress bar + percent */}
        <div className="flex flex-col gap-2">
          <div className="h-1.5 w-full overflow-hidden rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)]">
            <div
              className="h-full rounded-[var(--radius-sm)] bg-[var(--accent-primary)] transition-[width] duration-[var(--motion-move)]"
              style={{ width: `${state.percentDone}%` }}
              data-testid="sheet-progress-bar"
            />
          </div>
          <span className="font-mono text-sm font-semibold tabular-nums text-[var(--text-primary)]">
            {state.percentDone}%
          </span>
        </div>

        {/* 找到 · 解析 · 錯誤 — no 比對 counter, see ScanProgressCard. */}
        <ScanStats state={state} />

        {state.estimatedTime && (
          <p className="text-xs text-[var(--text-muted)]">預估剩餘：{state.estimatedTime}</p>
        )}

        {/* Cancel */}
        {showCancelConfirm ? (
          <div
            className="rounded-[var(--radius-md)] bg-[var(--bg-secondary)] p-3"
            data-testid="sheet-cancel-confirm"
          >
            <p className="mb-3 text-sm text-[var(--text-secondary)]">
              確定要取消掃描嗎？已處理的結果會保留。
            </p>
            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setShowCancelConfirm(false)}
                className="min-h-11 rounded-[var(--radius-md)] px-3 text-sm text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)]"
              >
                繼續掃描
              </button>
              <button
                type="button"
                onClick={handleCancelConfirm}
                disabled={isCancelling}
                className="min-h-11 rounded-[var(--radius-md)] bg-[var(--error)] px-3 text-sm text-[var(--text-on-scrim)] hover:bg-[var(--error-pressed)] disabled:opacity-50"
                data-testid="sheet-cancel-confirm-btn"
              >
                {isCancelling ? '取消中...' : '取消掃描'}
              </button>
            </div>
          </div>
        ) : (
          <div className="flex justify-center">
            <button
              type="button"
              onClick={() => setShowCancelConfirm(true)}
              className="min-h-11 rounded-[var(--radius-md)] border border-[var(--border-subtle)] px-5 text-sm font-medium text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)] active:bg-[var(--bg-tertiary)]"
              data-testid="sheet-cancel-btn"
            >
              取消掃描
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
