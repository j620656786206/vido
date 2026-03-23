/**
 * Mobile bottom sheet scan progress (Story 7.4, Task 4)
 * Peek state: 64px, full width. Expanded: half screen with drag handle.
 */

import { useState, useEffect, useRef, useCallback } from 'react';
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
}

export function ScanProgressSheet({
  state,
  onCancel,
  onDismiss,
  isCancelling = false,
}: ScanProgressSheetProps) {
  const [expanded, setExpanded] = useState(false);
  const [showCancelConfirm, setShowCancelConfirm] = useState(false);
  const autoDismissTimerRef = useRef<ReturnType<typeof setTimeout>>();
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
        className="w-full rounded-t-xl bg-slate-800 p-4 shadow-xl"
        data-testid="scan-progress-sheet"
        role="status"
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            {state.isCancelled ? (
              <XCircle className="h-5 w-5 text-slate-400" />
            ) : state.errorCount > 0 ? (
              <AlertTriangle className="h-5 w-5 text-yellow-400" />
            ) : (
              <CheckCircle className="h-5 w-5 text-green-400" />
            )}
            <span className="text-sm font-semibold text-slate-100">
              {state.isCancelled ? '掃描已取消' : '掃描完成'}
            </span>
          </div>
          <button
            type="button"
            onClick={onDismiss}
            className="rounded p-1 text-slate-400 hover:text-slate-200"
            aria-label="關閉"
            data-testid="sheet-dismiss-btn"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        <p className="mt-2 text-xs text-slate-400">
          {state.filesFound.toLocaleString()} 檔案 · 錯誤 {state.errorCount}
        </p>
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
        className="flex h-16 w-full items-center gap-3 rounded-t-xl bg-slate-800 px-4 shadow-xl"
        data-testid="scan-progress-sheet"
        aria-label="展開掃描進度"
      >
        <Loader className="h-4 w-4 animate-spin text-blue-400" />
        <span className="text-sm font-medium text-slate-200">掃描中 {state.percentDone}%</span>
        <span className="text-xs text-slate-400">{state.filesFound.toLocaleString()} 檔案</span>
      </button>
    );
  }

  // Expanded state
  return (
    <div
      className="w-full rounded-t-xl bg-slate-800 shadow-xl"
      data-testid="scan-progress-sheet"
      onTouchStart={handleTouchStart}
      onTouchEnd={handleTouchEnd}
      role="status"
    >
      {/* Drag handle */}
      <div className="flex justify-center pb-2 pt-3">
        <div className="h-1 w-10 rounded-full bg-slate-600" data-testid="sheet-drag-handle" />
      </div>

      <div className="px-4 pb-4">
        {/* Header */}
        <p className="mb-3 text-sm font-semibold text-slate-100">媒體庫掃描中</p>

        {/* Progress bar */}
        <div className="mb-3 flex items-center gap-3">
          <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-slate-700">
            <div
              className="h-full rounded-full bg-blue-500 transition-[width] duration-300"
              style={{ width: `${state.percentDone}%` }}
              data-testid="sheet-progress-bar"
            />
          </div>
          <span className="min-w-[3ch] text-right font-mono text-sm text-slate-200">
            {state.percentDone}%
          </span>
        </div>

        {/* Stats — two rows for narrow viewport (4 counters per design H5) */}
        <div className="mb-3 grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-slate-400">
          <span className="flex items-center gap-1">
            <File className="h-3.5 w-3.5" />
            找到{' '}
            <span className="font-mono text-slate-200">{state.filesFound.toLocaleString()}</span>
          </span>
          <span className="flex items-center gap-1">
            <FileCheck className="h-3.5 w-3.5" />
            解析{' '}
            <span className="font-mono text-slate-200">
              {state.filesProcessed.toLocaleString()}
            </span>
          </span>
          <span className="flex items-center gap-1">
            <Link className="h-3.5 w-3.5" />
            比對{' '}
            <span className="font-mono text-slate-200">
              {state.filesProcessed.toLocaleString()}
            </span>
          </span>
          <span className="flex items-center gap-1">
            <AlertTriangle className={cn('h-3.5 w-3.5', state.errorCount > 0 && 'text-red-400')} />
            錯誤{' '}
            <span
              className={cn('font-mono', state.errorCount > 0 ? 'text-red-400' : 'text-slate-200')}
            >
              {state.errorCount}
            </span>
          </span>
        </div>

        {/* ETA */}
        {state.estimatedTime && (
          <p className="mb-3 text-xs text-slate-500">預估剩餘: {state.estimatedTime}</p>
        )}

        {/* Cancel */}
        {showCancelConfirm ? (
          <div className="rounded-lg bg-slate-900 p-3" data-testid="sheet-cancel-confirm">
            <p className="mb-3 text-sm text-slate-300">確定要取消掃描嗎？已處理的結果會保留。</p>
            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setShowCancelConfirm(false)}
                className="rounded-md px-3 py-1.5 text-sm text-slate-300 hover:bg-slate-700"
              >
                繼續掃描
              </button>
              <button
                type="button"
                onClick={handleCancelConfirm}
                disabled={isCancelling}
                className="rounded-md bg-red-600 px-3 py-1.5 text-sm text-white hover:bg-red-700 disabled:opacity-50"
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
              className="text-sm text-slate-400 hover:text-slate-200"
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
