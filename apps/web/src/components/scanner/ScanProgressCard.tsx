// Design ref: ux-design.pen Screen E2-D (wyuhF) · E3-D (szzaW)
/**
 * Desktop scan progress (Story 7.4, Tasks 1+3).
 * Running: floating card, 400px, bottom-right (E2-D).
 * Complete / cancelled: summary toast, 480px, top-centre (E3-D).
 * Placement is ScanProgress.tsx's job; this component only draws the card.
 */

import { useState, useEffect, useRef, useCallback } from 'react';
import { useNavigate } from '@tanstack/react-router';
import {
  Loader,
  File,
  FileCheck,
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
  /**
   * sub-4-3 F17: 缺繁中字幕 count for the completion toast (prop-driven — the
   * container fetches it from the frozen preview endpoint). undefined/0 → the
   * line AND the 產生字幕 link stay hidden. The copy never implies the scan
   * itself generates anything.
   */
  missingSubtitleCount?: number;
}

/**
 * Where 查看無法匯入與錯誤 lands (dsr-5, Alexyu 2026-09-14「全部說真話」).
 * The toast used to offer 查看未比對項目 and 查看錯誤, both navigating to `/`
 * with a `status` param the homepage never reads. Worse, its「未比對」count
 * was files the scanner could NOT IMPORT (no episode number), which by
 * definition are not in the library — so no library filter could ever show
 * them. The scanner logs each one (SCANNER_UNMATCHED) and each error to the
 * system log, which is the one place both can be read.
 */
export const SCAN_PROBLEMS_DESTINATION = { to: '/settings/logs' } as const;

/**
 * 「找到 1,247 檔案 · 新增 30 · 更新 1,168 · 無法匯入 42 · 錯誤 7」— only numbers
 * the scanner reported. 新增／更新 are omitted when unknown (polling fallback).
 * There is no 比對成功: matching against TMDb happens after the scan, and the
 * old figure was found − unmatched, which counted errors as successes.
 */
export function scanSummaryParts(state: ScanProgressState): string[] {
  const n = (value: number) => value.toLocaleString();
  const parts = [`找到 ${n(state.filesFound)} 檔案`];
  if (state.isCancelled) {
    parts.push(`錯誤 ${n(state.errorCount)}`);
    return parts;
  }
  if (state.filesCreated !== undefined) parts.push(`新增 ${n(state.filesCreated)}`);
  if (state.filesUpdated !== undefined) parts.push(`更新 ${n(state.filesUpdated)}`);
  parts.push(`無法匯入 ${n(state.filesUnmatched ?? 0)}`);
  parts.push(`錯誤 ${n(state.errorCount)}`);
  return parts;
}

/** Something the user asked for did not happen: files not imported, or errors. */
export function scanHadProblems(state: ScanProgressState): boolean {
  return state.errorCount > 0 || (!state.isCancelled && (state.filesUnmatched ?? 0) > 0);
}

export function ScanProgressCard({
  state,
  onCancel,
  onToggleMinimize,
  onDismiss,
  missingSubtitleCount,
  isCancelling = false,
}: ScanProgressCardProps) {
  const navigate = useNavigate();
  const [showCancelConfirm, setShowCancelConfirm] = useState(false);
  const [isAutoDismissing, setIsAutoDismissing] = useState(false);
  const [isPaused, setIsPaused] = useState(false);
  // Bumped when the countdown restarts, so the bar remounts at full width.
  const [countdownCycle, setCountdownCycle] = useState(0);
  const autoDismissTimerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  const clearAutoDismiss = useCallback(() => {
    if (autoDismissTimerRef.current) clearTimeout(autoDismissTimerRef.current);
    setIsAutoDismissing(false);
    setIsPaused(false);
  }, []);

  const startAutoDismiss = useCallback(() => {
    if (autoDismissTimerRef.current) clearTimeout(autoDismissTimerRef.current);
    setIsAutoDismissing(true);
    autoDismissTimerRef.current = setTimeout(() => {
      clearAutoDismiss();
      onDismiss();
    }, AUTO_DISMISS_MS);
  }, [clearAutoDismiss, onDismiss]);

  // Auto-dismiss on completion
  useEffect(() => {
    if (state.isComplete || state.isCancelled) {
      startAutoDismiss();
    } else {
      clearAutoDismiss();
    }

    return clearAutoDismiss;
  }, [state.isComplete, state.isCancelled, startAutoDismiss, clearAutoDismiss]);

  const handleMouseEnter = () => {
    if (state.isComplete || state.isCancelled) {
      clearAutoDismiss();
      setIsPaused(true);
    }
  };

  // Hover pauses the countdown; leaving starts a fresh one. Before dsr-5 leaving
  // only cleared the paused flag, so a card you had pointed at once never went
  // away on its own.
  const handleMouseLeave = () => {
    if (isPaused) {
      setIsPaused(false);
      setCountdownCycle((cycle) => cycle + 1);
      startAutoDismiss();
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
        className="flex items-center gap-2 rounded-full border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-4 py-2 shadow-[var(--shadow-lg)]"
        data-testid="scan-progress-pill"
      >
        <Loader className="h-4 w-4 animate-spin text-[var(--accent-text)]" />
        <span className="text-sm font-medium text-[var(--text-primary)]">
          掃描中 {state.percentDone}%
        </span>
        <ChevronUp className="h-3.5 w-3.5 text-[var(--text-secondary)]" />
      </button>
    );
  }

  // Completion/cancelled summary — E3-D: 480px, radius-lg, hairline, floats (--shadow-lg)
  if (state.isComplete || state.isCancelled) {
    const hadProblems = scanHadProblems(state);
    const showMissing =
      !state.isCancelled && missingSubtitleCount !== undefined && missingSubtitleCount > 0;
    return (
      <div
        className="flex w-[480px] max-w-[calc(100vw-2rem)] flex-col gap-3 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-4 shadow-[var(--shadow-lg)]"
        data-testid="scan-progress-card"
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
        role="status"
      >
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            {state.isCancelled ? (
              <XCircle className="h-[18px] w-[18px] text-[var(--text-secondary)]" />
            ) : hadProblems ? (
              <AlertTriangle className="h-[18px] w-[18px] text-[var(--warning-text)]" />
            ) : (
              <CheckCircle className="h-[18px] w-[18px] text-[var(--success-text)]" />
            )}
            <span className="text-sm font-semibold text-[var(--text-primary)]">
              {state.isCancelled ? '掃描已取消' : '掃描完成'}
            </span>
          </div>
          <button
            type="button"
            onClick={onDismiss}
            className="rounded-[var(--radius-sm)] p-1 text-[var(--text-muted)] transition-colors hover:text-[var(--text-primary)]"
            aria-label="關閉"
            data-testid="scan-dismiss-btn"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Stats summary */}
        <p className="text-sm tabular-nums text-[var(--text-secondary)]" data-testid="scan-summary">
          {scanSummaryParts(state).join(' · ')}
        </p>

        {/* sub-4-3 F17: missing-subtitle line (only when a count is known and > 0). */}
        {showMissing && (
          <p
            data-testid="scan-missing-subtitle-line"
            className="flex items-center gap-[3px] text-sm text-[var(--text-secondary)]"
          >
            <span className="font-mono tabular-nums">{missingSubtitleCount.toLocaleString()}</span>
            部影片缺繁中字幕
          </p>
        )}

        {/* Action links */}
        {(hadProblems || showMissing) && (
          <div className="flex gap-4">
            {hadProblems && (
              <button
                type="button"
                onClick={() => {
                  onDismiss();
                  navigate(SCAN_PROBLEMS_DESTINATION);
                }}
                className="text-sm font-medium text-[var(--accent-text)] underline-offset-2 hover:underline"
                data-testid="view-scan-problems-link"
              >
                查看無法匯入與錯誤
              </button>
            )}
            {showMissing && (
              <button
                type="button"
                onClick={() => {
                  onDismiss();
                  // F17 deep link → the consent flow (library route, Rule 26-safe param).
                  navigate({ to: '/library', search: { generate: true } });
                }}
                className="text-sm font-medium text-[var(--accent-text)] underline-offset-2 hover:underline"
                data-testid="generate-subtitles-link"
              >
                產生字幕 →
              </button>
            )}
          </div>
        )}

        {/* Auto-dismiss progress bar.
            This card destroys itself after AUTO_DISMISS_MS while the reader may
            still be deciding what to open, and this 2px bar is the ONLY warning
            it gives — the copy never says the card is on a timer. It was frozen
            at full width from the day it shipped, because `animate-shrink` was
            declared in tailwind.config.js and Tailwind v4 never loads that file.
            So the one honest signal read「時間還很多」right up to the moment the
            card vanished.
            The duration is set inline from AUTO_DISMISS_MS rather than baked
            into the keyframe: the bar and the setTimeout must not be able to
            drift apart. Units are mandatory — a bare number in an animation
            shorthand parses as an iteration COUNT.
            `motion-reduce:animate-none` stays and is load-bearing: `forwards`
            plus the global 1ms clamp would otherwise empty the bar instantly
            and then sit at zero for the remaining ten seconds, which is a
            worse lie than the frozen one.
            Neutral, not gold: a countdown is not a job running (固定詞彙). */}
        <div className="h-0.5 w-full overflow-hidden rounded-full bg-[var(--bg-tertiary)]">
          <div
            key={countdownCycle}
            className={cn(
              'h-full origin-left bg-[var(--text-muted)] motion-reduce:animate-none',
              isAutoDismissing && 'animate-countdown'
            )}
            style={{
              animationDuration: `${AUTO_DISMISS_MS}ms`,
              ...(isPaused ? { animationPlayState: 'paused' } : {}),
            }}
            data-testid="auto-dismiss-bar"
          />
        </div>
      </div>
    );
  }

  // Active scanning state — E2-D
  return (
    <div
      className="flex w-[400px] flex-col gap-4 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-5 shadow-[var(--shadow-lg)]"
      data-testid="scan-progress-card"
      role="status"
    >
      {/* Header */}
      <div className="flex items-center justify-between">
        <span className="text-sm font-semibold text-[var(--text-primary)]">媒體庫掃描中</span>
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={onToggleMinimize}
            className="rounded-[var(--radius-sm)] p-1 text-[var(--text-muted)] transition-colors hover:text-[var(--text-primary)]"
            aria-label="最小化"
            data-testid="scan-minimize-btn"
          >
            <Minus className="h-4 w-4" />
          </button>
          <button
            type="button"
            onClick={handleCancelClick}
            className="rounded-[var(--radius-sm)] p-1 text-[var(--text-muted)] transition-colors hover:text-[var(--text-primary)]"
            aria-label="取消"
            data-testid="scan-close-btn"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Progress bar + percent */}
      <div className="flex flex-col gap-2">
        <div className="h-1.5 w-full overflow-hidden rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)]">
          <div
            className="h-full rounded-[var(--radius-sm)] bg-[var(--accent-primary)] transition-[width] duration-[var(--motion-move)]"
            style={{ width: `${state.percentDone}%` }}
            data-testid="scan-progress-bar"
          />
        </div>
        <span className="self-end font-mono text-sm font-semibold tabular-nums text-[var(--text-primary)]">
          {state.percentDone}%
        </span>
      </div>

      {/* Stats row. There is no 比對 counter (dsr-5): matching against TMDb
          happens after the scan, the scanner never reports a matched count, and
          the card used to print filesProcessed a second time under that label —
          a number that looked like a result but was a copy of its neighbour. */}
      <ScanStats state={state} />

      {/* Current file */}
      {state.currentFile && (
        <div className="flex flex-col gap-1">
          <p className="text-xs text-[var(--text-muted)]">正在處理：</p>
          <p
            className="truncate font-mono text-xs text-[var(--text-muted)]"
            title={state.currentFile}
            data-testid="scan-current-file"
          >
            {state.currentFile}
          </p>
        </div>
      )}

      {/* ETA */}
      {state.estimatedTime && (
        <p className="text-xs text-[var(--text-muted)]" data-testid="scan-eta">
          預估剩餘：{state.estimatedTime}
        </p>
      )}

      {/* Cancel button / Cancel confirmation */}
      {showCancelConfirm ? (
        <div
          className="rounded-[var(--radius-md)] bg-[var(--bg-primary)] p-3"
          data-testid="cancel-confirm-dialog"
        >
          <p className="mb-3 text-sm text-[var(--text-secondary)]">
            確定要取消掃描嗎？已處理的結果會保留。
          </p>
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={() => setShowCancelConfirm(false)}
              className="rounded-[var(--radius-md)] px-3 py-1.5 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)]"
              data-testid="cancel-continue-btn"
            >
              繼續掃描
            </button>
            <button
              type="button"
              onClick={handleCancelConfirm}
              disabled={isCancelling}
              className="rounded-[var(--radius-md)] bg-[var(--error)] px-3 py-1.5 text-sm text-[var(--text-on-scrim)] transition-colors hover:bg-[var(--error-pressed)] disabled:opacity-50"
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
            className="rounded-[var(--radius-md)] border border-[var(--border-subtle)] px-4 py-2 text-sm font-medium text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)] active:bg-[var(--bg-tertiary)]"
            data-testid="scan-cancel-btn"
          >
            取消掃描
          </button>
        </div>
      )}
    </div>
  );
}

/** 找到 · 解析 · 錯誤 — shared by the desktop card and the mobile sheet. */
export function ScanStats({ state }: { state: ScanProgressState }) {
  const hasErrors = state.errorCount > 0;
  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs tabular-nums text-[var(--text-secondary)]">
      <span className="flex items-center gap-1">
        <File className="h-3.5 w-3.5 text-[var(--text-muted)]" aria-hidden="true" />
        找到 {state.filesFound.toLocaleString()}
      </span>
      <span className="text-[var(--text-muted)]" aria-hidden="true">
        ·
      </span>
      <span className="flex items-center gap-1">
        <FileCheck className="h-3.5 w-3.5 text-[var(--text-muted)]" aria-hidden="true" />
        解析 {state.filesProcessed.toLocaleString()}
      </span>
      <span className="text-[var(--text-muted)]" aria-hidden="true">
        ·
      </span>
      <span
        className={cn('flex items-center gap-1', hasErrors && 'text-[var(--error-text)]')}
        data-testid="scan-error-stat"
      >
        <AlertTriangle
          className={cn(
            'h-3.5 w-3.5',
            hasErrors ? 'text-[var(--error-text)]' : 'text-[var(--text-muted)]'
          )}
          aria-hidden="true"
        />
        錯誤 {state.errorCount.toLocaleString()}
      </span>
    </div>
  );
}
