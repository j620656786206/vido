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
        <Loader2 className="h-6 w-6 animate-spin text-slate-400" />
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
        <span className="text-sm text-slate-400" data-testid="patterns-count">
          已記住 {stats?.totalPatterns ?? patterns.length} 個自訂規則
        </span>
      </div>

      {/* Stats summary */}
      {stats && stats.totalApplied > 0 && (
        <div
          className="rounded-lg bg-slate-800/50 px-4 py-3 text-sm text-slate-300"
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
          className="rounded-lg border border-dashed border-slate-700 py-8 text-center"
          data-testid="empty-patterns"
        >
          <Lightbulb className="mx-auto h-8 w-8 text-slate-600" />
          <p className="mt-2 text-slate-500">尚無自訂規則</p>
          <p className="mt-1 text-sm text-slate-600">
            在手動配對檔案後，可選擇學習規則以便未來自動套用
          </p>
        </div>
      ) : (
        <div className="space-y-2" data-testid="patterns-list">
          {patterns.map((pattern) => (
            <div
              key={pattern.id}
              className="rounded-lg bg-slate-800/50 overflow-hidden"
              data-testid={`pattern-item-${pattern.id}`}
            >
              {/* Pattern header */}
              <button
                onClick={() => toggleExpand(pattern.id)}
                className={cn(
                  'flex w-full items-center justify-between px-4 py-3',
                  'hover:bg-slate-700/50 transition-colors text-left'
                )}
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-sm text-white truncate">{pattern.pattern}</span>
                    <span
                      className={cn(
                        'px-1.5 py-0.5 rounded text-xs',
                        pattern.patternType === 'fansub'
                          ? 'bg-blue-500/20 text-blue-400'
                          : pattern.patternType === 'standard'
                            ? 'bg-green-500/20 text-green-400'
                            : 'bg-slate-500/20 text-slate-400'
                      )}
                    >
                      {pattern.patternType}
                    </span>
                  </div>
                  <div className="mt-1 text-xs text-slate-500">
                    {pattern.metadataType === 'series' ? '影集' : '電影'} · 套用 {pattern.useCount}{' '}
                    次
                  </div>
                </div>
                <ChevronRight
                  className={cn(
                    'h-5 w-5 text-slate-500 transition-transform',
                    expandedId === pattern.id && 'rotate-90'
                  )}
                />
              </button>

              {/* Expanded details */}
              {expandedId === pattern.id && (
                <div
                  className="border-t border-slate-700 px-4 py-3 space-y-2"
                  data-testid={`pattern-details-${pattern.id}`}
                >
                  {pattern.fansubGroup && (
                    <div className="flex gap-2 text-sm">
                      <span className="text-slate-500">字幕組：</span>
                      <span className="text-blue-400">{pattern.fansubGroup}</span>
                    </div>
                  )}
                  {pattern.titlePattern && (
                    <div className="flex gap-2 text-sm">
                      <span className="text-slate-500">標題：</span>
                      <span className="text-green-400">{pattern.titlePattern}</span>
                    </div>
                  )}
                  {pattern.tmdbId && (
                    <div className="flex gap-2 text-sm">
                      <span className="text-slate-500">TMDb ID：</span>
                      <span className="text-slate-300">{pattern.tmdbId}</span>
                    </div>
                  )}
                  <div className="flex gap-2 text-sm">
                    <span className="text-slate-500">建立時間：</span>
                    <span className="text-slate-300">
                      {new Date(pattern.createdAt).toLocaleDateString('zh-TW')}
                    </span>
                  </div>

                  {/* Delete button */}
                  <button
                    onClick={() => handleDelete(pattern.id)}
                    disabled={deletingId === pattern.id}
                    className={cn(
                      'mt-2 flex items-center gap-1.5 px-3 py-1.5 rounded',
                      'bg-red-500/20 text-red-400 text-sm',
                      'hover:bg-red-500/30 transition-colors',
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
