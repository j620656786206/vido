/**
 * LayeredProgressIndicator Component (Story 3.10 - Task 5)
 * Displays parse steps with status icons in a layered format
 * AC2: Step Progress Indicators (UX-3)
 */

import {
  CheckCircle,
  Loader2,
  XCircle,
  MinusCircle,
  Circle,
} from 'lucide-react';
import { cn } from '../../lib/utils';
import type { ParseStep, StepStatus } from './types';

export interface LayeredProgressIndicatorProps {
  /** List of parse steps with their status */
  steps: ParseStep[];
  /** Index of the currently active step */
  currentStep: number;
  /** Show compact view (icons only) */
  compact?: boolean;
  /** Additional CSS classes */
  className?: string;
}

const STEP_ICON_CONFIG: Record<
  StepStatus,
  { icon: typeof Circle; color: string; animate?: boolean }
> = {
  pending: { icon: Circle, color: 'text-slate-500' },
  in_progress: { icon: Loader2, color: 'text-blue-500', animate: true },
  success: { icon: CheckCircle, color: 'text-green-500' },
  failed: { icon: XCircle, color: 'text-red-500' },
  skipped: { icon: MinusCircle, color: 'text-slate-400' },
};

/**
 * Displays a single step in the progress indicator
 */
function ProgressStep({
  step,
  isActive,
  compact,
}: {
  step: ParseStep;
  isActive: boolean;
  compact?: boolean;
}) {
  const config = STEP_ICON_CONFIG[step.status];
  const Icon = config.icon;

  return (
    <div
      className={cn(
        'flex items-center gap-2',
        compact ? 'py-0.5' : 'py-1.5',
        isActive && 'font-medium'
      )}
      data-testid={`progress-step-${step.name}`}
      data-status={step.status}
    >
      <Icon
        className={cn(
          compact ? 'h-3.5 w-3.5' : 'h-4 w-4',
          config.color,
          config.animate && 'animate-spin'
        )}
        aria-hidden="true"
      />

      {!compact && (
        <>
          <span
            className={cn(
              'flex-1',
              step.status === 'pending' && 'text-slate-400',
              step.status === 'in_progress' && 'text-blue-400',
              step.status === 'success' && 'text-slate-200',
              step.status === 'failed' && 'text-red-400',
              step.status === 'skipped' && 'text-slate-500'
            )}
          >
            {step.label}
          </span>

          {step.status === 'in_progress' && (
            <span className="text-sm text-blue-400 animate-pulse">
              搜尋中...
            </span>
          )}

          {step.status === 'failed' && step.error && (
            <span className="text-sm text-red-400 truncate max-w-[150px]" title={step.error}>
              {step.error}
            </span>
          )}
        </>
      )}
    </div>
  );
}

/**
 * Layered progress indicator showing all parse steps
 * Implements AC2: Step Progress Indicators
 */
export function LayeredProgressIndicator({
  steps,
  currentStep,
  compact = false,
  className,
}: LayeredProgressIndicatorProps) {
  return (
    <div
      className={cn('space-y-0.5', className)}
      role="list"
      aria-label="解析步驟進度"
      data-testid="layered-progress-indicator"
    >
      {steps.map((step, index) => (
        <ProgressStep
          key={step.name}
          step={step}
          isActive={index === currentStep}
          compact={compact}
        />
      ))}
    </div>
  );
}

/**
 * Compact inline version showing step icons in a row
 */
export function InlineProgressIndicator({
  steps,
  className,
}: {
  steps: ParseStep[];
  className?: string;
}) {
  return (
    <div
      className={cn('flex items-center gap-1', className)}
      role="list"
      aria-label="解析步驟進度"
      data-testid="inline-progress-indicator"
    >
      {steps.map((step) => {
        const config = STEP_ICON_CONFIG[step.status];
        const Icon = config.icon;

        return (
          <div
            key={step.name}
            title={`${step.label}: ${getStatusLabel(step.status)}`}
            className="flex items-center"
          >
            <Icon
              className={cn(
                'h-3 w-3',
                config.color,
                config.animate && 'animate-spin'
              )}
              aria-hidden="true"
            />
          </div>
        );
      })}
    </div>
  );
}

/**
 * Source chain visualization (TMDb → Douban → Wikipedia)
 */
export function SourceChainIndicator({
  steps,
  className,
}: {
  steps: ParseStep[];
  className?: string;
}) {
  // Filter to only show search steps
  const searchSteps = steps.filter((step) =>
    ['tmdb_search', 'douban_search', 'wikipedia_search', 'ai_retry'].includes(step.name)
  );

  return (
    <div
      className={cn('flex items-center gap-2 text-sm', className)}
      data-testid="source-chain-indicator"
    >
      {searchSteps.map((step, index) => (
        <div key={step.name} className="flex items-center">
          <span
            className={cn(
              step.status === 'success' && 'text-green-500',
              step.status === 'failed' && 'text-red-500',
              step.status === 'skipped' && 'text-slate-500',
              step.status === 'pending' && 'text-slate-400',
              step.status === 'in_progress' && 'text-blue-500'
            )}
          >
            {getSourceName(step.name)}
            {step.status === 'success' && ' ✓'}
            {step.status === 'failed' && ' ✗'}
          </span>
          {index < searchSteps.length - 1 && (
            <span className="mx-1.5 text-slate-600">→</span>
          )}
        </div>
      ))}
    </div>
  );
}

function getStatusLabel(status: StepStatus): string {
  const labels: Record<StepStatus, string> = {
    pending: '等待中',
    in_progress: '進行中',
    success: '成功',
    failed: '失敗',
    skipped: '跳過',
  };
  return labels[status];
}

function getSourceName(stepName: string): string {
  const names: Record<string, string> = {
    tmdb_search: 'TMDb',
    douban_search: '豆瓣',
    wikipedia_search: 'Wikipedia',
    ai_retry: 'AI',
  };
  return names[stepName] || stepName;
}

export default LayeredProgressIndicator;
