/**
 * Desktop floating scan progress card (Story 7.4, Tasks 1+3)
 * Fixed bottom-right, z-50, 400px width, dark theme.
 * Shows progress bar, stats, current file, ETA, minimize/expand, cancel, completion summary.
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
  Minus,
  X,
  ChevronUp,
} from 'lucide-react';
import { cn } from '../../lib/utils';
import type { ScanProgressState } from '../../hooks/useScanProgress';

const AUTO_DISMISS_MS = 10000;

export interface ScanProgressCardProps {
  state: ScanProgressState;
  onCancel: () => void;
  onToggleMinimize: () => void;
  onDismiss: () => void;
  isCancelling?: boolean;
}

export function ScanProgressCard({
  state,
  onCancel,
  onToggleMinimize,
  onDismiss,
  isCancelling = false,
}: ScanProgressCardProps) {
  const navigate = useNavigate();
  const [showCancelConfirm, setShowCancelConfirm] = useState(false);
  const [isAutoDismissing, setIsAutoDismissing] = useState(false);
  const [isPaused, setIsPaused] = useState(false);
  const autoDismissTimerRef = useRef<ReturnType<typeof setTimeout>>();

  const clearAutoDismiss = useCallback(() => {
    if (autoDismissTimerRef.current) clearTimeout(autoDismissTimerRef.current);
    setIsAutoDismissing(false);
    setIsPaused(false);
  }, []);

  // Auto-dismiss on completion
  useEffect(() => {
    if (state.isComplete || state.isCancelled) {
      setIsAutoDismissing(true);

      autoDismissTimerRef.current = setTimeout(() => {
        clearAutoDismiss();
        onDismiss();
      }, AUTO_DISMISS_MS);
    } else {
      clearAutoDismiss();
    }

    return clearAutoDismiss;
  }, [state.isComplete, state.isCancelled, onDismiss, clearAutoDismiss]);

  const handleMouseEnter = () => {
    if (state.isComplete || state.isCancelled) {
      clearAutoDismiss();
      setIsPaused(true);
    }
  };

  const handleMouseLeave = () => {
    if (isPaused) {
      setIsPaused(false);
    }
  };

  const handleCancelClick = () => {
    setShowCancelConfirm(true);
  };

  const handleCancelConfirm = () => {
    setShowCancelConfirm(false);
    onCancel();
  };

  // Minimized pill
  if (state.isMinimized && state.isScanning) {
    return (
      <button
        type="button"
        onClick={onToggleMinimize}
        className="flex items-center gap-2 rounded-full bg-[var(--bg-secondary)] px-4 py-2 shadow-lg"
        data-testid="scan-progress-pill"
      >
        <Loader className="h-4 w-4 animate-spin text-[var(--accent-primary)]" />
        <span className="text-sm font-medium text-[var(--text-primary)]">
          掃描中 {state.percentDone}%
        </span>
        <ChevronUp className="h-3.5 w-3.5 text-[var(--text-secondary)]" />
      </button>
    );
  }

  // Completion/cancelled summary — H3 spec: 480px max-width, radius-lg
  if (state.isComplete || state.isCancelled) {
    return (
      <div
        className="w-[480px] max-w-[calc(100vw-2rem)] rounded-lg bg-[var(--bg-secondary)] p-4 shadow-xl"
        data-testid="scan-progress-card"
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
        role="status"
      >
        {/* Header */}
        <div className="mb-3 flex items-center justify-between">
          <div className="flex items-center gap-2">
            {state.isCancelled ? (
              <XCircle className="h-5 w-5 text-[var(--text-secondary)]" />
            ) : state.errorCount > 0 ? (
              <AlertTriangle className="h-5 w-5 text-[var(--warning)]" />
            ) : (
              <CheckCircle className="h-5 w-5 text-[var(--success)]" />
            )}
            <span className="text-sm font-semibold text-[var(--text-primary)]">
              {state.isCancelled ? '掃描已取消' : '掃描完成'}
            </span>
          </div>
          <button
            type="button"
            onClick={onDismiss}
            className="rounded p-1 text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)]"
            aria-label="關閉"
            data-testid="scan-dismiss-btn"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Stats summary */}
        <p className="mb-3 text-sm text-[var(--text-secondary)]">
          <span className="font-mono text-[var(--text-primary)]">
            {state.filesFound.toLocaleString()}
          </span>{' '}
          檔案 · 比對成功{' '}
          <span className="font-mono text-[var(--text-primary)]">
            {state.filesProcessed.toLocaleString()}
          </span>{' '}
          · 未比對{' '}
          <span className="font-mono text-[var(--text-primary)]">
            {Math.max(0, state.filesFound - state.filesProcessed).toLocaleString()}
          </span>{' '}
          · 錯誤{' '}
          <span className="font-mono text-[var(--text-primary)]">
            {state.errorCount.toLocaleString()}
          </span>
        </p>

        {/* Action links */}
        <div className="flex gap-4">
          <button
            type="button"
            onClick={() => {
              onDismiss();
              navigate({ to: '/', search: { status: 'unmatched' } });
            }}
            className="text-sm text-[var(--accent-primary)] underline-offset-2 hover:underline"
            data-testid="view-unmatched-link"
          >
            查看未比對項目
          </button>
          {state.errorCount > 0 && (
            <button
              type="button"
              onClick={() => {
                onDismiss();
                navigate({ to: '/', search: { status: 'error' } });
              }}
              className="text-sm text-[var(--accent-primary)] underline-offset-2 hover:underline"
              data-testid="view-errors-link"
            >
              查看錯誤
            </button>
          )}
        </div>

        {/* Auto-dismiss progress bar */}
        <div className="mt-3 h-0.5 w-full overflow-hidden rounded-full bg-[var(--bg-tertiary)]">
          <div
            className={cn(
              'h-full bg-[var(--text-muted)] motion-reduce:animate-none',
              isAutoDismissing && 'animate-shrink'
            )}
            style={isPaused ? { animationPlayState: 'paused' } : undefined}
            data-testid="auto-dismiss-bar"
          />
        </div>
      </div>
    );
  }

  // Active scanning state
  return (
    <div
      className="w-[400px] rounded-xl bg-[var(--bg-secondary)] p-4 shadow-xl"
      data-testid="scan-progress-card"
      role="status"
    >
      {/* Header */}
      <div className="mb-3 flex items-center justify-between">
        <span className="text-sm font-semibold text-[var(--text-primary)]">媒體庫掃描中</span>
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={onToggleMinimize}
            className="rounded p-1 text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)]"
            aria-label="最小化"
            data-testid="scan-minimize-btn"
          >
            <Minus className="h-4 w-4" />
          </button>
          <button
            type="button"
            onClick={handleCancelClick}
            className="rounded p-1 text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)]"
            aria-label="取消"
            data-testid="scan-close-btn"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Progress bar */}
      <div className="mb-3">
        <div className="mb-1 flex items-center justify-between">
          <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-[var(--bg-tertiary)]">
            <div
              className="h-full rounded-full bg-[var(--accent-primary)] transition-[width] duration-300"
              style={{ width: `${state.percentDone}%` }}
              data-testid="scan-progress-bar"
            />
          </div>
          <span className="ml-3 min-w-[3ch] text-right font-mono text-sm text-[var(--text-primary)]">
            {state.percentDone}%
          </span>
        </div>
      </div>

      {/* Stats row — 4 counters per design spec H2 */}
      <div className="mb-3 flex items-center gap-3 text-xs text-[var(--text-secondary)]">
        <span className="flex items-center gap-1">
          <File className="h-3.5 w-3.5" />
          <span>找到</span>
          <span className="font-mono text-[var(--text-primary)]">
            {state.filesFound.toLocaleString()}
          </span>
        </span>
        <span className="flex items-center gap-1">
          <FileCheck className="h-3.5 w-3.5" />
          <span>解析</span>
          <span className="font-mono text-[var(--text-primary)]">
            {state.filesProcessed.toLocaleString()}
          </span>
        </span>
        <span className="flex items-center gap-1">
          <Link className="h-3.5 w-3.5" />
          <span>比對</span>
          <span className="font-mono text-[var(--text-primary)]">
            {state.filesProcessed.toLocaleString()}
          </span>
        </span>
        <span className="flex items-center gap-1">
          <AlertTriangle
            className={cn('h-3.5 w-3.5', state.errorCount > 0 && 'text-[var(--error)]')}
          />
          <span>錯誤</span>
          <span
            className={cn(
              'font-mono',
              state.errorCount > 0 ? 'text-[var(--error)]' : 'text-[var(--text-primary)]'
            )}
          >
            {state.errorCount}
          </span>
        </span>
      </div>

      {/* Current file */}
      {state.currentFile && (
        <div className="mb-3">
          <p className="text-xs text-[var(--text-muted)]">正在處理:</p>
          <p
            className="truncate font-mono text-xs text-[var(--text-secondary)]"
            title={state.currentFile}
            data-testid="scan-current-file"
          >
            {state.currentFile}
          </p>
        </div>
      )}

      {/* ETA */}
      {state.estimatedTime && (
        <p className="mb-3 text-xs text-[var(--text-muted)]" data-testid="scan-eta">
          預估剩餘: {state.estimatedTime}
        </p>
      )}

      {/* Cancel button / Cancel confirmation */}
      {showCancelConfirm ? (
        <div className="rounded-lg bg-[var(--bg-primary)] p-3" data-testid="cancel-confirm-dialog">
          <p className="mb-3 text-sm text-[var(--text-secondary)]">
            確定要取消掃描嗎？已處理的結果會保留。
          </p>
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={() => setShowCancelConfirm(false)}
              className="rounded-md px-3 py-1.5 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)]"
              data-testid="cancel-continue-btn"
            >
              繼續掃描
            </button>
            <button
              type="button"
              onClick={handleCancelConfirm}
              disabled={isCancelling}
              className="rounded-md bg-[var(--error)] px-3 py-1.5 text-sm text-white transition-colors hover:bg-red-700 disabled:opacity-50"
              data-testid="cancel-confirm-btn"
            >
              {isCancelling ? '取消中...' : '取消掃描'}
            </button>
          </div>
        </div>
      ) : (
        <div className="flex justify-center">
          <button
            type="button"
            onClick={handleCancelClick}
            className="text-sm text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)]"
            data-testid="scan-cancel-btn"
          >
            取消掃描
          </button>
        </div>
      )}
    </div>
  );
}
