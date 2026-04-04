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
        <div className="text-sm text-[var(--text-secondary)] truncate" title={filename}>
          檔案：{filename}
        </div>
      )}
      {/* Failed Steps Summary */}
      {failedSteps.length > 0 && (
        <div className="bg-[var(--error)]/10 rounded-lg p-3 space-y-2">
          <h4 className="font-medium text-[var(--error)] flex items-center gap-2">
            <XCircle className="h-4 w-4" />
            失敗原因
          </h4>
          <ul className="space-y-1.5 text-sm" data-testid="failed-steps-list">
            {failedSteps.map((step) => (
              <li key={step.name} className="flex items-start gap-2">
                <XCircle className="h-3.5 w-3.5 text-[var(--error)] mt-0.5 flex-shrink-0" />
                <span className="text-[var(--text-secondary)]">
                  {step.label}
                  {step.error && (
                    <span className="text-[var(--text-secondary)]">：{step.error}</span>
                  )}
                  {!step.error && <span className="text-[var(--text-secondary)]">：無回應</span>}
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
                step.status === 'success' && 'text-[var(--success)]',
                step.status === 'failed' && 'text-[var(--error)]',
                step.status === 'skipped' && 'text-[var(--text-muted)]',
                step.status === 'pending' && 'text-[var(--text-secondary)]',
                step.status === 'in_progress' && 'text-[var(--accent-primary)]'
              )}
            >
              {getSourceDisplayName(step.name)}
              {step.status === 'success' && ' ✓'}
              {step.status === 'failed' && ' ✗'}
            </span>
            {index < searchSteps.length - 1 && (
              <ArrowRight className="h-3 w-3 text-[var(--text-muted)] mx-1.5" />
            )}
          </div>
        ))}
      </div>

      {/* Action Buttons (UX-4: Clear next steps) */}
      <div className="flex flex-col gap-2" data-testid="action-buttons">
        <button
          onClick={onManualSearch}
          className="w-full flex items-center justify-center gap-2 rounded-lg bg-[var(--accent-primary)] hover:bg-[var(--accent-pressed)] px-4 py-2.5 text-sm font-medium text-white transition-colors focus:outline-none focus:ring-2 focus:ring-[var(--accent-primary)]"
          data-testid="manual-search-button"
        >
          <Search className="h-4 w-4" />
          手動搜尋
        </button>

        <button
          onClick={onEditFilename}
          className="w-full flex items-center justify-center gap-2 rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-tertiary)] hover:bg-[var(--bg-tertiary)] px-4 py-2.5 text-sm font-medium text-white transition-colors focus:outline-none focus:ring-2 focus:ring-[var(--text-muted)]"
          data-testid="edit-filename-button"
        >
          <Edit className="h-4 w-4" />
          編輯檔名後重試
        </button>

        <button
          onClick={onSkip}
          className="w-full flex items-center justify-center gap-2 rounded-lg px-4 py-2 text-sm text-[var(--text-secondary)] hover:text-[var(--text-secondary)] transition-colors focus:outline-none focus:ring-2 focus:ring-[var(--text-muted)]"
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
    <div
      className={cn('text-xs text-[var(--error)]', className)}
      data-testid="compact-error-summary"
    >
      <span>{failedSteps.length} 個來源失敗</span>
      {failedSteps.length > 0 && failedSteps[0].error && (
        <span className="text-[var(--text-secondary)]">：{failedSteps[0].error}</span>
      )}
    </div>
  );
}

export default ErrorDetailsPanel;
