/**
 * FallbackStatusDisplay Component (Story 3.7 - UX-4)
 * Shows the fallback chain status when automatic parsing fails
 */

import { Check, X, SkipForward, ArrowRight } from 'lucide-react';
import { cn } from '../../lib/utils';
import type { FallbackStatus } from './ManualSearchDialog';

export interface FallbackStatusDisplayProps {
  status: FallbackStatus;
  className?: string;
}

// Source display names
const SOURCE_NAMES: Record<string, string> = {
  tmdb: 'TMDb',
  douban: '豆瓣',
  wikipedia: 'Wikipedia',
};

export function FallbackStatusDisplay({
  status,
  className,
}: FallbackStatusDisplayProps) {
  if (!status.attempts || status.attempts.length === 0) {
    return null;
  }

  const allFailed = status.attempts.every((a) => !a.success);

  return (
    <div className={cn('bg-slate-800/50 px-6 py-3 border-b border-slate-700', className)}>
      <h4 className="text-sm font-medium text-slate-300 mb-2">
        已嘗試的來源：
      </h4>
      <div className="flex items-center gap-2 flex-wrap">
        {status.attempts.map((attempt, index) => (
          <div key={attempt.source} className="flex items-center gap-1">
            {/* Source status */}
            <span
              className={cn(
                'flex items-center gap-1 px-2 py-1 rounded text-sm',
                attempt.success
                  ? 'bg-green-500/20 text-green-400'
                  : attempt.skipped
                  ? 'bg-slate-500/20 text-slate-400'
                  : 'bg-red-500/20 text-red-400'
              )}
            >
              {SOURCE_NAMES[attempt.source] || attempt.source}
              {attempt.success ? (
                <Check className="h-4 w-4" />
              ) : attempt.skipped ? (
                <SkipForward className="h-4 w-4" />
              ) : (
                <X className="h-4 w-4" />
              )}
            </span>

            {/* Arrow between sources */}
            {index < status.attempts.length - 1 && (
              <ArrowRight className="h-4 w-4 text-slate-500" />
            )}
          </div>
        ))}
      </div>

      {/* Guidance message (UX-4) */}
      {allFailed && (
        <p className="text-sm text-slate-400 mt-2">
          所有自動來源都無法找到匹配，請手動搜尋。
        </p>
      )}

      {status.cancelled && (
        <p className="text-sm text-yellow-400 mt-2">
          搜尋被取消，請重新嘗試。
        </p>
      )}
    </div>
  );
}

export default FallbackStatusDisplay;
