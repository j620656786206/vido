// Design ref: ux-design.pen — no current screen frame; 快取管理 tab was never given a frame — rides the designed settings shell (Screen C4-D, 6UCtX)
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

  // Second click within the armed state confirms; anywhere else disarms. Same
  // grammar as CacheTypeCard's per-type clear — the critique's Error-Prevention
  // finding was not that this action lacked ceremony, but that its ceremony
  // CONTRADICTED the pattern ten pixels below it.
  const [confirmingClearOld, setConfirmingClearOld] = useState(false);

  const handleClearOld = async () => {
    if (!confirmingClearOld) {
      setConfirmingClearOld(true);
      return;
    }
    try {
      const result = await clearByAge.mutateAsync(30);
      setLastResult(result);
    } finally {
      setConfirmingClearOld(false);
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20" data-testid="cache-loading">
        <Loader2 className="h-8 w-8 animate-spin text-[var(--text-secondary)]" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="py-10 text-center" data-testid="cache-error">
        <p className="text-[var(--error-text)]">無法載入快取資訊</p>
        <p className="mt-1 text-sm text-[var(--text-muted)]">{error.message}</p>
      </div>
    );
  }

  return (
    <div className="space-y-6" data-testid="cache-management">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Database className="h-5 w-5 text-[var(--text-secondary)]" />
          <div>
            <h2 className="text-lg font-semibold text-[var(--text-primary)]">快取管理</h2>
            <p className="text-sm text-[var(--text-secondary)]">
              總計 {stats ? formatBytes(stats.totalSizeBytes) : '—'}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {confirmingClearOld && !clearByAge.isPending && (
            <button
              onClick={() => setConfirmingClearOld(false)}
              className="rounded-lg px-3 py-2 text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
              data-testid="clear-old-cache-cancel-btn"
            >
              取消
            </button>
          )}
          <button
            onClick={handleClearOld}
            disabled={clearByAge.isPending}
            className={`flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-colors disabled:opacity-50 ${
              confirmingClearOld
                ? 'bg-[var(--error)] text-white hover:bg-[var(--error-pressed)]'
                : 'bg-[var(--bg-tertiary)] text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)]'
            }`}
            data-testid="clear-old-cache-btn"
          >
            {clearByAge.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Clock className="h-4 w-4" />
            )}
            {confirmingClearOld ? '確認清除 30 天前的快取' : '清除 30 天前的快取'}
          </button>
        </div>
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
          className="flex items-center gap-2 rounded-lg bg-[var(--success-tint)] px-4 py-3 text-sm text-[var(--success-text)]"
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
