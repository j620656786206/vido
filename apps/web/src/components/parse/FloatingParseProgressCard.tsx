/**
 * FloatingParseProgressCard Component (Story 3.10 - Task 4)
 * Non-blocking floating card showing parse progress
 * AC2: Step Progress Indicators (UX-3)
 * AC4: Non-Blocking Progress Card
 */

import { useState, useEffect } from 'react';
import {
  Loader2,
  CheckCircle,
  XCircle,
  ChevronUp,
  ChevronDown,
  X,
} from 'lucide-react';
import { cn } from '../../lib/utils';
import { useParseProgress } from './useParseProgress';
import { LayeredProgressIndicator } from './LayeredProgressIndicator';
import { ErrorDetailsPanel } from './ErrorDetailsPanel';
import type { ParseResult, ParseProgress } from './types';

export interface FloatingParseProgressCardProps {
  /** Task ID to track */
  taskId: string;
  /** Called when user closes the card */
  onClose: () => void;
  /** Called when parse completes successfully */
  onComplete?: (result: ParseResult | undefined) => void;
  /** Called when user clicks manual search */
  onManualSearch?: () => void;
  /** Called when user clicks edit filename */
  onEditFilename?: () => void;
  /** Called when user clicks skip */
  onSkip?: () => void;
  /** Auto-dismiss delay after success (ms), 0 to disable */
  autoDismissDelay?: number;
  /** Additional CSS classes */
  className?: string;
}

/**
 * Floating progress card showing parse progress in bottom-right corner
 * Implements AC4: Non-Blocking Progress Card
 */
export function FloatingParseProgressCard({
  taskId,
  onClose,
  onComplete,
  onManualSearch,
  onEditFilename,
  onSkip,
  autoDismissDelay = 3000,
  className,
}: FloatingParseProgressCardProps) {
  const [isMinimized, setIsMinimized] = useState(false);
  const [showSuccess, setShowSuccess] = useState(false);

  const { progress, status, isConnected, error } = useParseProgress(taskId, {
    onParseCompleted: (data) => {
      setShowSuccess(true);
      onComplete?.(data.result || undefined);
    },
  });

  // Auto-dismiss after success
  useEffect(() => {
    if (status === 'success' && autoDismissDelay > 0) {
      const timer = setTimeout(() => {
        onClose();
      }, autoDismissDelay);
      return () => clearTimeout(timer);
    }
  }, [status, autoDismissDelay, onClose]);

  const isParsing = status === 'pending' && isConnected;
  const isSuccess = status === 'success';
  const isFailed = status === 'failed';

  return (
    <div
      className={cn(
        'fixed bottom-6 right-6 w-[420px] bg-slate-800 rounded-xl shadow-2xl',
        'border border-slate-700',
        'animate-in slide-in-from-right-5 duration-300',
        isMinimized && 'h-auto',
        className
      )}
      role="status"
      aria-live="polite"
      data-testid="floating-parse-progress-card"
    >
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-slate-700">
        <div className="flex items-center gap-2">
          {isParsing && (
            <Loader2 className="h-4 w-4 text-blue-500 animate-spin" />
          )}
          {isSuccess && (
            <CheckCircle className="h-4 w-4 text-green-500" />
          )}
          {isFailed && (
            <XCircle className="h-4 w-4 text-red-500" />
          )}
          <span className="font-medium text-white">
            {isParsing && '正在解析...'}
            {isSuccess && '✅ 解析完成！'}
            {isFailed && '❌ 解析失敗'}
            {!isConnected && !isSuccess && !isFailed && '連線中...'}
          </span>
        </div>
        <div className="flex items-center gap-1">
          <button
            onClick={() => setIsMinimized(!isMinimized)}
            className="p-1.5 rounded-lg hover:bg-slate-700 text-slate-400 hover:text-white transition-colors"
            aria-label={isMinimized ? '展開' : '縮小'}
            data-testid="minimize-button"
          >
            {isMinimized ? (
              <ChevronUp className="h-4 w-4" />
            ) : (
              <ChevronDown className="h-4 w-4" />
            )}
          </button>
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg hover:bg-slate-700 text-slate-400 hover:text-white transition-colors"
            aria-label="關閉進度卡片"
            data-testid="close-button"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Progress Content */}
      {!isMinimized && progress && (
        <div className="p-4 space-y-4">
          {/* Overall Progress Bar */}
          <div className="space-y-1.5">
            <div className="flex justify-between text-sm">
              <span className="text-slate-400">進度</span>
              <span className="text-white">{progress.percentage}%</span>
            </div>
            <div
              className="h-2 bg-slate-700 rounded-full overflow-hidden"
              role="progressbar"
              aria-valuenow={progress.percentage}
              aria-valuemin={0}
              aria-valuemax={100}
              aria-label={`解析進度: ${progress.percentage}%`}
            >
              <div
                className={cn(
                  'h-full rounded-full transition-all duration-500 ease-out',
                  isSuccess && 'bg-green-500',
                  isFailed && 'bg-red-500',
                  isParsing && 'bg-blue-500'
                )}
                style={{ width: `${progress.percentage}%` }}
              />
            </div>
          </div>

          {/* Layered Steps */}
          <LayeredProgressIndicator
            steps={progress.steps}
            currentStep={progress.currentStep}
          />

          {/* Filename */}
          <div className="text-sm text-slate-400 truncate" title={progress.filename}>
            檔案：{progress.filename}
          </div>

          {/* Error Details on Failure */}
          {isFailed && (
            <ErrorDetailsPanel
              steps={progress.steps}
              filename={progress.filename}
              onManualSearch={onManualSearch}
              onEditFilename={onEditFilename}
              onSkip={onSkip}
            />
          )}

          {/* Success Message */}
          {isSuccess && progress.result && (
            <div className="bg-green-500/10 rounded-lg p-3 text-sm">
              <div className="text-green-400 font-medium">
                {progress.result.title}
                {progress.result.year && ` (${progress.result.year})`}
              </div>
              {progress.result.metadataSource && (
                <div className="text-slate-400 text-xs mt-1">
                  來源：{getSourceDisplayName(progress.result.metadataSource)}
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* Minimized View */}
      {isMinimized && progress && (
        <div className="px-4 py-2 flex items-center justify-between text-sm">
          <span className="text-slate-400 truncate flex-1 mr-2">
            {progress.filename}
          </span>
          <span className="text-white font-medium">{progress.percentage}%</span>
        </div>
      )}

      {/* Connection Error */}
      {error && (
        <div className="px-4 py-2 text-sm text-yellow-400 border-t border-slate-700">
          ⚠️ 連線中斷，嘗試重新連線...
        </div>
      )}
    </div>
  );
}

function getSourceDisplayName(source: string): string {
  const names: Record<string, string> = {
    tmdb: 'TMDb',
    douban: '豆瓣',
    wikipedia: 'Wikipedia',
    manual: '手動選擇',
  };
  return names[source] || source;
}

export default FloatingParseProgressCard;
