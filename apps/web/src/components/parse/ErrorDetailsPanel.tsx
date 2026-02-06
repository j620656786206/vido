/**
 * ErrorDetailsPanel Component (Story 3.10 - Task 7)
 * Displays failure reasons and action buttons for failed parses
 * AC3: Failure Reason Display (UX-4)
 */

import { XCircle, Search, Edit, SkipForward, ArrowRight } from 'lucide-react';
import { cn } from '../../lib/utils';
import type { ParseStep } from './types';
import { getSourceDisplayName } from './types';

export interface ErrorDetailsPanelProps {
  /** List of parse steps (to extract failures from) */
  steps: ParseStep[];
  /** Filename being parsed */
  filename: string;
  /** Called when user clicks "Manual Search" */
  onManualSearch?: () => void;
  /** Called when user clicks "Edit Filename" */
  onEditFilename?: () => void;
  /** Called when user clicks "Skip" */
  onSkip?: () => void;
  /** Additional CSS classes */
  className?: string;
}

/**
 * Panel showing failure details and action buttons
 * Implements AC3: Failure Reason Display (UX-4)
 */
export function ErrorDetailsPanel({
  steps,
  filename,
  onManualSearch,
  onEditFilename,
  onSkip,
  className,
}: ErrorDetailsPanelProps) {
  const failedSteps = steps.filter((step) => step.status === 'failed');
  const searchSteps = steps.filter((step) =>
    ['tmdb_search', 'douban_search', 'wikipedia_search', 'ai_retry'].includes(step.name)
  );

  return (
    <div className={cn('space-y-4', className)} data-testid="error-details-panel">
      {/* Filename display */}
      {filename && (
        <div className="text-sm text-slate-400 truncate" title={filename}>
          檔案：{filename}
        </div>
      )}
      {/* Failed Steps Summary */}
      {failedSteps.length > 0 && (
        <div className="bg-red-500/10 rounded-lg p-3 space-y-2">
          <h4 className="font-medium text-red-400 flex items-center gap-2">
            <XCircle className="h-4 w-4" />
            失敗原因
          </h4>
          <ul className="space-y-1.5 text-sm" data-testid="failed-steps-list">
            {failedSteps.map((step) => (
              <li key={step.name} className="flex items-start gap-2">
                <XCircle className="h-3.5 w-3.5 text-red-400 mt-0.5 flex-shrink-0" />
                <span className="text-slate-300">
                  {step.label}
                  {step.error && <span className="text-slate-400">：{step.error}</span>}
                  {!step.error && <span className="text-slate-400">：無回應</span>}
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* Source Chain Visualization */}
      <div className="flex items-center gap-2 text-sm flex-wrap" data-testid="source-chain">
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
              {getSourceDisplayName(step.name)}
              {step.status === 'success' && ' ✓'}
              {step.status === 'failed' && ' ✗'}
            </span>
            {index < searchSteps.length - 1 && (
              <ArrowRight className="h-3 w-3 text-slate-600 mx-1.5" />
            )}
          </div>
        ))}
      </div>

      {/* Action Buttons (UX-4: Clear next steps) */}
      <div className="flex flex-col gap-2" data-testid="action-buttons">
        <button
          onClick={onManualSearch}
          className="w-full flex items-center justify-center gap-2 rounded-lg bg-blue-600 hover:bg-blue-700 px-4 py-2.5 text-sm font-medium text-white transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500"
          data-testid="manual-search-button"
        >
          <Search className="h-4 w-4" />
          手動搜尋
        </button>

        <button
          onClick={onEditFilename}
          className="w-full flex items-center justify-center gap-2 rounded-lg border border-slate-600 bg-slate-700 hover:bg-slate-600 px-4 py-2.5 text-sm font-medium text-white transition-colors focus:outline-none focus:ring-2 focus:ring-slate-500"
          data-testid="edit-filename-button"
        >
          <Edit className="h-4 w-4" />
          編輯檔名後重試
        </button>

        <button
          onClick={onSkip}
          className="w-full flex items-center justify-center gap-2 rounded-lg px-4 py-2 text-sm text-slate-400 hover:text-slate-300 transition-colors focus:outline-none focus:ring-2 focus:ring-slate-500"
          data-testid="skip-button"
        >
          <SkipForward className="h-4 w-4" />
          跳過此檔案
        </button>
      </div>
    </div>
  );
}

/**
 * Compact error summary for card view
 */
export function CompactErrorSummary({
  steps,
  className,
}: {
  steps: ParseStep[];
  className?: string;
}) {
  const failedSteps = steps.filter((step) => step.status === 'failed');

  if (failedSteps.length === 0) {
    return null;
  }

  return (
    <div className={cn('text-xs text-red-400', className)} data-testid="compact-error-summary">
      <span>{failedSteps.length} 個來源失敗</span>
      {failedSteps.length > 0 && failedSteps[0].error && (
        <span className="text-slate-400">：{failedSteps[0].error}</span>
      )}
    </div>
  );
}

export default ErrorDetailsPanel;
