import { useState } from 'react';
import { Database, Trash2, Loader2, Clock } from 'lucide-react';
import { useCacheStats, useClearCacheByType, useClearCacheByAge } from '../../hooks/useCacheStats';
import { CacheTypeCard, formatBytes } from './CacheTypeCard';
import type { CleanupResult } from '../../services/cacheService';

export function CacheManagement() {
  const { data: stats, isLoading, error } = useCacheStats();
  const clearByType = useClearCacheByType();
  const clearByAge = useClearCacheByAge();
  const [lastResult, setLastResult] = useState<CleanupResult | null>(null);

  const handleClearByType = async (cacheType: string) => {
    const result = await clearByType.mutateAsync(cacheType);
    setLastResult(result);
  };

  const handleClearOld = async () => {
    const result = await clearByAge.mutateAsync(30);
    setLastResult(result);
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20" data-testid="cache-loading">
        <Loader2 className="h-8 w-8 animate-spin text-slate-400" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="py-10 text-center" data-testid="cache-error">
        <p className="text-red-400">無法載入快取資訊</p>
        <p className="mt-1 text-sm text-slate-500">{error.message}</p>
      </div>
    );
  }

  return (
    <div className="space-y-6" data-testid="cache-management">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Database className="h-5 w-5 text-slate-400" />
          <div>
            <h2 className="text-lg font-semibold text-slate-200">快取管理</h2>
            <p className="text-sm text-slate-400">
              總計 {stats ? formatBytes(stats.totalSizeBytes) : '—'}
            </p>
          </div>
        </div>

        <button
          onClick={handleClearOld}
          disabled={clearByAge.isPending}
          className="flex items-center gap-2 rounded-lg bg-slate-700 px-4 py-2 text-sm font-medium text-slate-200 transition-colors hover:bg-slate-600 disabled:opacity-50"
          data-testid="clear-old-cache-btn"
        >
          {clearByAge.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Clock className="h-4 w-4" />
          )}
          清除 30 天前的快取
        </button>
      </div>

      {/* Cache type cards */}
      <div className="space-y-3" data-testid="cache-types-list">
        {stats?.cacheTypes.map((ct) => (
          <CacheTypeCard key={ct.type} cacheType={ct} onClear={handleClearByType} />
        ))}
      </div>

      {/* Last result feedback */}
      {lastResult && (
        <div
          className="flex items-center gap-2 rounded-lg border border-green-700 bg-green-900/30 px-4 py-3 text-sm text-green-300"
          data-testid="cache-result"
        >
          <Trash2 className="h-4 w-4 flex-shrink-0" />
          <span>
            已清除 {lastResult.entriesRemoved.toLocaleString()} 筆快取
            {lastResult.bytesReclaimed > 0 && `，釋放 ${formatBytes(lastResult.bytesReclaimed)}`}
          </span>
        </div>
      )}
    </div>
  );
}
