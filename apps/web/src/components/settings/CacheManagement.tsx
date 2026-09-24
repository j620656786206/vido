// Design ref: ux-design.pen Screen C11-D (TrU8k) · C11-M (aYEWP)；二次點擊確認態見 C18-D (dfwSb)
import { useState } from 'react';
import { Database, Trash2, Loader2, Clock3, TriangleAlert } from 'lucide-react';
import { useCacheStats, useClearCacheByType, useClearCacheByAge } from '../../hooks/useCacheStats';
import { CacheTypeCard, formatBytes } from './CacheTypeCard';
import type { CleanupResult } from '../../services/cacheService';
import { SettingsErrorState } from './SettingsErrorState';

export function CacheManagement() {
  const { data: stats, isFetched, isFetching, error, refetch } = useCacheStats();
  const clearByType = useClearCacheByType();
  const clearByAge = useClearCacheByAge();
  const [lastResult, setLastResult] = useState<CleanupResult | null>(null);

  const handleClearByType = async (cacheType: string) => {
    const result = await clearByType.mutateAsync(cacheType);
    setLastResult(result);
  };

  // Second click confirms; 取消 (or completion) disarms. Same
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

  // Keyed on data (dsr-3c lesson): a failed read with nothing cached drops back
  // to pending on every refetch, which would swap the error page for the spinner.
  if (!stats && !error && !isFetched) {
    return (
      <div className="flex items-center justify-center py-20" data-testid="cache-loading">
        <Loader2 className="h-8 w-8 animate-spin text-[var(--text-secondary)]" />
      </div>
    );
  }

  if (!stats) {
    return (
      <SettingsErrorState
        testId="cache-error"
        title="無法載入快取資訊"
        description="與後端的連線中斷了。快取本身不受影響。"
        isRetrying={isFetching}
        onRetry={() => void refetch()}
      />
    );
  }

  return (
    <div className="space-y-4" data-testid="cache-management">
      {/* Header */}
      <div className="flex items-center justify-between">
        {/* Page title lives at the route level (one header contract for all
            settings tabs); this line is the live DATA readout, not a heading. */}
        <div className="flex items-center gap-2 text-sm text-[var(--text-secondary)]">
          <Database className="h-4 w-4" aria-hidden="true" />
          <span>總計 {stats ? formatBytes(stats.totalSizeBytes) : '—'}</span>
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
            // The phone shows「清除 30 天前」(C11-M lOTqc); the name is always the full sentence.
            aria-label={confirmingClearOld ? '確認清除 30 天前的快取' : '清除 30 天前的快取'}
            aria-describedby={
              confirmingClearOld && !clearByAge.isPending ? 'clear-old-cache-warning' : undefined
            }
            className={`flex min-h-10 items-center gap-2 rounded-[var(--radius-md)] px-4 text-sm font-medium transition-colors disabled:opacity-50 max-sm:min-h-11 ${
              confirmingClearOld
                ? 'bg-[var(--error)] text-[var(--text-on-scrim)] hover:bg-[var(--error-pressed)]'
                : 'bg-[var(--bg-tertiary)] text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)]'
            }`}
            data-testid="clear-old-cache-btn"
          >
            {clearByAge.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
            ) : (
              <Clock3 className="h-4 w-4" aria-hidden="true" />
            )}
            <span className="sm:hidden" aria-hidden="true">
              {confirmingClearOld ? '確認清除' : '清除 30 天前'}
            </span>
            <span className="hidden sm:inline" aria-hidden="true">
              {confirmingClearOld ? '確認清除 30 天前的快取' : '清除 30 天前的快取'}
            </span>
          </button>
        </div>
      </div>

      {/* C18-D N0tZx: the armed state says what the second press will do.
          Checked against the backend (dsr-3e Task 1): ClearCacheByAge deletes
          rows older than the cutoff from the four metadata/AI tables — never
          media or subtitle files, and (since bugfix-custom-posters-served-and-
          not-cache) never the posters users uploaded under data/posters. role="status": an explanation of a two-step confirm, not an error. */}
      {/* The live region is ALWAYS mounted and only its content changes: a
          region that appears already filled is usually not announced (dsr-3e
          CR A2), and focus stays on the same button, so a describedby that
          appears later is not re-read either. */}
      <div
        id="clear-old-cache-warning"
        role="status"
        // Empty most of the time: take no space in the space-y stack then.
        className="empty:mb-0"
        data-testid="clear-old-cache-warning-region"
      >
        {confirmingClearOld && !clearByAge.isPending && (
          <p
            className="flex items-start gap-2 rounded-[var(--radius-md)] bg-[var(--error-tint)] px-4 py-3 text-xs text-[var(--error-text)]"
            data-testid="clear-old-cache-warning"
          >
            <TriangleAlert className="mt-px size-4 shrink-0" aria-hidden="true" />
            {/* poster-upload-b AC #5 (Sally × John): named once uploading exists. */}
            再按一次才會真的清除。這會刪掉 30
            天前的所有快取，之後第一次瀏覽會比較慢，但不會影響影片、字幕與你上傳的海報。
          </p>
        )}
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
          className="flex items-center gap-2 rounded-lg bg-[var(--bg-tertiary)] px-4 py-3 text-sm text-[var(--text-secondary)]"
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
