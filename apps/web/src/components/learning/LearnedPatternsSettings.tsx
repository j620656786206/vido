/**
 * LearnedPatternsSettings Component (Story 3.9 - AC3)
 * Displays and manages learned filename patterns in settings
 * Shows "已記住 N 個自訂規則" count per spec
 */

import { useState } from 'react';
import { Trash2, Lightbulb, ChevronRight, Loader2 } from 'lucide-react';
import { cn } from '../../lib/utils';
import { useLearningPatterns, useDeletePattern } from '../../hooks/useLearning';

export interface LearnedPatternsSettingsProps {
  onError?: (error: Error) => void;
}

export function LearnedPatternsSettings({ onError }: LearnedPatternsSettingsProps) {
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const { data: response, isLoading, error: fetchError } = useLearningPatterns();

  const deletePatternMutation = useDeletePattern();

  // Report fetch error
  if (fetchError) {
    onError?.(fetchError);
  }

  const patterns = response?.patterns ?? [];
  const stats = response?.stats ?? null;

  const handleDelete = async (id: string) => {
    setDeletingId(id);
    try {
      await deletePatternMutation.mutateAsync(id);
    } catch (error) {
      onError?.(error as Error);
    } finally {
      setDeletingId(null);
    }
  };

  const toggleExpand = (id: string) => {
    setExpandedId(expandedId === id ? null : id);
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8" data-testid="patterns-loading">
        <Loader2 className="h-6 w-6 animate-spin text-[var(--text-secondary)]" />
      </div>
    );
  }

  return (
    <div className="space-y-4" data-testid="learned-patterns-settings">
      {/* Header with count (AC3) */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Lightbulb className="h-5 w-5 text-amber-400" />
          <h3 className="text-lg font-medium text-white">自訂規則</h3>
        </div>
        <span className="text-sm text-[var(--text-secondary)]" data-testid="patterns-count">
          已記住 {stats?.totalPatterns ?? patterns.length} 個自訂規則
        </span>
      </div>

      {/* Stats summary */}
      {stats && stats.totalApplied > 0 && (
        <div
          className="rounded-lg bg-[var(--bg-secondary)]/50 px-4 py-3 text-sm text-[var(--text-secondary)]"
          data-testid="patterns-stats"
        >
          <span>共套用 {stats.totalApplied} 次</span>
          {stats.mostUsedPattern && (
            <span className="ml-2">
              · 最常使用：<span className="text-amber-400">{stats.mostUsedPattern}</span> (
              {stats.mostUsedCount} 次)
            </span>
          )}
        </div>
      )}

      {/* Patterns list */}
      {patterns.length === 0 ? (
        <div
          className="rounded-lg border border-dashed border-[var(--border-subtle)] py-8 text-center"
          data-testid="empty-patterns"
        >
          <Lightbulb className="mx-auto h-8 w-8 text-[var(--text-muted)]" />
          <p className="mt-2 text-[var(--text-muted)]">尚無自訂規則</p>
          <p className="mt-1 text-sm text-[var(--text-muted)]">
            在手動配對檔案後，可選擇學習規則以便未來自動套用
          </p>
        </div>
      ) : (
        <div className="space-y-2" data-testid="patterns-list">
          {patterns.map((pattern) => (
            <div
              key={pattern.id}
              className="rounded-lg bg-[var(--bg-secondary)]/50 overflow-hidden"
              data-testid={`pattern-item-${pattern.id}`}
            >
              {/* Pattern header */}
              <button
                onClick={() => toggleExpand(pattern.id)}
                className={cn(
                  'flex w-full items-center justify-between px-4 py-3',
                  'hover:bg-[var(--bg-tertiary)]/50 transition-colors text-left'
                )}
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-sm text-white truncate">{pattern.pattern}</span>
                    <span
                      className={cn(
                        'px-1.5 py-0.5 rounded text-xs',
                        pattern.patternType === 'fansub'
                          ? 'bg-[var(--accent-primary)]/20 text-[var(--accent-primary)]'
                          : pattern.patternType === 'standard'
                            ? 'bg-[var(--success)]/20 text-[var(--success)]'
                            : 'bg-[var(--text-muted)]/20 text-[var(--text-secondary)]'
                      )}
                    >
                      {pattern.patternType}
                    </span>
                  </div>
                  <div className="mt-1 text-xs text-[var(--text-muted)]">
                    {pattern.metadataType === 'series' ? '影集' : '電影'} · 套用 {pattern.useCount}{' '}
                    次
                  </div>
                </div>
                <ChevronRight
                  className={cn(
                    'h-5 w-5 text-[var(--text-muted)] transition-transform',
                    expandedId === pattern.id && 'rotate-90'
                  )}
                />
              </button>

              {/* Expanded details */}
              {expandedId === pattern.id && (
                <div
                  className="border-t border-[var(--border-subtle)] px-4 py-3 space-y-2"
                  data-testid={`pattern-details-${pattern.id}`}
                >
                  {pattern.fansubGroup && (
                    <div className="flex gap-2 text-sm">
                      <span className="text-[var(--text-muted)]">字幕組：</span>
                      <span className="text-[var(--accent-primary)]">{pattern.fansubGroup}</span>
                    </div>
                  )}
                  {pattern.titlePattern && (
                    <div className="flex gap-2 text-sm">
                      <span className="text-[var(--text-muted)]">標題：</span>
                      <span className="text-[var(--success)]">{pattern.titlePattern}</span>
                    </div>
                  )}
                  {pattern.tmdbId && (
                    <div className="flex gap-2 text-sm">
                      <span className="text-[var(--text-muted)]">TMDb ID：</span>
                      <span className="text-[var(--text-secondary)]">{pattern.tmdbId}</span>
                    </div>
                  )}
                  <div className="flex gap-2 text-sm">
                    <span className="text-[var(--text-muted)]">建立時間：</span>
                    <span className="text-[var(--text-secondary)]">
                      {new Date(pattern.createdAt).toLocaleDateString('zh-TW')}
                    </span>
                  </div>

                  {/* Delete button */}
                  <button
                    onClick={() => handleDelete(pattern.id)}
                    disabled={deletingId === pattern.id}
                    className={cn(
                      'mt-2 flex items-center gap-1.5 px-3 py-1.5 rounded',
                      'bg-[var(--error)]/20 text-[var(--error)] text-sm',
                      'hover:bg-[var(--error)]/30 transition-colors',
                      'disabled:opacity-50 disabled:cursor-not-allowed'
                    )}
                    data-testid={`delete-pattern-${pattern.id}`}
                  >
                    {deletingId === pattern.id ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <Trash2 className="h-4 w-4" />
                    )}
                    刪除此規則
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default LearnedPatternsSettings;
