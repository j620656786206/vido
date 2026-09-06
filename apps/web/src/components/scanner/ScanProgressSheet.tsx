// Design ref: ux-design.pen Screen H5 Scan Progress Mobile (yezIo)
/**
 * Mobile bottom sheet scan progress (Story 7.4, Task 4)
 * Peek state: 64px, full width. Expanded: half screen with drag handle.
 */

import { useState, useEffect, useRef, useCallback } from 'react';
import { useNavigate } from '@tanstack/react-router';
import {
  Loader,
  File,
  FileCheck,
  Link,
  AlertTriangle,
  CheckCircle,
  XCircle,
  X,
} from 'lucide-react';
import { cn } from '../../lib/utils';
import type { ScanProgressState } from '../../hooks/useScanProgress';

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

  // Completion/cancelled toast
  if (state.isComplete || state.isCancelled) {
    return (
      <div
        className="w-full rounded-t-xl bg-[var(--bg-secondary)] p-4 shadow-xl"
        data-testid="scan-progress-sheet"
        role="status"
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            {state.isCancelled ? (
              <XCircle className="h-5 w-5 text-[var(--text-secondary)]" />
            ) : state.errorCount > 0 ? (
              <AlertTriangle className="h-5 w-5 text-[var(--warning-text)]" />
            ) : (
              <CheckCircle className="h-5 w-5 text-[var(--success-text)]" />
            )}
            <span className="text-sm font-semibold text-[var(--text-primary)]">
              {state.isCancelled ? '掃描已取消' : '掃描完成'}
            </span>
          </div>
          <button
            type="button"
            onClick={onDismiss}
            className="rounded p-1 text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
            aria-label="關閉"
            data-testid="sheet-dismiss-btn"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        <p className="mt-2 text-xs text-[var(--text-secondary)]">
          {state.filesFound.toLocaleString()} 檔案 · 錯誤 {state.errorCount}
        </p>
        {!state.isCancelled && missingSubtitleCount !== undefined && missingSubtitleCount > 0 && (
          <div className="mt-2 flex items-center justify-between">
            <p
              data-testid="scan-missing-subtitle-line"
              className="flex items-center gap-[3px] text-xs text-[var(--text-secondary)]"
            >
              <span className="font-mono tabular-nums">
                {missingSubtitleCount.toLocaleString()}
              </span>
              部影片缺繁中字幕
            </p>
            <button
              type="button"
              onClick={() => {
                onDismiss();
                navigate({ to: '/library', search: { generate: true } });
              }}
              className="text-xs text-[var(--accent-text)] underline-offset-2 hover:underline"
              data-testid="generate-subtitles-link"
            >
              產生字幕 →
            </button>
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
            auto-dismissing state, so the bar mounts exactly when the timer at
            :53 starts. Duration comes from the same constant for the same
            reason as the desktop card — they must not drift. */}
        <div className="mt-3 h-0.5 w-full overflow-hidden rounded-full bg-[var(--bg-tertiary)]">
          <div
            className="h-full origin-left animate-countdown bg-[var(--text-muted)] motion-reduce:animate-none"
            style={{ animationDuration: `${AUTO_DISMISS_MS}ms` }}
            data-testid="sheet-auto-dismiss-bar"
          />
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
        className="flex h-16 w-full items-center gap-3 rounded-t-xl bg-[var(--bg-secondary)] px-4 shadow-xl"
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

  // Expanded state
  return (
    <div
      className="w-full rounded-t-xl bg-[var(--bg-secondary)] shadow-xl"
      data-testid="scan-progress-sheet"
      onTouchStart={handleTouchStart}
      onTouchEnd={handleTouchEnd}
      role="status"
    >
      {/* Drag handle */}
      <div className="flex justify-center pb-2 pt-3">
        <div
          className="h-1 w-10 rounded-full bg-[var(--bg-tertiary)]"
          data-testid="sheet-drag-handle"
        />
      </div>

      <div className="px-4 pb-4">
        {/* Header */}
        <p className="mb-3 text-sm font-semibold text-[var(--text-primary)]">媒體庫掃描中</p>

        {/* Progress bar */}
        <div className="mb-3 flex items-center gap-3">
          <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-[var(--bg-tertiary)]">
            <div
              className="h-full rounded-full bg-[var(--accent-primary)] transition-[width] duration-[var(--motion-move)]"
              style={{ width: `${state.percentDone}%` }}
              data-testid="sheet-progress-bar"
            />
          </div>
          <span className="min-w-[3ch] text-right font-mono text-sm text-[var(--text-primary)]">
            {state.percentDone}%
          </span>
        </div>

        {/* Stats — two rows for narrow viewport (4 counters per design H5) */}
        <div className="mb-3 grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-[var(--text-secondary)]">
          <span className="flex items-center gap-1">
            <File className="h-3.5 w-3.5" />
            找到{' '}
            <span className="font-mono text-[var(--text-primary)]">
              {state.filesFound.toLocaleString()}
            </span>
          </span>
          <span className="flex items-center gap-1">
            <FileCheck className="h-3.5 w-3.5" />
            解析{' '}
            <span className="font-mono text-[var(--text-primary)]">
              {state.filesProcessed.toLocaleString()}
            </span>
          </span>
          <span className="flex items-center gap-1">
            <Link className="h-3.5 w-3.5" />
            比對{' '}
            <span className="font-mono text-[var(--text-primary)]">
              {state.filesProcessed.toLocaleString()}
            </span>
          </span>
          <span className="flex items-center gap-1">
            <AlertTriangle
              className={cn('h-3.5 w-3.5', state.errorCount > 0 && 'text-[var(--error-text)]')}
            />
            錯誤{' '}
            <span
              className={cn(
                'font-mono',
                state.errorCount > 0 ? 'text-[var(--error-text)]' : 'text-[var(--text-primary)]'
              )}
            >
              {state.errorCount}
            </span>
          </span>
        </div>

        {/* ETA */}
        {state.estimatedTime && (
          <p className="mb-3 text-xs text-[var(--text-muted)]">預估剩餘: {state.estimatedTime}</p>
        )}

        {/* Cancel */}
        {showCancelConfirm ? (
          <div className="rounded-lg bg-[var(--bg-primary)] p-3" data-testid="sheet-cancel-confirm">
            <p className="mb-3 text-sm text-[var(--text-secondary)]">
              確定要取消掃描嗎？已處理的結果會保留。
            </p>
            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setShowCancelConfirm(false)}
                className="rounded-md px-3 py-1.5 text-sm text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)]"
              >
                繼續掃描
              </button>
              <button
                type="button"
                onClick={handleCancelConfirm}
                disabled={isCancelling}
                className="rounded-md bg-[var(--error)] px-3 py-1.5 text-sm text-[var(--text-on-scrim)] hover:bg-[var(--error-pressed)] disabled:opacity-50"
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
              className="text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
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
