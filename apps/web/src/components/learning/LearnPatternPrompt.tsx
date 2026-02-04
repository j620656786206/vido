/**
 * LearnPatternPrompt Component (Story 3.9 - AC1, AC2)
 * Prompts user to learn a filename pattern after manual metadata correction
 * Shows UX-5 feedback: "✓ 已套用你之前的設定" when pattern applied
 */

import { useState } from 'react';
import { Lightbulb, X, Check, Loader2 } from 'lucide-react';
import { cn } from '../../lib/utils';
import { learningService, type LearnedPattern } from '../../services/learning';

export interface ExtractedPattern {
  fansubGroup?: string;
  titlePattern: string;
  patternType: 'fansub' | 'standard' | 'exact';
}

export interface LearnPatternPromptProps {
  filename: string;
  extractedPattern: ExtractedPattern;
  metadataId: string;
  metadataType: 'movie' | 'series';
  tmdbId?: number;
  onConfirm: (pattern: LearnedPattern) => void;
  onSkip: () => void;
  onError?: (error: Error) => void;
}

export function LearnPatternPrompt({
  filename,
  extractedPattern,
  metadataId,
  metadataType,
  tmdbId,
  onConfirm,
  onSkip,
  onError,
}: LearnPatternPromptProps) {
  const [isLoading, setIsLoading] = useState(false);

  const handleConfirm = async () => {
    setIsLoading(true);
    try {
      const pattern = await learningService.learnPattern({
        filename,
        metadataId,
        metadataType,
        tmdbId,
      });
      onConfirm(pattern);
    } catch (error) {
      onError?.(error as Error);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div
      className={cn(
        'rounded-lg border border-amber-500/30 bg-amber-500/10 p-4',
        'backdrop-blur-sm'
      )}
      role="alert"
      data-testid="learn-pattern-prompt"
    >
      <div className="flex items-start gap-3">
        <div className="flex-shrink-0 rounded-full bg-amber-500/20 p-2">
          <Lightbulb className="h-5 w-5 text-amber-400" />
        </div>
        <div className="flex-1 min-w-0">
          <h4 className="text-sm font-medium text-amber-200">學習此規則？</h4>
          <p className="mt-1 text-sm text-slate-300">
            系統偵測到以下規則，是否記住以便未來自動套用？
          </p>

          {/* Pattern Preview */}
          <div
            className="mt-3 rounded bg-slate-800/50 px-3 py-2 font-mono text-sm"
            data-testid="pattern-preview"
          >
            {extractedPattern.fansubGroup && (
              <span className="text-blue-400">[{extractedPattern.fansubGroup}]</span>
            )}{' '}
            <span className="text-green-400">{extractedPattern.titlePattern}</span>
          </div>

          {/* Action Buttons */}
          <div className="mt-4 flex items-center gap-2">
            <button
              onClick={handleConfirm}
              disabled={isLoading}
              className={cn(
                'inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5',
                'bg-amber-600 text-white text-sm font-medium',
                'hover:bg-amber-500 transition-colors',
                'disabled:opacity-50 disabled:cursor-not-allowed'
              )}
              data-testid="confirm-learn-button"
            >
              {isLoading ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Check className="h-4 w-4" />
              )}
              記住此規則
            </button>
            <button
              onClick={onSkip}
              disabled={isLoading}
              className={cn(
                'inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5',
                'bg-slate-700 text-slate-300 text-sm',
                'hover:bg-slate-600 transition-colors',
                'disabled:opacity-50 disabled:cursor-not-allowed'
              )}
              data-testid="skip-learn-button"
            >
              <X className="h-4 w-4" />
              這次不用
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

/**
 * PatternAppliedToast Component (UX-5)
 * Shows feedback when a learned pattern is automatically applied
 */
export interface PatternAppliedToastProps {
  patternTitle: string;
  onClose?: () => void;
}

export function PatternAppliedToast({ patternTitle, onClose }: PatternAppliedToastProps) {
  return (
    <div
      className={cn(
        'fixed bottom-6 left-1/2 -translate-x-1/2 z-50',
        'flex items-center gap-2 px-4 py-2 rounded-lg',
        'bg-green-600 text-white shadow-lg',
        'animate-in fade-in slide-in-from-bottom-4'
      )}
      role="status"
      aria-live="polite"
      data-testid="pattern-applied-toast"
    >
      <Check className="h-5 w-5 text-green-200" />
      <span>✓ 已套用你之前的設定</span>
      <span className="text-green-200 text-sm">（{patternTitle}）</span>
      {onClose && (
        <button
          onClick={onClose}
          className="ml-2 p-0.5 hover:bg-green-500 rounded"
          aria-label="關閉"
        >
          <X className="h-4 w-4" />
        </button>
      )}
    </div>
  );
}

/**
 * LearnSuccessToast Component
 * Shows feedback when a pattern is successfully learned
 */
export interface LearnSuccessToastProps {
  pattern: string;
  onClose?: () => void;
}

export function LearnSuccessToast({ pattern, onClose }: LearnSuccessToastProps) {
  return (
    <div
      className={cn(
        'fixed bottom-6 left-1/2 -translate-x-1/2 z-50',
        'flex items-center gap-2 px-4 py-2 rounded-lg',
        'bg-blue-600 text-white shadow-lg',
        'animate-in fade-in slide-in-from-bottom-4'
      )}
      role="status"
      aria-live="polite"
      data-testid="learn-success-toast"
    >
      <Lightbulb className="h-5 w-5 text-blue-200" />
      <span>已學習此規則</span>
      <span className="text-blue-200 text-sm">（{pattern}）</span>
      {onClose && (
        <button
          onClick={onClose}
          className="ml-2 p-0.5 hover:bg-blue-500 rounded"
          aria-label="關閉"
        >
          <X className="h-4 w-4" />
        </button>
      )}
    </div>
  );
}

export default LearnPatternPrompt;
